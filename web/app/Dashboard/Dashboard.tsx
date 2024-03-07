import '@patternfly/react-core/dist/styles/base.css';

import React, { useEffect, useRef } from 'react';
import { Grid, GridItem, PageSection } from '@patternfly/react-core';

import { ActionCard, KernelSpecList, KubernetesNodeList, KernelList } from '@app/Components';
import { KubernetesNode, DistributedJupyterKernel, JupyterKernelReplica, WorkloadPreset } from '@app/Data';
import { MigrationModal, StartWorkloadModal } from '@app/Components/Modals';

export interface DashboardProps {
    nodeRefreshInterval: number;
    workloadPresetRefreshInterval: number;
}

{
    /* <DropdownItem value={0} key="jun-aug" description="Trace data from June - August.">
June - August
</DropdownItem>
<DropdownItem value={1} key="july" description="Trace data from July.">
July
</DropdownItem>
<DropdownItem value={2} key="august" description="Trace data from August.">
August
</DropdownItem> */
}

const Dashboard: React.FunctionComponent<DashboardProps> = (props: DashboardProps) => {
    const [nodes, setNodes] = React.useState<KubernetesNode[]>([]);
    const [workloadPresets, setWorkloadPresets] = React.useState<WorkloadPreset[]>([]);
    const [isStartWorkloadModalOpen, setIsStartWorkloadOpen] = React.useState(true);
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
            console.log('Refreshing Kubernetes nodes.');

            // Make a network request to the backend. The server infrastructure handles proxying/routing the request to the correct host.
            // We're specifically targeting the API endpoint I setup called "nodes".
            const response = await fetch('/api/nodes');

            // Get the response, which will be in JSON format, and decode it into an array of KubernetesNode (which is a TypeScript interface that I defined).
            const respNodes: KubernetesNode[] = await response.json();

            if (!ignoreResponseForNodes.current) {
                // console.log('Received nodes: ' + JSON.stringify(respNodes));
                setNodes(respNodes);
                console.log('Successfully refreshed Kubernetes nodes.');
            }
        } catch (e) {
            console.error(e);
        }
    }

    /**
     * Retrieve the current workload presets from the backend.
     */
    async function fetchWorkloadPresets() {
        try {
            console.log('Refreshing workload presets.');

            // Make a network request to the backend. The server infrastructure handles proxying/routing the request to the correct host.
            // We're specifically targeting the API endpoint I setup called "nodes".
            const response = await fetch('/api/workload-presets');

            // Get the response, which will be in JSON format, and decode it into an array of WorkloadPreset (which is a TypeScript interface that I defined).
            const respWorkloadPresets: WorkloadPreset[] = await response.json();

            if (!ignoreResponseForWorkloadPresets.current) {
                setWorkloadPresets(respWorkloadPresets);
                console.log(
                    'Successfully refreshed workload presets. Discovered ' + respWorkloadPresets.length + ' preset(s).',
                );
            }
        } catch (e) {
            console.error(e);
        }
    }

    // Fetch the kubernetes nodes from the backend (which itself makes a network call to the Kubernetes API).
    useEffect(() => {
        ignoreResponseForNodes.current = false;
        fetchKubernetesNodes();

        // Periodically refresh the Kubernetes nodes every `props.nodeRefreshInterval` seconds, or when the user clicks the "refresh" button.
        setInterval(() => {
            ignoreResponseForNodes.current = false;
            fetchKubernetesNodes();
            ignoreResponseForNodes.current = true;
        }, props.nodeRefreshInterval * 1000);

        return () => {
            ignoreResponseForNodes.current = true;
        };
    }, [props.nodeRefreshInterval]);

    // Fetch the workload presets from the backend.
    useEffect(() => {
        ignoreResponseForWorkloadPresets.current = false;
        fetchWorkloadPresets();

        // Periodically refresh the Kubernetes nodes every `props.workloadPresetRefreshInterval` seconds, or when the user clicks the "refresh" button.
        setInterval(() => {
            ignoreResponseForWorkloadPresets.current = false;
            fetchWorkloadPresets();
            ignoreResponseForWorkloadPresets.current = true;
        }, props.workloadPresetRefreshInterval * 1000);

        return () => {
            ignoreResponseForWorkloadPresets.current = true;
        };
    }, [props.workloadPresetRefreshInterval]);

    async function manuallyRefreshNodes() {
        ignoreResponseForNodes.current = false;
        fetchKubernetesNodes();
        ignoreResponseForNodes.current = true;
    }

    const onConfirmMigrateReplica = () => {
        setIsMigrateModalOpen(false);
        setMigrateReplica(null);
        setMigrateKernel(null);
    };

    const onCancelMigrateReplica = () => {
        setIsMigrateModalOpen(false);
        setMigrateReplica(null);
        setMigrateKernel(null);
    };

    const onConfirmStartWorkload = () => {
        console.log('New workload started by user.');
        setIsStartWorkloadOpen(false);
    };

    const onCancelStartWorkload = () => {
        console.log('New workload cancelled by user before starting.');
        setIsStartWorkloadOpen(false);
    };

    const openMigrationModal = (kernel: DistributedJupyterKernel, replica: JupyterKernelReplica) => {
        setMigrateReplica(replica);
        setMigrateKernel(kernel);
        setIsMigrateModalOpen(true);
    };

    return (
        <PageSection>
            <Grid hasGutter>
                <GridItem span={6} rowSpan={1}>
                    <ActionCard
                        onLaunchWorkloadClicked={() => {
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
                onClose={() => onCancelMigrateReplica()}
                onConfirm={() => onConfirmMigrateReplica()}
                targetKernel={migrateKernel}
                targetReplica={migrateReplica}
            />
            <StartWorkloadModal
                isOpen={isStartWorkloadModalOpen}
                onClose={() => onCancelStartWorkload()}
                onConfirm={() => onConfirmStartWorkload()}
                workloadPresets={workloadPresets}
            />
        </PageSection>
    );
};

export { Dashboard };
