import '@patternfly/react-core/dist/styles/base.css';

import React, { createContext } from 'react';
import { Grid, GridItem, PageSection, gridSpans } from '@patternfly/react-core';

import { KernelList, KernelSpecList, KubernetesNodeList, LogViewCard, WorkloadCard } from '@app/Components/Cards/';
import { DistributedJupyterKernel, JupyterKernelReplica, KubernetesNode, VirtualGpuInfo } from '@app/Data';
import { AdjustVirtualGPUsModal, MigrationModal } from '@app/Components/Modals';

import toast, { Toaster } from 'react-hot-toast';

export interface DashboardProps {}

export type HeightFactorContext = {
    heightFactor: number;
    setHeightFactor: (value: number) => void;
};

export const KernelHeightFactorContext = createContext<HeightFactorContext>({
    heightFactor: 1,
    setHeightFactor: () => {},
});
export const KubernetesNodeHeightFactorContext = createContext<HeightFactorContext>({
    heightFactor: 3,
    setHeightFactor: () => {},
});
export const WorkloadsHeightFactorContext = createContext<HeightFactorContext>({
    heightFactor: 3,
    setHeightFactor: () => {},
});

const Dashboard: React.FunctionComponent<DashboardProps> = () => {
    const [isAdjustVirtualGPUsModalOpen, setIsAdjustVirtualGPUsModalOpen] = React.useState(false);
    const [isMigrateModalOpen, setIsMigrateModalOpen] = React.useState(false);
    const [migrateKernel, setMigrateKernel] = React.useState<DistributedJupyterKernel<JupyterKernelReplica> | null>(
        null,
    );
    const [migrateReplica, setMigrateReplica] = React.useState<JupyterKernelReplica | null>(null);
    const [adjustVirtualGPUsNode, setAdjustVirtualGPUsNode] = React.useState<KubernetesNode | null>(null);

    const [workloadHeightFactor, setWorkloadHeightFactor] = React.useState(3);
    const [kernelHeightFactor, setKernelHeightFactor] = React.useState(3);
    const [kubeNodeHeightFactor, setKubeNodeHeightFactor] = React.useState(3);

    const onConfirmMigrateReplica = (
        targetReplica: JupyterKernelReplica,
        targetKernel: DistributedJupyterKernel<JupyterKernelReplica>,
        targetNodeId: string,
    ) => {
        const requestOptions = {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                // 'Cache-Control': 'no-cache, no-transform, no-store',
            },
            body: JSON.stringify({
                targetReplica: {
                    replicaId: targetReplica.replicaId,
                    kernelId: targetKernel.kernelId,
                },
                targetNodeId: targetNodeId,
            }),
        };

        targetReplica.isMigrating = true;

        toast('Migrating kernel replica');

        fetch('/api/migrate', requestOptions).then((response) => {
            console.log(
                'Received response for migration operation of replica %d of kernel %s: %s',
                targetReplica.replicaId,
                targetKernel.kernelId,
                JSON.stringify(response),
            );
        });

        // Close the migration modal and reset its state.
        setIsMigrateModalOpen(false);
        setMigrateReplica(null);
        setMigrateKernel(null);
    };

    const closeMigrateReplicaModal = () => {
        // Close the migration modal and reset its state.
        setIsMigrateModalOpen(false);
        setMigrateReplica(null);
        setMigrateKernel(null);
    };

    const openMigrationModal = (
        kernel: DistributedJupyterKernel<JupyterKernelReplica>,
        replica: JupyterKernelReplica,
    ) => {
        setMigrateReplica(replica);
        setMigrateKernel(kernel);
        setIsMigrateModalOpen(true);
    };

    const onAdjustVirtualGPUsClicked = (node: KubernetesNode) => {
        setAdjustVirtualGPUsNode(node);
        setIsAdjustVirtualGPUsModalOpen(true);
    };

    const closeAdjustVirtualGPUsModal = () => {
        setIsAdjustVirtualGPUsModalOpen(false);
        setAdjustVirtualGPUsNode(null);
    };

    async function doAdjustVirtualGPUs(value: number) {
        toast('Adjusting vGPUs');

        if (adjustVirtualGPUsNode == null) {
            console.error("Field 'adjustVirtualGPUsNode' is null...");
            closeAdjustVirtualGPUsModal();
            return;
        }

        if (Number.isNaN(value)) {
            console.error('Specified value is NaN...');
            closeAdjustVirtualGPUsModal();
            return;
        }

        if (adjustVirtualGPUsNode.CapacityVGPUs == value) {
            console.log('Adjusted vGPUs value is same as current value. Doing nothing.');
            closeAdjustVirtualGPUsModal();
            return;
        }

        const requestOptions = {
            method: 'PATCH',
            headers: {
                'Content-Type': 'application/json',
                // 'Cache-Control': 'no-cache, no-transform, no-store'
            },
            body: JSON.stringify({
                value: value,
                kubernetesNodeName: adjustVirtualGPUsNode?.NodeId,
            }),
        };

        console.log(`Attempting to set vGPUs on node ${adjustVirtualGPUsNode?.NodeId} to ${value}`);

        const response: Response = await fetch('api/vgpus', requestOptions);
        const virtualGpuInfo: VirtualGpuInfo = await response.json().catch((reason) => {
            console.error(
                `Failed to update vGPUs for node ${adjustVirtualGPUsNode.NodeId} because: ${JSON.stringify(reason)}`,
            );
        });
        console.log(`Received updated virtual GPU info: ${JSON.stringify(virtualGpuInfo)}`);
    }

    const getWorkloadCardRowspan = () => {
        if (workloadHeightFactor <= 2) {
            return 1 as gridSpans;
        } else {
            return 2 as gridSpans;
        }
    };

    const getKernelCardRowspan = () => {
        if (kernelHeightFactor <= 2) {
            return 1 as gridSpans;
        } else {
            return 2 as gridSpans;
        }
    };

    const getKubeNodeCardRowspan = () => {
        if (kubeNodeHeightFactor <= 2) {
            return 1 as gridSpans;
        } else {
            return 2 as gridSpans;
        }
    };

    return (
        <PageSection>
            <Grid hasGutter>
                <GridItem span={12} rowSpan={8}>
                    <LogViewCard />
                </GridItem>
                <GridItem span={6} rowSpan={getKernelCardRowspan()}>
                    <KernelHeightFactorContext.Provider
                        value={{
                            heightFactor: kernelHeightFactor,
                            setHeightFactor: (newHeight: number) => setKernelHeightFactor(newHeight),
                        }}
                    >
                        <KernelList kernelsPerPage={3} openMigrationModal={openMigrationModal} />
                    </KernelHeightFactorContext.Provider>
                </GridItem>
                <GridItem span={6} rowSpan={getWorkloadCardRowspan()}>
                    <WorkloadsHeightFactorContext.Provider
                        value={{
                            heightFactor: workloadHeightFactor,
                            setHeightFactor: (value: number) => setWorkloadHeightFactor(value),
                        }}
                    >
                        <WorkloadCard workloadsPerPage={3} />
                    </WorkloadsHeightFactorContext.Provider>
                </GridItem>
                <GridItem span={6} rowSpan={1}>
                    <KernelSpecList />
                </GridItem>
                <GridItem span={6} rowSpan={getKubeNodeCardRowspan()}>
                    <KubernetesNodeHeightFactorContext.Provider
                        value={{
                            heightFactor: kubeNodeHeightFactor,
                            setHeightFactor: (value: number) => setKubeNodeHeightFactor(value),
                        }}
                    >
                        <KubernetesNodeList
                            isDashboardList={true}
                            hideAdjustVirtualGPUsButton={false}
                            onAdjustVirtualGPUsClicked={onAdjustVirtualGPUsClicked}
                            hideControlPlaneNode={true}
                            nodesPerPage={3}
                            selectableViaCheckboxes={false}
                            displayNodeToggleSwitch={true}
                        />
                    </KubernetesNodeHeightFactorContext.Provider>
                </GridItem>
            </Grid>
            <MigrationModal
                isOpen={isMigrateModalOpen}
                onClose={closeMigrateReplicaModal}
                onConfirm={onConfirmMigrateReplica}
                targetKernel={migrateKernel}
                targetReplica={migrateReplica}
            />
            <AdjustVirtualGPUsModal
                isOpen={isAdjustVirtualGPUsModalOpen}
                onClose={closeAdjustVirtualGPUsModal}
                onConfirm={doAdjustVirtualGPUs}
                node={adjustVirtualGPUsNode}
            />
            <Toaster position="bottom-right" />
        </PageSection>
    );
};

export { Dashboard };
