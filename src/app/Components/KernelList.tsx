import React, { useEffect, useRef } from 'react';
import {
  Badge,
  Button,
  Card,
  CardBody,
  CardExpandableContent,
  CardHeader,
  CardTitle,
  DataList,
  DataListAction,
  DataListCell,
  DataListContent,
  DataListItem,
  DataListItemCells,
  DataListItemRow,
  DataListToggle,
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
  Tooltip,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import { DistributedJupyterKernel } from '@data/Kernel';
import {
  CheckCircleIcon,
  CodeIcon,
  CubeIcon,
  ExclamationTriangleIcon,
  FilterIcon,
  HourglassHalfIcon,
  MigrationIcon,
  PlusIcon,
  RebootingIcon,
  SkullIcon,
  SpinnerIcon,
  StopCircleIcon,
  SyncIcon,
  TrashIcon,
} from '@patternfly/react-icons';

// Map from kernel status to the associated icon.
const kernelStatusIcons = {
  unknown: <ExclamationTriangleIcon />,
  starting: <SpinnerIcon className="loading-icon-spin-pulse" />,
  idle: <CheckCircleIcon />,
  busy: <HourglassHalfIcon />,
  terminating: <StopCircleIcon />,
  restarting: <RebootingIcon className="loading-icon-spin" />,
  autorestarting: <RebootingIcon className="loading-icon-spin" />,
  dead: <SkullIcon />,
};

export const KernelList: React.FunctionComponent = () => {
  const [searchValue, setSearchValue] = React.useState('');
  const [statusSelections, setStatusSelections] = React.useState<string[]>([]);
  const [isCardExpanded, setIsCardExpanded] = React.useState(true);
  const [expandedKernels, setExpandedKernels] = React.useState<string[]>([]);

  const onCardExpand = () => {
    setIsCardExpanded(!isCardExpanded);
  };

  const onSearchChange = (value: string) => {
    setSearchValue(value);
  };

  // Set up status single select
  const [isStatusMenuOpen, setIsStatusMenuOpen] = React.useState<boolean>(false);
  const statusToggleRef = React.useRef<HTMLButtonElement>(null);
  const statusMenuRef = React.useRef<HTMLDivElement>(null);
  const statusContainerRef = React.useRef<HTMLDivElement>(null);

  // Set up name search input
  const searchInput = (
    <SearchInput
      placeholder="Filter by kernel name"
      value={searchValue}
      onChange={(_event, value) => onSearchChange(value)}
      onClear={() => onSearchChange('')}
    />
  );

  useEffect(() => {
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

  const ignoreResponse = useRef(false);
  async function fetchKernels() {
    try {
      console.log('Refreshing kernels.');

      // Make a network request to the backend. The server infrastructure handles proxying/routing the request to the correct host.
      // We're specifically targeting the API endpoint I setup called "get-kernels".
      const response = await fetch('/api/get-kernels');

      const respKernels: DistributedJupyterKernel[] = await response.json();

      if (!ignoreResponse.current) {
        console.log('Received kernels: ' + JSON.stringify(respKernels));
        setKernels(respKernels);
      }
    } catch (e) {
      console.error(e);
    }
  }

  const [kernels, setKernels] = React.useState<DistributedJupyterKernel[]>([]);
  useEffect(() => {
    ignoreResponse.current = false;

    fetchKernels();

    // Periodically refresh the automatically kernels every 30 seconds.
    setInterval(fetchKernels, 30000);

    return () => {
      ignoreResponse.current = true;
    };
  }, []);

  function onStatusMenuSelect(_event: React.MouseEvent | undefined, itemId: string | number | undefined) {
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

  const ToolbarItems = (
    <ToolbarToggleGroup toggleIcon={<FilterIcon />} breakpoint="xl">
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
    </ToolbarToggleGroup>
  );

  const cardHeaderActions = (
    <React.Fragment>
      <ToolbarGroup variant="icon-button-group">
        <ToolbarItem>
          <Tooltip exitDelay={75} content={<div>Create a new kernel.</div>}>
            <Button id="create-kernel-button" variant="plain" onClick={fetchKernels}>
              <PlusIcon />
            </Button>
          </Tooltip>
          <Tooltip exitDelay={75} content={<div>Delete selected kernels.</div>}>
            <Button id="delete-kernels-button" variant="plain" onClick={fetchKernels}>
              <TrashIcon />
            </Button>
          </Tooltip>
          <Tooltip exitDelay={75} content={<div>Refresh kernels.</div>}>
            <Button id="refresh-kernels-button" variant="plain" onClick={fetchKernels}>
              <SyncIcon />
            </Button>
          </Tooltip>
        </ToolbarItem>
      </ToolbarGroup>
    </React.Fragment>
  );

  const expandedKernelContent = (kernel: DistributedJupyterKernel) => (
    <Table aria-label="Pods Table" variant={'compact'} borders={true}>
      <Thead>
        <Tr>
          <Th>ID</Th>
          <Th>Pod</Th>
          <Th>Node</Th>
          <Th />
          <Th />
        </Tr>
      </Thead>
      <Tbody>
        {kernel.replicas.map((replica) => (
          <Tr key={replica.replicaId}>
            <Td dataLabel="ID">{replica.replicaId}</Td>
            <Td dataLabel="Pod">{replica.podId}</Td>
            <Td dataLabel="Node">{replica.nodeId}</Td>
            <Td>
              <Tooltip exitDelay={20} entryDelay={175} content={<div>Execute Python code on this replica.</div>}>
                <Button variant={'link'} icon={<CodeIcon />}>
                  Execute
                </Button>
              </Tooltip>
            </Td>
            <Td>
              <Tooltip exitDelay={20} entryDelay={175} content={<div>Migrate this replica to another node.</div>}>
                <Button variant={'link'} icon={<MigrationIcon />}>
                  Migrate
                </Button>
              </Tooltip>
            </Td>
          </Tr>
        ))}
      </Tbody>
    </Table>
  );

  const toggleExpandedKernel = (id) => {
    const index = expandedKernels.indexOf(id);
    const newExpanded =
      index >= 0
        ? [...expandedKernels.slice(0, index), ...expandedKernels.slice(index + 1, expandedKernels.length)]
        : [...expandedKernels, id];
    setExpandedKernels(newExpanded);
  };

  return (
    <Card isCompact isRounded isExpanded={isCardExpanded}>
      <CardHeader
        onExpand={onCardExpand}
        actions={{ actions: cardHeaderActions, hasNoOffset: true }}
        toggleButtonProps={{
          id: 'toggle-button',
          'aria-label': 'Actions',
          'aria-labelledby': 'titleId toggle-button',
          'aria-expanded': isCardExpanded,
        }}
      >
        <CardTitle>
          <Title headingLevel="h4" size="xl">
            Active Kernels
          </Title>
        </CardTitle>
        <Toolbar
          id="content-padding-data-toolbar"
          usePageInsets
          clearAllFilters={() => {
            setStatusSelections([]);
          }}
        >
          <ToolbarContent>{ToolbarItems}</ToolbarContent>
        </Toolbar>
      </CardHeader>
      <CardExpandableContent>
        <CardBody>
          <DataList isCompact aria-label="data list">
            {filteredKernels.map((kernel, idx) => (
              <DataListItem
                isExpanded={expandedKernels.includes(kernel.kernelId)}
                key={kernel.kernelId}
                id={'content-padding-item-' + idx}
              >
                <DataListItemRow>
                  <DataListToggle
                    onClick={() => toggleExpandedKernel(kernel.kernelId)}
                    isExpanded={expandedKernels.includes(kernel.kernelId)}
                    id="ex-toggle1"
                    aria-controls="ex-expand1"
                  />
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
                        aria-labelledby={'content-padding-item-' + idx + ' content-action-item-' + idx}
                        id={'content-padding-item-' + idx}
                        aria-label="Actions"
                      >
                        <Stack>
                          <StackItem>
                            <Tooltip
                              exitDelay={75}
                              entryDelay={250}
                              content={<div>Execute Python code on this kernel.</div>}
                            >
                              <Button variant={'link'} icon={<CodeIcon />}>
                                Execute
                              </Button>
                            </Tooltip>
                          </StackItem>
                          <StackItem>
                            <Tooltip exitDelay={75} entryDelay={250} content={<div>Terminate this kernel.</div>}>
                              <Button variant={'link'} icon={<TrashIcon />} isDanger>
                                Terminate
                              </Button>
                            </Tooltip>
                          </StackItem>
                        </Stack>
                      </DataListAction>,
                    ]}
                  />
                </DataListItemRow>
                <DataListContent
                  aria-label={'kernel-' + kernel.kernelId + '-expandable-content'}
                  id={'kernel-' + kernel.kernelId + '-expandable-content'}
                  isHidden={!expandedKernels.includes(kernel.kernelId)}
                >
                  {expandedKernelContent(kernel)}
                </DataListContent>
              </DataListItem>
            ))}
          </DataList>
        </CardBody>
      </CardExpandableContent>
    </Card>
  );
};
