import React from 'react';
import {
  Badge,
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
  Menu,
  MenuContent,
  MenuItem,
  MenuList,
  MenuToggle,
  Popper,
  Progress,
  SearchInput,
  Stack,
  StackItem,
  Title,
  Toolbar,
  ToolbarContent,
  ToolbarFilter,
  ToolbarGroup,
  ToolbarItem,
  ToolbarToggleGroup,
} from '@patternfly/react-core';

import { DistributedJupyterKernel } from 'src/app/Data/Kernel';
import CubeIcon from '@patternfly/react-icons/dist/esm/icons/cube-icon';
import StopCircle from '@patternfly/react-icons/dist/esm/icons/stop-circle-icon';
import CheckCircleIcon from '@patternfly/react-icons/dist/esm/icons/check-circle-icon';
import ExclamationTriangleIcon from '@patternfly/react-icons/dist/esm/icons/exclamation-triangle-icon';
import FilterIcon from '@patternfly/react-icons/dist/esm/icons/filter-icon';
import { HourglassHalfIcon, RebootingIcon, SkullIcon, SpinnerIcon } from '@patternfly/react-icons';

// Map from kernel status to the associated icon.
const kernelStatusIcons = {
  unknown: <ExclamationTriangleIcon />,
  starting: <SpinnerIcon className="loading-icon-spin-pulse" />,
  idle: <CheckCircleIcon />,
  busy: <HourglassHalfIcon />,
  terminating: <StopCircle />,
  restarting: <RebootingIcon className="loading-icon-spin" />,
  autorestarting: <RebootingIcon className="loading-icon-spin" />,
  dead: <SkullIcon />,
};

// Hard-coded, dummy data.
const kernels: DistributedJupyterKernel[] = [
  {
    kernelId: '173d8f23-a5af-4998-8221-b510a73c832c',
    numReplicas: 3,
    status: 'idle',
    replicas: [
      {
        kernelId: '173d8f23-a5af-4998-8221-b510a73c832c',
        replicaId: 1,
        podId: '173d8f23-a5af-4998-8221-b510a73c832c-9042e',
        nodeId: 'node-1',
      },
      {
        kernelId: '173d8f23-a5af-4998-8221-b510a73c832c',
        replicaId: 2,
        podId: '173d8f23-a5af-4998-8221-b510a73c832c-b5f23',
        nodeId: 'node-2',
      },
      {
        kernelId: '173d8f23-a5af-4998-8221-b510a73c832c',
        replicaId: 3,
        podId: '173d8f23-a5af-4998-8221-b510a73c832c-7316b',
        nodeId: 'node-3',
      },
    ],
  },
  {
    kernelId: '62677bbf-359a-4f0b-96e7-6baf7ac65545',
    numReplicas: 3,
    status: 'terminating',
    replicas: [
      {
        kernelId: '62677bbf-359a-4f0b-96e7-6baf7ac65545',
        replicaId: 1,
        podId: '62677bbf-359a-4f0b-96e7-6baf7ac65545-7ad16',
        nodeId: 'node-1',
      },
      {
        kernelId: '62677bbf-359a-4f0b-96e7-6baf7ac65545',
        replicaId: 2,
        podId: '62677bbf-359a-4f0b-96e7-6baf7ac65545-9a75a',
        nodeId: 'node-2',
      },
      {
        kernelId: '62677bbf-359a-4f0b-96e7-6baf7ac65545',
        replicaId: 3,
        podId: '62677bbf-359a-4f0b-96e7-6baf7ac65545-04a02',
        nodeId: 'node-3',
      },
    ],
  },
  {
    kernelId: '51f66655-168b-4d77-a1e0-f8f2c8044d14',
    numReplicas: 3,
    status: 'restarting',
    replicas: [
      {
        kernelId: '51f66655-168b-4d77-a1e0-f8f2c8044d14',
        replicaId: 1,
        podId: '51f66655-168b-4d77-a1e0-f8f2c8044d14-jtqwg',
        nodeId: 'node-1',
      },
      {
        kernelId: '51f66655-168b-4d77-a1e0-f8f2c8044d14',
        replicaId: 2,
        podId: '51f66655-168b-4d77-a1e0-f8f2c8044d14-jth2a',
        nodeId: 'node-2',
      },
      {
        kernelId: '51f66655-168b-4d77-a1e0-f8f2c8044d14',
        replicaId: 3,
        podId: '51f66655-168b-4d77-a1e0-f8f2c8044d14-g31g4',
        nodeId: 'node-3',
      },
    ],
  },
  {
    kernelId: 'f692f0d4-852c-4b5f-9d21-e087f5a774e9',
    numReplicas: 3,
    status: 'starting',
    replicas: [
      {
        kernelId: 'f692f0d4-852c-4b5f-9d21-e087f5a774e9',
        replicaId: 1,
        podId: 'f692f0d4-852c-4b5f-9d21-e087f5a774e9-fq3qg',
        nodeId: 'node-1',
      },
      {
        kernelId: 'f692f0d4-852c-4b5f-9d21-e087f5a774e9',
        replicaId: 2,
        podId: 'f692f0d4-852c-4b5f-9d21-e087f5a774e9-geqgf',
        nodeId: 'node-2',
      },
      {
        kernelId: 'f692f0d4-852c-4b5f-9d21-e087f5a774e9',
        replicaId: 3,
        podId: 'f692f0d4-852c-4b5f-9d21-e087f5a774e9-1gasg',
        nodeId: 'node-3',
      },
    ],
  },
];

export const KernelList: React.FunctionComponent = () => {
  const [isDrawerExpanded, setIsDrawerExpanded] = React.useState(false);
  const [drawerPanelBodyContent, setDrawerPanelBodyContent] = React.useState('');
  const [selectedDataListItemId, setSelectedDataListItemId] = React.useState('');
  const [searchValue, setSearchValue] = React.useState('');
  const [statusSelections, setStatusSelections] = React.useState<string[]>([]);

  const onSelectDataListItem = (
    _event: React.MouseEvent<Element, MouseEvent> | React.KeyboardEvent<Element>,
    id: string,
  ) => {
    setSelectedDataListItemId(id);
    setIsDrawerExpanded(true);
    setDrawerPanelBodyContent(id.charAt(id.length - 1));
  };

  const onCloseDrawerClick = () => {
    setIsDrawerExpanded(false);
    setSelectedDataListItemId('');
  };

  const onSearchChange = (value: string) => {
    setSearchValue(value);
  };

  // Set up status single select
  const [isStatusMenuOpen, setIsStatusMenuOpen] = React.useState<boolean>(false);
  const statusToggleRef = React.useRef<HTMLButtonElement>(null);
  const statusMenuRef = React.useRef<HTMLDivElement>(null);
  const statusContainerRef = React.useRef<HTMLDivElement>(null);

  const onFilter = (repo: DistributedJupyterKernel) => {
    // Search name with search value
    let searchValueInput: RegExp;
    try {
      searchValueInput = new RegExp(searchValue, 'i');
    } catch (err) {
      searchValueInput = new RegExp(searchValue.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'i');
    }
    const matchesSearchValue = repo.kernelId.search(searchValueInput) >= 0;

    // Search status with status selection
    let matchesStatusValue = false;
    statusSelections.forEach(function (selectedStatus) {
      const match = repo.status.toLowerCase() === selectedStatus.toLowerCase();
      matchesStatusValue = matchesStatusValue || match;
    });

    return (searchValue === '' || matchesSearchValue) && (statusSelections.length === 0 || matchesStatusValue);
  };
  const filteredKernels = kernels.filter(onFilter);

  // Set up name search input
  const searchInput = (
    <SearchInput
      placeholder="Filter by kernel name"
      value={searchValue}
      onChange={(_event, value) => onSearchChange(value)}
      onClear={() => onSearchChange('')}
    />
  );

  React.useEffect(() => {
    const handleStatusClickOutside = (event: MouseEvent) => {
      if (isStatusMenuOpen && !statusMenuRef.current?.contains(event.target as Node)) {
        setIsStatusMenuOpen(false);
      }
    };

    const handleStatusMenuKeys = (event: KeyboardEvent) => {
      if (isStatusMenuOpen && statusMenuRef.current?.contains(event.target as Node)) {
        if (event.key === 'Escape' || event.key === 'Tab') {
          setIsStatusMenuOpen(!isStatusMenuOpen);
          statusToggleRef.current?.focus();
        }
      }
    };

    window.addEventListener('keydown', handleStatusMenuKeys);
    window.addEventListener('click', handleStatusClickOutside);
    return () => {
      window.removeEventListener('keydown', handleStatusMenuKeys);
      window.removeEventListener('click', handleStatusClickOutside);
    };
  }, [isStatusMenuOpen, statusMenuRef]);

  function onStatusMenuSelect(event: React.MouseEvent | undefined, itemId: string | number | undefined) {
    if (typeof itemId === 'undefined') {
      return;
    }

    const itemStr = itemId.toString();

    setStatusSelections(
      statusSelections.includes(itemStr)
        ? statusSelections.filter((selection) => selection !== itemStr)
        : [itemStr, ...statusSelections],
    );
  }

  const statusMenu = (
    <Menu ref={statusMenuRef} id="mixed-group-status-menu" onSelect={onStatusMenuSelect} selected={statusSelections}>
      <MenuContent>
        <MenuList>
          <MenuItem hasCheckbox isSelected={statusSelections.includes('unknown')} itemId="unknown">
            Unknown
          </MenuItem>
          <MenuItem hasCheckbox isSelected={statusSelections.includes('starting')} itemId="starting">
            Starting
          </MenuItem>
          <MenuItem hasCheckbox isSelected={statusSelections.includes('idle')} itemId="idle">
            Idle
          </MenuItem>
          <MenuItem hasCheckbox isSelected={statusSelections.includes('busy')} itemId="busy">
            Busy
          </MenuItem>
          <MenuItem hasCheckbox isSelected={statusSelections.includes('terminating')} itemId="terminating">
            Terminating
          </MenuItem>
          <MenuItem hasCheckbox isSelected={statusSelections.includes('restarting')} itemId="restarting">
            Restarting
          </MenuItem>
          <MenuItem hasCheckbox isSelected={statusSelections.includes('autorestarting')} itemId="autorestarting">
            Autorestarting
          </MenuItem>
          <MenuItem hasCheckbox isSelected={statusSelections.includes('dead')} itemId="dead">
            Dead
          </MenuItem>
        </MenuList>
      </MenuContent>
    </Menu>
  );

  const onStatusToggleClick = (ev: React.MouseEvent) => {
    ev.stopPropagation(); // Stop handleClickOutside from handling
    setTimeout(() => {
      if (statusMenuRef.current) {
        const firstElement = statusMenuRef.current.querySelector('li > button:not(:disabled)');
        firstElement && (firstElement as HTMLElement).focus();
      }
    }, 0);
    setIsStatusMenuOpen(!isStatusMenuOpen);
  };

  const statusToggle = (
    <MenuToggle
      ref={statusToggleRef}
      onClick={onStatusToggleClick}
      isExpanded={isStatusMenuOpen}
      {...(statusSelections.length > 0 && { badge: <Badge isRead>{statusSelections.length}</Badge> })}
      icon={<FilterIcon />}
      style={
        {
          width: '200px',
        } as React.CSSProperties
      }
    >
      Kernel Status
    </MenuToggle>
  );

  const statusSelect = (
    <div ref={statusContainerRef}>
      <Popper
        trigger={statusToggle}
        triggerRef={statusToggleRef}
        popper={statusMenu}
        popperRef={statusMenuRef}
        appendTo={statusContainerRef.current || undefined}
        isVisible={isStatusMenuOpen}
      />
    </div>
  );

  const toggleGroupItems = (
    <Flex alignItems={{ default: 'alignItemsCenter' }}>
      <ToolbarItem>
        <InputGroup>
          <InputGroupItem isFill>{searchInput}</InputGroupItem>
        </InputGroup>
      </ToolbarItem>
      <ToolbarGroup variant="filter-group">
        <ToolbarFilter
          chips={statusSelections}
          deleteChip={() => setStatusSelections([])}
          deleteChipGroup={() => setStatusSelections([])}
          categoryName="Status"
        >
          {statusSelect}
        </ToolbarFilter>
      </ToolbarGroup>
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
      <Toolbar
        id="content-padding-data-toolbar"
        usePageInsets
        clearAllFilters={() => {
          setStatusSelections([]);
        }}
      >
        <ToolbarContent>{ToolbarItems}</ToolbarContent>
      </Toolbar>
      <DataList
        aria-label="data list"
        selectedDataListItemId={selectedDataListItemId}
        onSelectDataListItem={onSelectDataListItem}
      >
        {filteredKernels.map((kernel) => (
          <DataListItem key={kernel.kernelId} id="content-padding-item1">
            <DataListItemRow>
              <DataListItemCells
                dataListCells={[
                  <DataListCell key="primary-content">
                    <Flex spaceItems={{ default: 'spaceItemsMd' }} direction={{ default: 'column' }}>
                      <FlexItem>
                        <p>Kernel {kernel.kernelId}</p>
                      </FlexItem>
                      <Flex spaceItems={{ default: 'spaceItemsMd' }}>
                        <FlexItem>
                          <CubeIcon /> {kernel.numReplicas}
                        </FlexItem>
                        <FlexItem>
                          {kernelStatusIcons[kernel.status]} {kernel.status}
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
      <CardTitle>
        <Title headingLevel="h2" size="xl">
          Active Kernels
        </Title>
      </CardTitle>
      <CardBody>
        <Drawer isExpanded={isDrawerExpanded}>
          <DrawerHead hasNoPadding></DrawerHead>
          <DrawerContent panelContent={panelContent} colorVariant="no-background">
            <DrawerContentBody>{drawerContent}</DrawerContentBody>
          </DrawerContent>
        </Drawer>
      </CardBody>
    </Card>
  );
};
