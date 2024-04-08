package main

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", "C:/Users/benrc/.kube/config")
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	nodes, _ := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	n, _ := json.Marshal(nodes)
	fmt.Printf("Nodes:\n%v\n", string(n))

	for _, node := range nodes.Items {
		if node.Name == "distributed-notebook-control-plane" {
			continue
		}
		selector := fmt.Sprintf("spec.nodeName=%s", node.Name)
		podsOnNode, err := clientset.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{
			FieldSelector: selector,
		})
		if err != nil {
			fmt.Printf("Failed to retrieve list of Pods running on node %s: %v\n", node.Name, err)
			fmt.Printf("Selector: \"%s\"\n", selector)
			continue
		}

		allocatedCPUs := 0.0
		allocatedMemory := 0.0
		allocatedVirtualGPUs := 0.0

		for _, pod := range podsOnNode.Items {
			for _, container := range pod.Spec.Containers {
				resources := container.Resources.Limits

				allocatedCPUs += resources.Cpu().AsApproximateFloat64()
				allocatedMemory += resources.Memory().AsApproximateFloat64()

				vgpus, ok := resources["ds2-lab.github.io/deflated-gpu"]
				if ok {
					allocatedVirtualGPUs += vgpus.AsApproximateFloat64()
				}
			}
		}
	}
}
