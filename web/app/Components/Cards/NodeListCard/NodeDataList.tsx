import { HeightFactorContext, NodeHeightFactorContext } from '@app/Dashboard/Dashboard';
import { GpuIcon } from '@app/Icons';
import { ClusterNode, PodOrContainer } from '@data/Cluster';
import {
  Button,
  DataList,
  DataListAction,
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
  FlexItem, Icon, Label,
  Pagination,
  PaginationVariant,
  Radio,
  Skeleton,
  Switch,
  Text,
  TextVariants,
  Tooltip
} from '@patternfly/react-core';
import {
  CheckCircleIcon,
  CheckIcon,
  CpuIcon, CrossIcon,
  CubeIcon,
  GlobeIcon,
  MemoryIcon,
  OutlinedClockIcon, TimesCircleIcon, TimesIcon,
  VirtualMachineIcon
} from '@patternfly/react-icons';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { useNodes } from '@providers/NodeProvider';
import React, { useReducer } from 'react';
import { toast } from 'react-hot-toast';

export interface NodeDataListProps {
    selectableViaCheckboxes: boolean;
    isDashboardList: boolean; // Indicates whether the node list is the primary list that appears on the dashboard
    disableRadiosWithKernel?: string; // KernelID such that, if a node has a Pod for that kernel, its radio button is disabled.
    hideControlPlaneNode?: boolean;
    onSelectNode?: (nodeId: string) => void; // Function to call when a node is selected; used in case parent wants to do something when node is selected, such as update state.
    nodesPerPage: number;
    hideAdjustVirtualGPUsButton: boolean;
    displayNodeToggleSwitch: boolean; // If true, show the Switch that is used to enable/disable the node.
    onFilter: (repo: ClusterNode) => boolean;
    onAdjustVirtualGPUsClicked: (nodes: ClusterNode[]) => void;
}

export const NodeDataList: React.FunctionComponent<NodeDataListProps> = (props: NodeDataListProps) => {
    const [expandedNodes, setExpandedNodes] = React.useState<string[]>([]);
    const [page, setPage] = React.useState(1);
    const [perPage, setPerPage] = React.useState(props.nodesPerPage);
    const { nodes } = useNodes();
    const [selectedNode, setSelectedNode] = React.useState('');

    const [, forceUpdate] = useReducer((x) => x + 1, 0);

    const heightFactorContext: HeightFactorContext = React.useContext(NodeHeightFactorContext);

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
                    title: '1 nodes',
                    value: 1,
                },
                {
                    title: '2 nodes',
                    value: 2,
                },
                {
                    title: '3 nodes',
                    value: 3,
                },
                {
                    title: '4 nodes',
                    value: 4,
                },
                {
                    title: '5 nodes',
                    value: 5,
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

    const expandedNodeContent = (clusterNode: ClusterNode) => (
        <Table isStriped aria-label="Pods Table" variant={'compact'} borders={true}>
            <Thead>
                <Tr>
                    <Th aria-label={'container-id'}>ID</Th>
                    <Th aria-label={'container-status'}>Status</Th>
                    <Th aria-label={'container-age'}>Age</Th>
                    <Th aria-label={'container-label'}>IP</Th>
                </Tr>
            </Thead>
            <Tbody>
                {clusterNode.PodsOrContainers.map((container) => (
                    <Tr key={container.Name}>
                        <Td dataLabel="ID" modifier={'truncate'}>
                            {container.Name}
                        </Td>
                        <Td dataLabel="Phase">{container.Phase}</Td>
                        <Td dataLabel="Age">{container.Age}</Td>
                        <Td dataLabel="IP">{container.IP}</Td>
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
    const shouldSelectBeDisabledForNode = (clusterNode: ClusterNode) => {
        if (props.disableRadiosWithKernel == '' || props.disableRadiosWithKernel == undefined) {
            return false;
        }

        const kernelId: string = props.disableRadiosWithKernel!;
        for (let i = 0; i < clusterNode.PodsOrContainers.length; i++) {
            const podOrContainer: PodOrContainer = clusterNode.PodsOrContainers[i];
            if (podOrContainer.Name.includes(kernelId)) {
                // console.log(
                //     `Pod/Container ${podOrContainer.Name} is a replica of kernel ${kernelId}. Disabling node ${clusterNode.NodeId}.`,
                // );
                return true;
            } else {
                // console.log(`Pod/Container ${podOrContainer.Name} is not a replica of kernel ${kernelId}...`);
            }
        }

        // console.log(`Node ${clusterNode.NodeId} has no replicas of kernel ${kernelId}.`);
        return false;
    };

    const enableOrDisableNode = (clusterNode: ClusterNode, checked: boolean) => {
        const requestBody = JSON.stringify({
            node_name: clusterNode.NodeId,
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

        fetch('api/nodes', requestOptions).then((resp) => {
            if (resp.status >= 300) {
                resp.text().then((text: string) => {
                    toast.error(
                        () => {
                            return (
                                <div>
                                    <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
                                        <FlexItem>
                                            <b>{`Failed to ${checked ? 'enable' : 'disable'} node ${clusterNode.NodeId}.`}</b>
                                        </FlexItem>
                                        <FlexItem>{`HTTP ${resp.status} - ${resp.statusText}: ${text}`}</FlexItem>
                                    </Flex>
                                </div>
                            );
                        },
                        { style: { maxWidth: 575 } },
                    );
                });
            } else {
                resp.json().then((updatedNode: ClusterNode) => {
                    console.log(`Received updated Kubernetes node: ${JSON.stringify(updatedNode)}`);
                    for (let i: number = 0; i < nodes.length; i++) {
                        if (nodes[i].NodeId == updatedNode.NodeId) {
                            nodes[i] = updatedNode;
                            break;
                        }
                    }

                    forceUpdate();
                });
            }
        });
    };

    const filteredNodes =
        nodes.length > 0
            ? nodes.filter(props.onFilter).slice(perPage * (page - 1), perPage * (page - 1) + perPage)
            : [];

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

    // The actions displayed at the right end of a row in the node list.
    const nodeDataListActions = (clusterNode: ClusterNode) => {
        return (
            <Flex
                spaceItems={{ default: 'spaceItemsMd', '2xl': 'spaceItemsMd' }}
                direction={{ default: 'row', '2xl': 'column' }}
                alignSelf={{ default: 'alignSelfCenter' }}
                align={{ default: 'alignRight' }}
            >
                <FlexItem hidden={props.hideAdjustVirtualGPUsButton} alignSelf={{ default: 'alignSelfCenter' }}>
                    <Button
                        variant="link"
                        onClick={(event: React.MouseEvent) => {
                            event.stopPropagation();
                            props.onAdjustVirtualGPUsClicked([clusterNode]);
                        }}
                    >
                        Adjust vGPUs
                    </Button>
                </FlexItem>
                <FlexItem
                    alignSelf={{ default: 'alignSelfCenter' }}
                    hidden={props.isDashboardList}
                >
                  <Tooltip
                    exitDelay={0.125}
                    content={shouldSelectBeDisabledForNode(clusterNode) ? "This node is not a a viable migration target." : "This node is a viable migration target."}
                    position={'right'}
                  >
                    <Label icon={shouldSelectBeDisabledForNode(clusterNode) ? <TimesCircleIcon/> : <CheckCircleIcon/> } color={shouldSelectBeDisabledForNode(clusterNode) ? "red" : "green"}>
                      {shouldSelectBeDisabledForNode(clusterNode) ? "Non-viable" : "Viable"}
                    </Label>
                  </Tooltip>
                </FlexItem>
                <FlexItem
                    alignSelf={{ default: 'alignSelfCenter' }}
                    hidden={clusterNode.NodeId.includes('control-plane') || !props.isDashboardList}
                >
                    <Tooltip
                        exitDelay={0.125}
                        content="Enable or disable a node, rendering it either available or unavailable, respectively, for hosting Distributed Notebook resources."
                        position={'left'}
                    >
                        <Switch
                            id={'node-' + clusterNode.NodeId + '-scheduling-switch'}
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
                            isChecked={clusterNode.Enabled}
                            ouiaId="node-scheduling-switch"
                            onChange={(_event: React.FormEvent<HTMLInputElement>, checked: boolean) => {
                                enableOrDisableNode(clusterNode, checked);
                            }}
                        />
                    </Tooltip>
                </FlexItem>
            </Flex>
        );
    };

    // The general info of the node (name, IP, and age).
    const nodeDescriptionList = (clusterNode: ClusterNode) => {
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
                    <DescriptionListTerm icon={<VirtualMachineIcon />}>Node ID</DescriptionListTerm>
                    <DescriptionListDescription>{clusterNode.NodeId}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup className="node-data-list-ip">
                    <DescriptionListTerm icon={<GlobeIcon />}>IP</DescriptionListTerm>
                    <DescriptionListDescription>{clusterNode.IP}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup className="node-data-list-age">
                    <DescriptionListTerm icon={<OutlinedClockIcon />}>Age</DescriptionListTerm>
                    <DescriptionListDescription>{clusterNode.Age}</DescriptionListDescription>
                </DescriptionListGroup>
            </DescriptionList>
        );
    };

    // The current resource usage of the node.
    const nodeResourceAmounts = (clusterNode: ClusterNode) => {
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
                        <Tooltip content="Number of Pods scheduled onto this node">
                            <CubeIcon className="node-pods-icon" />
                        </Tooltip>
                    </FlexItem>
                    <FlexItem>{clusterNode.PodsOrContainers.length}</FlexItem>
                </Flex>
                <Flex spaceItems={{ default: 'spaceItemsSm' }} alignSelf={{ default: 'alignSelfCenter' }}>
                    <FlexItem>
                        <Tooltip content="Committed/allocated millicpus (1/1000th of a CPU core)">
                            <CpuIcon className="node-cpu-icon" />
                        </Tooltip>
                    </FlexItem>
                    <FlexItem>
                        {clusterNode.AllocatedResources['CPU'].toFixed(2)} / {clusterNode.CapacityResources['CPU']}
                    </FlexItem>
                </Flex>
                <Flex spaceItems={{ default: 'spaceItemsSm' }} alignSelf={{ default: 'alignSelfCenter' }}>
                    <FlexItem>
                        <Tooltip content="Committed/allocated RAM in Gigabytes">
                            <MemoryIcon className="node-memory-icon" />
                        </Tooltip>
                    </FlexItem>
                    <FlexItem>
                        {(clusterNode.AllocatedResources['Memory'] * 0.001048576).toFixed(2)} /
                        {(clusterNode.CapacityResources['Memory'] * 0.001048576).toFixed(2)}
                    </FlexItem>
                </Flex>
                <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                    <FlexItem alignSelf={{ default: 'alignSelfCenter' }}>
                        <Tooltip content="Committed/allocated GPUs">
                            <GpuIcon scale={2.25} />
                        </Tooltip>
                    </FlexItem>
                    <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsMd' }}>
                        <Flex spaceItems={{ default: 'spaceItemsNone' }} direction={{ default: 'column' }}>
                            <FlexItem className="node-allocated-vgpu" align={{ default: 'alignRight' }}>
                                <Text component={TextVariants.p} className="node-allocated-vgpu">
                                    {clusterNode.AllocatedResources['vGPU'].toFixed(0)}
                                </Text>
                            </FlexItem>
                            <FlexItem className="node-allocated-gpu" align={{ default: 'alignRight' }}>
                                <Text component={TextVariants.p} className="node-allocated-gpu">
                                    {clusterNode.AllocatedResources['GPU'].toFixed(0)}
                                </Text>
                            </FlexItem>
                        </Flex>
                        <Flex spaceItems={{ default: 'spaceItemsNone' }} direction={{ default: 'column' }}>
                            <FlexItem align={{ default: 'alignRight' }}>/</FlexItem>
                            <FlexItem align={{ default: 'alignRight' }}>/</FlexItem>
                        </Flex>
                        <Flex spaceItems={{ default: 'spaceItemsNone' }} direction={{ default: 'column' }}>
                            <FlexItem align={{ default: 'alignRight' }}>
                                {' '}
                                {clusterNode.CapacityResources['vGPU']}
                            </FlexItem>
                            <FlexItem align={{ default: 'alignRight' }}>
                                {clusterNode.CapacityResources['GPU']}
                            </FlexItem>
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

    return (
        <React.Fragment>
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
                                    <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsXs' }}>
                                        <FlexItem>
                                            <Skeleton screenreaderText="Loading nodes" width="10%" />
                                        </FlexItem>
                                        <FlexItem>
                                            <div style={{ height: '90px' }}>
                                                <Skeleton screenreaderText="Loading nodes" width="100%" height="85%" />
                                            </div>
                                        </FlexItem>
                                    </Flex>
                                </DataListCell>,
                            ]}
                        />
                    </DataListItem>
                ))}
            </DataList>
            <DataList onSelectDataListItem={onClickNodeRow} isCompact aria-label="data list" hidden={nodes.length == 0}>
                {filteredNodes.map((clusterNode: ClusterNode, idx: number) => (
                    <DataListItem
                        key={clusterNode.NodeId}
                        id={'node-list-item-' + idx}
                        isExpanded={expandedNodes.includes(clusterNode.NodeId)}
                    >
                        <DataListItemRow>
                            {props.selectableViaCheckboxes && (
                                <DataListControl>
                                    <Radio
                                        id={'node-' + clusterNode.NodeId + '-radio'}
                                        aria-label={'node-' + clusterNode.NodeId + '-radio'}
                                        aria-labelledby={'node-' + clusterNode.NodeId + '-radio'}
                                        name={'node-list-radio-buttons'}
                                        hidden={!props.selectableViaCheckboxes}
                                        isDisabled={shouldSelectBeDisabledForNode(clusterNode)}
                                        onChange={() => {
                                            console.log('Selected node ' + clusterNode.NodeId);
                                            setSelectedNode(clusterNode.NodeId);
                                            if (props.onSelectNode != undefined) {
                                                props.onSelectNode(clusterNode.NodeId);
                                            }
                                        }}
                                        isChecked={clusterNode.NodeId == selectedNode}
                                    />
                                </DataListControl>
                            )}
                            <DataListToggle
                                className="node-list-toggle-button"
                                hidden={clusterNode.NodeId.includes('control-plane')}
                                onClick={() => expandedOrCollapseNode(clusterNode.NodeId)}
                                isExpanded={expandedNodes.includes(clusterNode.NodeId)}
                                id={'expand-node-' + clusterNode.NodeId + '-toggle'}
                                aria-controls={'expand-node-' + clusterNode.NodeId + '-toggle'}
                            />
                            <DataListItemCells
                                id={`node-detail-view-data-list`}
                                dataListCells={[
                                    <DataListCell key={`node-${clusterNode.NodeId}-primary-content`}>
                                        <Flex
                                            direction={{ default: 'row', '2xl': 'row' }}
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
                                                <FlexItem>{nodeDescriptionList(clusterNode)}</FlexItem>
                                                <FlexItem>{nodeResourceAmounts(clusterNode)}</FlexItem>
                                            </Flex>
                                        </Flex>
                                    </DataListCell>,
                                    <DataListAction
                                        id={`node-${clusterNode.NodeId}-data-list-actions`}
                                        key={`node-${clusterNode.NodeId}-data-list-actions`}
                                        aria-label={`Data list actions for node ${clusterNode.NodeId}`}
                                        aria-labelledby={`node-detail-view-data-list`}
                                    >
                                        {nodeDataListActions(clusterNode)}
                                    </DataListAction>,
                                ]}
                            />
                        </DataListItemRow>
                        <DataListContent
                            className="node-list-expandable-content"
                            aria-label={'node-' + clusterNode.NodeId + '-expandable-content'}
                            id={'node-' + clusterNode.NodeId + '-expandable-content'}
                            isHidden={!expandedNodes.includes(clusterNode.NodeId)}
                        >
                            {expandedNodeContent(clusterNode)}
                        </DataListContent>
                    </DataListItem>
                ))}
            </DataList>
            {pagination}
        </React.Fragment>
    );
};
