import '@patternfly/react-core/dist/styles/base.css';

import React, { createContext, useRef } from 'react';
import { Flex, FlexItem, Grid, GridItem, PageSection, Text, TextVariants, gridSpans } from '@patternfly/react-core';

import { KernelList, KernelSpecList, KubernetesNodeList, LogViewCard, WorkloadCard } from '@app/Components/Cards/';
import {
    DistributedJupyterKernel,
    JupyterKernelReplica,
    KubernetesNode,
    VirtualGpuInfo,
    WORKLOAD_STATE_RUNNING,
    Workload,
    WorkloadPreset,
} from '@app/Data';
import { AdjustVirtualGPUsModal, MigrationModal, RegisterWorkloadModal } from '@app/Components/Modals';

import { v4 as uuidv4 } from 'uuid';
import { useWorkloads } from '@providers/WorkloadProvider';

import toast, { Toaster } from 'react-hot-toast';
import { useKernels } from '@app/Providers';

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
    const [isRegisterWorkloadModalOpen, setIsRegisterWorkloadModalOpen] = React.useState(false);
    const [isAdjustVirtualGPUsModalOpen, setIsAdjustVirtualGPUsModalOpen] = React.useState(false);
    const [isMigrateModalOpen, setIsMigrateModalOpen] = React.useState(false);
    const [migrateKernel, setMigrateKernel] = React.useState<DistributedJupyterKernel | null>(null);
    const [migrateReplica, setMigrateReplica] = React.useState<JupyterKernelReplica | null>(null);
    const [adjustVirtualGPUsNode, setAdjustVirtualGPUsNode] = React.useState<KubernetesNode | null>(null);

    const defaultWorkloadTitle = useRef(uuidv4());

    const { kernels } = useKernels();
    const { workloads, sendJsonMessage } = useWorkloads();

    const [workloadItemsPerPage, setWorkloadItemsPerPage] = React.useState(3);
    const [kernelItemsPerPage, setKernelItemsPerPage] = React.useState(3);
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

    const onConfirmRegisterWorkload = (
        workloadName: string,
        selectedPreset: WorkloadPreset,
        workloadSeedString: string,
        debugLoggingEnabled: boolean,
    ) => {
        toast('Registering workload now.', {
            icon: '🛈',
        });

        console.log(
            "New workload '%s' registered by user with preset:\n%s",
            workloadName,
            JSON.stringify(selectedPreset),
        );
        setIsRegisterWorkloadModalOpen(false);

        let workloadSeed = -1;

        if (workloadSeedString != '') {
            workloadSeed = parseInt(workloadSeedString);
        }

        const messageId: string = uuidv4();
        sendJsonMessage({
            op: 'register_workload',
            msg_id: messageId,
            workloadRegistrationRequest: {
                adjust_gpu_reservations: false,
                seed: workloadSeed,
                key: selectedPreset.key,
                name: workloadName,
                debug_logging: debugLoggingEnabled,
            },
        });
    };

    const onCancelStartWorkload = () => {
        console.log('New workload cancelled by user before starting.');
        setIsRegisterWorkloadModalOpen(false);
    };

    const openMigrationModal = (kernel: DistributedJupyterKernel, replica: JupyterKernelReplica) => {
        setMigrateReplica(replica);
        setMigrateKernel(kernel);
        setIsMigrateModalOpen(true);
    };

    const toggleDebugLogs = (workloadId: string, enabled: boolean) => {
        if (enabled) {
            console.log("Enabling debug logging for workload '%s'", workloadId);
        } else {
            console.log("Disabling debug logging for workload '%s'", workloadId);
        }

        const messageId: string = uuidv4();
        // const callback = (result: SingleWorkloadResponse) => {
        //     setWorkloads(new Map(workloads.set(result.workload.id, result.workload)));
        // };
        // websocketCallbacks.current.set(messageId, callback);
        sendJsonMessage({
            op: 'toggle_debug_logs',
            msg_id: messageId,
            workload_id: workloadId,
            enabled: enabled,
        });
    };

    const onStartWorkloadClicked = (workload: Workload) => {
        toast(() => (
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
                <FlexItem>
                    <b>Starting workload {workload.name}</b>
                </FlexItem>
                <FlexItem>
                    <Text component={TextVariants.small}>
                        <b>Workload ID: </b>
                        {workload.id}
                    </Text>
                </FlexItem>
            </Flex>
        ));

        console.log(`Starting workload '${workload.name}' (ID=${workload.id})`);

        const messageId: string = uuidv4();
        // const callback = (result: SingleWorkloadResponse) => {
        //     setWorkloads(new Map(workloads.set(result.workload.id, result.workload)));
        // };
        // websocketCallbacks.current.set(messageId, callback);
        sendJsonMessage({
            op: 'start_workload',
            msg_id: messageId,
            workload_id: workload.id,
        });
    };

    const onStopWorkloadClicked = (workload: Workload) => {
        toast('Stop workload');

        console.log("Stopping workload '%s' (ID=%s)", workload.name, workload.id);

        const messageId: string = uuidv4();
        // const callback = (result: SingleWorkloadResponse) => {
        //     setWorkloads(new Map(workloads.set(result.workload.id, result.workload)));
        // };
        // websocketCallbacks.current.set(messageId, callback);
        sendJsonMessage({
            op: 'stop_workload',
            msg_id: messageId,
            workload_id: workload.id,
        });
    };

    const onStopAllWorkloadsClicked = () => {
        toast('Stopping all workload');

        const activeWorkloadsIDs: string[] = [];
        workloads.forEach((workload: Workload) => {
            if (workload.workload_state == WORKLOAD_STATE_RUNNING) {
                activeWorkloadsIDs.push(workload.id);
            }
        });

        const messageId: string = uuidv4();
        // const callback = (result: WorkloadsResponse) => {
        //     result.workloads.forEach((workload: Workload) => {
        //         setWorkloads((w) => new Map(w.set(workload.id, workload)));
        //     });
        // };
        // websocketCallbacks.current.set(messageId, callback);
        sendJsonMessage({
            op: 'stop_workloads',
            msg_id: messageId,
            workload_ids: activeWorkloadsIDs,
        });
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
        const heightFactor: number = Math.min(workloads.length, workloadItemsPerPage);
        if (heightFactor <= 2) {
            return 1 as gridSpans;
        } else {
            return 2 as gridSpans;
        }
    };

    const getKernelCardRowspan = () => {
        const heightFactor: number = Math.min(kernels.length, kernelItemsPerPage);
        if (heightFactor <= 2) {
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
                            heightFactor: kernelItemsPerPage,
                            setHeightFactor: (newHeight: number) => setKernelItemsPerPage(newHeight),
                        }}
                    >
                        <KernelList kernelsPerPage={3} openMigrationModal={openMigrationModal} />
                    </KernelHeightFactorContext.Provider>
                </GridItem>
                <GridItem span={6} rowSpan={getWorkloadCardRowspan()}>
                    <WorkloadsHeightFactorContext.Provider
                        value={{
                            heightFactor: workloadItemsPerPage,
                            setHeightFactor: (value: number) => setWorkloadItemsPerPage(value),
                        }}
                    >
                        <WorkloadCard
                            workloadsPerPage={3}
                            toggleDebugLogs={toggleDebugLogs}
                            onStartWorkloadClicked={onStartWorkloadClicked}
                            onStopWorkloadClicked={onStopWorkloadClicked}
                            onStopAllWorkloadsClicked={onStopAllWorkloadsClicked}
                            onLaunchWorkloadClicked={() => {
                                defaultWorkloadTitle.current = uuidv4(); // Regenerate the default workload title as we're opening the modal again.
                                setIsRegisterWorkloadModalOpen(true);
                            }}
                        />
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
            <RegisterWorkloadModal
                isOpen={isRegisterWorkloadModalOpen}
                onClose={onCancelStartWorkload}
                onConfirm={onConfirmRegisterWorkload}
                defaultWorkloadTitle={defaultWorkloadTitle.current}
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
