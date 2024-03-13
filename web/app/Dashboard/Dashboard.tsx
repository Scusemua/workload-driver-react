import '@patternfly/react-core/dist/styles/base.css';

import React, { useEffect, useRef } from 'react';
import { Grid, GridItem, PageSection } from '@patternfly/react-core';

import { KernelList, KernelSpecList, KubernetesNodeList, WorkloadCard } from '@app/Components';
import { DistributedJupyterKernel, JupyterKernelReplica, KubernetesNode, WorkloadPreset } from '@app/Data';
import { MigrationModal, StartWorkloadModal } from '@app/Components/Modals';

export interface DashboardProps {
    nodeRefreshInterval: number;
    workloadPresetRefreshInterval: number;
}

function wait<T>(ms: number, value: T) {
    return new Promise<T>((resolve) => setTimeout(resolve, ms, value));
}

const Dashboard: React.FunctionComponent<DashboardProps> = (props: DashboardProps) => {
    const [nodes, setNodes] = React.useState<KubernetesNode[]>([]);
    const [workloadPresets, setWorkloadPresets] = React.useState<WorkloadPreset[]>([]);
    const [isStartWorkloadModalOpen, setIsStartWorkloadOpen] = React.useState(false);
    const [isMigrateModalOpen, setIsMigrateModalOpen] = React.useState(false);
    const [migrateKernel, setMigrateKernel] = React.useState<DistributedJupyterKernel | null>(null);
    const [migrateReplica, setMigrateReplica] = React.useState<JupyterKernelReplica | null>(null);

    /**
     * The following references are used to handle the fact that network responses can return at random/arbitrary/misordered times.
     * We ignore network responses except when we're expecting one.
     */

    // Coordinate acceptance of network responses for kubernetes nodes.
    const ignoreResponseForNodes = useRef(false);

    // Coordinate acceptance of network responses for workload presets.
    const ignoreResponseForWorkloadPresets = useRef(false);

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

    const onConfirmStartWorkload = (workloadName: string, selectedPreset: WorkloadPreset) => {
        console.log("New workload '%s' started by user with preset:\n%s", workloadName, JSON.stringify(selectedPreset));
        setIsStartWorkloadOpen(false);

        const requestOptions = {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                adjust_gpu_reservations: false,
                seed: 1,
                key: selectedPreset.key,
                name: workloadName,
            }),
        };

        fetch('/api/workload', requestOptions).then((response) => {
            console.log('Received response for launch of new workload: %s', JSON.stringify(response));
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

    return (
        <PageSection>
            <Grid hasGutter>
                <GridItem span={6} rowSpan={1}>
                    <WorkloadCard
                        refreshWorkloadPresets={fetchWorkloadPresets}
                        onLaunchWorkloadClicked={() => {
                            // If we have no workload presets, then refresh them when the user opens the 'Start Workload' modal.
                            if (workloadPresets.length == 0) {
                                ignoreResponseForWorkloadPresets.current = false;
                                fetchWorkloadPresets(() => {}).then(() => {
                                    ignoreResponseForWorkloadPresets.current = true;
                                });
                            }

                            setIsStartWorkloadOpen(true);
                        }}
                    />
                </GridItem>
                <GridItem span={6} rowSpan={6}>
                    <KernelList openMigrationModal={openMigrationModal} />
                </GridItem>
                <GridItem span={6} rowSpan={2}>
                    <KubernetesNodeList
                        manuallyRefreshNodes={manuallyRefreshNodes}
                        nodes={nodes}
                        refreshInterval={120}
                        selectable={false}
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
            <StartWorkloadModal
                isOpen={isStartWorkloadModalOpen}
                onClose={onCancelStartWorkload}
                onConfirm={onConfirmStartWorkload}
                workloadPresets={workloadPresets}
            />
        </PageSection>
    );
};

export { Dashboard };
