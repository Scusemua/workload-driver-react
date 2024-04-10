interface KubernetesNode {
    NodeId: string;
    Pods: KubernetesPod[];
    Age: string;
    IP: string;
    AllocatedResources: Map<string, number>;
    CapacityResources: Map<string, number>;
    Enabled: boolean;
}

interface KubernetesPod {
    PodName: string;
    PodPhase: string;
    PodAge: string;
    PodIP: string;
    Valid: boolean;
}

interface VirtualGpuInfo {
    totalVirtualGPUs: number; // Total available vGPUs on the node.
    allocatedVirtualGPUs: number; // Number of allocated vGPUs on the node.
    freeVirtualGPUs: number; // Free (i.e., idle) vGPUs on the node.
}

export type { KubernetesPod as KubernetesPod };
export type { KubernetesNode as KubernetesNode };
export type { VirtualGpuInfo as VirtualGpuInfo };
