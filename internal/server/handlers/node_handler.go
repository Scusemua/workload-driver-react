package handlers

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

type KubeNodeHttpHandler struct {
	*GrpcClient

	metricsClient *metrics.Clientset
	clientset     *kubernetes.Clientset
	spoof         bool
}

func NewKubeNodeHttpHandler(opts *domain.Configuration) domain.BackendHttpGetPatchHandler {
	handler := &KubeNodeHttpHandler{
		GrpcClient: NewGrpcClient(opts, !opts.SpoofKernels),
		spoof:      opts.SpoofKubeNodes,
	}
	handler.BackendHttpGetHandler = handler

	handler.logger.Info("Creating server-side KubeNodeHttpHandler.")

	if !opts.SpoofKubeNodes {
		handler.createKubernetesClient(opts)
	}

	handler.logger.Info("Successfully created server-side HTTP handler.")

	return handler
}

func (h *KubeNodeHttpHandler) createKubernetesClient(opts *domain.Configuration) {
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

		metricsConfig, err := rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}

		metricsClient, err := metrics.NewForConfig(metricsConfig)
		if err != nil {
			panic(err)
		}

		h.clientset = clientset
		h.metricsClient = metricsClient
	} else {
		// use the current context in kubeconfig
		config, err := clientcmd.BuildConfigFromFlags("", opts.KubeConfig)
		if err != nil {
			panic(err.Error())
		}

		// create the clientset
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}

		metricsConfig, err := clientcmd.BuildConfigFromFlags("", opts.KubeConfig)
		if err != nil {
			panic(err.Error())
		}

		metricsClient, err := metrics.NewForConfig(metricsConfig)
		if err != nil {
			panic(err)
		}

		h.clientset = clientset
		h.metricsClient = metricsClient
	}
}

func (h *KubeNodeHttpHandler) parseKubernetesNode(node *corev1.Node) *domain.KubernetesNode {
	allocatableCPU := node.Status.Capacity[corev1.ResourceCPU]
	allocatableMemory := node.Status.Capacity[corev1.ResourceMemory]

	var allocVGPU float64 = 0.0
	allocatableVirtualGPUs, ok := node.Status.Capacity["ds2-lab.github.io/deflated-gpu"]
	if !ok {
		allocVGPU = 0
	} else {
		allocVGPU = allocatableVirtualGPUs.AsApproximateFloat64()
	}

	allocCpu := allocatableCPU.AsApproximateFloat64()
	allocMem := allocatableMemory.AsApproximateFloat64()

	// h.logger.Info("Memory as inf.Dec.", zap.String("node-id", node.Name), zap.Any("mem inf.Dec", allocatableMemory.AsDec().String()))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	pods, err := h.clientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + node.Name,
	})

	if err != nil {
		h.logger.Error("Could not retrieve Pods running on node.", zap.String("node", node.Name), zap.Error(err))
	}

	var schedulingDisabled bool = false
	var executionDisabled bool = false
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

	var kubePods []*domain.KubernetesPod
	if pods != nil {
		kubePods = make([]*domain.KubernetesPod, 0, len(pods.Items))

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
		return kubePods[i].PodName < kubePods[j].PodName
	})

	return &domain.KubernetesNode{
		NodeId:         node.Name,
		CapacityCPU:    allocCpu,
		CapacityMemory: allocMem / 976600.0, // Convert from Ki to GB.
		CapacityVGPUs:  allocVGPU,
		Pods:           kubePods,
		Age:            time.Since(node.GetCreationTimestamp().Time).Round(time.Second).String(),
		IP:             node.Status.Addresses[0].Address,
		Enabled:        !schedulingDisabled && !executionDisabled,
	}

}

func (h *KubeNodeHttpHandler) HandleRequest(c *gin.Context) {
	if h.spoof {
		h.spoofNodes(c)
		return
	}

	nodes, err := h.clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		h.logger.Error("Failed to retrieve nodes from Kubernetes.", zap.Error(err))
		h.WriteError(c, "Failed to retrieve nodes from Kubernetes.")
		return
	}

	nodeUsageMetrics, err := h.metricsClient.MetricsV1beta1().NodeMetricses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		h.logger.Error("Failed to retrieve node metrics from Kubernetes.", zap.Error(err))
		h.WriteError(c, "Failed to retrieve node metrics from Kubernetes.")
		return
	}

	h.logger.Info(fmt.Sprintf("Sending a list of %d nodes back to the client.", len(nodes.Items)), zap.Int("num-nodes", len(nodes.Items)))

	var kubernetesNodes map[string]*domain.KubernetesNode = make(map[string]*domain.KubernetesNode, len(nodes.Items))
	val := nodes.Items[0].Status.Capacity[corev1.ResourceCPU]
	val.AsInt64()
	for _, node := range nodes.Items {
		kubernetesNodes[node.Name] = h.parseKubernetesNode(&node)
	}

	for _, nodeMetric := range nodeUsageMetrics.Items {
		nodeName := nodeMetric.ObjectMeta.Name
		kubeNode := kubernetesNodes[nodeName]

		cpu := nodeMetric.Usage.Cpu().AsApproximateFloat64()
		mem := nodeMetric.Usage.Memory().AsApproximateFloat64()

		kubeNode.AllocatedCPU = cpu
		kubeNode.AllocatedMemory = mem / 976600.0 // Convert from Ki to GB.

		kubernetesNodes[nodeName] = kubeNode
	}

	var resp []*domain.KubernetesNode = make([]*domain.KubernetesNode, 0, len(kubernetesNodes))
	for _, node := range kubernetesNodes {
		if node == nil {
			continue
		}

		resp = append(resp, node)
	}

	if len(resp) > 0 {
		// This could be more efficient (converting from map to slice and then sorting; I could just do it in a single step).
		sort.Slice(resp, func(i, j int) bool {
			return resp[i].NodeId < resp[j].NodeId
		})
	}

	h.logger.Info("Sending nodes back to client now.", zap.Int("num-nodes", len(resp)))
	c.JSON(http.StatusOK, resp)
}

// Handle enabling/disabling a particular kubernetes node, which amounts to adding/removing taints from the node.
func (h *KubeNodeHttpHandler) HandlePatchRequest(c *gin.Context) {
	var req *domain.EnableDisableNodeRequest
	err := c.BindJSON(&req)
	if err != nil {
		h.logger.Error("Failed to extract 'EnableDisableNodeRequest'")
		return
	}

	var nodeName string = req.NodeName

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

		updatedNode := h.parseKubernetesNode(resp)
		h.logger.Debug("Sending updated Kubernetes node back to frontend.", zap.String("node-name", nodeName), zap.String("updated-node", updatedNode.String()))
		c.JSON(http.StatusOK, updatedNode)
	}
}

func (h *KubeNodeHttpHandler) spoofNodes(c *gin.Context) {
	c.JSON(http.StatusOK, []*domain.KubernetesNode{
		{
			NodeId: "spoofed-kubernetes-node-0",
			Pods: []*domain.KubernetesPod{
				{
					PodName:  "spoofed-kubernetes-pod-0",
					PodAge:   "121hr24m18sec",
					PodPhase: "running",
					PodIP:    "148.122.32.1",
				},
				{
					PodName:  "spoofed-kubernetes-pod-1",
					PodAge:   "121hr25m43sec",
					PodPhase: "running",
					PodIP:    "148.122.32.2",
				},
				{
					PodName:  "spoofed-kubernetes-pod-2",
					PodAge:   "121hr12m59sec",
					PodPhase: "running",
					PodIP:    "148.122.32.3",
				},
			},
			Age:             "121hr32m14sec",
			IP:              "10.0.0.1",
			CapacityCPU:     64,
			CapacityMemory:  64,
			CapacityGPUs:    8,
			CapacityVGPUs:   72,
			AllocatedCPU:    24,
			AllocatedMemory: 54,
			AllocatedGPUs:   2,
			AllocatedVGPUs:  18,
		},
		{
			NodeId: "spoofed-kubernetes-node-1",
			Pods: []*domain.KubernetesPod{
				{
					PodName:  "spoofed-kubernetes-pod-3",
					PodAge:   "121hr44m28sec",
					PodPhase: "running",
					PodIP:    "157.137.61.1",
				},
				{
					PodName:  "spoofed-kubernetes-pod-4",
					PodAge:   "121hr22m42sec",
					PodPhase: "running",
					PodIP:    "157.137.61.2",
				},
				{
					PodName:  "spoofed-kubernetes-pod-5",
					PodAge:   "121hr13m49sec",
					PodPhase: "running",
					PodIP:    "157.137.61.3",
				},
			},
			Age:             "121hr32m14sec",
			IP:              "10.0.0.2",
			CapacityCPU:     64,
			CapacityMemory:  64,
			CapacityGPUs:    8,
			CapacityVGPUs:   72,
			AllocatedCPU:    48,
			AllocatedMemory: 60,
			AllocatedGPUs:   4,
			AllocatedVGPUs:  36,
		},
	})
}
