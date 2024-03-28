import { IKernelConnection } from '@jupyterlab/services/lib/kernel/kernel';

interface DistributedJupyterKernel {
    kernelId: string;
    numReplicas: number;
    status: string;
    aggregateBusyStatus: string;
    replicas: JupyterKernelReplica[];
    kernelSpec: DistributedKernelSpec;
    kernel?: IKernelConnection;
}

interface JupyterKernelReplica {
    kernelId: string;
    replicaId: number;
    podId: string;
    nodeId: string;
    isMigrating: boolean;
}

// The KernelSpec used within the Distributed Notebook cluster.
interface DistributedKernelSpec {
    id: string;
    session: string;
    argv: string[];
    signatureScheme: string;
    key: string;
    resource: ResourceSpec;
}

interface ResourceSpec {
    cpu: number;
    memory: number;
    gpu: number;
}

// The KernelSpec used within JupyterServer when provisioning kernels.
interface JupyterKernelSpec {
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
export type { JupyterKernelSpec as KernelSpec };
export type { JupyterKernelSpec as KernelProvisioner };
