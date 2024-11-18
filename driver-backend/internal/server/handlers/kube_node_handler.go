package handlers

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/enriquebris/goconcurrentqueue"
	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeNodeHttpHandler handles HTTP GET and HTTP PATCH requests that respectively retrieve and modify
// nodes within the distributed cluster. These nodes are represented by the domain.ClusterNode interface.
//
// KubeNodeHttpHandler is used as the internal node handler by the NodeHttpHandler struct when the cluster is
// running/deployed in Kubernetes mode.
//
// KubeNodeHttpHandler uses the Go Kubernetes API to retrieve information about the nodes available within the cluster.
type KubeNodeHttpHandler struct {
	*BaseHandler
	grpcClient *ClusterDashboardHandler

	// The handler will return 0 nodes until this flag is flipped to true.
	nodeTypeRegistered bool

	clientsets goconcurrentqueue.Queue
	clientset  *kubernetes.Clientset
	//spoof      bool
}

// NewKubeNodeHttpHandler creates a new NewKubeNodeHttpHandler struct and return a pointer to it.
func NewKubeNodeHttpHandler(opts *domain.Configuration, grpcClient *ClusterDashboardHandler, atom *zap.AtomicLevel) *KubeNodeHttpHandler {
	if grpcClient == nil {
		panic("gRPC Client cannot be nil.")
	}

	handler := &KubeNodeHttpHandler{
		BaseHandler: newBaseHandler(opts, atom),
		grpcClient:  grpcClient,
		//spoof:              opts.SpoofKubeNodes,
		nodeTypeRegistered: false,
		clientsets:         goconcurrentqueue.NewFixedFIFO(128),
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side KubeNodeHttpHandler.")

	//if !opts.SpoofKubeNodes {
	//	handler.clientset = handler.createKubernetesClient(opts)
	//}

	handler.logger.Info("Successfully created server-side KubeNodeHttpHandler handler.")

	return handler
}

func (h *KubeNodeHttpHandler) createKubernetesClient(opts *domain.Configuration) *kubernetes.Clientset {
	if opts.InCluster {
		// creates the in-cluster config
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}

		// creates the clientset
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}

		return clientset
	} else {
		// use the current context in kube config
		config, err := clientcmd.BuildConfigFromFlags("", opts.KubeConfig)
		if err != nil {
			panic(err.Error())
		}

		// create the clientset
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}

		return clientset
	}
}

func (h *KubeNodeHttpHandler) getOrCreateClientset() *kubernetes.Clientset {
	// Use a cached clientset if one is available.
	var clientset *kubernetes.Clientset
	val, err := h.clientsets.Dequeue()
	if err != nil {
		// We create a new clientset here, rather than reuse the clientset of the handler, as this method
		// is called in an individual goroutine for each node. We want to be able to issue the requests
		// in parallel, so we want each thread to have its own clientset.
		clientset = h.createKubernetesClient(h.opts)
	} else {
		clientset = val.(*kubernetes.Clientset)
	}

	return clientset
}

func (h *KubeNodeHttpHandler) parseKubernetesNode(node *corev1.Node, actualGpuInformation *gateway.ClusterActualGpuInfo) (*domain.KubernetesNode, error) {
	capacityCpuAsQuantity := node.Status.Allocatable[corev1.ResourceCPU]
	capacityMemoryAsQuantity := node.Status.Allocatable[corev1.ResourceMemory]

	var capacityVirtualGPUs = 0.0
	capacityVirtualGPUsAsQuantity, ok := node.Status.Allocatable["ds2-lab.github.io/deflated-gpu"]
	if !ok {
		capacityVirtualGPUs = 0
	} else {
		capacityVirtualGPUs = float64(capacityVirtualGPUsAsQuantity.Value())
	}

	capacityCPUs := float64(capacityCpuAsQuantity.MilliValue()) / 1000.0 // Convert from mCPU to CPU.
	capacityMemory := float64(capacityMemoryAsQuantity.Value() / (1024 * 1024))

	// h.logger.Info("Memory as inf.Dec.", zap.String("node-id", node.Name), zap.Any("mem inf.Dec", capacityMemory.AsDec().String()))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	clientset := h.getOrCreateClientset()

	st := time.Now()
	pods, err := clientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + node.Name,
	})
	if err != nil {
		h.logger.Error("Could not retrieve Pods running on node.", zap.String("node", node.Name), zap.Error(err))
	} else {
		h.sugaredLogger.Debugf("Retrieved Pods running on node %s in %v.", node.Name, time.Since(st))
	}

	var schedulingDisabled = false
	var executionDisabled = false
	if len(node.Spec.Taints) > 0 {
		h.sugaredLogger.Debugf("Discovered %d taint(s) on node %s.", len(node.Spec.Taints), node.Name)

		for _, taint := range node.Spec.Taints {
			h.logger.Debug("Discovered taint.", zap.String("effect", string(taint.Effect)), zap.String("taint-key", taint.Key), zap.String("taint-value", taint.Value))

			if string(taint.Effect) == "NoSchedule" {
				schedulingDisabled = true
			} else if string(taint.Effect) == "NoExecute" {
				executionDisabled = true
			}
		}
	}

	var kubePods domain.ContainerList
	if pods != nil {
		kubePods = make(domain.ContainerList, 0, len(pods.Items))

		for _, pod := range pods.Items {
			kubePod := &domain.KubernetesPod{
				PodName:  pod.ObjectMeta.Name,
				PodPhase: string(pod.Status.Phase),
				PodIP:    pod.Status.PodIP,
				PodAge:   time.Since(pod.GetCreationTimestamp().Time).Round(time.Second).String(),
				Valid:    true,
			}

			kubePods = append(kubePods, kubePod)
		}
	}

	sort.Slice(kubePods, func(i, j int) bool {
		return kubePods[i].GetName() < kubePods[j].GetName()
	})

	podsOnNode, err := clientset.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", node.Name),
	})
	if err != nil {
		h.logger.Error("Failed to retrieve list of Pods running on node.", zap.String("node", node.Name), zap.Error(err))
		return nil, err
	}

	allocatedCPUs := 0.0
	allocatedMemory := 0.0
	allocatedVirtualGPUs := 0.0
	allocatedGPUs := 0.0
	capacityGPUs := 0.0
	allocatedResources := make(map[domain.ResourceName]float64)
	capacityResources := make(map[domain.ResourceName]float64)

	for _, pod := range podsOnNode.Items {
		for _, container := range pod.Spec.Containers {
			resources := container.Resources.Limits

			allocatedCPUs += float64(resources.Cpu().MilliValue() / 1000.0)
			allocatedMemory += float64(resources.Memory().Value() / (1024 * 1024))

			vgpus, ok := resources["ds2-lab.github.io/deflated-gpu"]
			if ok {
				allocatedVirtualGPUs += float64(vgpus.Value())
			}
		}
	}

	// The control-plane node won't have any GPU information whatsoever.
	if !strings.HasSuffix(node.Name, "control-plane") && actualGpuInformation != nil {
		if gpuInfo, ok := actualGpuInformation.GetGpuInfo()[node.Name]; ok {
			allocatedGPUs = float64(gpuInfo.CommittedGPUs)
			capacityGPUs = float64(gpuInfo.SpecGPUs)
		} else {
			h.logger.Error("Could not retrieve 'actual' GPU information for node.", zap.String("node", node.Name))
		}
	}

	// TODO: VRAM not supported for Kubernetes yet.

	allocatedResources[domain.CpuResource] = allocatedCPUs
	capacityResources[domain.CpuResource] = capacityCPUs
	allocatedResources[domain.MemoryResource] = allocatedMemory
	capacityResources[domain.MemoryResource] = capacityMemory
	allocatedResources[domain.GpuResource] = allocatedGPUs
	capacityResources[domain.GpuResource] = capacityGPUs
	allocatedResources[domain.VirtualGpuResource] = allocatedVirtualGPUs
	capacityResources[domain.VirtualGpuResource] = capacityVirtualGPUs
	allocatedResources[domain.VRAMResource] = -1
	capacityResources[domain.VRAMResource] = -1

	parsedNode := &domain.KubernetesNode{
		NodeId:             node.Name,
		NodeName:           node.Name,
		Pods:               kubePods,
		Age:                time.Since(node.GetCreationTimestamp().Time).Round(time.Second).String(),
		CreatedAt:          node.GetCreationTimestamp().Time.UnixMilli(),
		IP:                 node.Status.Addresses[0].Address,
		Enabled:            !schedulingDisabled && !executionDisabled,
		AllocatedResources: allocatedResources,
		CapacityResources:  capacityResources,
	}

	err = h.clientsets.Enqueue(clientset)
	if err != nil {
		h.logger.Error("Failed to cache clientset.", zap.Error(err))
	}

	return parsedNode, nil
}

func (h *KubeNodeHttpHandler) HandleRequest(c *gin.Context) {
	st := time.Now()
	h.logger.Debug("Handling 'get-nodes' request now.")
	//if h.spoof {
	//	h.spoofNodes(c)
	//	return
	//}

	h.logger.Debug("Serving HTTP GET request for Kubernetes nodes.")

	nodes, err := h.clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		h.logger.Error("Failed to retrieve nodes from Kubernetes.", zap.Error(err))
		_ = c.AbortWithError(502, err)
		return
	}
	h.sugaredLogger.Debugf("Listed Kubernetes nodes via Kubernetes API in %v.", time.Since(st))

	st2 := time.Now()
	actualGpuInformation, err := h.grpcClient.GetClusterActualGpuInfo(context.TODO(), &gateway.Void{})
	if err != nil {
		domain.LogErrorWithoutStacktrace(h.logger, "Failed to retrieve 'actual' GPU usage from Cluster Gateway.", zap.Error(err))
		_ = c.Error(fmt.Errorf("failed to retrieve 'actual' GPU usage from Cluster Gateway: %v", err.Error()))
		h.grpcClient.HandleConnectionError()
	}
	h.sugaredLogger.Debugf("Retrieved 'actual' GPU info from Cluster Gateway in %v. Total time elapsed: %v.", time.Since(st2), time.Since(st))
	st3 := time.Now()

	var resp = make([]domain.ClusterNode, 0, len(nodes.Items)-1)
	val := nodes.Items[0].Status.Capacity[corev1.ResourceCPU]
	val.AsInt64()

	nodesChannel := make(chan *domain.KubernetesNode, len(nodes.Items)-1)
	var (
		waitGroup sync.WaitGroup
		done      bool
	)
	waitGroup.Add(len(nodes.Items) - 1)

	// Using goroutines to process/parse the nodes in parallel.
	// Each node will require a Kubernetes API call and a gRPC call, so doing this in parallel should generally be faster.
	for _, n := range nodes.Items {
		if strings.HasSuffix(n.Name, "control-plane") {
			continue
		}
		go func(resultChannel chan *domain.KubernetesNode, node corev1.Node, wg *sync.WaitGroup) {
			parsedNode, err := h.parseKubernetesNode(&node, actualGpuInformation)

			if err != nil {
				_ = c.Error(err)
			} else {
				nodesChannel <- parsedNode
			}

			wg.Done()
		}(nodesChannel, n, &waitGroup)
	}

	// Wait for the goroutines to finish processing the nodes.
	waitGroup.Wait()
	for !done {
		select {
		case parsedNode := <-nodesChannel:
			{
				resp = append(resp, parsedNode)
			}
		default:
			{
				done = true
				break
			}
		}
	}

	// Sort the nodes.
	sort.Slice(resp, func(i, j int) bool {
		return resp[i].GetNodeId() < resp[j].GetNodeId()
	})

	h.sugaredLogger.Debugf("Parsed %d Kubernetes nodes in %v. Total time elapsed: %v.", len(resp), time.Since(st3), time.Since(st))
	c.JSON(http.StatusOK, resp)
}

// HandlePatchRequest handles enabling/disabling a particular kubernetes node,
// which amounts to adding/removing taints from the node.
func (h *KubeNodeHttpHandler) HandlePatchRequest(c *gin.Context) {
	var req *domain.EnableDisableNodeRequest
	err := c.BindJSON(&req)
	if err != nil {
		h.logger.Error("Failed to extract 'EnableDisableNodeRequest'")
		return
	}

	var nodeName = req.NodeName

	h.logger.Debug("Received HTTP PATCH request to enable/disable kubernetes node.", zap.String("node-name", nodeName), zap.String("request", req.String()))

	var applyConfig *v1.NodeApplyConfiguration
	if req.Enable {
		h.sugaredLogger.Debugf("Will be enabling node %s and therefore removing taints from the node.", nodeName)
		applyConfig = v1.Node(nodeName).WithSpec(v1.NodeSpec().WithTaints())
	} else {
		h.sugaredLogger.Debugf("Will be disabling node %s.", nodeName)
		applyConfig = v1.Node(nodeName).WithSpec(v1.NodeSpec().WithTaints(
			v1.Taint().WithKey("key1").WithValue("value1").WithEffect(corev1.TaintEffectNoExecute),
			v1.Taint().WithKey("key1").WithValue("value1").WithEffect(corev1.TaintEffectNoSchedule)))
	}

	resp, err := h.clientset.CoreV1().Nodes().Apply(context.Background(), applyConfig, metav1.ApplyOptions{
		FieldManager: "application/apply-patch",
	})

	// resp, err := h.clientset.CoreV1().Nodes().Patch(context.Background(), nodeName, types.StrategicMergePatchType, []byte(patchData), metav1.PatchOptions{FieldValidation: "Strict"})
	if err != nil {
		// Error message depends on whether we're enabling/disabling the node.
		if req.Enable {
			h.logger.Error("Failed to remove taints from Kubernetes node.", zap.String("node-name", nodeName), zap.Error(err))
		} else {
			h.logger.Error("Failed to add 'NoExecute' and 'NoSchedule' taints to Kubernetes node.", zap.String("node-name", nodeName), zap.Error(err))
		}

		// TODO(Ben): We need a proper way to handle this.
		c.JSON(http.StatusInternalServerError, &domain.ErrorMessage{
			ErrorMessage: err.Error(),
			Valid:        true,
		})
	} else {
		if req.Enable {
			h.logger.Debug("Successfully removed the 'NoExecute' and 'NoSchedule' taints from the Kubernetes node.", zap.String("node-name", nodeName))
		} else {
			h.logger.Debug("Successfully added 'NoExecute' and 'NoSchedule' taints to the Kubernetes node.", zap.String("node-name", nodeName))
		}

		updatedNode, err := h.parseKubernetesNode(resp, nil)

		if err != nil {
			_ = c.AbortWithError(502, err)
			return
		} else {
			h.logger.Debug("Sending updated Kubernetes node back to frontend.", zap.String("node-name", nodeName), zap.String("updated-node", updatedNode.String()))
			c.JSON(http.StatusOK, updatedNode)
		}
	}
}

func (h *KubeNodeHttpHandler) spoofNodes(c *gin.Context) {
	c.JSON(http.StatusOK, []*domain.KubernetesNode{
		{
			NodeId: "spoofed-kubernetes-node-0",
			Pods: domain.ContainerList{
				&domain.KubernetesPod{
					PodName:  "spoofed-kubernetes-pod-0",
					PodAge:   "121hr24m18sec",
					PodPhase: "running",
					PodIP:    "148.122.32.1",
				},
				&domain.KubernetesPod{
					PodName:  "spoofed-kubernetes-pod-1",
					PodAge:   "121hr25m43sec",
					PodPhase: "running",
					PodIP:    "148.122.32.2",
				},
				&domain.KubernetesPod{
					PodName:  "spoofed-kubernetes-pod-2",
					PodAge:   "121hr12m59sec",
					PodPhase: "running",
					PodIP:    "148.122.32.3",
				},
			},
			Age: "121hr32m14sec",
			IP:  "10.0.0.1",
		},
		{
			NodeId: "spoofed-kubernetes-node-1",
			Pods: domain.ContainerList{
				&domain.KubernetesPod{
					PodName:  "spoofed-kubernetes-pod-3",
					PodAge:   "121hr44m28sec",
					PodPhase: "running",
					PodIP:    "157.137.61.1",
				},
				&domain.KubernetesPod{
					PodName:  "spoofed-kubernetes-pod-4",
					PodAge:   "121hr22m42sec",
					PodPhase: "running",
					PodIP:    "157.137.61.2",
				},
				&domain.KubernetesPod{
					PodName:  "spoofed-kubernetes-pod-5",
					PodAge:   "121hr13m49sec",
					PodPhase: "running",
					PodIP:    "157.137.61.3",
				},
			},
			Age: "121hr32m14sec",
			IP:  "10.0.0.2",
		},
	})
}
