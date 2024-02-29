import React from 'react';
import {
    Button,
    ButtonVariant,
    Card,
    CardBody,
    CardTitle,
    DataList,
    DataListAction,
    DataListCell,
    DataListItem,
    DataListItemCells,
    DataListItemRow,
    Toolbar,
    ToolbarItem,
    ToolbarContent,
    ToolbarToggleGroup,
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
    Stack,
    StackItem,
    SearchInput,
    Title,
    SelectOptionProps,
    CardHeader
} from '@patternfly/react-core';

import GpuIcon from 'src/app/Icons/GpuIcon';
import { KubernetesNode } from 'src/app/Data/Kubernetes';
import CubeIcon from '@patternfly/react-icons/dist/esm/icons/cube-icon';
import FilterIcon from '@patternfly/react-icons/dist/esm/icons/filter-icon';
import { CpuIcon, MemoryIcon } from '@patternfly/react-icons';

interface SelectOptionType extends Omit<SelectOptionProps, 'children'> {
    label: string;
}

// Hard-coded, dummy data.
const kubeNodes: KubernetesNode[] = [
    {
        NodeId: "distributed-notebook-worker",
        Pods: [
            {
                PodName: "62677bbf-359a-4f0b-96e7-6baf7ac65545-7ad16",
                PodPhase: "running",
                PodAge: "127h2m45s",
                PodIP: "10.0.0.1",
            },
        ],
        Age: "147h4m53s",
        IP: "172.20.0.3",
        CapacityCPU: 64,
        CapacityMemory: 64000,
        CapacityGPUs: 8,
        CapacityVGPUs: 72,
        AllocatedCPU: 0.24,
        AllocatedMemory: 1557.10,
        AllocatedGPUs: 2,
        AllocatedVGPUs: 4,
    },
];

export const KubernetesNodeList: React.FunctionComponent = () => {
    const [isDrawerExpanded, setIsDrawerExpanded] = React.useState(false);
    const [drawerPanelBodyContent, setDrawerPanelBodyContent] = React.useState('');
    const [statusIsOpen, setStatusIsOpen] = React.useState(false);
    const [statusSelected, setStatusSelected] = React.useState<string | number | undefined>('Status');
    const [selectedDataListItemId, setSelectedDataListItemId] = React.useState('');
    const [searchValue, setSearchValue] = React.useState('');
    const [statusSelection, setStatusSelection] = React.useState('');

    const onStatusSelect = (_event: React.MouseEvent<Element> | undefined, value: string | number | undefined) => {
        setStatusSelected(value);
        setStatusIsOpen(false);
    };

    const onSelectDataListItem = (
        _event: React.MouseEvent<Element, MouseEvent> | React.KeyboardEvent<Element>,
        id: string
    ) => {
        setSelectedDataListItemId(id);
        setIsDrawerExpanded(true);
        setDrawerPanelBodyContent(id.charAt(id.length - 1));
    };

    const onCloseDrawerClick = (_event: React.MouseEvent<HTMLDivElement>) => {
        setIsDrawerExpanded(false);
        setSelectedDataListItemId('');
    };

    const onSearchChange = (value: string) => {
        setSearchValue(value);
    };

    const onFilter = (repo: KubernetesNode) => {
        // Search name with search value
        let searchValueInput: RegExp;
        try {
            searchValueInput = new RegExp(searchValue, 'i');
        } catch (err) {
            searchValueInput = new RegExp(searchValue.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'i');
        }
        const matchesSearchValue = repo.NodeId.search(searchValueInput) >= 0;

        return (
            (searchValue === '' || matchesSearchValue)
        );
    };
    const filteredRepos = kubeNodes.filter(onFilter);

    // Set up name search input
    const searchInput = (
        <SearchInput
            placeholder="Filter by kubeNode name"
            value={searchValue}
            onChange={(_event, value) => onSearchChange(value)}
            onClear={() => onSearchChange('')}
        />
    );

    const toggleGroupItems = (
        <Flex alignItems={{ default: 'alignItemsCenter' }}>
            <ToolbarItem>
                <InputGroup>
                    <InputGroupItem isFill>
                        {searchInput}
                    </InputGroupItem>
                </InputGroup>
            </ToolbarItem>
        </Flex>
    );

    const ToolbarItems = (
        <ToolbarToggleGroup toggleIcon={<FilterIcon />} breakpoint="xl">
            {toggleGroupItems}
        </ToolbarToggleGroup>
    );

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

    const drawerContent = (
        <React.Fragment>
            <Toolbar id="content-padding-data-toolbar" usePageInsets>
                <ToolbarContent>{ToolbarItems}</ToolbarContent>
            </Toolbar>
            <DataList
                aria-label="data list"
                selectedDataListItemId={selectedDataListItemId}
                onSelectDataListItem={onSelectDataListItem}
            >
                {
                    kubeNodes.map(kubeNode => (
                        <DataListItem id="content-padding-item1">
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
                                                        <CpuIcon /> {kubeNode.AllocatedCPU} / {kubeNode.CapacityCPU}
                                                    </FlexItem>
                                                    <FlexItem>
                                                        <MemoryIcon /> {kubeNode.AllocatedMemory} / {kubeNode.CapacityMemory}
                                                    </FlexItem>
                                                    <FlexItem>
                                                        <GpuIcon /> {kubeNode.AllocatedCPU} / {kubeNode.CapacityCPU}
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
                                        </DataListAction>
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
                    <Title headingLevel='h2' size='xl'>
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
