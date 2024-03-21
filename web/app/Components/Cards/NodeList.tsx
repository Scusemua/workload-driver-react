import React from 'react';
import {
    Button,
    Card,
    CardBody,
    CardExpandableContent,
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
    Title,
    ToolbarGroup,
    ToolbarItem,
    ToolbarToggleGroup,
    Tooltip,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import GpuIcon from '@app/Icons/GpuIcon';
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
import { useNodes } from '../Providers/NodeProvider';

export interface NodeListProps {
    selectable: boolean;
    disableRadiosWithKernel?: string; // KernelID such that, if a node has a Pod for that kernel, its radio button is disabled.
    hideControlPlaneNode?: boolean;
    onSelectNode?: (nodeId: string) => void; // Function to call when a node is selected; used in case parent wants to do something when node is selected, such as update state.
    nodesPerPage: number;
}

export const KubernetesNodeList: React.FunctionComponent<NodeListProps> = (props: NodeListProps) => {
    const [searchValue, setSearchValue] = React.useState('');
    const [isCardExpanded, setIsCardExpanded] = React.useState(true);
    const [expandedNodes, setExpandedNodes] = React.useState<string[]>([]);
    const [selectedNode, setSelectedNode] = React.useState('');
    const [page, setPage] = React.useState(1);
    const [perPage, setPerPage] = React.useState(props.nodesPerPage);
    const { nodes, nodesAreLoading, refreshNodes } = useNodes();

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
        console.log(
            'onPerPageSelect: Displaying workloads %d through %d.',
            newPerPage * (newPage - 1),
            newPerPage * (newPage - 1) + newPerPage,
        );
    };

    const onCardExpand = () => {
        setIsCardExpanded(!isCardExpanded);
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
    const filteredNodes = nodes.length > 0 ? nodes.filter(onFilter) : [];

    const toolbar = (
        <React.Fragment>
            <ToolbarToggleGroup toggleIcon={<FilterIcon />} breakpoint="xl">
                <Flex alignItems={{ default: 'alignItemsCenter' }}>
                    <FlexItem hidden={!props.selectable}>
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
                                refreshNodes();
                            }}
                            isDisabled={nodesAreLoading}
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

    const toggleExpandedNode = (id) => {
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

    return (
        <Card isRounded isExpanded={isCardExpanded}>
            <CardHeader
                onExpand={onCardExpand}
                actions={{ actions: toolbar, hasNoOffset: true }}
                toggleButtonProps={{
                    id: 'expand-kube-nodes-button',
                    'aria-label': 'expand-kube-nodes-button',
                    'aria-expanded': isCardExpanded,
                }}
            >
                <CardTitle>
                    <Title headingLevel="h1" size="xl">
                        Kubernetes Nodes
                    </Title>
                </CardTitle>
            </CardHeader>
            <CardExpandableContent>
                <CardBody>
                    <DataList isCompact aria-label="data list" hidden={nodes.length == 0}>
                        {filteredNodes
                            .slice(perPage * (page - 1), perPage * (page - 1) + perPage)
                            .map((kubeNode: KubernetesNode, idx: number) => (
                                <DataListItem
                                    key={kubeNode.NodeId}
                                    id={'node-list-item-' + idx}
                                    isExpanded={expandedNodes.includes(kubeNode.NodeId)}
                                >
                                    <DataListItemRow>
                                        {props.selectable && (
                                            <DataListControl>
                                                <Radio
                                                    id={'node-' + kubeNode.NodeId + '-radio'}
                                                    aria-label={'node-' + kubeNode.NodeId + '-radio'}
                                                    aria-labelledby={'node-' + kubeNode.NodeId + '-radio'}
                                                    name={'node-list-radio-buttons'}
                                                    hidden={!props.selectable}
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
                                            onClick={() => toggleExpandedNode(kubeNode.NodeId)}
                                            isExpanded={expandedNodes.includes(kubeNode.NodeId)}
                                            id={'expand-node-' + kubeNode.NodeId + '-toggle'}
                                            aria-controls={'expand-node-' + kubeNode.NodeId + '-toggle'}
                                        />
                                        <DataListItemCells
                                            dataListCells={[
                                                <DataListCell key="primary-content">
                                                    <Flex
                                                        spaceItems={{ default: 'spaceItemsMd' }}
                                                        direction={{ default: 'column' }}
                                                    >
                                                        <FlexItem>
                                                            <DescriptionList
                                                                className="node-list-description-list"
                                                                columnModifier={{
                                                                    sm: '2Col',
                                                                    md: '2Col',
                                                                    lg: '3Col',
                                                                    xl: '3Col',
                                                                }}
                                                            >
                                                                <DescriptionListGroup>
                                                                    <DescriptionListTerm icon={<VirtualMachineIcon />}>
                                                                        Node
                                                                    </DescriptionListTerm>
                                                                    <DescriptionListDescription>
                                                                        {kubeNode.NodeId}
                                                                    </DescriptionListDescription>
                                                                </DescriptionListGroup>
                                                                <DescriptionListGroup>
                                                                    <DescriptionListTerm icon={<GlobeIcon />}>
                                                                        IP
                                                                    </DescriptionListTerm>
                                                                    <DescriptionListDescription>
                                                                        {kubeNode.IP}
                                                                    </DescriptionListDescription>
                                                                </DescriptionListGroup>
                                                                <DescriptionListGroup>
                                                                    <DescriptionListTerm icon={<OutlinedClockIcon />}>
                                                                        Age
                                                                    </DescriptionListTerm>
                                                                    <DescriptionListDescription>
                                                                        {kubeNode.Age}
                                                                    </DescriptionListDescription>
                                                                </DescriptionListGroup>
                                                            </DescriptionList>
                                                        </FlexItem>
                                                        <FlexItem>
                                                            <Flex spaceItems={{ default: 'spaceItems4xl' }}>
                                                                <FlexItem>
                                                                    <CubeIcon /> {kubeNode.Pods.length}
                                                                </FlexItem>
                                                                <FlexItem>
                                                                    <CpuIcon /> {kubeNode.AllocatedCPU.toFixed(2)} /{' '}
                                                                    {kubeNode.CapacityCPU}
                                                                </FlexItem>
                                                                <FlexItem>
                                                                    <MemoryIcon /> {kubeNode.AllocatedMemory.toFixed(2)}{' '}
                                                                    / {kubeNode.CapacityMemory.toFixed(0)}
                                                                </FlexItem>
                                                                <FlexItem>
                                                                    <GpuIcon /> {kubeNode.AllocatedCPU.toFixed(2)} /{' '}
                                                                    {kubeNode.CapacityCPU}
                                                                </FlexItem>
                                                            </Flex>
                                                        </FlexItem>
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
                    <Pagination
                        isDisabled={nodes.length == 0}
                        itemCount={nodes.length}
                        widgetId="bottom-example"
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
                            {
                                title: '4',
                                value: 4,
                            },
                            {
                                title: '5',
                                value: 5,
                            },
                        ]}
                        onSetPage={onSetPage}
                        onPerPageSelect={onPerPageSelect}
                    />
                </CardBody>
            </CardExpandableContent>
        </Card>
    );
};
