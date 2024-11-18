package handlers

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"go.uber.org/zap"
)

type KernelHttpHandler struct {
	*BaseHandler
	grpcClient *ClusterDashboardHandler

	//spoofKernels   bool                                                           // If we're creating and returning fake/mocked kernels.
	//spoofedKernels *cmap.ConcurrentMap[string, *gateway.DistributedJupyterKernel] // Latest spoofedKernels.
}

func NewKernelHttpHandler(opts *domain.Configuration, grpcClient *ClusterDashboardHandler, atom *zap.AtomicLevel) *KernelHttpHandler {
	if grpcClient == nil {
		panic("gRPC Client cannot be nil.")
	}

	handler := &KernelHttpHandler{
		BaseHandler: newBaseHandler(opts, atom),
		//spoofKernels: opts.SpoofKernels,
		grpcClient: grpcClient,
	}
	handler.BackendHttpGetHandler = handler

	//if handler.spoofKernels {
	//	spoofedKernels := cmap.New[*gateway.DistributedJupyterKernel]()
	//	handler.spoofedKernels = &spoofedKernels
	//}

	handler.logger.Info("Creating server-side KernelHttpHandler.")

	return handler
}

func (h *KernelHttpHandler) HandleRequest(c *gin.Context) {
	if !h.grpcClient.ConnectedToGateway() {
		h.logger.Warn("Connection with Cluster Gateway has not been established. Aborting.")

		h.grpcClient.HandleConnectionError()

		_ = c.AbortWithError(http.StatusServiceUnavailable, fmt.Errorf("connection with Cluster Gateway is inactive"))
		return
	}

	var (
		kernels []*gateway.DistributedJupyterKernel
		err     error
	)

	kernels, err = h.getKernelsFromClusterGateway()
	if err != nil {
		// We already attempt to reconnect gRPC in the `getKernelsFromClusterGateway` method. So, just abort the request here.
		_ = c.AbortWithError(500, err)
		return
	}

	if kernels == nil {
		// Write error back to front-end.
		h.logger.Error("Failed to retrieve list of kernels from Jupyter Server.")
		h.WriteError(c, "Failed to retrieve list of kernels from Jupyter Server.")
		return
	}

	for _, kernel := range kernels {
		sort.SliceStable(kernel.Replicas, func(i, j int) bool {
			return kernel.Replicas[i].ReplicaId < kernel.Replicas[j].ReplicaId
		})
	}
	//}

	h.sugaredLogger.Infof("Sending %d kernel(s) back to client now.", len(kernels))
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
//func (h *KernelHttpHandler) spoofInitialKernels() {
//	numKernels := rand.Intn(8-2) + 2
//
//	for i := 0; i < numKernels; i++ {
//		kernel := h.spoofKernel()
//		h.spoofedKernels.Set(kernel.GetKernelId(), kernel)
//	}
//
//	h.logger.Sugar().Debugf("Created an initial batch of %d spoofed kernels.", numKernels)
//}
//
//func (h *KernelHttpHandler) doSpoofKernels() []*gateway.DistributedJupyterKernel {
//	// If we've already generated some kernels, then we'll randomly remove a few and add a few.
//	if h.spoofedKernels.Count() > 0 {
//		h.logger.Debug("Spoofing kernels.")
//
//		var maxAdd int
//
//		if h.spoofedKernels.Count() <= 2 {
//			// If there's 2 kernels or fewer, then add up to 5.
//			maxAdd = 5
//		} else {
//			maxAdd = int(math.Ceil(0.25 * float64(h.spoofedKernels.Count()))) // Add and remove up to 25% of the existing number of the spoofed kernels.
//		}
//
//		maxDelete := int(math.Ceil(0.50 * float64(h.spoofedKernels.Count()))) // Add and remove up to 50% of the existing number of the spoofed kernels.
//		numToDelete := rand.Intn(int(math.Max(2, float64(maxDelete+1))))      // Delete UP TO this many.
//		numToAdd := rand.Intn(int(math.Max(2, float64(maxAdd+1))))
//
//		h.logger.Sugar().Debugf("Adding %d new kernel(s) and removing up to %d existing kernel(s).", numToAdd, numToDelete)
//
//		if numToDelete > 0 {
//			currentKernels := h.spoofedKernelsToSlice()
//			toDelete := make([]string, 0, numToDelete)
//
//			for i := 0; i < numToDelete; i++ {
//				// We may select the same victim multiple times. It will only be deleted once, of course.
//				victimIdx := rand.Intn(len(currentKernels))
//				toDelete = append(toDelete, currentKernels[victimIdx].GetKernelId())
//			}
//
//			numDeleted := 0
//			// Delete the victims.
//			for _, id := range toDelete {
//				// Make sure we didn't already delete this one.
//				if _, ok := h.spoofedKernels.Get(id); ok {
//					h.spoofedKernels.Remove(id)
//					numDeleted++
//				}
//			}
//
//			h.logger.Sugar().Debugf("Removed %d kernel(s).", numDeleted)
//		}
//
//		for i := 0; i < numToAdd; i++ {
//			kernel := h.spoofKernel()
//			h.spoofedKernels.Set(kernel.GetKernelId(), kernel)
//		}
//
//		h.logger.Sugar().Debugf("There are now %d kernel(s).", h.spoofedKernels.Count())
//	} else {
//		h.logger.Debug("Spoofing kernels for the first time.")
//		h.spoofInitialKernels()
//	}
//
//	// Convert to a slice before returning.
//	return h.spoofedKernelsToSlice()
//}
//
//func (h *KernelHttpHandler) spoofedKernelsToSlice() []*gateway.DistributedJupyterKernel {
//	spoofedKernelsSlice := make([]*gateway.DistributedJupyterKernel, 0, h.spoofedKernels.Count())
//	for kvPair := range h.spoofedKernels.IterBuffered() {
//		spoofedKernelsSlice = append(spoofedKernelsSlice, kvPair.Val)
//	}
//	return spoofedKernelsSlice
//}

func (h *KernelHttpHandler) getKernelsFromClusterGateway() ([]*gateway.DistributedJupyterKernel, error) {
	resp, err := h.grpcClient.ListKernels(context.TODO(), &gateway.Void{})
	if err != nil {
		domain.LogErrorWithoutStacktrace(h.logger, "Failed to fetch list of active kernels from the Cluster Gateway.", zap.Error(err))
		h.grpcClient.HandleConnectionError()
		return nil, err
	} else if resp.Kernels == nil {
		// We successfully retrieved the kernels, so return them.
		// The response can be nil if there are no kernels on the Gateway.
		return make([]*gateway.DistributedJupyterKernel, 0), nil
	} else {
		// We successfully retrieved the kernels, so return them.
		return resp.Kernels, nil
	}
}
