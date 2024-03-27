import '@patternfly/react-core/dist/styles/base.css';

import React, { useRef } from 'react';
import { Grid, GridItem, PageSection, gridSpans } from '@patternfly/react-core';

import { KernelList, KernelSpecList, KubernetesNodeList, WorkloadCard } from '@app/Components/Cards/';
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
import { useNodes } from '@providers/NodeProvider';
import { useKernels } from '@app/Providers/KernelProvider';

export interface DashboardProps {}

const Dashboard: React.FunctionComponent<DashboardProps> = () => {
    const [isRegisterWorkloadModalOpen, setIsRegisterWorkloadModalOpen] = React.useState(false);
    const [isAdjustVirtualGPUsModalOpen, setIsAdjustVirtualGPUsModalOpen] = React.useState(false);
    const [isMigrateModalOpen, setIsMigrateModalOpen] = React.useState(false);
    const [migrateKernel, setMigrateKernel] = React.useState<DistributedJupyterKernel | null>(null);
    const [migrateReplica, setMigrateReplica] = React.useState<JupyterKernelReplica | null>(null);
    const [adjustVirtualGPUsNode, setAdjustVirtualGPUsNode] = React.useState<KubernetesNode | null>(null);

    const { nodes } = useNodes();
    const { kernels } = useKernels();
    const { workloads, sendJsonMessage } = useWorkloads();

    // const { sendJsonMessage, lastMessage, lastJsonMessage } = useWebSocket<Record<string, unknown>>(
    //     'ws://localhost:8000/workload',
    //     {
    //         share: false,
    //         shouldReconnect: () => true,
    //     },
    // );

    // Run when the connection state (readyState) changes.
    // useEffect(() => {
    //     console.log('Connection state changed: %s', readyState.toString());
    //     if (readyState === ReadyState.OPEN) {
    //         sendJsonMessage({
    //             event: 'subscribe',
    //             data: {
    //                 channel: 'general-chatroom',
    //             },
    //         });
    //     }
    // }, [readyState]);

    // const handleMessage = useCallback(
    //     (message: Record<string, unknown>) => {
    //         console.log(`Got a new message: ${JSON.stringify(message)}`);

    //         const handleActiveWorkloadsUpdate = (updatedWorkloads: Workload[]) => {
    //             console.log('Received update about %d active workload(s).', updatedWorkloads.length);

    //             updatedWorkloads.forEach((workload: Workload) => {
    //                 setWorkloads((w) => new Map(w.set(workload.id, workload)));
    //             });

    //             if (workloads.size > 0) {
    //                 setWorkloadCardRowspan(2);
    //             } else {
    //                 setWorkloadCardRowspan(1);
    //             }
    //         };

    //         // If there is a callback, then call it.
    //         if (message) {
    //             if (websocketCallbacks.current.has(message['msg_id'])) {
    //                 console.log(`Found callback for message ${message['msg_id']}`);
    //                 websocketCallbacks.current.get(message['msg_id'])(message);
    //             } else {
    //                 console.log(`No callback found for message ${message['msg_id']}`);
    //                 const op = message['op'];

    //                 if (op == 'active_workloads_update') {
    //                     const updatedWorkloads: Workload[] | unknown = message['updated_workloads'];
    //                     if (!updatedWorkloads) {
    //                         throw new Error("Unexpected response for 'updated_workloads' key.");
    //                     }
    //                     handleActiveWorkloadsUpdate(updatedWorkloads as Workload[]);
    //                 }
    //             }
    //         }
    //     },
    //     [workloads.size],
    // );

    // Run when a new WebSocket message is received (lastJsonMessage).
    // useEffect(() => {
    //     handleMessage(lastJsonMessage);
    // }, [lastJsonMessage, handleMessage]);

    // useEffect(() => {
    //     if (lastMessage && lastMessage.data) {
    //         const promise: Promise<string> = lastMessage.data.text();
    //         promise.then((data) => {
    //             console.log(data);
    //             const messageJson: Record<string, unknown> = JSON.parse(data);
    //             handleMessage(messageJson);
    //         });
    //     }
    // }, [lastMessage, handleMessage]);

    /**
     * The following references are used to handle the fact that network responses can return at random/arbitrary/misordered times.
     * We ignore network responses except when we're expecting one.
     */

    const defaultWorkloadTitle = useRef(uuidv4());

    // /**
    //  * Retrieve the current workloads from the backend.
    //  */
    // const fetchWorkloads = useCallback(
    //     (callback: (presets: Workload[]) => void | undefined) => {
    //         const startTime = performance.now();
    //         try {
    //             console.log('Refreshing workloads.');

    //             const messageId: string = uuidv4();
    //             const onResponse = (response: WorkloadsResponse) => {
    //                 const respWorkloads: Workload[] = response.workloads;

    //                 if (!ignoreResponseForWorkloads.current) {
    //                     setWorkloads(new Map());
    //                     respWorkloads.forEach((workload: Workload) => {
    //                         setWorkloads((w) => new Map(w.set(workload.id, workload)));
    //                     });
    //                     console.log(
    //                         'Successfully refreshed workloads. Discovered %d workload(s):\n%s',
    //                         respWorkloads.length,
    //                         JSON.stringify(respWorkloads),
    //                     );

    //                     if (workloads.size > 0) {
    //                         setWorkloadCardRowspan(2);
    //                     } else {
    //                         setWorkloadCardRowspan(1);
    //                     }

    //                     if (callback != undefined) {
    //                         callback(respWorkloads);
    //                     }
    //                     ignoreResponseForWorkloads.current = true;
    //                 } else {
    //                     console.log("Refreshed workloads, but we're ignoring the response...");
    //                 }
    //             };
    //             websocketCallbacks.current.set(messageId, onResponse);
    //             sendJsonMessage({
    // op: 'get_workloads',
    // msg_id: messageId,
    //             });
    //         } catch (e) {
    //             console.error(e);
    //         }
    //         console.log(`Refresh workloads: ${(performance.now() - startTime).toFixed(4)} ms`);
    //     },
    //     [sendJsonMessage, workloads.size],
    // );

    // // Fetch the workloads from the backend.
    // useEffect(() => {
    //     ignoreResponseForWorkloads.current = false;
    //     fetchWorkloads(() => {});

    //     // Periodically refresh the Kubernetes nodes every `props.workloadPresetRefreshInterval` seconds, or when the user clicks the "refresh" button.
    //     setInterval(() => {
    //         ignoreResponseForWorkloads.current = false;
    //         fetchWorkloads(() => {});
    //     }, props.workloadRefreshInterval * 1000);

    //     return () => {
    //         ignoreResponseForWorkloads.current = true;
    //     };
    // }, [props.workloadRefreshInterval, fetchWorkloads]);

    const onConfirmMigrateReplica = (
        targetReplica: JupyterKernelReplica,
        targetKernel: DistributedJupyterKernel,
        targetNodeId: string,
    ) => {
        const requestOptions = {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                targetReplica: {
                    replicaId: targetReplica.replicaId,
                    kernelId: targetKernel.kernelId,
                },
                targetNodeId: targetNodeId,
            }),
        };

        targetReplica.isMigrating = true;

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
        // const callback = (result: SingleWorkloadResponse) => {
        //     console.log('Successfully registered workload %s', result.workload.id);
        //     setWorkloads(new Map(workloads.set(result.workload.id, result.workload)));

        //     if (workloads.size >= 1) {
        //         setWorkloadCardRowspan(2);
        //     } else {
        //         setWorkloadCardRowspan(1);
        //     }
        // };
        // websocketCallbacks.current.set(messageId, callback);
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
        console.log("Starting workload '%s' (ID=%s)", workload.name, workload.id);

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

    const getWorkloadCardRowspan = () => {
        if (workloads.length == 0) {
            return 1 as gridSpans;
        } else if (workloads.length == 1) {
            return 1 as gridSpans;
        }

        return 2 as gridSpans;
    };

    const getKernelCardRowspan = () => {
        if (kernels.length == 0) {
            return 1 as gridSpans;
        } else if (kernels.length == 1) {
            return 1 as gridSpans;
        }

        return 2 as gridSpans;
    };

    const onAdjustVirtualGPUsClicked = (node: KubernetesNode) => {
        setAdjustVirtualGPUsNode(node);
        setIsAdjustVirtualGPUsModalOpen(true);
    };

    const closeAdjustVirtualGPUsModal = () => {
        setIsAdjustVirtualGPUsModalOpen(false);
        setAdjustVirtualGPUsNode(null);
    };

    const doAdjustVirtualGPUs = (value: number) => {
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
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                value: value,
                kubernetesNodeName: adjustVirtualGPUsNode?.NodeId,
            }),
        };

        console.log(`Attempting to set vGPUs on node ${adjustVirtualGPUsNode?.NodeId} to ${value}`);

        fetch('api/vgpus', requestOptions).then((response) =>
            response
                .json()
                .catch((reason) => {
                    console.error(
                        `Failed to update vGPUs for node ${adjustVirtualGPUsNode.NodeId} because: ${JSON.stringify(
                            reason,
                        )}`,
                    );
                })
                .then((virtualGpuInfo: VirtualGpuInfo) => {
                    console.log(`Received updated virtual GPU info: ${JSON.stringify(virtualGpuInfo)}`);
                }),
        );

        closeAdjustVirtualGPUsModal();
    };

    return (
        <PageSection>
            <Grid hasGutter>
                <GridItem span={6} rowSpan={getKernelCardRowspan()}>
                    <KernelList kernelsPerPage={3} openMigrationModal={openMigrationModal} />
                </GridItem>
                <GridItem span={6} rowSpan={getWorkloadCardRowspan()}>
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
                </GridItem>
                <GridItem span={6} rowSpan={1}>
                    <KernelSpecList />
                </GridItem>
                <GridItem span={6} rowSpan={nodes.length == 0 ? 1 : 2}>
                    <KubernetesNodeList
                        hideAdjustVirtualGPUsButton={false}
                        onAdjustVirtualGPUsClicked={onAdjustVirtualGPUsClicked}
                        hideControlPlaneNode={true}
                        nodesPerPage={3}
                        selectableViaCheckboxes={false}
                        displayNodeToggleSwitch={true}
                    />
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
        </PageSection>
    );
};

export { Dashboard };
