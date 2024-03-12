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
    DataListCheck,
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

import {
    CheckCircleIcon,
    CodeIcon,
    CubesIcon,
    ExclamationTriangleIcon,
    FilterIcon,
    HourglassHalfIcon,
    MigrationIcon,
    PauseIcon,
    PlusIcon,
    RebootingIcon,
    SearchIcon,
    SkullIcon,
    SpinnerIcon,
    StopCircleIcon,
    SyncIcon,
    TrashIcon,
} from '@patternfly/react-icons';

import { IKernelConnection } from '@jupyterlab/services/lib/kernel/kernel';
import { IInfoReplyMsg } from '@jupyterlab/services/lib/kernel/messages';

import {
    ConfirmationModal,
    ConfirmationWithTextInputModal,
    ExecuteCodeOnKernelModal,
    InformationModal,
} from '@app/Components/Modals';
import { DistributedJupyterKernel, JupyterKernelReplica } from '@data/Kernel';

function isNumber(value?: string | number): boolean {
    return value != null && value !== '' && !isNaN(Number(value.toString()));
}

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

export interface KernelListProps {
    openMigrationModal: (DistributedJupyterKernel, JupyterKernelReplica) => void;
}

export const KernelList: React.FunctionComponent<KernelListProps> = (props: KernelListProps) => {
    const [searchValue, setSearchValue] = React.useState('');
    const [statusSelections, setStatusSelections] = React.useState<string[]>([]);
    const [isCardExpanded, setIsCardExpanded] = React.useState(true);
    const [expandedKernels, setExpandedKernels] = React.useState<string[]>([]);
    const [isConfirmCreateModalOpen, setIsConfirmCreateModalOpen] = React.useState(false);
    const [isConfirmDeleteKernelsModalOpen, setIsConfirmDeleteKernelsModalOpen] = React.useState(false);
    const [isConfirmDeleteKernelModalOpen, setIsConfirmDeleteKernelModalOpen] = React.useState(false);
    const [isErrorModalOpen, setIsErrorModalOpen] = React.useState(false);
    const [errorMessage, setErrorMessage] = React.useState('');
    const [errorMessagePreamble, setErrorMessagePreamble] = React.useState('');
    const [isExecuteCodeModalOpen, setIsExecuteCodeModalOpen] = React.useState(false);
    const [executeCodeKernel, setExecuteCodeKernel] = React.useState<DistributedJupyterKernel | null>(null);
    const [executeCodeKernelReplica, setExecuteCodeKernelReplica] = React.useState<JupyterKernelReplica | null>(null);
    const [selectedKernels, setSelectedKernels] = React.useState<string[]>([]);
    const [kernelToDelete, setKernelToDelete] = React.useState<string>('');
    const kernelManager = useRef<KernelManager | null>(null);

    async function initializeKernelManagers() {
        if (kernelManager.current === null) {
            const kernelSpecManagerOptions: KernelManager.IOptions = {
                serverSettings: ServerConnection.makeSettings({
                    token: '',
                    appendToken: false,
                    baseUrl: '/jupyter',
                    wsUrl: 'ws://localhost:8888/',
                    fetch: fetch,
                }),
            };
            kernelManager.current = new KernelManager(kernelSpecManagerOptions);

            console.log('Waiting for Kernel Manager to be ready.');

            kernelManager.current.connectionFailure.connect((_sender: KernelManager, err: Error) => {
                console.log(
                    '[ERROR] An error has occurred while preparing the Kernel Manager. ' +
                        err.name +
                        ': ' +
                        err.message,
                );
            });

            await kernelManager.current.ready.then(() => {
                console.log('Kernel Manager is ready!');
            });
        }
    }

    const onSelectKernel = (
        _event: React.FormEvent<HTMLInputElement>,
        _checked: boolean,
        kernelId: string | undefined,
    ) => {
        const item = kernelId as string;

        // console.log('onSelectKernel: ' + item);

        if (selectedKernels.includes(item)) {
            setSelectedKernels(selectedKernels.filter((id) => id !== item));
        } else {
            setSelectedKernels([...selectedKernels, item]);
        }

        console.log('selectedKernels: ' + selectedKernels);
    };

    useEffect(() => {
        initializeKernelManagers();
    }, []);

    const onCardExpand = () => {
        setIsCardExpanded(!isCardExpanded);
    };

    const onSearchChange = (value: string) => {
        setSearchValue(value);
    };

    const onCancelCreateKernelClicked = () => {
        setIsConfirmCreateModalOpen(false);
    };

    const onCancelDeleteKernelClicked = () => {
        setIsConfirmDeleteKernelModalOpen(false);
    };

    const onCancelDeleteKernelsClicked = () => {
        setIsConfirmDeleteKernelsModalOpen(false);
    };

    const onCancelExecuteCodeClicked = () => {
        setIsExecuteCodeModalOpen(false);
        setExecuteCodeKernel(null);
        setExecuteCodeKernelReplica(null);
    };

    const onExecuteCodeClicked = (kernel: DistributedJupyterKernel, replicaIdx?: number | undefined) => {
        // If we clicked the 'Execute' button associated with a specific replica, then set the state for that replica.
        if (replicaIdx !== undefined) {
            // Need to use "!== undefined" because a `replicaIdx` of 0 will be coerced to false if by itself.
            console.log(
                'Will be executing code on replica %d of kernel %s.',
                kernel.replicas[replicaIdx].replicaId,
                kernel.kernelId,
            );
            setExecuteCodeKernelReplica(kernel.replicas[replicaIdx]);
        } else {
            setExecuteCodeKernelReplica(null);
        }

        setExecuteCodeKernel(kernel);
        setIsExecuteCodeModalOpen(true);
    };

    async function onInspectKernelClicked(kernelIndex: number) {
        const kernelId: string | undefined = filteredKernels[kernelIndex].kernelId;
        console.log('User is inspecting kernel ' + kernelId);

        if (!kernelManager.current) {
            console.log('[ERROR] Kernel Manager is not available. Will try to connect...');
            initializeKernelManagers();
            return;
        }

        const kernelConnection: IKernelConnection = kernelManager.current.connectTo({
            model: { id: kernelId, name: kernelId },
        });

        if (kernelConnection.connectionStatus == 'connected') {
            kernelConnection.requestKernelInfo().then((resp: IInfoReplyMsg | undefined) => {
                if (resp == undefined) {
                    console.log('Failed to retrieve information about kernel ' + kernelId);
                } else {
                    console.log('Received info from kernel ' + kernelId + ': ' + JSON.stringify(resp));
                }
            });
        } else {
            console.log('[ERROR] Could not retrieve information for kernel %s. Not connected to the kernel.', kernelId);
            setErrorMessage(
                'Could not retrieve information about kernel ' +
                    kernelId +
                    ' as a connection to the kernel was not established successfully.',
            );
            setIsErrorModalOpen(true);
        }
    }

    const onInterruptKernelClicked = (index: number) => {
        const kernelId: string | undefined = filteredKernels[index].kernelId;
        console.log('User is interrupting kernel ' + kernelId);

        if (!kernelManager.current) {
            console.log('[ERROR] Kernel Manager is not available. Will try to connect...');
            initializeKernelManagers();
            return;
        }

        const kernelConnection: IKernelConnection = kernelManager.current.connectTo({
            model: { id: kernelId, name: kernelId },
        });

        if (kernelConnection.connectionStatus == 'connected') {
            kernelConnection.interrupt().then(() => {
                console.log('Successfully interrupted kernel ' + kernelId);
            });
        } else {
            console.log('[ERROR] Could not interrupt kernel %s. Not connected to the kernel.', kernelId);
            setErrorMessage(
                'Could not interrupt kernel ' +
                    kernelId +
                    ' as a connection to the kernel was not established successfully.',
            );
            setIsErrorModalOpen(true);
        }
    };

    async function startKernel() {
        // Precondition: The KernelManager is defined.
        const manager: KernelManager = kernelManager.current!;

        console.log('Starting kernel now...');

        // Start a python kernel
        const kernel: IKernelConnection = await manager.startNew({ name: 'distributed' });

        console.log('Successfully started kernel!');

        // Register a callback for when the kernel changes state.
        kernel.statusChanged.connect((_, status) => {
            console.log(`New Kernel Status Update: ${status}`);
        });

        // Update/refresh the kernels since we know a new one was just created.
        setTimeout(() => {
            ignoreResponse.current = false;
            fetchKernels();
        }, 3000);
    }

    async function onConfirmExecuteCodeClicked(code: string, logConsumer: (logMessage: string) => void) {
        console.log(
            'Executing code on kernel %s, replica %d:\n%s',
            executeCodeKernel?.kernelId,
            executeCodeKernelReplica?.replicaId,
            code,
        );

        const kernelId: string | undefined = executeCodeKernel?.kernelId;

        if (kernelId == undefined) {
            console.log("Couldn't determiner kernel ID of target kernel for code execution...");
            return;
        }

        if (!kernelManager.current) {
            console.log('[ERROR] Kernel Manager is not available. Will try to connect...');
            initializeKernelManagers();
            return;
        }

        const kernelConnection: IKernelConnection = kernelManager.current.connectTo({
            model: { id: kernelId, name: kernelId },
        });

        const future = kernelConnection.requestExecute({ code: code }, undefined, {
            target_replica_key: executeCodeKernelReplica?.replicaId || -1,
        });

        // Handle iopub messages
        future.onIOPub = (msg) => {
            console.log('Received IOPub message:\n%s\n', JSON.stringify(msg));
            if (msg.header.msg_type == 'status') {
                logConsumer(
                    msg['header']['date'] +
                        ': Execution state changed to ' +
                        JSON.stringify(msg.content['execution_state']) +
                        '\n',
                );
            } else if (msg.header.msg_type == 'execute_input') {
                // Do nothing.
            } else if (msg.header.msg_type == 'stream') {
                if (msg['content']['name'] == 'stderr') {
                    logConsumer(msg['header']['date'] + ' <ERROR>: ' + JSON.stringify(msg.content['text']) + '\n');
                } else if (msg['content']['name'] == 'stdout') {
                    logConsumer(msg['header']['date'] + ': ' + JSON.stringify(msg.content['text']) + '\n');
                } else {
                    logConsumer(msg['header']['date'] + ': ' + JSON.stringify(msg.content['text']) + '\n');
                }
            } else {
                logConsumer(msg['header']['date'] + ': ' + JSON.stringify(msg.content) + '\n');
            }

            // if (msg.header.msg_type !== 'status') {
            //     logConsumer(JSON.stringify(msg.content));
            // }
        };
        await future.done;
        console.log('Execution on Kernel ' + kernelId + ' is done.');
    }

    const onConfirmDeleteKernelsClicked = (kernelIds: string[]) => {
        // Close the confirmation dialogue.
        setIsConfirmDeleteKernelsModalOpen(false);
        setIsConfirmDeleteKernelModalOpen(false);

        // Create a new kernel.
        if (!kernelManager.current) {
            console.log('[ERROR] Kernel Manager is not available. Will try to connect...');
            initializeKernelManagers();
            return;
        }

        /**
         * Delete the specified kernel.
         *
         * @param id The ID of the kernel to be deleted.
         */
        async function delete_kernel(id: string) {
            console.log('Deleting Kernel ' + id + ' now.');
            await kernelManager.current?.shutdown(id).then(() => {
                console.log('Successfully deleted Kernel ' + id + ' now.');

                // Update/refresh the kernels since we know that we just deleted one of them.
                ignoreResponse.current = false;
                fetchKernels();
            });
        }

        setSelectedKernels([]);
        setKernelToDelete('');
        kernelIds.forEach((kernelId) => {
            delete_kernel(kernelId);
        });
    };

    const onConfirmCreateKernelClicked = (input: string) => {
        let numKernelsToCreate: number = 1;
        input = input.trim();

        // If the user specified som
        if (input != '') {
            if (isNumber(input)) {
                numKernelsToCreate = parseInt(input);
            } else {
                console.log('[ERROR] Failed to convert number of kernels to a number: "' + input + '"');
                setErrorMessage('Failed to convert number of kernels to a number: "' + input + '"');
                setIsErrorModalOpen(true);
            }
        }

        console.log(`Creating ${numKernelsToCreate} new Kernel(s).`);

        // Close the confirmation dialogue.
        setIsConfirmCreateModalOpen(false);

        // Create a new kernel.
        if (!kernelManager.current) {
            console.log('[ERROR] Kernel Manager is not available. Will try to connect...');
            initializeKernelManagers();
            return;
        }

        let errorOccurred = false;
        for (let i = 0; i < numKernelsToCreate; i++) {
            if (errorOccurred) break;

            console.log(`Creating kernel ${i + 1} / ${numKernelsToCreate} now.`);

            // Create a new kernel.
            startKernel().catch((error) => {
                console.error('Error while trying to start a new kernel:\n' + error);
                setErrorMessagePreamble('An error occurred while trying to start a new kernel:');
                setErrorMessage(error.toString());
                setIsErrorModalOpen(true);
                errorOccurred = true;
            });
        }
    };

    // Set up status single select
    const [isStatusMenuOpen, setIsStatusMenuOpen] = React.useState<boolean>(false);
    const statusToggleRef = React.useRef<HTMLButtonElement>(null);
    const statusMenuRef = React.useRef<HTMLDivElement>(null);
    const statusContainerRef = React.useRef<HTMLDivElement>(null);

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
            console.log('Refreshing kernels now.');
            // Make a network request to the backend. The server infrastructure handles proxying/routing the request to the correct host.
            // We're specifically targeting the API endpoint I setup called "get-kernels".
            const response = await fetch('api/get-kernels');

            if (response.status == 200) {
                const respKernels: DistributedJupyterKernel[] = await response.json();

                if (!ignoreResponse.current) {
                    console.log('Received kernels: ' + JSON.stringify(respKernels));
                    setKernels(respKernels);
                    ignoreResponse.current = true;
                } else {
                    console.log("Received %d kernel(s), but we're ignoring the response.", respKernels.length);
                }
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
        setInterval(() => {
            ignoreResponse.current = false;
            fetchKernels();
        }, 120000);

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
        <Menu
            ref={statusMenuRef}
            id="mixed-group-status-menu"
            onSelect={onStatusMenuSelect}
            selected={statusSelections}
        >
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
                    <MenuItem
                        hasCheckbox
                        isSelected={statusSelections.includes('autorestarting')}
                        itemId="autorestarting"
                    >
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
            <Flex alignItems={{ default: 'alignItemsFlexEnd' }}>
                <ToolbarItem>
                    <InputGroup>
                        <InputGroupItem isFill>
                            <SearchInput
                                placeholder="Filter by kernel name"
                                value={searchValue}
                                onChange={(_event, value) => onSearchChange(value)}
                                onClear={() => onSearchChange('')}
                            />
                        </InputGroupItem>
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
                        <Button
                            id="create-kernel-button"
                            variant="plain"
                            onClick={() => setIsConfirmCreateModalOpen(!isConfirmCreateModalOpen)}
                        >
                            <PlusIcon />
                        </Button>
                    </Tooltip>
                    <Tooltip exitDelay={75} content={<div>Terminate selected kernels.</div>}>
                        <Button
                            id="delete-kernels-button"
                            variant="plain"
                            onClick={() => setIsConfirmDeleteKernelsModalOpen(true)}
                        >
                            <TrashIcon />
                        </Button>
                    </Tooltip>
                    <Tooltip exitDelay={75} content={<div>Refresh kernels.</div>}>
                        <Button
                            id="refresh-kernels-button"
                            variant="plain"
                            onClick={() => {
                                ignoreResponse.current = false;
                                console.log('Manually refreshing kernels.');
                                fetchKernels();
                            }}
                        >
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
                {kernel.replicas != null &&
                    kernel.replicas.map((replica, replicaIdx) => (
                        <Tr key={replica.replicaId}>
                            <Td dataLabel="ID">{replica.replicaId}</Td>
                            <Td dataLabel="Pod">{replica.podId}</Td>
                            <Td dataLabel="Node">{replica.nodeId}</Td>
                            <Td>
                                <Tooltip
                                    exitDelay={20}
                                    entryDelay={175}
                                    content={
                                        <div>
                                            Execute Python code on replica {kernel.replicas[replicaIdx].replicaId}.
                                        </div>
                                    }
                                >
                                    <Button
                                        variant={'link'}
                                        icon={<CodeIcon />}
                                        onClick={() => onExecuteCodeClicked(kernel, replicaIdx)}
                                    >
                                        Execute
                                    </Button>
                                </Tooltip>
                            </Td>
                            <Td>
                                <Tooltip
                                    exitDelay={20}
                                    entryDelay={175}
                                    content={<div>Migrate this replica to another node.</div>}
                                >
                                    <Button
                                        variant={'link'}
                                        isLoading={replica.isMigrating}
                                        isDisabled={replica.isMigrating}
                                        icon={replica.isMigrating ? null : <MigrationIcon />}
                                        onClick={() => {
                                            props.openMigrationModal(kernel, replica);
                                        }}
                                    >
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
        <Card isRounded isExpanded={isCardExpanded}>
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
                                className="kernel-list-row"
                                id={'content-padding-item-' + idx}
                            >
                                <DataListItemRow>
                                    <DataListCheck
                                        aria-labelledby={'kernel-' + kernel.kernelId + '-check'}
                                        name={'kernel-' + kernel.kernelId + '-check'}
                                        onChange={(event: React.FormEvent<HTMLInputElement>, checked: boolean) =>
                                            onSelectKernel(event, checked, kernel.kernelId)
                                        }
                                        defaultChecked={kernel.kernelId in selectedKernels}
                                    />
                                    <DataListToggle
                                        onClick={() => toggleExpandedKernel(kernel.kernelId)}
                                        isExpanded={expandedKernels.includes(kernel.kernelId)}
                                        id="ex-toggle1"
                                        aria-controls="ex-expand1"
                                    />
                                    <DataListItemCells
                                        dataListCells={[
                                            <DataListCell key="primary-content">
                                                <Flex
                                                    spaceItems={{ default: 'spaceItemsMd' }}
                                                    direction={{ default: 'column' }}
                                                >
                                                    <FlexItem>
                                                        <p>Kernel {kernel.kernelId}</p>
                                                    </FlexItem>
                                                    <Flex
                                                        className="kernel-list-stat-icons"
                                                        spaceItems={{ default: 'spaceItemsMd' }}
                                                    >
                                                        <FlexItem>
                                                            <CubesIcon /> {kernel.numReplicas}
                                                        </FlexItem>
                                                        <FlexItem>
                                                            {kernelStatusIcons[kernel.aggregateBusyStatus]}{' '}
                                                            {kernel.aggregateBusyStatus}
                                                        </FlexItem>
                                                    </Flex>
                                                </Flex>
                                            </DataListCell>,
                                            <DataListAction
                                                key="actions"
                                                aria-labelledby={
                                                    'content-padding-item-' + idx + ' content-action-item-' + idx
                                                }
                                                id={'content-padding-item-' + idx}
                                                aria-label="Actions"
                                            >
                                                <Flex
                                                    spaceItems={{ default: 'spaceItemsNone' }}
                                                    direction={{ default: 'column' }}
                                                >
                                                    <FlexItem>
                                                        <Flex
                                                            spaceItems={{ default: 'spaceItemsSm' }}
                                                            direction={{ default: 'row' }}
                                                        >
                                                            <FlexItem>
                                                                <Tooltip
                                                                    exitDelay={75}
                                                                    entryDelay={250}
                                                                    content={
                                                                        <div>Execute Python code on this kernel.</div>
                                                                    }
                                                                >
                                                                    <Button
                                                                        variant={'link'}
                                                                        icon={<CodeIcon />}
                                                                        onClick={() => onExecuteCodeClicked(kernel)}
                                                                    >
                                                                        Execute
                                                                    </Button>
                                                                </Tooltip>
                                                            </FlexItem>
                                                            <FlexItem>
                                                                <Tooltip
                                                                    exitDelay={75}
                                                                    entryDelay={250}
                                                                    content={
                                                                        <div>
                                                                            Inspect and obtain information about this
                                                                            kernel.
                                                                        </div>
                                                                    }
                                                                >
                                                                    <Button
                                                                        variant={'link'}
                                                                        icon={<SearchIcon />}
                                                                        onClick={() => onInspectKernelClicked(idx)}
                                                                    >
                                                                        Inspect
                                                                    </Button>
                                                                </Tooltip>
                                                            </FlexItem>
                                                        </Flex>
                                                    </FlexItem>
                                                    <FlexItem>
                                                        <Flex
                                                            spaceItems={{ default: 'spaceItemsSm' }}
                                                            direction={{ default: 'row' }}
                                                        >
                                                            <FlexItem>
                                                                <Tooltip
                                                                    exitDelay={75}
                                                                    entryDelay={250}
                                                                    content={<div>Interrupt this kernel.</div>}
                                                                >
                                                                    <Button
                                                                        variant={'link'}
                                                                        isDanger
                                                                        icon={<PauseIcon />}
                                                                        onClick={() => onInterruptKernelClicked(idx)}
                                                                    >
                                                                        Interrupt
                                                                    </Button>
                                                                </Tooltip>
                                                            </FlexItem>
                                                            <FlexItem>
                                                                <Tooltip
                                                                    exitDelay={75}
                                                                    entryDelay={250}
                                                                    content={<div>Terminate this kernel.</div>}
                                                                >
                                                                    <Button
                                                                        variant={'link'}
                                                                        icon={<TrashIcon />}
                                                                        isDanger
                                                                        onClick={() => {
                                                                            // We're trying to delete a specific kernel.
                                                                            setKernelToDelete(kernel.kernelId);
                                                                            setIsConfirmDeleteKernelModalOpen(true);
                                                                        }}
                                                                    >
                                                                        Terminate
                                                                    </Button>
                                                                </Tooltip>
                                                            </FlexItem>
                                                        </Flex>
                                                    </FlexItem>
                                                </Flex>
                                            </DataListAction>,
                                        ]}
                                    />
                                </DataListItemRow>
                                <DataListContent
                                    aria-label={'kernel-' + kernel.kernelId + '-expandable-content'}
                                    id={'kernel-' + kernel.kernelId + '-expandable-content'}
                                    className="kernel-list-expandable-content"
                                    isHidden={!expandedKernels.includes(kernel.kernelId)}
                                >
                                    {kernel != null && expandedKernelContent(kernel)}
                                </DataListContent>
                            </DataListItem>
                        ))}
                    </DataList>
                    <ConfirmationWithTextInputModal
                        isOpen={isConfirmCreateModalOpen}
                        onConfirm={onConfirmCreateKernelClicked}
                        onClose={onCancelCreateKernelClicked}
                        title="Create a New Kernel"
                        message="How many kernels would you like to create?"
                        hint="1"
                    />
                    <ConfirmationModal
                        isOpen={isConfirmDeleteKernelsModalOpen}
                        onConfirm={() => onConfirmDeleteKernelsClicked(selectedKernels)}
                        onClose={onCancelDeleteKernelsClicked}
                        title={'Terminate Selected Kernels'}
                        message={"Are you sure you'd like to delete the specified kernel(s)?"}
                    />
                    <ConfirmationModal
                        isOpen={isConfirmDeleteKernelModalOpen}
                        onConfirm={() => onConfirmDeleteKernelsClicked([kernelToDelete])}
                        onClose={onCancelDeleteKernelClicked}
                        title={'Terminate Kernel'}
                        message={"Are you sure you'd like to delete the specified kernel?"}
                    />
                    <ExecuteCodeOnKernelModal
                        kernelId={executeCodeKernel?.kernelId || 'N/A'}
                        replicaId={executeCodeKernelReplica?.replicaId}
                        isOpen={isExecuteCodeModalOpen}
                        onClose={onCancelExecuteCodeClicked}
                        onSubmit={onConfirmExecuteCodeClicked}
                    />
                    <InformationModal
                        isOpen={isErrorModalOpen}
                        onClose={() => {
                            setIsErrorModalOpen(false);
                            setErrorMessage('');
                            setErrorMessagePreamble('');
                        }}
                        title="An Error has Occurred"
                        titleIconVariant="danger"
                        message1={errorMessagePreamble}
                        message2={errorMessage}
                    />
                </CardBody>
            </CardExpandableContent>
        </Card>
    );
};
