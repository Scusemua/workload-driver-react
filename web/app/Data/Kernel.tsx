import { IKernelConnection } from '@jupyterlab/services/lib/kernel/kernel';

interface DistributedJupyterKernel {
    kernelId: string;
    numReplicas: number;
    status: string;
    aggregateBusyStatus: string;
    replicas: JupyterKernelReplica[];
    kernel?: IKernelConnection;
}

interface JupyterKernelReplica {
    kernelId: string;
    replicaId: number;
    podId: string;
    nodeId: string;
}

interface KernelSpec {
    name: string;
    displayName: string;
    language: string;
    interruptMode: string;
    kernelProvisioner: KernelProvisioner;
    argV: string[];
}

interface KernelProvisioner {
    name: string;
    gateway: string;
    valid: boolean;
}

export type { DistributedJupyterKernel as DistributedJupyterKernel };
export type { JupyterKernelReplica as JupyterKernelReplica };
export type { KernelSpec as KernelSpec };
export type { KernelSpec as KernelProvisioner };
