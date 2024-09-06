import '@patternfly/react-core/dist/styles/base.css';

import { KernelList, KernelSpecList, NodeList, UtilizationCard, WorkloadCard } from '@app/Components/Cards/';
import { MigrationModal } from '@app/Components/Modals';
import { DistributedJupyterKernel, JupyterKernelReplica } from '@app/Data';
import { Grid, GridItem, gridSpans, PageSection } from '@patternfly/react-core';

import React, { createContext } from 'react';

import toast from 'react-hot-toast';

export interface DashboardProps {}

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

const Dashboard: React.FunctionComponent<DashboardProps> = () => {
    const [isMigrateModalOpen, setIsMigrateModalOpen] = React.useState(false);
    const [migrateKernel, setMigrateKernel] = React.useState<DistributedJupyterKernel | null>(null);
    const [migrateReplica, setMigrateReplica] = React.useState<JupyterKernelReplica | null>(null);

    const [workloadHeightFactor, setWorkloadHeightFactor] = React.useState(3);
    const [kernelHeightFactor, setKernelHeightFactor] = React.useState(3);
    const [kubeNodeHeightFactor, setKubeNodeHeightFactor] = React.useState(3);

    const onConfirmMigrateReplica = (
        targetReplica: JupyterKernelReplica,
        targetKernel: DistributedJupyterKernel,
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
                {/* <GridItem span={12} rowSpan={8}>
                    <LogViewCard />
                </GridItem> */}
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
                  <NodeList
                    isDashboardList={true}
                    hideAdjustVirtualGPUsButton={false}
                    hideControlPlaneNode={true}
                    nodesPerPage={4}
                    selectableViaCheckboxes={false}
                    displayNodeToggleSwitch={true}
                  />
                </NodeHeightFactorContext.Provider>
              </GridItem>
            </Grid>
            <MigrationModal
                isOpen={isMigrateModalOpen}
                onClose={closeMigrateReplicaModal}
                onConfirm={onConfirmMigrateReplica}
                targetKernel={migrateKernel}
                targetReplica={migrateReplica}
            />
        </PageSection>
    );
};

export { Dashboard };
