import '@patternfly/react-core/dist/styles/base.css';

import { KernelManager, ServerConnection } from '@jupyterlab/services';
import { IKernelConnection } from '@jupyterlab/services/lib/kernel/kernel';

import React, { useCallback, useEffect, useRef } from 'react';
import { Grid, GridItem, gridSpans, PageSection } from '@patternfly/react-core';

import {
    KernelList,
    KernelSpecList,
    KubernetesNodeList,
    WorkloadCard,
    ClusterComponentsCard,
} from '@app/Components/Cards';
import {
    DistributedJupyterKernel,
    JupyterKernelReplica,
    KubernetesNode,
    SingleWorkloadResponse,
    WORKLOAD_STATE_RUNNING,
    Workload,
    WorkloadPreset,
    WorkloadsResponse,
} from '@app/Data';
import { InformationModal, MigrationModal, RegisterWorkloadModal } from '@app/Components/Modals';

import useWebSocket from 'react-use-websocket';

import { v4 as uuidv4 } from 'uuid';
import { number } from 'prop-types';

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
    const [kernels, setKernels] = React.useState<DistributedJupyterKernel[]>([]);
    const [workloads, setWorkloads] = React.useState(new Map());
    const [workloadPresets, setWorkloadPresets] = React.useState<WorkloadPreset[]>([]);
    const [isStartWorkloadModalOpen, setIsStartWorkloadOpen] = React.useState(false);
    const [isMigrateModalOpen, setIsMigrateModalOpen] = React.useState(false);
    const [migrateKernel, setMigrateKernel] = React.useState<DistributedJupyterKernel | null>(null);
    const [migrateReplica, setMigrateReplica] = React.useState<JupyterKernelReplica | null>(null);
    const [refreshingKernels, setRefreshingKernels] = React.useState(false);
    const [numKernelsCreating, setNumKernelsCreating] = React.useState(0);
    const [isErrorModalOpen, setIsErrorModalOpen] = React.useState(false);
    const [errorMessage, setErrorMessage] = React.useState('');
    const [errorMessagePreamble, setErrorMessagePreamble] = React.useState('');

    // const [kernelCardRowSpan, setKernelCardRowSpan] = React.useState<gridSpans>(2);
    const [nodeCardRowSpan, setNodeCardRowSpan] = React.useState<gridSpans>(4);
    const [workloadsCardRowSpan, setWorkloadsCardRowSpan] = React.useState<gridSpans>(1);

    const kernelManager = React.useRef<KernelManager | null>(null);

    const websocketCallbacks = React.useRef(new Map());

    const { sendJsonMessage, lastMessage, lastJsonMessage } = useWebSocket<Record<string, unknown>>(
        'ws://localhost:8000/workload',
        {
            share: false,
            shouldReconnect: () => true,
        },
    );

    const handleMessage = useCallback((message: Record<string, unknown>) => {
        console.log(`Got a new message: ${JSON.stringify(message)}`);

        const handleActiveWorkloadsUpdate = (updatedWorkloads: Workload[]) => {
            console.log('Received update about %d active workload(s).', updatedWorkloads.length);

            updatedWorkloads.forEach((workload: Workload) => {
                setWorkloads((w) => new Map(w.set(workload.id, workload)));
            });

            const rowSpan: gridSpans = Math.min(Math.max(workloads.size, 1), 3) as gridSpans;
            setWorkloadsCardRowSpan(rowSpan);
        };

        // If there is a callback, then call it.
        if (message) {
            if (websocketCallbacks.current.has(message['msg_id'])) {
                console.log(`Found callback for message ${message['msg_id']}`);
                websocketCallbacks.current.get(message['msg_id'])(message);
            } else {
                console.log(`No callback found for message ${message['msg_id']}`);
                const op = message['op'];

                if (op == 'active_workloads_update') {
                    const updatedWorkloads: Workload[] | unknown = message['updated_workloads'];
                    if (!updatedWorkloads) {
                        throw new Error("Unexpected response for 'updated_workloads' key.");
                    }
                    handleActiveWorkloadsUpdate(updatedWorkloads as Workload[]);
                }
            }
        }
    }, []);

    // Run when a new WebSocket message is received (lastJsonMessage).
    useEffect(() => {
        handleMessage(lastJsonMessage);
    }, [lastJsonMessage, handleMessage]);

    useEffect(() => {
        if (lastMessage && lastMessage.data) {
            const promise: Promise<string> = lastMessage.data.text();
            promise.then((data) => {
                console.log(data);
                const messageJson: Record<string, unknown> = JSON.parse(data);
                handleMessage(messageJson);
            });
        }
    }, [lastMessage, handleMessage]);

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
                const onResponse = (response: WorkloadsResponse) => {
                    const respWorkloads: Workload[] = response.workloads;

                    if (!ignoreResponseForWorkloads.current) {
                        setWorkloads(new Map());
                        respWorkloads.forEach((workload: Workload) => {
                            setWorkloads((w) => new Map(w.set(workload.id, workload)));
                        });
                        console.log(
                            'Successfully refreshed workloads. Discovered %d workload(s):\n%s',
                            respWorkloads.length,
                            JSON.stringify(respWorkloads),
                        );

                        const rowSpan: gridSpans = Math.min(Math.max(workloads.size, 1), 3) as gridSpans;
                        setWorkloadsCardRowSpan(rowSpan);

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

    const ignoreResponseForKernels = useRef(false);
    const fetchKernels = useCallback(() => {
        const startTime = performance.now();
        try {
            setRefreshingKernels(true);
            console.log('Refreshing kernels now.');
            // Make a network request to the backend. The server infrastructure handles proxying/routing the request to the correct host.
            // We're specifically targeting the API endpoint I setup called "get-kernels".
            fetch('api/get-kernels').then((response: Response) => {
                if (response.status == 200) {
                    response.json().then((respKernels: DistributedJupyterKernel[]) => {
                        if (!ignoreResponseForKernels.current) {
                            console.log('Received kernels: ' + JSON.stringify(respKernels));
                            console.log("We're currently creating %d kernel(s).", numKernelsCreating);

                            // Only bother with this next bit if we're waiting on some kernels that we just created.
                            if (numKernelsCreating > 0) {
                                // For each kernel that we receive, we'll check if it is a new kernel.
                                respKernels.forEach((newKernel) => {
                                    for (let i = 0; i < kernels.length; i++) {
                                        // If we've already seen this kernel, then return immediately.
                                        // No need to compare it against all the other kernels; we already know that it isn't new.
                                        if (kernels[i].kernelId == newKernel.kernelId) {
                                            console.log(
                                                'Kernel %s is NOT a new kernel (i.e., we already knew about it).',
                                                newKernel.kernelId,
                                            );
                                            return;
                                        }
                                    }

                                    // If we're currently creating any kernels and we just received a kernel that we've never seen before,
                                    // then this must be one of the newly-created kernels that we're waiting on! So, we decrement the
                                    // 'numKernelsCreating' counter.
                                    if (numKernelsCreating > 0) {
                                        console.log(
                                            'Kernel %s is a NEW kernel (i.e., it was just created).',
                                            newKernel.kernelId,
                                        );
                                        setNumKernelsCreating(numKernelsCreating - 1);
                                    }
                                });
                            }

                            setKernels(respKernels);
                            ignoreResponseForKernels.current = true;
                        } else {
                            console.log("Received %d kernel(s), but we're ignoring the response.", respKernels.length);
                        }
                        setRefreshingKernels(false);
                    });
                }
            });
        } catch (e) {
            console.error(e);
        }
        console.log(`Refresh kernels: ${(performance.now() - startTime).toFixed(4)} ms`);
    }, []);

    // const kernels = React.useRef<DistributedJupyterKernel[]>([]);
    useEffect(() => {
        ignoreResponseForKernels.current = false;
        fetchKernels();

        // Periodically refresh the automatically kernels every 30 seconds.
        setInterval(() => {
            ignoreResponseForKernels.current = false;
            fetchKernels();
        }, 120000);

        return () => {
            ignoreResponseForKernels.current = true;
        };
    }, [fetchKernels]);

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

    async function manuallyRefreshKernels(callback: () => void | undefined) {
        const startTime = performance.now();
        ignoreResponseForKernels.current = false;
        fetchKernels();
        ignoreResponseForNodes.current = true;
        console.log(`Refresh kernels nodes: ${(performance.now() - startTime).toFixed(4)} ms`);

        if (callback != undefined) {
            callback();
        }
        // await fetchKernels().then(() => {
        //     ignoreResponseForNodes.current = true;
        //     console.log(`Refresh kernels nodes: ${(performance.now() - startTime).toFixed(4)} ms`);

        //     if (callback != undefined) {
        //         callback();
        //     }
        // });
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
        const callback = (result: SingleWorkloadResponse) => {
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
        if (enabled) {
            console.log("Enabling debug logging for workload '%s'", workloadId);
        } else {
            console.log("Disabling debug logging for workload '%s'", workloadId);
        }

        const messageId: string = uuidv4();
        const callback = (result: SingleWorkloadResponse) => {
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
        const callback = (result: SingleWorkloadResponse) => {
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
        const callback = (result: SingleWorkloadResponse) => {
            setWorkloads(new Map(workloads.set(result.workload.id, result.workload)));
        };
        websocketCallbacks.current.set(messageId, callback);
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
        const callback = (result: WorkloadsResponse) => {
            result.workloads.forEach((workload: Workload) => {
                setWorkloads((w) => new Map(w.set(workload.id, workload)));
            });
        };
        websocketCallbacks.current.set(messageId, callback);
        sendJsonMessage({
            op: 'stop_workloads',
            msg_id: messageId,
            workload_ids: activeWorkloadsIDs,
        });
    };

    const kernelsPerPageChanged = (value: number) => {};

    const WithGutters = () => (
        <Grid hasGutter>
            <GridItem span={8}>span = 8</GridItem>
            <GridItem span={4} rowSpan={2}>
                span = 4, rowSpan = 2
            </GridItem>
            <GridItem span={2} rowSpan={3}>
                span = 2, rowSpan = 3
            </GridItem>
            <GridItem span={2}>span = 2</GridItem>
            <GridItem span={4}>span = 4</GridItem>
            <GridItem span={2}>span = 2</GridItem>
            <GridItem span={2}>span = 2</GridItem>
            <GridItem span={2}>span = 2</GridItem>
            <GridItem span={4}>span = 4</GridItem>
            <GridItem span={2}>span = 2</GridItem>
            <GridItem span={4}>span = 4</GridItem>
            <GridItem span={4}>span = 4</GridItem>
        </Grid>
    );

    const displayErrorMessage = (message: string, preamble: string) => {
        setErrorMessage(message);
        setErrorMessagePreamble(preamble);
        setIsErrorModalOpen(true);
    };

    async function initializeKernelManagers() {
        if (kernelManager.current === null) {
            const kernelSpecManagerOptions: KernelManager.IOptions = {
                serverSettings: ServerConnection.makeSettings({
                    token: '',
                    appendToken: false,
                    baseUrl: '/jupyter',
                    wsUrl: 'ws://localhost:8888/',
                    fetch: fetch,
                }),
            };
            kernelManager.current = new KernelManager(kernelSpecManagerOptions);

            console.log('Waiting for Kernel Manager to be ready.');

            kernelManager.current.connectionFailure.connect((_sender: KernelManager, err: Error) => {
                console.error(
                    'An error has occurred while preparing the Kernel Manager. ' + err.name + ': ' + err.message,
                );
            });

            await kernelManager.current.ready.then(() => {
                console.log('Kernel Manager is ready!');
            });
        }
    }

    useEffect(() => {
        initializeKernelManagers();
    }, []);

    const onInterruptKernelClicked = (kernelId: string) => {
        console.log('User is interrupting kernel ' + kernelId);

        if (!kernelManager.current) {
            console.error('Kernel Manager is not available. Will try to connect...');
            initializeKernelManagers().then(() => {}); // Wait for promise to resolve.
        }

        const kernelConnection: IKernelConnection = kernelManager.current!.connectTo({
            model: { id: kernelId, name: kernelId },
        });

        if (kernelConnection.connectionStatus == 'connected') {
            kernelConnection.interrupt().then(() => {
                console.log('Successfully interrupted kernel ' + kernelId);
            });
        }
    };

    async function startKernel() {
        // Create a new kernel.
        if (!kernelManager.current) {
            console.error('Kernel Manager is not available. Will try to connect...');
            await initializeKernelManagers();
        }

        setNumKernelsCreating(numKernelsCreating + 1);

        // Precondition: The KernelManager is defined.
        const manager: KernelManager = kernelManager.current!;

        console.log('Starting kernel now...');

        // Start a python kernel
        const kernel: IKernelConnection = await manager.startNew({ name: 'distributed' });

        console.log('Successfully started kernel!');

        // Register a callback for when the kernel changes state.
        kernel.statusChanged.connect((_, status) => {
            console.log(`New Kernel Status Update: ${status}`);
        });

        // Update/refresh the kernels since we know a new one was just created.
        setTimeout(() => {
            // ignoreResponse.current = false;
            // fetchKernels();
            manuallyRefreshKernels(() => {});
        }, 3000);
    }

    return (
        <PageSection>
            <Grid hasGutter>
                <GridItem span={6} rowSpan={2}>
                    <KernelList
                        kernels={kernels}
                        kernelManager={kernelManager.current!}
                        onInterruptKernelClicked={onInterruptKernelClicked}
                        startKernel={startKernel}
                        displayErrorMessage={displayErrorMessage}
                        numKernelsCreating={numKernelsCreating}
                        manuallyRefreshKernels={manuallyRefreshKernels}
                        refreshingKernels={refreshingKernels}
                        onChangeKernelsPerPage={kernelsPerPageChanged}
                        kernelsPerPageInitialValue={1}
                        openMigrationModal={openMigrationModal}
                    />
                </GridItem>
                <GridItem span={6} rowSpan={2}>
                    <ClusterComponentsCard />
                </GridItem>
                <GridItem span={6} rowSpan={nodeCardRowSpan}>
                    <KubernetesNodeList
                        nodesPerPageInitialValue={3}
                        manuallyRefreshNodes={manuallyRefreshNodes}
                        nodes={nodes}
                        refreshInterval={120}
                        selectable={false}
                    />
                </GridItem>
                <GridItem span={6} rowSpan={workloadsCardRowSpan}>
                    <WorkloadCard
                        workloadsPerPage={3}
                        toggleDebugLogs={toggleDebugLogs}
                        onStartWorkloadClicked={onStartWorkloadClicked}
                        onStopWorkloadClicked={onStopWorkloadClicked}
                        onStopAllWorkloadsClicked={onStopAllWorkloadsClicked}
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
            <InformationModal
                isOpen={isErrorModalOpen}
                onClose={() => {
                    setIsErrorModalOpen(false);
                    setErrorMessage('');
                    setErrorMessagePreamble('');
                }}
                title="An Error has Occurred"
                titleIconVariant="danger"
                message1={errorMessagePreamble}
                message2={errorMessage}
            />
        </PageSection>
    );
};

export { Dashboard };
