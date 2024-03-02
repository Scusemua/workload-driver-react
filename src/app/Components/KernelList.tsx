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
  Modal,
  ModalVariant,
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
import { KernelManager, ServerConnection } from '@jupyterlab/services';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import { ConfirmationModal } from '@app/Components/ConfirmationModal';
import { CodeEditorComponent } from '@app/Components/CodeEditor';
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
import { IKernelConnection } from '@jupyterlab/services/lib/kernel/kernel';

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

interface ExecuteCodeOnKernelProps {
  children?: React.ReactNode;
  kernelId: string;
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (code: string) => void;
}

const ExecuteCodeOnKernelModal: React.FunctionComponent<ExecuteCodeOnKernelProps> = (props) => {
  const [code, setCode] = React.useState('');

  const onSubmit = () => {
    props.onSubmit(code);
  };

  const onChange = (code) => {
    setCode(code);
  };

  return (
    <Modal
      variant={ModalVariant.large}
      title={'Execute Code on Kernel ' + props.kernelId}
      isOpen={props.isOpen}
      onClose={props.onClose}
      actions={[
        <Button key="submit" variant="primary" onClick={onSubmit}>
          Submit
        </Button>,
        <Button key="cancel" variant="link" onClick={props.onClose}>
          Cancel
        </Button>,
      ]}
    >
      Enter the code to be executed below. Once you&apos;re ready, press &apos;Submit&apos; to submit the code to the
      kernel for execution.
      <CodeEditorComponent onChange={onChange} />
    </Modal>
  );
};

export const KernelList: React.FunctionComponent = () => {
  const [searchValue, setSearchValue] = React.useState('');
  const [statusSelections, setStatusSelections] = React.useState<string[]>([]);
  const [isCardExpanded, setIsCardExpanded] = React.useState(true);
  const [expandedKernels, setExpandedKernels] = React.useState<string[]>([]);
  const [isConfirmCreateModalOpen, setIsConfirmCreateModalOpen] = React.useState(false);
  const [isExecuteCodeModalOpen, setIsExecuteCodeModalOpen] = React.useState(false);
  const [executeCodeKernel, setExecuteCodeKernel] = React.useState<DistributedJupyterKernel | null>(null);
  const kernelManager = useRef<KernelManager | null>(null);

  async function initializeKernelManagers() {
    if (kernelManager.current === null) {
      const kernelSpecManagerOptions: KernelManager.IOptions = {
        serverSettings: ServerConnection.makeSettings({
          token: '',
          appendToken: false,
          baseUrl: '/jupyter',
          fetch: fetch,
        }),
      };
      kernelManager.current = new KernelManager(kernelSpecManagerOptions);

      console.log('Waiting for Kernel Manager to be ready.');

      kernelManager.current.connectionFailure.connect((_sender: KernelManager, err: Error) => {
        console.log(
          '[ERROR] An error has occurred while preparing the Kernel Manager. ' + err.name + ': ' + err.message,
        );
      });

      await kernelManager.current.ready.then(() => {
        console.log('Kernel Manager is ready!');
      });
    }
  }

  useEffect(() => {
    initializeKernelManagers();
  }, []);

  const onCardExpand = () => {
    setIsCardExpanded(!isCardExpanded);
  };

  const onSearchChange = (value: string) => {
    setSearchValue(value);
  };

  const onCreateKernelClicked = () => {
    setIsConfirmCreateModalOpen(!isConfirmCreateModalOpen);
  };

  const onCancelCreateKernelClicked = () => {
    setIsConfirmCreateModalOpen(false);
  };

  const onCancelExecuteCodeClicked = () => {
    setIsExecuteCodeModalOpen(false);
    setExecuteCodeKernel(null);
  };

  const onExecuteCodeClicked = (index: number) => {
    setIsExecuteCodeModalOpen(true);
    setExecuteCodeKernel(filteredKernels[index]);
  };

  async function startKernel() {
    // Precondition: The KernelManager is defined.
    const manager: KernelManager = kernelManager.current!;

    // Start a python kernel
    const kernel: IKernelConnection = await manager.startNew({ name: 'distributed' });

    // Register a callback for when the kernel changes state.
    kernel.statusChanged.connect((_, status) => {
      console.log(`New Kernel Status Update: ${status}`);
    });
  }

  async function onConfirmExecuteCodeClicked(code: string) {
    console.log('Executing code:\n' + code);

    const kernelId: string | undefined = executeCodeKernel?.kernelId;

    if (kernelId == undefined) {
      console.log("Couldn't determiner kernel ID of target kernel for code execution...");
      return;
    }

    if (!kernelManager.current) {
      console.log('ERROR: Kernel Manager is not available. Will try to connect...');
      initializeKernelManagers();
      return;
    }

    const kernelConnection: IKernelConnection = kernelManager.current.connectTo({
      model: { id: kernelId, name: kernelId },
    });

    const future = kernelConnection.requestExecute({ code: code });

    // Handle iopub messages
    future.onIOPub = (msg) => {
      if (msg.header.msg_type !== 'status') {
        console.log(msg.content);
      }
    };
    await future.done;
    console.log('Execution on Kernel ' + kernelId + ' is done.');
  }

  const onConfirmCreateKernelClicked = () => {
    // _event: KeyboardEvent | React.MouseEvent
    console.log('Creating a new Kernel.');

    // Close the confirmation dialogue.
    setIsConfirmCreateModalOpen(!isConfirmCreateModalOpen);

    // Create a new kernel.
    if (!kernelManager.current) {
      console.log('ERROR: Kernel Manager is not available. Will try to connect...');
      initializeKernelManagers();
      return;
    }

    // Create a new kernel.
    startKernel();
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
            <Button id="create-kernel-button" variant="plain" onClick={onCreateKernelClicked}>
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
          {/* <Th /> */}
          <Th />
        </Tr>
      </Thead>
      <Tbody>
        {kernel.replicas.map((replica) => (
          <Tr key={replica.replicaId}>
            <Td dataLabel="ID">{replica.replicaId}</Td>
            <Td dataLabel="Pod">{replica.podId}</Td>
            <Td dataLabel="Node">{replica.nodeId}</Td>
            {/* <Td>
              <Tooltip exitDelay={20} entryDelay={175} content={<div>Execute Python code on this replica.</div>}>
                <Button variant={'link'} icon={<CodeIcon />} onClick={() => onExecuteCodeClicked(replica.kernelId)}>
                  Execute
                </Button>
              </Tooltip>
            </Td> */}
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
                              <Button variant={'link'} icon={<CodeIcon />} onClick={() => onExecuteCodeClicked(idx)}>
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
          <ConfirmationModal
            isOpen={isConfirmCreateModalOpen}
            onConfirm={onConfirmCreateKernelClicked}
            onClose={onCancelCreateKernelClicked}
            message="Are you sure you'd like to create a new kernel?"
          />
          <ExecuteCodeOnKernelModal
            kernelId={executeCodeKernel?.kernelId || 'N/A'}
            isOpen={isExecuteCodeModalOpen}
            onClose={onCancelExecuteCodeClicked}
            onSubmit={onConfirmExecuteCodeClicked}
          />
        </CardBody>
      </CardExpandableContent>
    </Card>
  );
};
