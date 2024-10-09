import { AdjustNumNodesModal } from '@app/Components/Modals/AdjustNumNodesModal';
import { GpuIcon } from '@app/Icons';
import { GetToastContentWithHeaderAndBody, ToastFetch } from '@app/utils/toast_utils';
import { NodeDataList } from '@cards/NodeListCard/NodeDataList';
import { NodeResourceView } from '@cards/NodeListCard/NodeResourceView';
import { ClusterNode, GetNodeId, GetNodeSpecResource } from '@data/Cluster';
import {
    Button,
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Flex,
    FlexItem,
    InputGroup,
    InputGroupItem,
    SearchInput,
    Title,
    ToolbarGroup,
    ToolbarItem,
    ToolbarToggleGroup,
    Tooltip,
} from '@patternfly/react-core';
import { FilterIcon, ListIcon, MonitoringIcon, ReplicatorIcon, SyncIcon } from '@patternfly/react-icons';
import { useNodes } from '@providers/NodeProvider';
import React from 'react';
import { AdjustVirtualGPUsModal, RoundToTwoDecimalPlaces } from '../../Modals';

export interface NodeListProps {
    selectableViaCheckboxes: boolean;
    isDashboardList: boolean; // Indicates whether the node list is the primary list that appears on the dashboard
    disableRadiosWithKernel?: string; // KernelID such that, if a node has a Pod for that kernel, its radio button is disabled.
    hideControlPlaneNode?: boolean;
    onSelectNode?: (nodeId: string) => void; // Function to call when a node is selected; used in case parent wants to do something when node is selected, such as update state.
    nodesPerPage: number;
    hideAdjustVirtualGPUsButton: boolean;
    displayNodeToggleSwitch: boolean; // If true, show the Switch that is used to enable/disable the node.
}

export const NodeList: React.FunctionComponent<NodeListProps> = (props: NodeListProps) => {
    const [searchValue, setSearchValue] = React.useState('');
    const [selectedNode, setSelectedNode] = React.useState('');
    const [resourceModeToggled, setResourceModeToggled] = React.useState<boolean>(false);
    const { nodes, nodesAreLoading, refreshNodes } = useNodes();
    const [adjustVirtualGPUsNodes, setAdjustVirtualGPUsNodes] = React.useState<ClusterNode[]>([]);
    const [isAdjustVirtualGPUsModalOpen, setIsAdjustVirtualGPUsModalOpen] = React.useState(false);
    const [isAdjustNumNodesModalOpen, setIsAdjustNumNodesModalOpen] = React.useState(false);

    // When the user types something into the node name filter, we update the associated state.
    const onSearchChange = (value: string) => {
        setSearchValue(value);
    };

    const onAdjustVirtualGPUsClicked = (nodes: ClusterNode[]) => {
        setAdjustVirtualGPUsNodes(nodes);
        setIsAdjustVirtualGPUsModalOpen(true);
    };

    const onAdjustNumNodesClicked = () => {
        setIsAdjustNumNodesModalOpen(true);
    };

    const closeAdjustVirtualGPUsModal = () => {
        setIsAdjustVirtualGPUsModalOpen(false);
        setAdjustVirtualGPUsNodes([]);
    };

    const closeAdjustNumNodesModal = () => {
        setIsAdjustNumNodesModalOpen(false);
    };

    async function doAdjustVirtualGPUs(value: number) {
        if (adjustVirtualGPUsNodes.length == 0) {
            console.error("Field 'adjustVirtualGPUsNode' is null...");
            closeAdjustVirtualGPUsModal();
            return;
        }

        if (Number.isNaN(value)) {
            console.error('Specified value is NaN...');
            closeAdjustVirtualGPUsModal();
            return;
        }

        adjustVirtualGPUsNodes.forEach((node: ClusterNode) => {
            if (GetNodeSpecResource(node, 'GPU') == value) {
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
                    kubernetesNodeName: GetNodeId(node),
                }),
            };

            console.log(`Attempting to set vGPUs on node ${GetNodeId(node)} to ${value}`);

            ToastFetch(
                `Adjusting number of vGPUs on node ${GetNodeId(node)} to ${value}`,
                GetToastContentWithHeaderAndBody(
                    `Successfully updated vGPU capacity for node ${GetNodeId(node)}`,
                    'It may take several seconds for the updated value to appear.',
                ),
                (_, reason) => {
                    return GetToastContentWithHeaderAndBody(
                        `Failed to update vGPUs for node ${GetNodeId(node)}`,
                        JSON.stringify(reason),
                    );
                },
                'api/vgpus',
                requestOptions,
            ).then(() => {});
        });
    }

    async function doAdjustNumNodes(value: number, operation: 'set_nodes' | 'add_nodes' | 'remove_nodes') {
        closeAdjustNumNodesModal();

        const startTime: number = performance.now();

        const getToastLoadingMessage = (numNodes: number, op: 'set_nodes' | 'add_nodes' | 'remove_nodes') => {
            if (op == 'set_nodes') {
                return `Setting cluster scale to ${numNodes} ${numNodes == 1 ? 'node' : 'nodes'}...`;
            } else if (op == 'add_nodes') {
                return `Adding ${numNodes} ${numNodes == 1 ? 'node' : 'nodes'} to the cluster...`;
            } else {
                return `Removing ${numNodes} ${numNodes == 1 ? 'node' : 'nodes'} from the cluster...`;
            }
        };

        const getToastSuccessMessage = (numNodes: number, op: 'set_nodes' | 'add_nodes' | 'remove_nodes') => {
            if (op == 'set_nodes') {
                return `Successfully scaled number of nodes in cluster to ${numNodes} nodes.`;
            } else if (op == 'add_nodes') {
                return `Successfully added ${numNodes} ${numNodes == 1 ? 'node' : 'nodes'} to the cluster.`;
            } else {
                return `Successfully removed ${numNodes} ${numNodes == 1 ? 'node' : 'nodes'} from the cluster.`;
            }
        };

        const getToastFailureMessage = (numNodes: number, op: 'set_nodes' | 'add_nodes' | 'remove_nodes') => {
            if (op == 'set_nodes') {
                return `Failed to scale the number of nodes in cluster to ${numNodes} nodes.`;
            } else if (op == 'add_nodes') {
                return `Failed to add ${numNodes} ${numNodes == 1 ? 'node' : 'nodes'} to the cluster.`;
            } else {
                return `Failed to remove ${numNodes} ${numNodes == 1 ? 'node' : 'nodes'} from the cluster.`;
            }
        };

        console.log(`Attempting to set number of nodes in cluster to ${value}`);

        const requestOptions = {
            method: 'PATCH',
            headers: {
                'Content-Type': 'application/json',
                // 'Cache-Control': 'no-cache, no-transform, no-store'
            },
            body: JSON.stringify({
                target_num_nodes: value,
                op: operation,
            }),
        };

        await ToastFetch(
            getToastLoadingMessage(value, operation),
            GetToastContentWithHeaderAndBody(
                getToastSuccessMessage(value, operation),
                `It may take several seconds for the nodes list to update. (Time elapsed: ${RoundToTwoDecimalPlaces(performance.now() - startTime)} seconds.)`,
            ),
            (res, reason) => {
                return GetToastContentWithHeaderAndBody(
                    getToastFailureMessage(value, operation),
                    `HTTP ${res.status} - ${res.statusText}: ${JSON.stringify(reason)}`,
                );
            },
            'api/nodes',
            requestOptions,
        );

        refreshNodes(false);
    }

    // Handler for when the user filters by node name.
    const onFilter = (repo: ClusterNode) => {
        if (props.hideControlPlaneNode && GetNodeId(repo).includes('control-plane')) {
            return false;
        }

        // Search name with search value
        let searchValueInput: RegExp;
        try {
            searchValueInput = new RegExp(searchValue, 'i');
        } catch (err) {
            searchValueInput = new RegExp(searchValue.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'i');
        }
        const matchesSearchValue = GetNodeId(repo).search(searchValueInput) >= 0;

        // If the filter text box is empty, then match against everything. Otherwise, match against node ID.
        return searchValue === '' || matchesSearchValue;
    };

    const toolbar = (
        <React.Fragment>
            <ToolbarToggleGroup toggleIcon={<FilterIcon />} breakpoint="xl">
                <Flex alignItems={{ default: 'alignItemsCenter' }}>
                    <FlexItem hidden={!props.selectableViaCheckboxes}>
                        <ToolbarItem>
                            <Tooltip exitDelay={75} content={<div>Clear selected node.</div>}>
                                <Button
                                    label="clear-selected-node-button"
                                    aria-label="clear-selected-node-button"
                                    variant="link"
                                    isDisabled={selectedNode == ''}
                                    onClick={() => {
                                        setSelectedNode('');
                                        if (props.onSelectNode != undefined) {
                                            props.onSelectNode('');
                                        }
                                    }}
                                >
                                    Clear Selection
                                </Button>
                            </Tooltip>
                        </ToolbarItem>
                    </FlexItem>
                    <FlexItem hidden={nodes.length == 0 || resourceModeToggled}>
                        <ToolbarItem>
                            <InputGroup>
                                <InputGroupItem isFill>
                                    <SearchInput
                                        placeholder="Filter by node ID"
                                        value={searchValue}
                                        onChange={(_event, value) => onSearchChange(value)}
                                        onClear={() => onSearchChange('')}
                                    />
                                </InputGroupItem>
                            </InputGroup>
                        </ToolbarItem>
                    </FlexItem>
                </Flex>
            </ToolbarToggleGroup>
            <ToolbarGroup variant="icon-button-group">
                <ToolbarItem>
                    <Tooltip content="Adjust the number of vGPUs available on ALL nodes.">
                        <Button
                            variant="plain"
                            onClick={(event: React.MouseEvent) => {
                                event.stopPropagation();
                                onAdjustVirtualGPUsClicked(nodes);
                            }}
                            icon={<GpuIcon scale={1.5} />}
                        />
                    </Tooltip>
                </ToolbarItem>
                <ToolbarItem>
                    <Tooltip content="Adjust the number of nodes within the cluster.">
                        <Button
                            variant="plain"
                            onClick={(event: React.MouseEvent) => {
                                event.stopPropagation();
                                onAdjustNumNodesClicked();
                            }}
                            icon={<ReplicatorIcon />}
                        />
                    </Tooltip>
                </ToolbarItem>
                <ToolbarItem>
                    <Tooltip
                        exitDelay={75}
                        content={
                            <div>
                                {!resourceModeToggled ? "Switch to 'resource' view." : "Switch to 'detail' view."}
                            </div>
                        }
                    >
                        <Button
                            variant={'plain'}
                            isDisabled={nodesAreLoading}
                            onClick={() => {
                                setResourceModeToggled((toggled) => !toggled);
                            }}
                            label="toggle-view-button"
                            aria-label="Toggle between resource and detail view"
                            icon={!resourceModeToggled ? <MonitoringIcon /> : <ListIcon />}
                        />
                    </Tooltip>
                    <Tooltip exitDelay={75} content={<div>Refresh nodes.</div>}>
                        <Button
                            variant="plain"
                            isDisabled={nodesAreLoading}
                            onClick={() => {
                                console.log('Refreshing nodes now.');
                                refreshNodes();
                            }}
                            // isDisabled={nodesAreLoading}
                            label="refresh-nodes-button"
                            aria-label="refresh-nodes-button"
                            className={
                                (nodesAreLoading && 'loading-icon-spin-toggleable') ||
                                'loading-icon-spin-toggleable paused'
                            }
                            icon={<SyncIcon />}
                        />
                    </Tooltip>
                </ToolbarItem>
            </ToolbarGroup>
        </React.Fragment>
    );

    return (
        <div>
            <Card
                isFullHeight
                isRounded
                id={props.isDashboardList ? 'primary-node-list-card' : 'migration-node-list-card'}
                style={{ minHeight: '30em' }}
            >
                <CardHeader
                    actions={{ actions: toolbar, hasNoOffset: false }}
                    toggleButtonProps={{
                        id: 'expand-kube-nodes-button',
                        'aria-label': 'expand-kube-nodes-button',
                    }}
                >
                    <CardTitle>
                        <Title headingLevel="h1" size="xl">
                            {resourceModeToggled
                                ? `Nodes (Resource View): ${nodes.length}`
                                : `Nodes (Detailed View): ${nodes.length}`}
                        </Title>
                    </CardTitle>
                </CardHeader>
                <CardBody>
                    {!resourceModeToggled && (
                        <NodeDataList
                            selectableViaCheckboxes={props.selectableViaCheckboxes}
                            isDashboardList={props.isDashboardList}
                            nodesPerPage={props.nodesPerPage}
                            hideAdjustVirtualGPUsButton={props.hideAdjustVirtualGPUsButton}
                            displayNodeToggleSwitch={props.displayNodeToggleSwitch}
                            disableRadiosWithKernel={props.disableRadiosWithKernel}
                            onFilter={onFilter}
                            onSelectNode={props.onSelectNode}
                            onAdjustVirtualGPUsClicked={onAdjustVirtualGPUsClicked}
                        />
                    )}
                    {resourceModeToggled && <NodeResourceView />}
                    <AdjustVirtualGPUsModal
                        isOpen={isAdjustVirtualGPUsModalOpen}
                        onClose={closeAdjustVirtualGPUsModal}
                        onConfirm={doAdjustVirtualGPUs}
                        nodes={adjustVirtualGPUsNodes}
                    />
                    <AdjustNumNodesModal
                        isOpen={isAdjustNumNodesModalOpen}
                        onClose={closeAdjustNumNodesModal}
                        onConfirm={doAdjustNumNodes}
                    />
                </CardBody>
            </Card>
        </div>
    );
};
