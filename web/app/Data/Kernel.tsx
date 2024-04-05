import { IKernelConnection } from '@jupyterlab/services/lib/kernel/kernel';

interface DistributedJupyterKernel<T extends JupyterKernelReplica> {
    kernelId: string;
    numReplicas: number;
    status: string;
    aggregateBusyStatus: string;
    kernelSpec: DistributedKernelSpec;
    replicas: T[];
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
    resourceSpec: ResourceSpec;
}

interface ResourceSpec {
    cpu: number;
    memory: number;
    gpu: number;
}

// The KernelSpec used within JupyterServer when provisioning kernels.
interface JupyterKernelSpecWrapper {
    /**
     * The name of the kernel spec.
     */
    name: string;
    spec: JupyterKernelSpec;
}

interface JupyterKernelSpec {
    /**
     * The name of the language of the kernel.
     */
    language: string;
    /**
     * The kernel’s name as it should be displayed in the UI.
     */
    display_name: string;
    interrupt_mode: string;
    metadata: JupyterKernelSpecMetadata;
    argv: string[];
}

interface JupyterKernelSpecMetadata {
    kernel_provisioner: KernelProvisioner;
}

interface KernelProvisioner {
    provisioner_name: string;
    config: KernelProvisionerConfig;
}

interface KernelProvisionerConfig {
    gateway: string;
}

export type { DistributedJupyterKernel as DistributedJupyterKernel };
export type { JupyterKernelReplica as JupyterKernelReplica };
export type { JupyterKernelSpecWrapper as JupyterKernelSpecWrapper };
export type { JupyterKernelSpec as JupyterKernelSpec };
export type { JupyterKernelSpecMetadata as JupyterKernelSpecMetadata };
export type { KernelProvisioner as KernelProvisioner };
export type { KernelProvisionerConfig as KernelProvisionerConfig };
export type { ResourceSpec as ResourceSpec };
