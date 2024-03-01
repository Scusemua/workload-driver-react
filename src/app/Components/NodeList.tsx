import React, { useEffect } from 'react';
import {
  Button,
  ButtonVariant,
  Card,
  CardBody,
  CardHeader,
  CardTitle,
  DataList,
  DataListAction,
  DataListCell,
  DataListItem,
  DataListItemCells,
  DataListItemRow,
  Drawer,
  DrawerActions,
  DrawerCloseButton,
  DrawerContent,
  DrawerContentBody,
  DrawerHead,
  DrawerPanelBody,
  DrawerPanelContent,
  Flex,
  FlexItem,
  InputGroup,
  InputGroupItem,
  Progress,
  SearchInput,
  Stack,
  StackItem,
  Title,
  Toolbar,
  ToolbarContent,
  ToolbarItem,
  ToolbarToggleGroup,
} from '@patternfly/react-core';

import GpuIcon from '@app/Icons/GpuIcon';
import { KubernetesNode } from '@data/Kubernetes';
import { CpuIcon, CubeIcon, FilterIcon, MemoryIcon } from '@patternfly/react-icons';

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
  const [isDrawerExpanded, setIsDrawerExpanded] = React.useState(false);
  const [drawerPanelBodyContent, setDrawerPanelBodyContent] = React.useState('');
  const [selectedDataListItemId, setSelectedDataListItemId] = React.useState('');
  const [searchValue, setSearchValue] = React.useState('');

  // Clicking one of the nodes to open its associated drawer.
  const onSelectDataListItem = (
    _event: React.MouseEvent<Element, MouseEvent> | React.KeyboardEvent<Element>,
    id: string,
  ) => {
    setSelectedDataListItemId(id);
    setIsDrawerExpanded(true);
    setDrawerPanelBodyContent(id.charAt(id.length - 1));
  };

  // Handle closing the drawer.
  const onCloseDrawerClick = () => {
    setIsDrawerExpanded(false);
    setSelectedDataListItemId('');
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

  // This is the drawer that is opened when clicking a node.
  // Presently it's just a placeholder.
  const panelContent = (
    <DrawerPanelContent>
      <DrawerHead>
        <Title headingLevel="h2" size="xl">
          node-{drawerPanelBodyContent}
        </Title>
        <DrawerActions>
          <DrawerCloseButton onClick={onCloseDrawerClick} />
        </DrawerActions>
      </DrawerHead>
      <DrawerPanelBody>
        <Flex spaceItems={{ default: 'spaceItemsLg' }} direction={{ default: 'column' }}>
          <FlexItem>
            <p>
              The content of the drawer really is up to you. It could have form fields, definition lists, text lists,
              labels, charts, progress bars, etc. Spacing recommendation is 24px margins. You can put tabs in here, and
              can also make the drawer scrollable.
            </p>
          </FlexItem>
          <FlexItem>
            <Progress value={parseInt(drawerPanelBodyContent) * 10} title="Title" />
          </FlexItem>
          <FlexItem>
            <Progress value={parseInt(drawerPanelBodyContent) * 5} title="Title" />
          </FlexItem>
        </Flex>
      </DrawerPanelBody>
    </DrawerPanelContent>
  );

  // Fetch the kubernetes nodes from the backend (which itself makes a network call to the Kubernetes API).
  const [nodes, setNodes] = React.useState<KubernetesNode[]>([]);
  useEffect(() => {
    let ignoreResponse = false;
    async function fetchKubernetesNodes() {
      try {
        console.log('Refreshing Kubernetes nodes.');

        // Make a network request to the backend. The server infrastructure handles proxying/routing the request to the correct host.
        // We're specifically targeting the API endpoint I setup called "nodes".
        const response = await fetch('/api/node');

        // Get the response, which will be in JSON format, and decode it into an array of KubernetesNode (which is a TypeScript interface that I defined).
        const respNodes: KubernetesNode[] = await response.json();

        if (!ignoreResponse) {
          console.log('Received nodes: ' + JSON.stringify(respNodes));
          setNodes(respNodes);
        }
      } catch (e) {
        console.error(e);
      }
    }

    fetchKubernetesNodes();

    // Periodically refresh the Kubernetes nodes every 120,000ms, or when the user clicks the "refresh" button.
    setInterval(fetchKubernetesNodes, 120000);

    return () => {
      ignoreResponse = true;
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

  const drawerContent = (
    <React.Fragment>
      <Toolbar id="content-padding-data-toolbar" usePageInsets>
        <ToolbarContent>
          <ToolbarToggleGroup toggleIcon={<FilterIcon />} breakpoint="xl">
            <Flex alignItems={{ default: 'alignItemsCenter' }}>
              <ToolbarItem>
                <InputGroup>
                  <InputGroupItem isFill>{searchInput}</InputGroupItem>
                </InputGroup>
              </ToolbarItem>
            </Flex>
          </ToolbarToggleGroup>
        </ToolbarContent>
      </Toolbar>
      <DataList
        aria-label="data list"
        selectedDataListItemId={selectedDataListItemId}
        onSelectDataListItem={onSelectDataListItem}
      >
        {filteredNodes.map((kubeNode) => (
          <DataListItem key={kubeNode.NodeId} id="content-padding-item1">
            <DataListItemRow>
              <DataListItemCells
                dataListCells={[
                  <DataListCell key="primary-content">
                    <Flex spaceItems={{ default: 'spaceItemsMd' }} direction={{ default: 'column' }}>
                      <FlexItem>
                        <p>Node {kubeNode.NodeId}</p>
                      </FlexItem>
                      <Flex spaceItems={{ default: 'spaceItemsMd' }}>
                        <FlexItem>
                          <CubeIcon /> {kubeNode.Pods.length}
                        </FlexItem>
                      </Flex>
                      <Flex spaceItems={{ default: 'spaceItemsMd' }}>
                        <FlexItem>
                          <CpuIcon /> {kubeNode.AllocatedCPU.toFixed(4)} / {kubeNode.CapacityCPU}
                        </FlexItem>
                        <FlexItem>
                          <MemoryIcon /> {kubeNode.AllocatedMemory.toFixed(4)} / {kubeNode.CapacityMemory.toFixed(0)}
                        </FlexItem>
                        <FlexItem>
                          <GpuIcon /> {kubeNode.AllocatedCPU.toFixed(4)} / {kubeNode.CapacityCPU}
                        </FlexItem>
                      </Flex>
                    </Flex>
                  </DataListCell>,
                  <DataListAction
                    key="actions"
                    aria-labelledby="content-padding-item1 content-padding-action1"
                    id="content-padding-action1"
                    aria-label="Actions"
                  >
                    <Stack>
                      <StackItem>
                        <Button variant={ButtonVariant.secondary}>Secondary</Button>
                      </StackItem>
                      <StackItem>
                        <Button variant={ButtonVariant.link}>Link Button</Button>
                      </StackItem>
                    </Stack>
                  </DataListAction>,
                ]}
              />
            </DataListItemRow>
          </DataListItem>
        ))}
      </DataList>
    </React.Fragment>
  );

  return (
    <Card isCompact isRounded>
      <CardHeader>
        <CardTitle>
          <Title headingLevel="h2" size="xl">
            Kubernetes Nodes
          </Title>
        </CardTitle>
      </CardHeader>
      <CardBody>
        <Drawer isExpanded={isDrawerExpanded}>
          <DrawerContent panelContent={panelContent} colorVariant="no-background">
            <DrawerContentBody>{drawerContent}</DrawerContentBody>
          </DrawerContent>
        </Drawer>
      </CardBody>
    </Card>
  );
};
