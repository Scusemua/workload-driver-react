import React, { useReducer } from 'react';
import {
    Button,
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    DataList,
    DataListCell,
    DataListContent,
    DataListControl,
    DataListItem,
    DataListItemCells,
    DataListItemRow,
    DataListToggle,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    FlexItem,
    InputGroup,
    InputGroupItem,
    Pagination,
    PaginationVariant,
    Radio,
    SearchInput,
    Skeleton,
    Switch,
    Text,
    TextVariants,
    Title,
    ToolbarGroup,
    ToolbarItem,
    ToolbarToggleGroup,
    Tooltip,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { KubernetesNode, KubernetesPod } from '@data/Kubernetes';
import {
    CpuIcon,
    CubeIcon,
    FilterIcon,
    GlobeIcon,
    MemoryIcon,
    OutlinedClockIcon,
    SyncIcon,
    VirtualMachineIcon,
} from '@patternfly/react-icons';
import { useNodes } from '@providers/NodeProvider';
import { GpuIcon } from '@app/Icons';
import { toast } from 'react-hot-toast';
import { HeightFactorContext, KubernetesNodeHeightFactorContext } from '@app/Dashboard/Dashboard';

export interface NodeListProps {
    selectableViaCheckboxes: boolean;
    isDashboardList: boolean; // Indicates whether the node list is the primary list that appears on the dashboard
    disableRadiosWithKernel?: string; // KernelID such that, if a node has a Pod for that kernel, its radio button is disabled.
    hideControlPlaneNode?: boolean;
    onSelectNode?: (nodeId: string) => void; // Function to call when a node is selected; used in case parent wants to do something when node is selected, such as update state.
    nodesPerPage: number;
    hideAdjustVirtualGPUsButton: boolean;
    onAdjustVirtualGPUsClicked?: (node: KubernetesNode) => void;
    displayNodeToggleSwitch: boolean; // If true, show the Switch that is used to enable/disable the node.
}

export const KubernetesNodeList: React.FunctionComponent<NodeListProps> = (props: NodeListProps) => {
    const [searchValue, setSearchValue] = React.useState('');
    const [expandedNodes, setExpandedNodes] = React.useState<string[]>([]);
    const [selectedNode, setSelectedNode] = React.useState('');
    const [page, setPage] = React.useState(1);
    const [perPage, setPerPage] = React.useState(props.nodesPerPage);
    const { nodes, nodesAreLoading, refreshNodes } = useNodes();
    const [, forceUpdate] = useReducer((x) => x + 1, 0);
    const heightFactorContext: HeightFactorContext = React.useContext(KubernetesNodeHeightFactorContext);

    const onSetPage = (_event: React.MouseEvent | React.KeyboardEvent | MouseEvent, newPage: number) => {
        setPage(newPage);
        console.log(
            'onSetPage: Displaying workloads %d through %d.',
            perPage * (newPage - 1),
            perPage * (newPage - 1) + perPage,
        );
    };

    const onPerPageSelect = (
        _event: React.MouseEvent | React.KeyboardEvent | MouseEvent,
        newPerPage: number,
        newPage: number,
    ) => {
        setPerPage(newPerPage);
        setPage(newPage);
        // console.log(
        //     'onPerPageSelect: Displaying workloads %d through %d.',
        //     newPerPage * (newPage - 1),
        //     newPerPage * (newPage - 1) + newPerPage,
        // );

        if (props.isDashboardList) {
            heightFactorContext.setHeightFactor(Math.min(newPerPage, nodes.length));
        }
    };

    // When the user types something into the node name filter, we update the associated state.
    const onSearchChange = (value: string) => {
        setSearchValue(value);
    };

    // Set up name search input
    const searchInput = (
        <SearchInput
            placeholder="Filter by node ID"
            value={searchValue}
            onChange={(_event, value) => onSearchChange(value)}
            onClear={() => onSearchChange('')}
        />
    );

    // Handler for when the user filters by node name.
    const onFilter = (repo: KubernetesNode) => {
        if (props.hideControlPlaneNode && repo.NodeId.includes('control-plane')) {
            return false;
        }

        // Search name with search value
        let searchValueInput: RegExp;
        try {
            searchValueInput = new RegExp(searchValue, 'i');
        } catch (err) {
            searchValueInput = new RegExp(searchValue.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'i');
        }
        const matchesSearchValue = repo.NodeId.search(searchValueInput) >= 0;

        // If the filter text box is empty, then match against everything. Otherwise, match against node ID.
        return searchValue === '' || matchesSearchValue;
    };
    const filteredNodes =
        nodes.length > 0 ? nodes.filter(onFilter).slice(perPage * (page - 1), perPage * (page - 1) + perPage) : [];

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
                    <FlexItem hidden={nodes.length == 0}>
                        <ToolbarItem>
                            <InputGroup>
                                <InputGroupItem isFill>{searchInput}</InputGroupItem>
                            </InputGroup>
                        </ToolbarItem>
                    </FlexItem>
                </Flex>
            </ToolbarToggleGroup>
            <ToolbarGroup variant="icon-button-group">
                <ToolbarItem>
                    <Tooltip exitDelay={75} content={<div>Refresh nodes.</div>}>
                        <Button
                            variant="plain"
                            onClick={() => {
                                toast.promise(refreshNodes(), {
                                    loading: <b>Refreshing Kubernetes nodes...</b>,
                                    success: <b>Refreshed Kubernetes nodes!</b>,
                                    error: (reason: Error) => {
                                        let explanation: string = reason.message;
                                        if (reason.name === 'SyntaxError') {
                                            explanation = 'HTTP 504 Gateway Timeout';
                                        }

                                        return (
                                            <div>
                                                <Flex
                                                    direction={{ default: 'column' }}
                                                    spaceItems={{ default: 'spaceItemsNone' }}
                                                >
                                                    <FlexItem>
                                                        <b>Could not refresh Kubernetes nodes.</b>
                                                    </FlexItem>
                                                    <FlexItem>{explanation}</FlexItem>
                                                </Flex>
                                            </div>
                                        );
                                    },
                                });
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

    const expandedNodeContent = (kubeNode: KubernetesNode) => (
        <Table isStriped aria-label="Pods Table" variant={'compact'} borders={true}>
            <Thead>
                <Tr>
                    <Th>Pod ID</Th>
                    <Th>Phase</Th>
                    <Th>Age</Th>
                    <Th>IP</Th>
                </Tr>
            </Thead>
            <Tbody>
                {kubeNode.Pods.map((pod) => (
                    <Tr key={pod.PodName}>
                        <Td dataLabel="Pod ID" modifier={'truncate'}>
                            {pod.PodName}
                        </Td>
                        <Td dataLabel="Phase">{pod.PodPhase}</Td>
                        <Td dataLabel="Age">{pod.PodAge}</Td>
                        <Td dataLabel="IP">{pod.PodIP}</Td>
                    </Tr>
                ))}
            </Tbody>
        </Table>
    );

    const expandedOrCollapseNode = (id: string) => {
        const index = expandedNodes.indexOf(id);
        const newExpanded =
            index >= 0
                ? [...expandedNodes.slice(0, index), ...expandedNodes.slice(index + 1, expandedNodes.length)]
                : [...expandedNodes, id];
        setExpandedNodes(newExpanded);
    };

    // Returns true if the node's radio button should be disabled.
    const shouldSelectBeDisabledForNode = (kubeNode: KubernetesNode) => {
        if (props.disableRadiosWithKernel == '' || props.disableRadiosWithKernel == undefined) {
            return false;
        }

        const kernelId: string = props.disableRadiosWithKernel!;
        for (let i = 0; i < kubeNode.Pods.length; i++) {
            const pod: KubernetesPod = kubeNode.Pods[i];
            if (pod.PodName.includes(kernelId)) return true;
        }

        return false;
    };

    const onClickNodeRow = (_event: React.MouseEvent | React.KeyboardEvent, id: string) => {
        const filteredNodeIndex: number = Number.parseInt(id.slice(id.lastIndexOf('-') + 1, id.length));
        const filteredNodeName: string = filteredNodes[filteredNodeIndex].NodeId;

        // Don't expand the control plane node.
        if (filteredNodeName.includes('control-plane')) {
            return;
        }

        // If the row is already expanded, then collapse it.
        // If the row is currently collapsed, then expand it.
        expandedOrCollapseNode(filteredNodeName);
    };

    const enableOrDisableNode = (kubeNode: KubernetesNode, checked: boolean) => {
        const requestBody = JSON.stringify({
            node_name: kubeNode.NodeId,
            enable: checked,
        });

        const requestOptions = {
            method: 'PATCH',
            headers: {
                'Content-Type': 'application/json',
                // 'Cache-Control': 'no-cache, no-transform, no-store',
            },
            body: requestBody,
        };

        fetch('api/nodes', requestOptions).then((resp) =>
            resp.json().then((updatedNode: KubernetesNode) => {
                console.log(`Received updated Kubernetes node: ${JSON.stringify(updatedNode)}`);
                for (let i: number = 0; i < nodes.length; i++) {
                    if (nodes[i].NodeId == updatedNode.NodeId) {
                        nodes[i] = updatedNode;
                        break;
                    }
                }

                forceUpdate();
            }),
        );
    };

    // The general info of the node (name, IP, and age).
    const nodeDescriptionList = (kubeNode: KubernetesNode) => {
        return (
            <DescriptionList
                isAutoColumnWidths
                className="node-list-description-list"
                columnModifier={{
                    sm: '2Col',
                    md: '2Col',
                    lg: '2Col',
                    xl: '3Col',
                    '2xl': '3Col',
                }}
            >
                <DescriptionListGroup>
                    <DescriptionListTerm icon={<VirtualMachineIcon />}>Node</DescriptionListTerm>
                    <DescriptionListDescription>{kubeNode.NodeId}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup className="node-data-list-ip">
                    <DescriptionListTerm icon={<GlobeIcon />}>IP</DescriptionListTerm>
                    <DescriptionListDescription>{kubeNode.IP}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup className="node-data-list-age">
                    <DescriptionListTerm icon={<OutlinedClockIcon />}>Age</DescriptionListTerm>
                    <DescriptionListDescription>{kubeNode.Age}</DescriptionListDescription>
                </DescriptionListGroup>
            </DescriptionList>
        );
    };

    // The current resource usage of the node.
    const nodeResourceAmounts = (kubeNode: KubernetesNode) => {
        return (
            <Flex
                spaceItems={{
                    md: 'spaceItemsLg',
                    lg: 'spaceItemsLg',
                    xl: 'spaceItemsXl',
                    '2xl': 'spaceItemsXl',
                }}
            >
                <Flex spaceItems={{ default: 'spaceItemsSm' }} alignSelf={{ default: 'alignSelfCenter' }}>
                    <FlexItem>
                        <CubeIcon className="node-pods-icon" />
                    </FlexItem>
                    <FlexItem>{kubeNode.Pods.length}</FlexItem>
                </Flex>
                <Flex spaceItems={{ default: 'spaceItemsSm' }} alignSelf={{ default: 'alignSelfCenter' }}>
                    <FlexItem>
                        <CpuIcon className="node-cpu-icon" />
                    </FlexItem>
                    <FlexItem>
                        {kubeNode.AllocatedCPU.toFixed(2)} / {kubeNode.CapacityCPU}
                    </FlexItem>
                </Flex>
                <Flex spaceItems={{ default: 'spaceItemsSm' }} alignSelf={{ default: 'alignSelfCenter' }}>
                    <FlexItem>
                        <MemoryIcon className="node-memory-icon" />
                    </FlexItem>
                    <FlexItem>
                        {kubeNode.AllocatedMemory.toFixed(2)} / {kubeNode.CapacityMemory.toFixed(0)}
                    </FlexItem>
                </Flex>
                <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                    <FlexItem alignSelf={{ default: 'alignSelfCenter' }}>
                        <GpuIcon scale={2.25} />
                    </FlexItem>
                    <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsMd' }}>
                        <Flex spaceItems={{ default: 'spaceItemsNone' }} direction={{ default: 'column' }}>
                            <FlexItem className="node-allocated-vgpu" align={{ default: 'alignRight' }}>
                                <Text component={TextVariants.p} className="node-allocated-vgpu">
                                    {kubeNode.AllocatedVGPUs.toFixed(0)}
                                </Text>
                            </FlexItem>
                            <FlexItem className="node-allocated-gpu" align={{ default: 'alignRight' }}>
                                <Text component={TextVariants.p} className="node-allocated-gpu">
                                    {kubeNode.AllocatedGPUs.toFixed(0)}
                                </Text>
                            </FlexItem>
                        </Flex>
                        <Flex spaceItems={{ default: 'spaceItemsNone' }} direction={{ default: 'column' }}>
                            <FlexItem align={{ default: 'alignRight' }}>/</FlexItem>
                            <FlexItem align={{ default: 'alignRight' }}>/</FlexItem>
                        </Flex>
                        <Flex spaceItems={{ default: 'spaceItemsNone' }} direction={{ default: 'column' }}>
                            <FlexItem align={{ default: 'alignRight' }}> {kubeNode.CapacityVGPUs}</FlexItem>
                            <FlexItem align={{ default: 'alignRight' }}>{kubeNode.CapacityGPUs}</FlexItem>
                        </Flex>
                        <Flex spaceItems={{ default: 'spaceItemsNone' }} direction={{ default: 'column' }}>
                            <FlexItem align={{ default: 'alignRight' }}>
                                <Text component={TextVariants.small}>(vGPUs)</Text>
                            </FlexItem>
                            <FlexItem align={{ default: 'alignRight' }}>
                                <Text component={TextVariants.small}>(GPUs)</Text>
                            </FlexItem>
                        </Flex>
                    </Flex>
                </Flex>
            </Flex>
        );
    };

    // The actions displayed at the right end of a row in the node list.
    const nodeDataListActions = (kubeNode: KubernetesNode) => {
        return (
            <Flex
                spaceItems={{ default: 'spaceItemsMd', '2xl': 'spaceItemsXs' }}
                direction={{ default: 'row', '2xl': 'column' }}
                alignSelf={{ default: 'alignSelfCenter' }}
                align={{ default: 'alignRight' }}
            >
                <FlexItem hidden={props.hideAdjustVirtualGPUsButton} alignSelf={{ default: 'alignSelfCenter' }}>
                    <Button
                        variant="link"
                        onClick={(event: React.MouseEvent) => {
                            event.stopPropagation();
                            if (props.onAdjustVirtualGPUsClicked) {
                                props.onAdjustVirtualGPUsClicked(kubeNode);
                            }
                        }}
                    >
                        Adjust vGPUs
                    </Button>
                </FlexItem>
                <FlexItem alignSelf={{ default: 'alignSelfCenter' }} hidden={kubeNode.NodeId.includes('control-plane')}>
                    <Tooltip
                        exitDelay={0.125}
                        content="Enable or disable a node, rendering it either available or unavailable, respectively, for hosting Distributed Notebook resources."
                        position={'bottom'}
                    >
                        <Switch
                            id={'node-' + kubeNode.NodeId + '-scheduling-switch'}
                            label={
                                <React.Fragment>
                                    <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsXs' }}>
                                        <Text component={TextVariants.h2}>Enabled</Text>
                                    </Flex>
                                </React.Fragment>
                            }
                            labelOff={
                                <React.Fragment>
                                    <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsXs' }}>
                                        <Text component={TextVariants.h2}>Disabled</Text>
                                    </Flex>
                                </React.Fragment>
                            }
                            aria-label="node-scheduling-switch"
                            isChecked={kubeNode.Enabled}
                            ouiaId="node-scheduling-switch"
                            onChange={(_event: React.FormEvent<HTMLInputElement>, checked: boolean) => {
                                enableOrDisableNode(kubeNode, checked);
                            }}
                        />
                    </Tooltip>
                </FlexItem>
            </Flex>
        );
    };

    const pagination = (
        <Pagination
            isDisabled={nodes.length == 0}
            itemCount={nodes.length}
            widgetId="node-list-pagination"
            perPage={perPage}
            page={page}
            variant={PaginationVariant.bottom}
            perPageOptions={[
                {
                    title: '1',
                    value: 1,
                },
                {
                    title: '2',
                    value: 2,
                },
                {
                    title: '3',
                    value: 3,
                },
            ]}
            onSetPage={onSetPage}
            onPerPageSelect={onPerPageSelect}
        />
    );

    let loadingNodes: number[] = [];
    if (nodes.length == 0) {
        loadingNodes = [0, 1];
    }

    return (
        <Card isFullHeight isRounded>
            <CardHeader
                actions={{ actions: toolbar, hasNoOffset: false }}
                toggleButtonProps={{
                    id: 'expand-kube-nodes-button',
                    'aria-label': 'expand-kube-nodes-button',
                }}
            >
                <CardTitle>
                    <Title headingLevel="h1" size="xl">
                        Kubernetes Nodes
                    </Title>
                </CardTitle>
            </CardHeader>
            <CardBody>
                <DataList isCompact aria-label="nodes-loading-list" hidden={nodes.length > 0}>
                    {loadingNodes.map((idx: number) => (
                        <DataListItem
                            key={`loading-kube-node-${idx}`}
                            id={'loading-kube-node-list-item-' + idx}
                            isExpanded={false}
                        >
                            <DataListItemCells
                                dataListCells={[
                                    <DataListCell key={`loading-node-${idx}-primary-content`}>
                                        <Flex
                                            direction={{ default: 'column' }}
                                            spaceItems={{ default: 'spaceItemsXs' }}
                                        >
                                            <FlexItem>
                                                <Skeleton screenreaderText="Loading Kubernetes nodes" width="10%" />
                                            </FlexItem>
                                            <FlexItem>
                                                <div style={{ height: '90px' }}>
                                                    <Skeleton
                                                        screenreaderText="Loading Kubernetes nodes"
                                                        width="100%"
                                                        height="85%"
                                                    />
                                                </div>
                                            </FlexItem>
                                        </Flex>
                                    </DataListCell>,
                                ]}
                            />
                        </DataListItem>
                    ))}
                </DataList>
                <DataList
                    onSelectDataListItem={onClickNodeRow}
                    isCompact
                    aria-label="data list"
                    hidden={nodes.length == 0}
                >
                    {filteredNodes.map((kubeNode: KubernetesNode, idx: number) => (
                        <DataListItem
                            key={kubeNode.NodeId}
                            id={'node-list-item-' + idx}
                            isExpanded={expandedNodes.includes(kubeNode.NodeId)}
                        >
                            <DataListItemRow>
                                {props.selectableViaCheckboxes && (
                                    <DataListControl>
                                        <Radio
                                            id={'node-' + kubeNode.NodeId + '-radio'}
                                            aria-label={'node-' + kubeNode.NodeId + '-radio'}
                                            aria-labelledby={'node-' + kubeNode.NodeId + '-radio'}
                                            name={'node-list-radio-buttons'}
                                            hidden={!props.selectableViaCheckboxes}
                                            isDisabled={shouldSelectBeDisabledForNode(kubeNode)}
                                            onChange={() => {
                                                console.log('Selected node ' + kubeNode.NodeId);
                                                setSelectedNode(kubeNode.NodeId);
                                                if (props.onSelectNode != undefined) {
                                                    props.onSelectNode(kubeNode.NodeId);
                                                }
                                            }}
                                            isChecked={kubeNode.NodeId == selectedNode}
                                        />
                                    </DataListControl>
                                )}
                                <DataListToggle
                                    className="node-list-toggle-button"
                                    hidden={kubeNode.NodeId.includes('control-plane')}
                                    onClick={() => expandedOrCollapseNode(kubeNode.NodeId)}
                                    isExpanded={expandedNodes.includes(kubeNode.NodeId)}
                                    id={'expand-node-' + kubeNode.NodeId + '-toggle'}
                                    aria-controls={'expand-node-' + kubeNode.NodeId + '-toggle'}
                                />
                                <DataListItemCells
                                    dataListCells={[
                                        <DataListCell key={`node-${kubeNode.NodeId}-primary-content`}>
                                            <Flex
                                                direction={{ default: 'column', '2xl': 'row' }}
                                                spaceItems={{
                                                    default: 'spaceItemsNone',
                                                    '2xl': 'spaceItems2xl',
                                                }}
                                            >
                                                <Flex
                                                    className="node-list-content"
                                                    spaceItems={{
                                                        default: 'spaceItemsNone',
                                                        sm: 'spaceItemsNone',
                                                        md: 'spaceItemsNone',
                                                        lg: 'spaceItemsNone',
                                                        xl: 'spaceItemsSm',
                                                    }}
                                                    direction={{ default: 'column' }}
                                                >
                                                    <FlexItem>{nodeDescriptionList(kubeNode)}</FlexItem>
                                                    <FlexItem>{nodeResourceAmounts(kubeNode)}</FlexItem>
                                                </Flex>
                                                {nodeDataListActions(kubeNode)}
                                            </Flex>
                                        </DataListCell>,
                                    ]}
                                />
                            </DataListItemRow>
                            <DataListContent
                                className="node-list-expandable-content"
                                aria-label={'node-' + kubeNode.NodeId + '-expandable-content'}
                                id={'node-' + kubeNode.NodeId + '-expandable-content'}
                                isHidden={!expandedNodes.includes(kubeNode.NodeId)}
                            >
                                {expandedNodeContent(kubeNode)}
                            </DataListContent>
                        </DataListItem>
                    ))}
                </DataList>
                {pagination}
            </CardBody>
        </Card>
    );
};
