interface KubernetesNode {
    NodeId: string;
    Pods: KubernetesPod[];
    Age: string;
    IP: string;
    CapacityCPU: number;
    CapacityMemory: number;
    CapacityGPUs: number;
    CapacityVGPUs: number;
    AllocatedCPU: number;
    AllocatedMemory: number;
    AllocatedGPUs: number;
    AllocatedVGPUs: number;
    Enabled: boolean;
}

interface KubernetesPod {
    PodName: string;
    PodPhase: string;
    PodAge: string;
    PodIP: string;
    Valid: boolean;
}

export type { KubernetesPod as KubernetesPod };
export type { KubernetesNode as KubernetesNode };
