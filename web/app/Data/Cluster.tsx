// The reason we use ClusterNode is that we need a platform-agnostic node type.
// The fields of ClusterNode correspond to VirtualDockerNode and KubernetesNode, which have the same JSON fields.
// The corresponding protobuffers structs for those types are not interchangeable. So, we convert them
// to a more generic format.
//
// TODO: Though it might make the most sense to have a single ClusterNode type in the Golang backend, rather than two
//       "generic" structs for docker and kubernetes nodes. (They're generic in the sense that they have the same JSON
//       fields. But then, why not just unify the protobuffer structs under a single type in the backend?)
export interface ClusterNode {
    NodeId: string;
    NodeName: string;
    PodsOrContainers: PodOrContainer[];
    Age: string;
    CreatedAt: number;
    IP: string;
    AllocatedResources: Map<string, number>;
    IdleResources: Map<string, number>;
    PendingResources: Map<string, number>;
    CapacityResources: Map<string, number>;
    Enabled: boolean;
}

// export interface ClusterNode {
//     nodeId: string;
//     containers?: PodOrContainer[];
//     specCpu: number;
//     specMemory: number;
//     specGpu: number;
//     specVRAM: number;
//     allocatedCpu?: number;
//     allocatedMemory?: number;
//     allocatedGpu?: number;
//     allocatedVRAM?: number;
//     pendingCpu?: number;
//     pendingMemory?: number;
//     pendingGpu?: number;
//     pendingVRAM?: number;
//     nodeName: string;
//     address: string;
//     createdAt: ProtoTimestamp;
// }

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

// I keep changing the struct definition, so this function makes it, so I only have to update
// things in one place (here) rather than everywhere the uses the NodeName field.
export function GetNodeName(node: ClusterNode): string {
    return node.NodeName;
}

// I keep changing the struct definition, so this function makes it, so I only have to update
// things in one place (here) rather than everywhere the uses the NodeId field.
export function GetNodeId(node: ClusterNode): string {
    return node.NodeId;
}

export function GetNodePendingResource(node: ClusterNode, resource: 'CPU' | 'GPU' | 'VRAM' | 'Memory'): number {
    // if (resource == 'CPU') {
    //     return node.pendingCpu || 0;
    // } else if (resource == 'GPU') {
    //     return node.pendingGpu || 0;
    // } else if (resource == 'VRAM') {
    //     return node.pendingVRAM || 0;
    // } else {
    //     return node.pendingMemory || 0;
    // }

    return node.PendingResources[resource];
}

export function GetNodeAllocatedResource(node: ClusterNode, resource: 'CPU' | 'GPU' | 'VRAM' | 'Memory'): number {
    // if (resource == 'CPU') {
    //     return node.allocatedCpu || 0;
    // } else if (resource == 'GPU') {
    //     return node.allocatedGpu || 0;
    // } else if (resource == 'VRAM') {
    //     return node.allocatedVRAM || 0;
    // } else {
    //     return node.allocatedMemory || 0;
    // }

    return node.AllocatedResources[resource];
}

export function GetNodeSpecResource(node: ClusterNode, resource: 'CPU' | 'GPU' | 'VRAM' | 'Memory'): number {
    // if (resource == 'CPU') {
    //     return node.specCpu;
    // } else if (resource == 'GPU') {
    //     return node.specGpu;
    // } else if (resource == 'VRAM') {
    //     return node.specVRAM;
    // } else {
    //     return node.specMemory;
    // }

    return node.CapacityResources[resource];
}

export function GetNodeIdleResource(node: ClusterNode, resource: 'CPU' | 'GPU' | 'VRAM' | 'Memory'): number {
    // if (resource == 'CPU') {
    //     return node.specCpu - (node.allocatedCpu || 0);
    // } else if (resource == 'GPU') {
    //     return node.specGpu - (node.allocatedGpu || 0);
    // } else if (resource == 'VRAM') {
    //     return node.specVRAM - (node.allocatedVRAM || 0);
    // } else {
    //     return node.specMemory - (node.allocatedMemory || 0);
    // }

    return node.CapacityResources[resource] - node.AllocatedResources[resource];
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
