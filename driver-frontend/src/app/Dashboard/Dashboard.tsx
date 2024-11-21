import '@patternfly/react-core/dist/styles/base.css';

import { KernelListCard, KernelSpecList, NodeListCard, UtilizationCard, WorkloadCard } from '@Components/Cards/';
import { MigrationModal } from '@Components/Modals';
import { Grid, GridItem, PageSection, gridSpans } from '@patternfly/react-core';
import { MigrateKernelReplica } from '@src/Components';

import React, { createContext } from 'react';
import { DistributedJupyterKernel, JupyterKernelReplica } from 'src/Data';

export type HeightFactorContext = {
    heightFactor: number;
    setHeightFactor: (value: number) => void;
};

export const KernelHeightFactorContext = createContext<HeightFactorContext>({
    heightFactor: 1,
    setHeightFactor: () => {},
});
export const NodeHeightFactorContext = createContext<HeightFactorContext>({
    heightFactor: 3,
    setHeightFactor: () => {},
});
export const WorkloadsHeightFactorContext = createContext<HeightFactorContext>({
    heightFactor: 3,
    setHeightFactor: () => {},
});

const Dashboard: React.FunctionComponent = () => {
    const [isMigrateModalOpen, setIsMigrateModalOpen] = React.useState(false);
    const [migrateKernel, setMigrateKernel] = React.useState<DistributedJupyterKernel | null>(null);
    const [migrateReplica, setMigrateReplica] = React.useState<JupyterKernelReplica | null>(null);

    const [workloadHeightFactor, setWorkloadHeightFactor] = React.useState(3);
    const [kernelHeightFactor, setKernelHeightFactor] = React.useState(3);
    const [kubeNodeHeightFactor, setKubeNodeHeightFactor] = React.useState(3);

    const onConfirmMigrateReplica = async (
        targetReplica: JupyterKernelReplica,
        targetKernel: DistributedJupyterKernel,
        targetNodeId: string,
    ) => {
        // Close the migration modal and reset its state.
        setIsMigrateModalOpen(false);
        setMigrateReplica(null);
        setMigrateKernel(null);

        await MigrateKernelReplica(targetReplica, targetKernel, targetNodeId);
    };

    const closeMigrateReplicaModal = () => {
        // Close the migration modal and reset its state.
        setIsMigrateModalOpen(false);
        setMigrateReplica(null);
        setMigrateKernel(null);
    };

    const openMigrationModal = (kernel: DistributedJupyterKernel, replica: JupyterKernelReplica) => {
        setMigrateReplica(replica);
        setMigrateKernel(kernel);
        setIsMigrateModalOpen(true);
    };

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
                <GridItem span={6} rowSpan={getKernelCardRowspan()}>
                    <KernelHeightFactorContext.Provider
                        value={{
                            heightFactor: kernelHeightFactor,
                            setHeightFactor: (newHeight: number) => setKernelHeightFactor(newHeight),
                        }}
                    >
                        <KernelListCard
                            kernelsPerPage={3}
                            openMigrationModal={openMigrationModal}
                            perPageOption={[
                                {
                                    title: '1 kernels',
                                    value: 1,
                                },
                                {
                                    title: '2 kernels',
                                    value: 2,
                                },
                                {
                                    title: '3 kernels',
                                    value: 3,
                                },
                                {
                                    title: '5 kernels',
                                    value: 5,
                                },
                            ]}
                        />
                    </KernelHeightFactorContext.Provider>
                </GridItem>
                <GridItem span={6} rowSpan={getWorkloadCardRowspan()}>
                    <WorkloadsHeightFactorContext.Provider
                        value={{
                            heightFactor: workloadHeightFactor,
                            setHeightFactor: (value: number) => setWorkloadHeightFactor(value),
                        }}
                    >
                        <WorkloadCard
                            useCreationModal={true}
                            workloadsPerPage={3}
                            inspectInModal={true}
                            perPageOption={[
                                {
                                    title: '1 workloads',
                                    value: 1,
                                },
                                {
                                    title: '2 workloads',
                                    value: 2,
                                },
                                {
                                    title: '3 workloads',
                                    value: 3,
                                },
                            ]}
                        />
                    </WorkloadsHeightFactorContext.Provider>
                </GridItem>
                <GridItem span={6} rowSpan={1}>
                    <KernelSpecList />
                </GridItem>
                <GridItem span={6} rowSpan={1}>
                    <UtilizationCard chartHeight={320} chartWidth={400} />
                </GridItem>
                <GridItem span={12} rowSpan={getKubeNodeCardRowspan()}>
                    <NodeHeightFactorContext.Provider
                        value={{
                            heightFactor: kubeNodeHeightFactor,
                            setHeightFactor: (value: number) => setKubeNodeHeightFactor(value),
                        }}
                    >
                        <NodeListCard
                            isDashboardList={true}
                            hideAdjustVirtualGPUsButton={false}
                            hideControlPlaneNode={true}
                            nodesPerPage={10}
                            selectableViaCheckboxes={false}
                            displayNodeToggleSwitch={true}
                        />
                    </NodeHeightFactorContext.Provider>
                </GridItem>
                {/*<GridItem span={12} rowSpan={2}>*/}
                {/*    <DockerLogViewCard />*/}
                {/*</GridItem>*/}
            </Grid>
            {migrateKernel && migrateReplica && (
                <MigrationModal
                    isOpen={isMigrateModalOpen}
                    onClose={closeMigrateReplicaModal}
                    onConfirm={onConfirmMigrateReplica}
                    targetKernel={migrateKernel}
                    targetReplica={migrateReplica}
                />
            )}
        </PageSection>
    );
};

export { Dashboard };
