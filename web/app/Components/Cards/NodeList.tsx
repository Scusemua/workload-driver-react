import { HeightFactorContext, NodeHeightFactorContext } from '@app/Dashboard/Dashboard';
import { GpuIcon } from '@app/Icons';
import { ClusterNode, PodOrContainer } from '@data/Cluster';
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
  Tooltip
} from '@patternfly/react-core';
import {
  CpuIcon,
  CubeIcon,
  FilterIcon,
  GlobeIcon,
  MemoryIcon,
  OutlinedClockIcon,
  SyncIcon,
  VirtualMachineIcon
} from '@patternfly/react-icons';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { useNodes } from '@providers/NodeProvider';
import React, { useReducer } from 'react';
import { toast } from 'react-hot-toast';
import { AdjustVirtualGPUsModal } from '../Modals';

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

function roundToTwo(num: number) {
  return +(Math.round(Number.parseFloat(num.toString() + 'e+2')).toString() + 'e-2');
}

export const NodeList: React.FunctionComponent<NodeListProps> = (props: NodeListProps) => {
  const [searchValue, setSearchValue] = React.useState('');
  const [expandedNodes, setExpandedNodes] = React.useState<string[]>([]);
  const [selectedNode, setSelectedNode] = React.useState('');
  const [page, setPage] = React.useState(1);
  const [perPage, setPerPage] = React.useState(props.nodesPerPage);
  const { nodes, nodesAreLoading, refreshNodes } = useNodes();
  const [, forceUpdate] = useReducer((x) => x + 1, 0);
  const heightFactorContext: HeightFactorContext = React.useContext(NodeHeightFactorContext);
  const [adjustVirtualGPUsNodes, setAdjustVirtualGPUsNodes] = React.useState<ClusterNode[]>([]);
  const [isAdjustVirtualGPUsModalOpen, setIsAdjustVirtualGPUsModalOpen] = React.useState(false);

  const onSetPage = (_event: React.MouseEvent | React.KeyboardEvent | MouseEvent, newPage: number) => {
    setPage(newPage);
    console.log(
      'onSetPage: Displaying workloads %d through %d.',
      perPage * (newPage - 1),
      perPage * (newPage - 1) + perPage
    );
  };

  const onPerPageSelect = (
    _event: React.MouseEvent | React.KeyboardEvent | MouseEvent,
    newPerPage: number,
    newPage: number
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

  const onAdjustVirtualGPUsClicked = (nodes: ClusterNode[]) => {
    setAdjustVirtualGPUsNodes(nodes);
    setIsAdjustVirtualGPUsModalOpen(true);
  };

  const closeAdjustVirtualGPUsModal = () => {
    setIsAdjustVirtualGPUsModalOpen(false);
    setAdjustVirtualGPUsNodes([]);
  };

  async function doAdjustVirtualGPUs(value: number) {
    if (adjustVirtualGPUsNodes.length == 0) {
      console.error('Field \'adjustVirtualGPUsNode\' is null...');
      closeAdjustVirtualGPUsModal();
      return;
    }

    if (Number.isNaN(value)) {
      console.error('Specified value is NaN...');
      closeAdjustVirtualGPUsModal();
      return;
    }

    adjustVirtualGPUsNodes.forEach((node: ClusterNode) => {
      if (node.CapacityResources['vGPU'] == value) {
        console.log('Adjusted vGPUs value is same as current value. Doing nothing.');
        closeAdjustVirtualGPUsModal();
        return;
      }

      const requestOptions = {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json'
          // 'Cache-Control': 'no-cache, no-transform, no-store'
        },
        body: JSON.stringify({
          value: value,
          kubernetesNodeName: node?.NodeId
        })
      };

      console.log(`Attempting to set vGPUs on node ${node?.NodeId} to ${value}`);

      toast.promise(
        fetch('api/vgpus', requestOptions),
        {
          loading: 'Adjusting GPUs...',
          success: (
            <div>
              <Flex>
                <FlexItem>
                  <Text component={TextVariants.p}>
                    <b>Successfully updated vGPU capacity for node {node.NodeId}.</b>
                  </Text>
                </FlexItem>
                <FlexItem>
                  <Text component={TextVariants.small}>
                    It may take several seconds for the updated value to appear.
                  </Text>
                </FlexItem>
              </Flex>
            </div>
          ),
          error: (reason) => (
            <div>
              <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
                <FlexItem>
                  <b>Failed to update vGPUs for node ${node.NodeId} because:</b>
                </FlexItem>
                <FlexItem>{JSON.stringify(reason)}</FlexItem>
              </Flex>
            </div>
          )
        },
        {
          duration: 5000,
          style: { maxWidth: 450 }
        }
      );
    });
  }

  // Handler for when the user filters by node name.
  const onFilter = (repo: ClusterNode) => {
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

  // The message displayed in a Toast when a node refresh completes successfully.
  const successfulRefreshMessage = (st: number) => {
    const et: number = performance.now();
    console.log(`Successful refresh. Start time: ${st}. End time: ${et}. Time elapsed: ${et - st} ms.`);
    return (
      <div>
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
          <FlexItem>
            <b>Refreshed nodes.</b>
          </FlexItem>
          <FlexItem>
            <Text component={TextVariants.small}>Time elapsed: {roundToTwo(et - st)} ms.</Text>
          </FlexItem>
        </Flex>
      </div>
    );
  };

  // The message displayed in a Toast when a node refresh fails to complete.
  const failedRefreshMessage = (reason: Error) => {
    let explanation: string = reason.message;
    if (reason.name === 'SyntaxError') {
      explanation = 'HTTP 504 Gateway Timeout. (Is your kubeconfig correct?)';
    }

    return (
      <div>
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
          <FlexItem>
            <b>Could not refresh nodes.</b>
          </FlexItem>
          <FlexItem>{explanation}</FlexItem>
        </Flex>
      </div>
    );
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
          <FlexItem hidden={nodes.length == 0}>
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
      <ToolbarGroup variant="button-group">
        <ToolbarItem>
          <Tooltip content="Adjust the number of vGPUs available on ALL nodes.">
            <Button
              variant="link"
              onClick={(event: React.MouseEvent) => {
                event.stopPropagation();
                onAdjustVirtualGPUsClicked(nodes);
              }}
            >
              Adjust vGPUs
            </Button>
          </Tooltip>
        </ToolbarItem>
      </ToolbarGroup>
      <ToolbarGroup variant="icon-button-group">
        <ToolbarItem>
          <Tooltip exitDelay={75} content={<div>Refresh nodes.</div>}>
            <Button
              variant="plain"
              isDisabled={nodesAreLoading}
              onClick={() => {
                console.log('Refreshing nodes now.');
                const st: number = performance.now();
                return toast.promise(
                  refreshNodes(),
                  {
                    loading: <b>Refreshing nodes...</b>,
                    success: () => successfulRefreshMessage(st),
                    error: (reason: Error) => failedRefreshMessage(reason)
                  },
                  {
                    style: { maxWidth: 450 }
                  }
                );
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
      if (podOrContainer.Name.includes(kernelId)) return true;
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

  const enableOrDisableNode = (clusterNode: ClusterNode, checked: boolean) => {
    const requestBody = JSON.stringify({
      node_name: clusterNode.NodeId,
      enable: checked
    });

    const requestOptions = {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json'
        // 'Cache-Control': 'no-cache, no-transform, no-store',
      },
      body: requestBody
    };

    fetch('api/nodes', requestOptions)
      .then((resp) => {
        if (resp.status >= 300) {
          resp.text().then((text: string) => {
            toast.error(() => {
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
            }, { style: { maxWidth: 575 } });
          })
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
  }

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
          '2xl': '3Col'
        }}
      >
        <DescriptionListGroup>
          <DescriptionListTerm icon={<VirtualMachineIcon />}>Node</DescriptionListTerm>
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
          '2xl': 'spaceItemsXl'
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
            <Tooltip content="millicpus (1/1000th of a CPU core)">
              <CpuIcon className="node-cpu-icon" />
            </Tooltip>
          </FlexItem>
          <FlexItem>
            {clusterNode.AllocatedResources['CPU'].toFixed(2)} / {clusterNode.CapacityResources['CPU']}
          </FlexItem>
        </Flex>
        <Flex spaceItems={{ default: 'spaceItemsSm' }} alignSelf={{ default: 'alignSelfCenter' }}>
          <FlexItem>
            <Tooltip content="RAM in Gigabytes">
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
            <GpuIcon scale={2.25} />
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

  // The actions displayed at the right end of a row in the node list.
  const nodeDataListActions = (clusterNode: ClusterNode) => {
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
              onAdjustVirtualGPUsClicked([clusterNode]);
            }}
          >
            Adjust vGPUs
          </Button>
        </FlexItem>
        <FlexItem
          alignSelf={{ default: 'alignSelfCenter' }}
          hidden={clusterNode.NodeId.includes('control-plane')}
        >
          <Tooltip
            exitDelay={0.125}
            content="Enable or disable a node, rendering it either available or unavailable, respectively, for hosting Distributed Notebook resources."
            position={'bottom'}
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
          value: 1
        },
        {
          title: '2 nodes',
          value: 2
        },
        {
          title: '3 nodes',
          value: 3
        },
        {
          title: '4 nodes',
          value: 4
        },
        {
          title: '5 nodes',
          value: 5
        }
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
    <Card isFullHeight isRounded id={props.isDashboardList ? 'primary-node-list-card' : 'migration-node-list-card'}>
      <CardHeader
        actions={{ actions: toolbar, hasNoOffset: false }}
        toggleButtonProps={{
          id: 'expand-kube-nodes-button',
          'aria-label': 'expand-kube-nodes-button'
        }}
      >
        <CardTitle>
          <Title headingLevel="h1" size="xl">
            Nodes
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
                        <Skeleton screenreaderText="Loading nodes" width="10%" />
                      </FlexItem>
                      <FlexItem>
                        <div style={{ height: '90px' }}>
                          <Skeleton
                            screenreaderText="Loading nodes"
                            width="100%"
                            height="85%"
                          />
                        </div>
                      </FlexItem>
                    </Flex>
                  </DataListCell>
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
                  dataListCells={[
                    <DataListCell key={`node-${clusterNode.NodeId}-primary-content`}>
                      <Flex
                        direction={{ default: 'column', '2xl': 'row' }}
                        spaceItems={{
                          default: 'spaceItemsNone',
                          '2xl': 'spaceItems2xl'
                        }}
                      >
                        <Flex
                          className="node-list-content"
                          spaceItems={{
                            default: 'spaceItemsNone',
                            sm: 'spaceItemsNone',
                            md: 'spaceItemsNone',
                            lg: 'spaceItemsNone',
                            xl: 'spaceItemsSm'
                          }}
                          direction={{ default: 'column' }}
                        >
                          <FlexItem>{nodeDescriptionList(clusterNode)}</FlexItem>
                          <FlexItem>{nodeResourceAmounts(clusterNode)}</FlexItem>
                        </Flex>
                        {nodeDataListActions(clusterNode)}
                      </Flex>
                    </DataListCell>
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
        <AdjustVirtualGPUsModal
          isOpen={isAdjustVirtualGPUsModalOpen}
          onClose={closeAdjustVirtualGPUsModal}
          onConfirm={doAdjustVirtualGPUs}
          nodes={adjustVirtualGPUsNodes}
        />
      </CardBody>
    </Card>
  );
};
