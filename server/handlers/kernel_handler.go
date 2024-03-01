package handlers

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	cmap "github.com/orcaman/concurrent-map/v2"
	gateway "github.com/scusemua/workload-driver-react/m/v2/server/api/proto"
	"github.com/scusemua/workload-driver-react/m/v2/server/config"
	"github.com/scusemua/workload-driver-react/m/v2/server/domain"
	"go.uber.org/zap"
)

type KernelHttpHandler struct {
	*BaseGRPCHandler

	spoofKernels   bool                                                           // If we're creating and returning fake/mocked kernels.
	spoofedKernels *cmap.ConcurrentMap[string, *gateway.DistributedJupyterKernel] // Latest spoofedKernels.
}

func NewKernelHttpHandler(opts *config.Configuration) domain.BackendHttpGetHandler {
	handler := &KernelHttpHandler{
		spoofKernels:    opts.SpoofKernels,
		BaseGRPCHandler: newBaseGRPCHandler(opts, !opts.SpoofKernels),
	}
	handler.BackendHttpGetHandler = handler

	if handler.spoofKernels {
		spoofedKernels := cmap.New[*gateway.DistributedJupyterKernel]()
		handler.spoofedKernels = &spoofedKernels
	}

	handler.logger.Info(fmt.Sprintf("Creating server-side KernelHttpHandler.\nOptions: %s", opts))

	return handler
}

func (h *KernelHttpHandler) HandleRequest(c *gin.Context) {
	var kernels []*gateway.DistributedJupyterKernel

	// If we're spoofing the cluster, then just return some made up kernels for testing/debugging purposes.
	if h.spoofKernels {
		h.logger.Info("Spoofing Jupyter kernels now.")
		kernels = h.doSpoofKernels()
	} else {
		h.logger.Info("Retrieving Jupyter kernels from the Jupyter Server now.", zap.String("jupyter-server-ip", h.gatewayAddress))
		kernels = h.getKernelSpecsFromClusterGateway()

		if kernels == nil {
			// Write error back to front-end.
			h.logger.Error("Failed to retrieve list of kernels from Jupyter Server.")
			h.WriteError(c, "Failed to retrieve list of kernels from Jupyter Server.")
			return
		}
	}

	h.logger.Info(fmt.Sprintf("Sending %d kernel(s) back to client now.", len(kernels)))
	c.JSON(http.StatusOK, kernels)
}

// Create an individual spoofed/fake kernel.
func (h *KernelHttpHandler) spoofKernel() *gateway.DistributedJupyterKernel {
	status := domain.KernelStatuses[rand.Intn(len(domain.KernelStatuses))]
	numReplicas := rand.Intn(5-2) + 2
	kernelId := uuid.New().String()
	// Spoof the kernel itself.
	kernel := &gateway.DistributedJupyterKernel{
		KernelId:            kernelId,
		NumReplicas:         int32(numReplicas),
		Status:              status,
		AggregateBusyStatus: status,
		Replicas:            make([]*gateway.JupyterKernelReplica, 0, numReplicas),
	}

	// Spoof the kernel's replicas.
	for j := 0; j < numReplicas; j++ {
		podId := fmt.Sprintf("kernel-%s-%s", kernelId, uuid.New().String()[0:5])
		replica := &gateway.JupyterKernelReplica{
			ReplicaId: int32(j + 1),
			KernelId:  kernelId,
			PodId:     podId,
			NodeId:    fmt.Sprintf("Node-%d", rand.Intn(4-1)+1),
		}
		kernel.Replicas = append(kernel.Replicas, replica)
	}

	return kernel
}

// Called when spoofing kernels for the first time.
func (h *KernelHttpHandler) spoofInitialKernels() {
	numKernels := rand.Intn(8-2) + 2

	for i := 0; i < numKernels; i++ {
		kernel := h.spoofKernel()
		h.spoofedKernels.Set(kernel.GetKernelId(), kernel)
	}

	h.logger.Sugar().Debugf("Created an initial batch of %d spoofed kernels.", numKernels)
}

func (h *KernelHttpHandler) doSpoofKernels() []*gateway.DistributedJupyterKernel {
	// If we've already generated some kernels, then we'll randomly remove a few and add a few.
	if h.spoofedKernels.Count() > 0 {
		h.logger.Debug("Spoofing kernels.")

		var maxAdd int

		if h.spoofedKernels.Count() <= 2 {
			// If ther's 2 kernels or less, then add up to 5.
			maxAdd = 5
		} else {
			maxAdd = int(math.Ceil((0.25 * float64(h.spoofedKernels.Count())))) // Add and remove up to 25% of the existing number of the spoofed kernels.
		}

		maxDelete := int(math.Ceil((0.50 * float64(h.spoofedKernels.Count())))) // Add and remove up to 50% of the existing number of the spoofed kernels.
		numToDelete := rand.Intn(int(math.Max(2, float64(maxDelete+1))))        // Delete UP TO this many.
		numToAdd := rand.Intn(int(math.Max(2, float64(maxAdd+1))))

		h.logger.Sugar().Debugf("Adding %d new kernel(s) and removing up to %d existing kernel(s).", numToAdd, numToDelete)

		if numToDelete > 0 {
			currentKernels := h.spoofedKernelsToSlice()
			toDelete := make([]string, 0, numToDelete)

			for i := 0; i < numToDelete; i++ {
				// We may select the same victim multiple times. It will only be deleted once, of course.
				victimIdx := rand.Intn(len(currentKernels))
				toDelete = append(toDelete, currentKernels[victimIdx].GetKernelId())
			}

			numDeleted := 0
			// Delete the victims.
			for _, id := range toDelete {
				// Make sure we didn't already delete this one.
				if _, ok := h.spoofedKernels.Get(id); ok {
					h.spoofedKernels.Remove(id)
					numDeleted++
				}
			}

			h.logger.Sugar().Debugf("Removed %d kernel(s).", numDeleted)
		}

		for i := 0; i < numToAdd; i++ {
			kernel := h.spoofKernel()
			h.spoofedKernels.Set(kernel.GetKernelId(), kernel)
		}

		h.logger.Sugar().Debugf("There are now %d kernel(s).", h.spoofedKernels.Count())
	} else {
		h.logger.Debug("Spoofing kernels for the first time.")
		h.spoofInitialKernels()
	}

	// Convert to a slice before returning.
	return h.spoofedKernelsToSlice()
}

func (h *KernelHttpHandler) spoofedKernelsToSlice() []*gateway.DistributedJupyterKernel {
	spoofedKernelsSlice := make([]*gateway.DistributedJupyterKernel, 0, h.spoofedKernels.Count())
	for kvPair := range h.spoofedKernels.IterBuffered() {
		spoofedKernelsSlice = append(spoofedKernelsSlice, kvPair.Val)
	}
	return spoofedKernelsSlice
}

func (h *KernelHttpHandler) getKernelSpecsFromClusterGateway() []*gateway.DistributedJupyterKernel {
	h.logger.Debug("Kernel Querier is refreshing kernels now.")
	resp, err := h.rpcClient.ListKernels(context.TODO(), &gateway.Void{})
	if err != nil || resp == nil {
		h.logger.Error("[ERROR] Failed to fetch list of active kernels from the Cluster Gateway.", zap.Error(err))
		return nil
	}

	return resp.Kernels
}
