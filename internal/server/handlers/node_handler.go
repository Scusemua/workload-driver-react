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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

type KubeNodeHttpHandler struct {
	*BaseHandler

	metricsClient *metrics.Clientset
	clientset     *kubernetes.Clientset
	spoof         bool
}

func NewKubeNodeHttpHandler(opts *domain.Configuration) domain.BackendHttpGetHandler {
	handler := &KubeNodeHttpHandler{
		BaseHandler: newBaseHandler(opts),
		spoof:       opts.SpoofKubeNodes,
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
		allocatableCPU := node.Status.Capacity[corev1.ResourceCPU]
		allocatableMemory := node.Status.Capacity[corev1.ResourceMemory]

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

		kubernetesNode := domain.KubernetesNode{
			NodeId:         node.Name,
			CapacityCPU:    allocCpu,
			CapacityMemory: allocMem / 976600.0, // Convert from Ki to GB.
			Pods:           kubePods,
			Age:            time.Since(node.GetCreationTimestamp().Time).Round(time.Second).String(),
			IP:             node.Status.Addresses[0].Address,
			// CapacityGPUs:    0,
			// CapacityVGPUs:   0,
			// AllocatedCPU:    0,
			// AllocatedMemory: 0,
			// AllocatedGPUs:   0,
			// AllocatedVGPUs:  0,
		}

		kubernetesNodes[node.Name] = &kubernetesNode
	}

	for _, nodeMetric := range nodeUsageMetrics.Items {
		nodeName := nodeMetric.ObjectMeta.Name
		kubeNode := kubernetesNodes[nodeName]
		// h.logger.Info("Node metric.", zap.String("node", nodeName), zap.Any("metric", nodeMetric))

		cpu := nodeMetric.Usage.Cpu().AsApproximateFloat64()
		// if !ok {
		// 	h.logger.Error("Could not convert CPU usage metric to Int64.", zap.Any("cpu-metric", nodeMetric.Usage.Cpu()))
		// }
		// h.logger.Info("CPU metric.", zap.String("node-id", nodeName), zap.Float64("cpu", cpu))

		mem := nodeMetric.Usage.Memory().AsApproximateFloat64()
		// if !ok {
		// 	h.logger.Error("Could not convert 	memory usage metric to Int64.", zap.Any("mem-metric", nodeMetric.Usage.Memory()))
		// }
		// h.logger.Info("Memory metric.", zap.String("node-id", nodeName), zap.Float64("memory", cpu))

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
