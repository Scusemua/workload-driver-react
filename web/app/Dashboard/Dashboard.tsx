import '@patternfly/react-core/dist/styles/base.css';

import React, { useCallback, useEffect, useRef } from 'react';
import { Grid, GridItem, PageSection } from '@patternfly/react-core';

import { KernelList, KernelSpecList, KubernetesNodeList, WorkloadCard } from '@app/Components';
import { DistributedJupyterKernel, JupyterKernelReplica, KubernetesNode, Workload, WorkloadPreset } from '@app/Data';
import { MigrationModal, RegisterWorkloadModal } from '@app/Components/Modals';

import useWebSocket from 'react-use-websocket';

import { v4 as uuidv4 } from 'uuid';

export interface DashboardProps {
    nodeRefreshInterval: number;
    workloadPresetRefreshInterval: number;
    workloadRefreshInterval: number;
}

function wait<T>(ms: number, value: T) {
    return new Promise<T>((resolve) => setTimeout(resolve, ms, value));
}

const Dashboard: React.FunctionComponent<DashboardProps> = (props: DashboardProps) => {
    const [nodes, setNodes] = React.useState<KubernetesNode[]>([]);
    const [workloads, setWorkloads] = React.useState(new Map());
    const [workloadPresets, setWorkloadPresets] = React.useState<WorkloadPreset[]>([]);
    const [isStartWorkloadModalOpen, setIsStartWorkloadOpen] = React.useState(false);
    const [isMigrateModalOpen, setIsMigrateModalOpen] = React.useState(false);
    const [migrateKernel, setMigrateKernel] = React.useState<DistributedJupyterKernel | null>(null);
    const [migrateReplica, setMigrateReplica] = React.useState<JupyterKernelReplica | null>(null);

    const websocketCallbacks = React.useRef(new Map());

    const { sendJsonMessage, lastJsonMessage } = useWebSocket<Record<string, unknown>>('ws://localhost:8000/workload', {
        share: false,
        shouldReconnect: () => true,
    });

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

    // Run when a new WebSocket message is received (lastJsonMessage).
    useEffect(() => {
        console.log(`Got a new message: ${JSON.stringify(lastJsonMessage)}`);

        const handleActiveWorkloadsUpdate = (updatedWorkloads: Workload[]) => {
            console.log('Received update about %d active workload(s).', updatedWorkloads.length);

            updatedWorkloads.forEach((workload: Workload) => {
                setWorkloads(new Map(workloads.set(workload.id, workload)));
            });
        };

        // If there is a callback, then call it.
        if (lastJsonMessage) {
            if (websocketCallbacks.current.has(lastJsonMessage['msg_id'])) {
                console.log(`Found callback for message ${lastJsonMessage['msg_id']}`);
                websocketCallbacks.current.get(lastJsonMessage['msg_id'])(lastJsonMessage);
            } else {
                console.log(`No callback found for message ${lastJsonMessage['msg_id']}`);
                const op = lastJsonMessage['op'];

                if (op == 'active_workloads_update') {
                    const updatedWorkloads: Workload[] | unknown = lastJsonMessage['updated_workloads'];
                    if (!updatedWorkloads) {
                        throw new Error("Unexpected response for 'updated_workloads' key.");
                    }
                    handleActiveWorkloadsUpdate(updatedWorkloads as Workload[]);
                }
            }
        }
    }, [workloads, lastJsonMessage]);

    /**
     * The following references are used to handle the fact that network responses can return at random/arbitrary/misordered times.
     * We ignore network responses except when we're expecting one.
     */

    // Coordinate acceptance of network responses for kubernetes nodes.
    const ignoreResponseForNodes = useRef(false);

    // Coordinate acceptance of network responses for workload presets.
    const ignoreResponseForWorkloadPresets = useRef(false);

    const ignoreResponseForWorkloads = useRef(false);

    const defaultWorkloadTitle = useRef(uuidv4());

    /**
     * Retrieve the current Kubernetes nodes from the backend.
     */
    async function fetchKubernetesNodes() {
        try {
            console.log(
                'Refreshing Kubernetes nodes. ignoreResponseForNodes.current = ' + ignoreResponseForNodes.current,
            );

            // Make a network request to the backend. The server infrastructure handles proxying/routing the request to the correct host.
            // We're specifically targeting the API endpoint I setup called "nodes".
            const response = await fetch('/api/nodes');

            if (response.status == 200) {
                // Get the response, which will be in JSON format, and decode it into an array of KubernetesNode (which is a TypeScript interface that I defined).
                const respNodes: KubernetesNode[] = await response.json();

                if (!ignoreResponseForNodes.current) {
                    // console.log('Received nodes: ' + JSON.stringify(respNodes));
                    setNodes(respNodes);
                    console.log('Successfully refreshed Kubernetes nodes.');
                } else {
                    console.log("Refreshed Kubernetes nodes, but we're ignoring the response.");
                }
            }
        } catch (e) {
            console.error(e);
        }
    }

    /**
     * Retrieve the current workload presets from the backend.
     */
    async function fetchWorkloadPresets(callback: () => void | undefined) {
        const startTime = performance.now();
        try {
            console.log('Refreshing workload presets.');

            // Make a network request to the backend. The server infrastructure handles proxying/routing the request to the correct host.
            // We're specifically targeting the API endpoint I setup called "nodes".
            const response = await fetch('/api/workload-presets');

            if (response.status == 200) {
                // Get the response, which will be in JSON format, and decode it into an array of WorkloadPreset (which is a TypeScript interface that I defined).
                const respWorkloadPresets: WorkloadPreset[] = await response.json();

                if (!ignoreResponseForWorkloadPresets.current) {
                    setWorkloadPresets(respWorkloadPresets);
                    console.log(
                        'Successfully refreshed workload presets. Discovered %d preset(s).',
                        respWorkloadPresets.length,
                    );

                    if (callback != undefined) {
                        callback();
                    }
                }
            }
        } catch (e) {
            console.error(e);
        }
        console.log(`Refresh workload presets: ${(performance.now() - startTime).toFixed(4)} ms`);
    }

    /**
     * Retrieve the current workloads from the backend.
     */
    const fetchWorkloads = useCallback(
        (callback: (presets: Workload[]) => void | undefined) => {
            const startTime = performance.now();
            try {
                console.log('Refreshing workloads.');

                const messageId: string = uuidv4();
                const onResponse = (response: Record<string, any>) => {
                    const respWorkloads: Workload[] = response.workloads;

                    if (!ignoreResponseForWorkloads.current) {
                        setWorkloads(new Map());
                        respWorkloads.forEach((workload: Workload) => {
                            setWorkloads(new Map(workloads.set(workload.id, workload)));
                        });
                        console.log(
                            'Successfully refreshed workloads. Discovered %d workload(s):\n%s',
                            respWorkloads.length,
                            JSON.stringify(respWorkloads),
                        );

                        if (callback != undefined) {
                            callback(respWorkloads);
                        }
                        ignoreResponseForWorkloads.current = true;
                    } else {
                        console.log("Refreshed workloads, but we're ignoring the response...");
                    }
                };
                websocketCallbacks.current.set(messageId, onResponse);
                sendJsonMessage({
                    op: 'get_workloads',
                    msg_id: messageId,
                });
            } catch (e) {
                console.error(e);
            }
            console.log(`Refresh workloads: ${(performance.now() - startTime).toFixed(4)} ms`);
        },
        [sendJsonMessage],
    );

    // Fetch the kubernetes nodes from the backend (which itself makes a network call to the Kubernetes API).
    useEffect(() => {
        ignoreResponseForNodes.current = false;
        fetchKubernetesNodes();

        // Periodically refresh the Kubernetes nodes every `props.nodeRefreshInterval` seconds, or when the user clicks the "refresh" button.
        setInterval(() => {
            ignoreResponseForNodes.current = false;
            fetchKubernetesNodes().then(() => {
                ignoreResponseForNodes.current = true;
            });
        }, props.nodeRefreshInterval * 1000);

        return () => {
            ignoreResponseForNodes.current = true;
        };
    }, [props.nodeRefreshInterval]);

    // Fetch the workload presets from the backend.
    useEffect(() => {
        ignoreResponseForWorkloadPresets.current = false;
        fetchWorkloadPresets(() => {});

        // Periodically refresh the Kubernetes nodes every `props.workloadPresetRefreshInterval` seconds, or when the user clicks the "refresh" button.
        setInterval(() => {
            ignoreResponseForWorkloadPresets.current = false;
            fetchWorkloadPresets(() => {}).then(() => {
                ignoreResponseForWorkloadPresets.current = true;
            });
        }, props.workloadPresetRefreshInterval * 1000);

        return () => {
            ignoreResponseForWorkloadPresets.current = true;
        };
    }, [props.workloadPresetRefreshInterval]);

    // Fetch the workloads from the backend.
    useEffect(() => {
        ignoreResponseForWorkloads.current = false;
        fetchWorkloads(() => {});

        // Periodically refresh the Kubernetes nodes every `props.workloadPresetRefreshInterval` seconds, or when the user clicks the "refresh" button.
        setInterval(() => {
            ignoreResponseForWorkloads.current = false;
            fetchWorkloads(() => {});
        }, props.workloadRefreshInterval * 1000);

        return () => {
            ignoreResponseForWorkloads.current = true;
        };
    }, [props.workloadRefreshInterval, fetchWorkloads]);

    async function manuallyRefreshNodes(callback: () => void | undefined) {
        const startTime = performance.now();
        ignoreResponseForNodes.current = false;
        await fetchKubernetesNodes().then(() => {
            ignoreResponseForNodes.current = true;
            console.log(`Refresh Kubernetes nodes: ${(performance.now() - startTime).toFixed(4)} ms`);

            if (callback != undefined) {
                callback();
            }
        });
    }

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

    const onConfirmStartWorkload = (
        workloadName: string,
        selectedPreset: WorkloadPreset,
        workloadSeedString: string,
        debugLoggingEnabled: boolean,
    ) => {
        console.log("New workload '%s' started by user with preset:\n%s", workloadName, JSON.stringify(selectedPreset));
        setIsStartWorkloadOpen(false);

        let workloadSeed = -1;

        if (workloadSeedString != '') {
            workloadSeed = parseInt(workloadSeedString);
        }

        const messageId: string = uuidv4();
        const callback = (result: any) => {
            console.log('Successfully registered workload %s', result.workload.id);
            setWorkloads(new Map(workloads.set(result.workload.id, result.workload)));
        };
        websocketCallbacks.current.set(messageId, callback);
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
        setIsStartWorkloadOpen(false);
    };

    const openMigrationModal = (kernel: DistributedJupyterKernel, replica: JupyterKernelReplica) => {
        const refreshComplete: Promise<void> = manuallyRefreshNodes(() => {});

        const delayMilliseconds = 5;
        // Basically, we'll open the modal after either 'delayMilliseconds' ms or when the node refresh completes, whichever comes first.
        Promise.race([wait(delayMilliseconds, 'timeout'), refreshComplete]).then((value) => {
            if (value == 'timeout') {
                console.warn('Node refresh took longer than %dms to complete', delayMilliseconds);
            }
            console.log('value: ' + value);
            setMigrateReplica(replica);
            setMigrateKernel(kernel);
            setIsMigrateModalOpen(true);
        });
    };

    const toggleDebugLogs = (workloadId: string, enabled: boolean) => {
        console.log('Toggling debug logs for workload ID=%s', workloadId);

        const messageId: string = uuidv4();
        const callback = (result: any) => {
            setWorkloads(new Map(workloads.set(result.workload.id, result.workload)));
        };
        websocketCallbacks.current.set(messageId, callback);
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
        const callback = (result: any) => {
            // const updatedWorkloads: Workload[] = workloads.map((workload: Workload) => {
            //     if (workload.id == result.workload.id) {
            //         return result.workload;
            //     } else {
            //         return workload;
            //     }
            // });

            // setWorkloads(updatedWorkloads);
            setWorkloads(new Map(workloads.set(result.workload.id, result.workload)));
        };
        websocketCallbacks.current.set(messageId, callback);
        sendJsonMessage({
            op: 'start_workload',
            msg_id: messageId,
            workload_id: workload.id,
        });
    };

    const onStopWorkloadClicked = (workload: Workload) => {
        console.log("Stopping workload '%s' (ID=%s)", workload.name, workload.id);

        const messageId: string = uuidv4();
        const callback = (result: any) => {
            // const updatedWorkloads: Workload[] = workloads.map((workload: Workload) => {
            //     if (workload.id == result.workload.id) {
            //         return result.workload;
            //     } else {
            //         return workload;
            //     }
            // });

            // setWorkloads(updatedWorkloads);
            setWorkloads(new Map(workloads.set(result.workload.id, result.workload)));
        };
        websocketCallbacks.current.set(messageId, callback);
        sendJsonMessage({
            op: 'stop_workload',
            msg_id: messageId,
            workload_id: workload.id,
        });
    };

    return (
        <PageSection>
            <Grid hasGutter>
                <GridItem span={6} rowSpan={2}>
                    <KernelList kernelsPerPage={5} openMigrationModal={openMigrationModal} />
                </GridItem>
                <GridItem span={6} rowSpan={6}>
                    <KubernetesNodeList
                        nodesPerPage={5}
                        manuallyRefreshNodes={manuallyRefreshNodes}
                        nodes={nodes}
                        refreshInterval={120}
                        selectable={false}
                    />
                </GridItem>
                <GridItem span={6} rowSpan={1}>
                    <WorkloadCard
                        workloadsPerPage={5}
                        toggleDebugLogs={toggleDebugLogs}
                        onStartWorkloadClicked={onStartWorkloadClicked}
                        onStopWorkloadClicked={onStopWorkloadClicked}
                        workloads={Array.from(workloads.values())}
                        refreshWorkloads={(callback: () => void | undefined) => {
                            ignoreResponseForWorkloads.current = false;
                            fetchWorkloads(callback);
                        }}
                        onLaunchWorkloadClicked={() => {
                            // If we have no workload presets, then refresh them when the user opens the 'Start Workload' modal.
                            if (workloadPresets.length == 0) {
                                ignoreResponseForWorkloadPresets.current = false;
                                fetchWorkloadPresets(() => {}).then(() => {
                                    ignoreResponseForWorkloadPresets.current = true;
                                });
                            }

                            defaultWorkloadTitle.current = uuidv4(); // Regenerate the default workload title as we're opening the modal again.
                            setIsStartWorkloadOpen(true);
                        }}
                    />
                </GridItem>
                <GridItem span={6} rowSpan={1}>
                    <KernelSpecList />
                </GridItem>
            </Grid>
            <MigrationModal
                nodes={nodes}
                manuallyRefreshNodes={manuallyRefreshNodes}
                isOpen={isMigrateModalOpen}
                onClose={closeMigrateReplicaModal}
                onConfirm={onConfirmMigrateReplica}
                targetKernel={migrateKernel}
                targetReplica={migrateReplica}
            />
            <RegisterWorkloadModal
                isOpen={isStartWorkloadModalOpen}
                onClose={onCancelStartWorkload}
                onConfirm={onConfirmStartWorkload}
                workloadPresets={workloadPresets}
                defaultWorkloadTitle={defaultWorkloadTitle.current}
            />
        </PageSection>
    );
};

export { Dashboard };
