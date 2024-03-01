import React, { useEffect, useRef } from 'react';
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
  SearchInput,
  Title,
  // Toolbar,
  // ToolbarContent,
  ToolbarGroup,
  ToolbarItem,
  ToolbarToggleGroup,
  Tooltip,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import GpuIcon from '@app/Icons/GpuIcon';
import { KubernetesNode } from '@data/Kubernetes';
import { CpuIcon, CubeIcon, FilterIcon, MemoryIcon, SyncIcon } from '@patternfly/react-icons';

// Hard-coded, dummy data.
// const kubeNodes: KubernetesNode[] = [
//     {
//         NodeId: 'distributed-notebook-worker',
//         Pods: [
//             {
//                 PodName: '62677bbf-359a-4f0b-96e7-6baf7ac65545-7ad16',
//                 PodPhase: 'running',
//                 PodAge: '127h2m45s',
//                 PodIP: '10.0.0.1',
//             },
//         ],
//         Age: '147h4m53s',
//         IP: '172.20.0.3',
//         CapacityCPU: 64,
//         CapacityMemory: 64000,
//         CapacityGPUs: 8,
//         CapacityVGPUs: 72,
//         AllocatedCPU: 0.24,
//         AllocatedMemory: 1557.1,
//         AllocatedGPUs: 2,
//         AllocatedVGPUs: 4,
//     },
// ];

export const KubernetesNodeList: React.FunctionComponent = () => {
  const [searchValue, setSearchValue] = React.useState('');
  const [isCardExpanded, setIsCardExpanded] = React.useState(true);
  const [expandedNodes, setExpandedNodes] = React.useState<string[]>([]);

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

  const ignoreResponse = useRef(false);
  async function fetchKubernetesNodes() {
    try {
      console.log('Refreshing Kubernetes nodes.');

      // Make a network request to the backend. The server infrastructure handles proxying/routing the request to the correct host.
      // We're specifically targeting the API endpoint I setup called "nodes".
      const response = await fetch('/api/nodes');

      // Get the response, which will be in JSON format, and decode it into an array of KubernetesNode (which is a TypeScript interface that I defined).
      const respNodes: KubernetesNode[] = await response.json();

      if (!ignoreResponse.current) {
        // console.log('Received nodes: ' + JSON.stringify(respNodes));
        setNodes(respNodes);
      }
    } catch (e) {
      console.error(e);
    }
  }

  // Fetch the kubernetes nodes from the backend (which itself makes a network call to the Kubernetes API).
  const [nodes, setNodes] = React.useState<KubernetesNode[]>([]);
  useEffect(() => {
    ignoreResponse.current = false;
    fetchKubernetesNodes();

    // Periodically refresh the Kubernetes nodes every 120,000ms, or when the user clicks the "refresh" button.
    setInterval(fetchKubernetesNodes, 120000);

    return () => {
      ignoreResponse.current = true;
    };
  }, []);

  // Handler for when the user filters by node name.
  const onFilter = (repo: KubernetesNode) => {
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
  const filteredNodes = nodes.filter(onFilter);

  const toolbar = (
    <React.Fragment>
      <ToolbarToggleGroup toggleIcon={<FilterIcon />} breakpoint="xl">
        <Flex alignItems={{ default: 'alignItemsCenter' }}>
          <ToolbarItem>
            <InputGroup>
              <InputGroupItem isFill>{searchInput}</InputGroupItem>
            </InputGroup>
          </ToolbarItem>
        </Flex>
      </ToolbarToggleGroup>
      <ToolbarGroup variant="icon-button-group">
        <ToolbarItem>
          <Tooltip exitDelay={75} content={<div>Refresh nodes.</div>}>
            <Button variant="plain" onClick={fetchKubernetesNodes}>
              <SyncIcon />
            </Button>
          </Tooltip>
        </ToolbarItem>
      </ToolbarGroup>
    </React.Fragment>
  );

  const expandedNodeContent = (kubeNode: KubernetesNode) => (
    <Table aria-label="Pods Table" variant={'compact'} borders={true}>
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
            <Td dataLabel="Pod ID">{pod.PodName}</Td>
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

  return (
    <Card isCompact isRounded isExpanded={isCardExpanded}>
      <CardHeader
        onExpand={onCardExpand}
        actions={{ actions: toolbar, hasNoOffset: true }}
        toggleButtonProps={{
          id: 'toggle-button',
          'aria-label': 'Actions',
          'aria-labelledby': 'titleId toggle-button',
          'aria-expanded': isCardExpanded,
        }}
      >
        <CardTitle>
          <Title headingLevel="h4" size="xl">
            Kubernetes Nodes
          </Title>
        </CardTitle>
      </CardHeader>
      <CardExpandableContent>
        <CardBody>
          <DataList isCompact aria-label="data list">
            {filteredNodes.map((kubeNode: KubernetesNode, idx: number) => (
              <DataListItem
                key={kubeNode.NodeId}
                id={'node-list-item-' + idx}
                isExpanded={expandedNodes.includes(kubeNode.NodeId)}
              >
                <DataListItemRow>
                  <DataListToggle
                    onClick={() => toggleExpandedNode(kubeNode.NodeId)}
                    isExpanded={expandedNodes.includes(kubeNode.NodeId)}
                    id="ex-toggle1"
                    aria-controls="ex-expand1"
                  />
                  <DataListItemCells
                    dataListCells={[
                      <DataListCell key="primary-content">
                        <Flex spaceItems={{ default: 'spaceItemsMd' }} direction={{ default: 'column' }}>
                          <FlexItem>
                            <DescriptionList isCompact columnModifier={{ lg: '3Col' }}>
                              <DescriptionListGroup>
                                <DescriptionListTerm>Node ID</DescriptionListTerm>
                                <DescriptionListDescription>{kubeNode.NodeId}</DescriptionListDescription>
                              </DescriptionListGroup>
                              <DescriptionListGroup>
                                <DescriptionListTerm>IP</DescriptionListTerm>
                                <DescriptionListDescription>{kubeNode.IP}</DescriptionListDescription>
                              </DescriptionListGroup>
                              <DescriptionListGroup>
                                <DescriptionListTerm>Age</DescriptionListTerm>
                                <DescriptionListDescription>{kubeNode.Age}</DescriptionListDescription>
                              </DescriptionListGroup>
                            </DescriptionList>
                          </FlexItem>
                          <FlexItem>
                            <Flex spaceItems={{ default: 'spaceItems4xl' }}>
                              <FlexItem>
                                <CubeIcon /> {kubeNode.Pods.length}
                              </FlexItem>
                              <FlexItem>
                                <CpuIcon /> {kubeNode.AllocatedCPU.toFixed(4)} / {kubeNode.CapacityCPU}
                              </FlexItem>
                              <FlexItem>
                                <MemoryIcon /> {kubeNode.AllocatedMemory.toFixed(4)} /{' '}
                                {kubeNode.CapacityMemory.toFixed(0)}
                              </FlexItem>
                              <FlexItem>
                                <GpuIcon /> {kubeNode.AllocatedCPU.toFixed(4)} / {kubeNode.CapacityCPU}
                              </FlexItem>
                            </Flex>
                          </FlexItem>
                        </Flex>
                      </DataListCell>,
                    ]}
                  />
                </DataListItemRow>
                <DataListContent
                  aria-label={'node-' + kubeNode.NodeId + '-expandable-content'}
                  id={'node-' + kubeNode.NodeId + '-expandable-content'}
                  isHidden={!expandedNodes.includes(kubeNode.NodeId)}
                >
                  {expandedNodeContent(kubeNode)}
                </DataListContent>
              </DataListItem>
            ))}
          </DataList>
        </CardBody>
      </CardExpandableContent>
    </Card>
  );
};
