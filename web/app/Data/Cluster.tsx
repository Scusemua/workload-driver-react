// interface ClusterNode {
//     NodeId: string;
//     NodeName: string;
//     PodsOrContainers: PodOrContainer[];
//     Age: string;
//     IP: string;
//     AllocatedResources: Map<string, number>;
//     IdleResources: Map<string, number>;
//     PendingResources: Map<string, number>;
//     CapacityResources: Map<string, number>;
//     Enabled: boolean;
// }

export interface ClusterNode {
    nodeId: string;
    containers?: PodOrContainer[];
    specCpu: number;
    specMemory: number;
    specGpu: number;
    specVRAM: number;
    allocatedCpu?: number;
    allocatedMemory?: number;
    allocatedGpu?: number;
    allocatedVRAM?: number;
    pendingCpu?: number;
    pendingMemory?: number;
    pendingGpu?: number;
    pendingVRAM?: number;
    nodeName: string;
    address: string;
    createdAt: ProtoTimestamp;
}

export interface ProtoTimestamp {
    // Represents seconds of UTC time since Unix epoch
    // 1970-01-01T00:00:00Z. Must be from 0001-01-01T00:00:00Z to
    // 9999-12-31T23:59:59Z inclusive.
    seconds: number;

    // Non-negative fractions of a second at nanosecond resolution. Negative
    // second values with fractions must still have non-negative nanos values
    // that count forward in time. Must be from 0 to 999,999,999
    // inclusive.
    nanos: number;
}

export function GetNodePendingResource(node: ClusterNode, resource: 'CPU' | 'GPU' | 'VRAM' | 'Memory'): number {
    if (resource == 'CPU') {
        return node.pendingCpu || 0;
    } else if (resource == 'GPU') {
        return node.pendingGpu || 0;
    } else if (resource == 'VRAM') {
        return node.pendingVRAM || 0;
    } else {
        return node.pendingMemory || 0;
    }
}

export function GetNodeAllocatedResource(node: ClusterNode, resource: 'CPU' | 'GPU' | 'VRAM' | 'Memory'): number {
    if (resource == 'CPU') {
        return node.allocatedCpu || 0;
    } else if (resource == 'GPU') {
        return node.allocatedGpu || 0;
    } else if (resource == 'VRAM') {
        return node.allocatedVRAM || 0;
    } else {
        return node.allocatedMemory || 0;
    }
}

export function GetNodeSpecResource(node: ClusterNode, resource: 'CPU' | 'GPU' | 'VRAM' | 'Memory'): number {
    if (resource == 'CPU') {
        return node.specCpu;
    } else if (resource == 'GPU') {
        return node.specGpu;
    } else if (resource == 'VRAM') {
        return node.specVRAM;
    } else {
        return node.specMemory;
    }
}

export function GetNodeIdleResource(node: ClusterNode, resource: 'CPU' | 'GPU' | 'VRAM' | 'Memory'): number {
    if (resource == 'CPU') {
        return node.specCpu - (node.allocatedCpu || 0);
    } else if (resource == 'GPU') {
        return node.specGpu - (node.allocatedGpu || 0);
    } else if (resource == 'VRAM') {
        return node.specVRAM - (node.allocatedVRAM || 0);
    } else {
        return node.specMemory - (node.allocatedMemory || 0);
    }

    // return node.CapacityResources[resource] - node.AllocatedResources[resource];
}

export interface PodOrContainer {
    Name: string;
    Phase: string;
    Age: string;
    IP: string;
    Valid: boolean;
    Type: string;
}

export interface VirtualGpuInfo {
    totalVirtualGPUs: number; // Total available vGPUs on the node.
    allocatedVirtualGPUs: number; // Number of allocated vGPUs on the node.
    freeVirtualGPUs: number; // Free (i.e., idle) vGPUs on the node.
}
