interface ClusterNode {
    NodeId: string;
    PodsOrContainers: PodOrContainer[];
    Age: string;
    IP: string;
    AllocatedResources: Map<string, number>;
    PendingResources: Map<string, number>;
    CapacityResources: Map<string, number>;
    Enabled: boolean;
}

interface PodOrContainer {
    Name: string;
    Phase: string;
    Age: string;
    IP: string;
    Valid: boolean;
    Type: string;
}

interface VirtualGpuInfo {
    totalVirtualGPUs: number; // Total available vGPUs on the node.
    allocatedVirtualGPUs: number; // Number of allocated vGPUs on the node.
    freeVirtualGPUs: number; // Free (i.e., idle) vGPUs on the node.
}

export type { PodOrContainer as PodOrContainer };
export type { ClusterNode as ClusterNode };
export type { VirtualGpuInfo as VirtualGpuInfo };
