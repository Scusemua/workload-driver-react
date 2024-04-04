import React, { useEffect, useReducer, useRef } from 'react';
import {
    Badge,
    Button,
    Card,
    CardBody,
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
    Dropdown,
    DropdownList,
    Flex,
    FlexItem,
    InputGroup,
    InputGroupItem,
    Menu,
    MenuContent,
    MenuItem,
    MenuList,
    MenuToggle,
    OverflowMenu,
    OverflowMenuContent,
    OverflowMenuControl,
    OverflowMenuDropdownItem,
    OverflowMenuItem,
    Pagination,
    PaginationVariant,
    Popper,
    SearchInput,
    Text,
    TextVariants,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarFilter,
    ToolbarGroup,
    ToolbarItem,
    ToolbarToggleGroup,
    Tooltip,
} from '@patternfly/react-core';

import toast from 'react-hot-toast';

import { KernelManager, ServerConnection, SessionManager } from '@jupyterlab/services';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import {
    BundleIcon,
    CheckCircleIcon,
    CodeIcon,
    CpuIcon,
    CubesIcon,
    EllipsisVIcon,
    ExclamationTriangleIcon,
    FilterIcon,
    HourglassHalfIcon,
    InfoAltIcon,
    MemoryIcon,
    MigrationIcon,
    PauseIcon,
    PlusIcon,
    RebootingIcon,
    SkullIcon,
    SpinnerIcon,
    StopCircleIcon,
    SyncIcon,
    TrashIcon,
    VirtualMachineIcon,
} from '@patternfly/react-icons';

import {
    ConfirmationModal,
    CreateKernelsModal,
    ExecuteCodeOnKernelModal,
    InformationModal,
} from '@app/Components/Modals';
import { DistributedJupyterKernel, JupyterKernelReplica, ResourceSpec } from '@data/Kernel';
import { useKernels } from '@providers/KernelProvider';
import { GpuIcon } from '@app/Icons';
import { HeightFactorContext, KernelHeightFactorContext } from '@app/Dashboard/Dashboard';
import { IKernelConnection } from '@jupyterlab/services/lib/kernel/kernel';
import { ISessionConnection } from '@jupyterlab/services/lib/session/session';
import { IModel as ISessionModel } from '@jupyterlab/services/lib/session/session';

function range(start: number, end: number) {
    const nums: number[] = [];
    for (let i: number = start; i < end; i++) nums.push(i);
    return nums;
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
    kernelsPerPage: number;
}

export const KernelList: React.FunctionComponent<KernelListProps> = (props: KernelListProps) => {
    const [searchValue, setSearchValue] = React.useState('');
    const [statusSelections, setStatusSelections] = React.useState<string[]>([]);
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
    const [page, setPage] = React.useState(1);
    const [perPage, setPerPage] = React.useState(props.kernelsPerPage);
    const { kernels, kernelsAreLoading, refreshKernels } = useKernels();
    const [openReplicaDropdownMenu, setOpenReplicaDropdownMenu] = React.useState<string>('');
    const [openKernelDropdownMenu, setOpenKernelDropdownMenu] = React.useState<string>('');
    const heightFactorContext: HeightFactorContext = React.useContext(KernelHeightFactorContext);

    const [, forceUpdate] = useReducer((x) => x + 1, 0);

    const kernelIdSet = useRef<Set<string>>(new Set()); // Keep track of kernels we've seen before.
    const numKernelsCreating = useRef(0); // Used to display "pending" entries in the kernel list.
    const kernelManager = useRef<KernelManager | null>(null);
    const sessionManager = useRef<SessionManager | null>(null);

    const onToggleOrSelectReplicaDropdown = (replica: JupyterKernelReplica) => {
        const entryId: string = `${replica.kernelId}-${replica.replicaId}`;
        if (openReplicaDropdownMenu === entryId) {
            setOpenReplicaDropdownMenu('');
        } else {
            setOpenReplicaDropdownMenu(entryId);
            setOpenKernelDropdownMenu('');
        }
    };

    const onToggleOrSelectKernelDropdown = (kernel: DistributedJupyterKernel | null) => {
        if (openKernelDropdownMenu === kernel?.kernelId) {
            setOpenKernelDropdownMenu('');
        } else {
            setOpenKernelDropdownMenu(kernel?.kernelId || '');
            setOpenReplicaDropdownMenu('');
        }
    };

    // If there are any new kernels, decrement `numKernelsCreating`.
    kernels.forEach((kernel: DistributedJupyterKernel) => {
        if (!kernelIdSet.current.has(kernel.kernelId)) {
            kernelIdSet.current.add(kernel.kernelId);
            numKernelsCreating.current -= 1;

            if (numKernelsCreating.current < 0) {
                console.warn("Tried to decrement 'numKernelsCreating' below 0...");
                numKernelsCreating.current = 0;
            }
        }
    });

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

        heightFactorContext.setHeightFactor(Math.min(newPerPage, kernels.length));
    };

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
                console.error(
                    'An error has occurred while preparing the Kernel Manager. ' + err.name + ': ' + err.message,
                );
            });

            await kernelManager.current.ready.then(() => {
                console.log('Kernel Manager is ready!');
            });
        }

        if (sessionManager.current === null) {
            if (kernelManager.current === null) {
                console.error('Cannot create Session Manager as Kernel Mangaer is null.');
                return;
            }

            sessionManager.current = new SessionManager({
                kernelManager: kernelManager.current,
                serverSettings: ServerConnection.makeSettings({
                    token: '',
                    appendToken: false,
                    baseUrl: '/jupyter',
                    wsUrl: 'ws://localhost:8888/',
                    fetch: fetch,
                }),
            });

            await sessionManager.current.ready.then(() => {
                console.log('Session Manager is ready!');
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
    };

    useEffect(() => {
        initializeKernelManagers();
    }, []);

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

    const onExecuteCodeClicked = (kernel: DistributedJupyterKernel | null, replicaIdx?: number | undefined) => {
        if (kernel == null) {
            return;
        }

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

    async function onInspectKernelClicked(kernel: DistributedJupyterKernel) {
        const kernelId: string = kernel.kernelId;
        console.log('User is inspecting kernel ' + kernelId);
    }

    const onInterruptKernelClicked = (index: number) => {
        const kernelId: string | undefined = filteredKernels[index].kernelId;
        console.log('User is interrupting kernel ' + kernelId);

        if (!kernelManager.current) {
            console.error('Kernel Manager is not available. Will try to connect...');
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
        }
    };

    async function startKernel(kernelId: string, sessionId: string, resourceSpec: ResourceSpec) {
        numKernelsCreating.current = numKernelsCreating.current + 1;

        console.log(
            `Starting kernel ${kernelId} (sessionId=${sessionId}) now. ResourceSpec: ${JSON.stringify(resourceSpec)}`,
        );

        const username: string = sessionId;

        console.log(`Starting new 'distributed' kernel for user ${username} with clientID=${sessionId}.`);
        console.log(`Creating new Jupyter Session ${sessionId} now...`);

        if (!sessionManager.current || !sessionManager.current.isReady) {
            console.error(
                `SessionManager is NOT ready... will try to initialize the SessionManager before proceeding.`,
            );
            await initializeKernelManagers();
            console.warn(`Trying again...`);

            if (!sessionManager.current || !sessionManager.current.isReady) {
                toast.error('Cannot establish connection with Jupyter Server.');
                numKernelsCreating.current -= 1;
                return;
            }
        }

        console.log(`sessionManager.current.isReady: ${sessionManager.current.isReady}`);

        const req: RequestInit = {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                // 'Cache-Control': 'no-cache, no-transform, no-store',
            },
            body: JSON.stringify({
                id: sessionId,
                kernel: {
                    name: 'distributed',
                    kernel_id: kernelId,
                },
                name: sessionId,
                path: sessionId,
                type: 'notebook',
                resource_spec: resourceSpec,
            }),
        };

        async function start_session() {
            const response: Response = await fetch('jupyter/api/sessions', req);

            if (response.status != 201) {
                numKernelsCreating.current -= 1;
                const responseText: string = await response.text();
                let err: Error | null = null;
                try {
                    const responseJson = JSON.parse(responseText);
                    console.error(
                        `Failed to create new Session. Received (${response.status} ${response.statusText}): ${responseJson.message}`,
                    );
                    err = {
                        name: `${response.status} ${response.statusText}`,
                        message: `${response.status} ${response.statusText}: ${responseJson.message}`,
                        stack: new Error().stack,
                    };
                } catch (e) {
                    console.log(e);
                    console.error(
                        `Failed to create new Session. Received (${response.status} ${response.statusText}): ${responseText}`,
                    );
                    err = {
                        name: `${response.status} ${response.statusText}`,
                        message: `${response.status} ${response.statusText}: ${responseText}`,
                        stack: new Error().stack,
                    };
                }

                throw err;
            }

            return await response.json();
        }

        const sessionModel: ISessionModel = await toast.promise(start_session(), {
            loading: <b>Starting new Jupyter Session and Jupyter Kernel...</b>,
            success: <b>Successfully started new Jupyter Session and Jupyter Kernel.</b>,
            error: (reason: Error) => {
                return (
                    <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
                        <FlexItem>
                            <b>Failed to start new Jupyter Session and Jupyter Kernel.</b>
                        </FlexItem>
                        <FlexItem>
                            <b>Reason:</b> {reason.message}
                        </FlexItem>
                    </Flex>
                );
            },
        });

        const session: ISessionConnection = sessionManager.current.connectTo({
            model: sessionModel,
            kernelConnectionOptions: {
                handleComms: true,
            },
            username: sessionId,
            clientId: sessionId,
        });

        console.log(
            `Successfully created new Jupyter Session. ClientID=${sessionId}, SessionID=${session.id}, SessionName=${session.name}, SessionKernelClientID=${session.kernel?.clientId}, SessionKernelName=${session.kernel?.name}, SessionKernelID=${session.kernel?.id}.`,
        );

        if (session.kernel === null) {
            toast.error(`Kernel for newly-created Session ${session.id} is null...`);
            return;
        }
        const kernel: IKernelConnection = session.kernel!;

        console.log(`Successfully launched new kernel: kernel ${kernel.id}`);

        const requestOptions = {
            method: 'PATCH',
            headers: {
                'Content-Type': 'application/json',
                // 'Cache-Control': 'no-cache, no-transform, no-store',
            },
            body: JSON.stringify({
                resourceSpec: resourceSpec,
                kernelId: kernel.id,
            }),
        };

        fetch('api/resourcespecs', requestOptions).catch((reason) => {
            console.error(
                `Failed to register ResourceSpec for newly-created kernel, kernel ${kernel.id}, because: ${reason}`,
            );
        });

        // Register a callback for when the kernel changes state.
        kernel.statusChanged.connect((_, status) => {
            console.log(`New Kernel Status Update: ${status}`);
        });

        refreshKernels();

        // Update/refresh the kernels since we know a new one was just created.
        // setTimeout(() => {
        //     refreshKernels();
        // }, 3000);
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
            console.error("Couldn't determiner kernel ID of target kernel for code execution...");
            return;
        }

        if (!kernelManager.current) {
            console.error('Kernel Manager is not available. Will try to connect...');
            initializeKernelManagers();
            return;
        }

        const kernelConnection: IKernelConnection = kernelManager.current.connectTo({
            model: { id: kernelId, name: kernelId },
        });

        const future = kernelConnection.requestExecute({ code: code }, undefined, {
            target_replica: executeCodeKernelReplica?.replicaId || -1,
        });

        // Handle iopub messages
        future.onIOPub = (msg) => {
            console.log('Received IOPub message:\n%s\n', JSON.stringify(msg));
            const messageType: string = msg.header.msg_type;
            if (messageType == 'execute_input') {
                // Do nothing.
            } else if (messageType == 'status') {
                logConsumer(
                    msg['header']['date'] +
                        ': Execution state changed to ' +
                        JSON.stringify(msg.content['execution_state']) +
                        '\n',
                );
            } else if (messageType == 'stream') {
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
            console.error('Kernel Manager is not available. Will try to connect...');
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
                refreshKernels();
            });
        }

        for (let i: number = 0; i < kernelIds.length; i++) {
            const kernelId: string = kernelIds[i];
            toast.promise(delete_kernel(kernelId), {
                loading: <b>Deleting kernel {kernelId}</b>,
                success: <b>Successfully deleted kernel {kernelId}</b>,
                error: <b>Failed to delete kernel {kernelId}</b>,
            });
        }

        setSelectedKernels([]);
        setKernelToDelete('');
    };

    const onConfirmCreateKernelClicked = (
        numKernelsToCreate: number,
        kernelIds: string[],
        sessionIds: string[],
        resourceSpecs: ResourceSpec[],
    ) => {
        console.log(`Creating ${numKernelsToCreate} new Kernel(s).`);

        // Close the confirmation dialogue.
        setIsConfirmCreateModalOpen(false);

        // Create a new kernel.
        if (!kernelManager.current) {
            console.error('Kernel Manager is not available. Will try to connect...');
            initializeKernelManagers();
            return;
        } else if (!kernelManager.current.isReady) {
            console.warn("Kernel Manager isn't ready yet!");
            toast.error("Kernel Manager isn't ready yet.");
            return;
        }

        if (!sessionManager.current) {
            console.error('Session Manager is not available. Will try to connect...');
            initializeKernelManagers();
            return;
        } else if (!sessionManager.current.isReady) {
            console.warn("Session Manager isn't ready yet!");
            toast.error("Session Manager isn't ready yet.");
            return;
        }

        console.log("We're now creating %d kernel(s).", numKernelsToCreate);
        forceUpdate();

        let errorOccurred = false;
        for (let i = 0; i < numKernelsToCreate; i++) {
            if (errorOccurred) break;

            console.log(
                `Creating kernel ${i + 1} / ${numKernelsToCreate} now. KernelID: ${kernelIds[i]}, SessionID: ${
                    sessionIds[i]
                }, ResourceSpec: ${JSON.stringify(resourceSpecs[i])}`,
            );

            // Create a new kernel.
            startKernel(kernelIds[i], sessionIds[i], resourceSpecs[i]).catch((error) => {
                console.error('Error while trying to start a new kernel:\n' + JSON.stringify(error));
                setErrorMessagePreamble(`An error occurred while trying to start a new kernel: ${error.name}`);
                setErrorMessage(`${error.message}`);
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

    const filteredKernels = kernels.filter(onFilter).slice(perPage * (page - 1), perPage * (page - 1) + perPage);

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
        <ToolbarToggleGroup toggleIcon={<FilterIcon />} breakpoint="md">
            <Flex
                alignSelf={{ default: 'alignSelfFlexEnd' }}
                alignItems={{ default: 'alignItemsFlexEnd' }}
                spaceItems={{ default: 'spaceItemsSm' }}
            >
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
            <ToolbarGroup className="kernel-list-card-actions" variant="icon-button-group">
                <ToolbarItem>
                    <Tooltip exitDelay={75} content={<div>Create a new kernel.</div>}>
                        <Button
                            label="create-kernels-button"
                            aria-label="create-kernels-button"
                            id="create-kernel-button"
                            variant="plain"
                            onClick={() => setIsConfirmCreateModalOpen(!isConfirmCreateModalOpen)}
                        >
                            <PlusIcon />
                        </Button>
                    </Tooltip>
                    <Tooltip exitDelay={75} content={<div>Terminate selected kernels.</div>}>
                        <Button
                            label="delete-kernels-button"
                            aria-label="delete-kernels-button"
                            id="delete-kernels-button"
                            variant="plain"
                            isDanger
                            isDisabled={kernels.length == 0 || selectedKernels.length == 0}
                            onClick={() => setIsConfirmDeleteKernelsModalOpen(true)}
                        >
                            <TrashIcon />
                        </Button>
                    </Tooltip>
                    <Tooltip exitDelay={75} content={<div>Refresh kernels.</div>}>
                        <Button
                            label="refresh-kernels-button"
                            aria-label="refresh-kernels-button"
                            id="refresh-kernels-button"
                            variant="plain"
                            isDisabled={kernelsAreLoading}
                            className={
                                (kernelsAreLoading && 'loading-icon-spin-toggleable') ||
                                'loading-icon-spin-toggleable paused'
                            }
                            onClick={() => {
                                toast.promise(refreshKernels(), {
                                    loading: <b>Refreshing kernels...</b>,
                                    success: <b>Refreshed kernels!</b>,
                                    error: (reason: Error) => {
                                        let explanation: string = reason.message;
                                        if (reason.name === 'SyntaxError') {
                                            explanation = 'HTTP 504 Gateway Timeout';
                                        }

                                        return (
                                            <Flex
                                                direction={{ default: 'column' }}
                                                spaceItems={{ default: 'spaceItemsNone' }}
                                            >
                                                <FlexItem>
                                                    <b>Could not refresh kernels.</b>
                                                </FlexItem>
                                                <FlexItem>{explanation}</FlexItem>
                                            </Flex>
                                        );
                                    },
                                });
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
        <Table isStriped aria-label="Pods Table" variant={'compact'} borders={true}>
            <Thead>
                <Tr>
                    <Th>ID</Th>
                    <Th>
                        <BundleIcon />
                        {' Pod'}
                    </Th>
                    <Th>
                        <VirtualMachineIcon />
                        {' Node'}
                    </Th>
                    <Th />
                </Tr>
            </Thead>
            <Tbody>
                {kernel.replicas != null &&
                    kernel.replicas.map((replica, replicaIdx) => (
                        <Tr key={replica.replicaId}>
                            <Td dataLabel="ID">{replica.replicaId}</Td>
                            <Td dataLabel="Pod" width={25} modifier="truncate">
                                {replica.podId}
                            </Td>
                            <Td dataLabel="Node" width={25} modifier="truncate">
                                {replica.nodeId}
                            </Td>
                            <Td width={45}>
                                <OverflowMenu breakpoint="xl">
                                    <OverflowMenuContent>
                                        <OverflowMenuItem>
                                            <Tooltip
                                                exitDelay={20}
                                                entryDelay={175}
                                                position={'right'}
                                                content={
                                                    <div>
                                                        Execute Python code on replica{' '}
                                                        {kernel.replicas[replicaIdx].replicaId}.
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
                                        </OverflowMenuItem>
                                        <OverflowMenuItem>
                                            <Tooltip
                                                exitDelay={20}
                                                entryDelay={175}
                                                position={'right'}
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
                                        </OverflowMenuItem>
                                    </OverflowMenuContent>
                                    <OverflowMenuControl>
                                        <Dropdown
                                            onSelect={() => {
                                                onToggleOrSelectReplicaDropdown(replica);
                                            }}
                                            isOpen={
                                                openReplicaDropdownMenu === `${replica.kernelId}-${replica.replicaId}`
                                            }
                                            toggle={(toggleRef) => (
                                                <MenuToggle
                                                    ref={toggleRef}
                                                    aria-label="Replica dropdown toggle"
                                                    variant="plain"
                                                    onClick={() => {
                                                        onToggleOrSelectReplicaDropdown(replica);
                                                    }}
                                                    isExpanded={
                                                        openReplicaDropdownMenu ===
                                                        `${replica.kernelId}-${replica.replicaId}`
                                                    }
                                                >
                                                    <EllipsisVIcon />
                                                </MenuToggle>
                                            )}
                                        >
                                            <DropdownList>
                                                <OverflowMenuDropdownItem
                                                    itemId={0}
                                                    key="execute-code-replica-dropdown"
                                                    aria-label="execute-code-replica-dropdown"
                                                    isShared
                                                    icon={<CodeIcon />}
                                                    onClick={() => {
                                                        onExecuteCodeClicked(kernel, replicaIdx);
                                                    }}
                                                >
                                                    Execute
                                                </OverflowMenuDropdownItem>
                                                <OverflowMenuDropdownItem
                                                    itemId={1}
                                                    key="migrate-replica-dropdown"
                                                    aria-label="migrate-replica-dropdown"
                                                    icon={<MigrationIcon />}
                                                    onClick={() => {
                                                        props.openMigrationModal(kernel, replica);
                                                    }}
                                                >
                                                    Migrate
                                                </OverflowMenuDropdownItem>
                                            </DropdownList>
                                        </Dropdown>
                                    </OverflowMenuControl>
                                </OverflowMenu>
                            </Td>
                        </Tr>
                    ))}
            </Tbody>
        </Table>
    );

    const onTerminateKernelClicked = (kernel: DistributedJupyterKernel | null) => {
        if (kernel == null) {
            return;
        }

        // We're trying to delete a specific kernel.
        setKernelToDelete(kernel.kernelId);
        setIsConfirmDeleteKernelModalOpen(true);
    };

    const toggleExpandedKernel = (id) => {
        const index = expandedKernels.indexOf(id);
        const newExpanded =
            index >= 0
                ? [...expandedKernels.slice(0, index), ...expandedKernels.slice(index + 1, expandedKernels.length)]
                : [...expandedKernels, id];
        setExpandedKernels(newExpanded);
    };

    const getKernelDataListRow = (kernel: DistributedJupyterKernel | null, idx: number) => {
        return (
            <DataListItem
                isExpanded={expandedKernels.includes(kernel?.kernelId || 'Pending...')}
                key={'kernel-data-row-' + idx}
                className="kernel-list-row"
                id={'kernel-data-list-' + idx}
            >
                <DataListItemRow>
                    <DataListCheck
                        label={'kernel-' + kernel?.kernelId + '-check'}
                        aria-label={'kernel-' + kernel?.kernelId + '-check'}
                        aria-labelledby={'kernel-' + kernel?.kernelId + '-check'}
                        name={'kernel-' + kernel?.kernelId + '-check'}
                        onChange={(event: React.FormEvent<HTMLInputElement>, checked: boolean) =>
                            onSelectKernel(event, checked, kernel?.kernelId)
                        }
                        isDisabled={kernel == null}
                        defaultChecked={kernel != null && kernel.kernelId in selectedKernels}
                    />
                    <DataListToggle
                        onClick={() => {
                            if (kernel == null) {
                                return;
                            }

                            toggleExpandedKernel(kernel?.kernelId);
                        }}
                        isExpanded={kernel != null && expandedKernels.includes(kernel.kernelId)}
                        id={'expand-kernel-' + kernel?.kernelId + '-button'}
                        aria-controls={'expand-kernel-' + kernel?.kernelId + '-button'}
                        label={'expand-kernel-' + kernel?.kernelId + '-button'}
                        aria-label={'expand-kernel-' + kernel?.kernelId + '-button'}
                    />
                    <DataListItemCells
                        dataListCells={[
                            <DataListCell key="primary-content">
                                <Flex spaceItems={{ default: 'spaceItemsMd' }} direction={{ default: 'column' }}>
                                    <FlexItem>
                                        {kernel != null && <p>Kernel {kernel.kernelId}</p>}
                                        {kernel == null && <p className="loading">Pending</p>}
                                    </FlexItem>
                                    <Flex className="kernel-list-stat-icons" spaceItems={{ default: 'spaceItemsMd' }}>
                                        <FlexItem>
                                            <CubesIcon /> {kernel != null && kernel.numReplicas}{' '}
                                            {kernel == null && 'TBD'}
                                        </FlexItem>
                                        <FlexItem>
                                            {kernel != null && kernelStatusIcons[kernel.aggregateBusyStatus]}{' '}
                                            {kernel != null && kernel.aggregateBusyStatus}
                                            {kernel == null && kernelStatusIcons['starting']}{' '}
                                            {kernel == null && 'starting'}
                                        </FlexItem>
                                        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                                            <FlexItem>
                                                <CpuIcon className="node-cpu-icon" />
                                            </FlexItem>
                                            <FlexItem>
                                                {(kernel != null &&
                                                    kernel.kernelSpec.resourceSpec.cpu != null &&
                                                    kernel.kernelSpec.resourceSpec.cpu.toFixed(2)) ||
                                                    '0'}
                                            </FlexItem>
                                        </Flex>
                                        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                                            <FlexItem>
                                                <MemoryIcon className="node-memory-icon" />{' '}
                                            </FlexItem>
                                            <FlexItem>
                                                {(kernel != null &&
                                                    kernel.kernelSpec.resourceSpec.memory != null &&
                                                    kernel.kernelSpec.resourceSpec.memory.toFixed(2)) ||
                                                    '0'}
                                            </FlexItem>
                                        </Flex>
                                        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                                            <FlexItem>
                                                <GpuIcon className="node-memory-icon" />{' '}
                                            </FlexItem>
                                            <FlexItem>
                                                {(kernel != null &&
                                                    kernel.kernelSpec.resourceSpec.gpu != null &&
                                                    kernel.kernelSpec.resourceSpec.gpu.toFixed(0)) ||
                                                    '0'}
                                            </FlexItem>
                                        </Flex>
                                    </Flex>
                                </Flex>
                            </DataListCell>,
                            <DataListAction
                                key={'kernel-' + idx + '-actions'}
                                aria-labelledby={'kernel-data-list-' + idx + ' kernel-data-list-action-item-' + idx}
                                id={'kernel-data-list-' + idx}
                                aria-label="Actions"
                            >
                                <OverflowMenu breakpoint="xl">
                                    <OverflowMenuContent>
                                        <OverflowMenuItem>
                                            <Flex
                                                direction={{ default: 'column' }}
                                                spaceItems={{ default: 'spaceItemsNone' }}
                                            >
                                                <FlexItem>
                                                    <Tooltip
                                                        exitDelay={75}
                                                        entryDelay={250}
                                                        content={<div>Execute Python code on this kernel.</div>}
                                                    >
                                                        <Button
                                                            variant={'link'}
                                                            icon={<CodeIcon />}
                                                            isDisabled={kernel == null}
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
                                                        content={<div>Interrupt this kernel.</div>}
                                                    >
                                                        <Button
                                                            variant={'link'}
                                                            isDanger
                                                            icon={<PauseIcon />}
                                                            isDisabled={kernel == null}
                                                            onClick={() => onInterruptKernelClicked(idx)}
                                                        >
                                                            Interrupt
                                                        </Button>
                                                    </Tooltip>
                                                </FlexItem>
                                            </Flex>
                                        </OverflowMenuItem>
                                        <OverflowMenuItem>
                                            <Flex
                                                direction={{ default: 'column' }}
                                                spaceItems={{ default: 'spaceItemsNone' }}
                                            >
                                                <FlexItem>
                                                    {' '}
                                                    <Tooltip
                                                        exitDelay={75}
                                                        entryDelay={250}
                                                        content={<div>View details about kernel.</div>}
                                                    >
                                                        <Button
                                                            variant={'link'}
                                                            icon={<InfoAltIcon />}
                                                            isDisabled={kernel == null}
                                                            onClick={() => onInspectKernelClicked(filteredKernels[idx])}
                                                        >
                                                            Inspect
                                                        </Button>
                                                    </Tooltip>
                                                </FlexItem>
                                                <FlexItem>
                                                    <OverflowMenuItem>
                                                        <Tooltip
                                                            exitDelay={75}
                                                            entryDelay={250}
                                                            content={<div>Terminate this kernel.</div>}
                                                        >
                                                            <Button
                                                                variant={'link'}
                                                                icon={<TrashIcon />}
                                                                isDanger
                                                                isDisabled={kernel == null}
                                                                onClick={() => onTerminateKernelClicked(kernel)}
                                                            >
                                                                Terminate
                                                            </Button>
                                                        </Tooltip>
                                                    </OverflowMenuItem>
                                                </FlexItem>
                                            </Flex>
                                        </OverflowMenuItem>
                                    </OverflowMenuContent>
                                    <OverflowMenuControl>
                                        <Dropdown
                                            onSelect={() => {
                                                onToggleOrSelectKernelDropdown(kernel);
                                            }}
                                            isOpen={openKernelDropdownMenu === kernel?.kernelId}
                                            toggle={(toggleRef) => (
                                                <MenuToggle
                                                    ref={toggleRef}
                                                    aria-label="Kernel dropdown menu"
                                                    variant="plain"
                                                    onClick={() => {
                                                        onToggleOrSelectKernelDropdown(kernel);
                                                    }}
                                                    isExpanded={openKernelDropdownMenu === kernel?.kernelId}
                                                >
                                                    <EllipsisVIcon />
                                                </MenuToggle>
                                            )}
                                        >
                                            <DropdownList>
                                                <OverflowMenuDropdownItem
                                                    itemId={0}
                                                    key="execute-code-kernel-dropdown"
                                                    isShared
                                                    icon={<CodeIcon />}
                                                    onClick={() => {
                                                        onExecuteCodeClicked(kernel);
                                                    }}
                                                >
                                                    Execute
                                                </OverflowMenuDropdownItem>
                                                ,
                                                <OverflowMenuDropdownItem
                                                    itemId={0}
                                                    key="inspect-code-kernel-dropdown"
                                                    isShared
                                                    icon={<InfoAltIcon />}
                                                    onClick={() => {
                                                        onInspectKernelClicked(kernel!);
                                                    }}
                                                >
                                                    Inspect
                                                </OverflowMenuDropdownItem>
                                                ,
                                                <OverflowMenuDropdownItem
                                                    itemId={1}
                                                    key="terminate-kernel-dropdown"
                                                    icon={<TrashIcon />}
                                                    isDanger
                                                    onClick={() => onTerminateKernelClicked(kernel)}
                                                >
                                                    Terminate
                                                </OverflowMenuDropdownItem>
                                                ,
                                                <OverflowMenuDropdownItem
                                                    itemId={1}
                                                    key="interrupt-kernel-dropdown"
                                                    isDanger
                                                    icon={<PauseIcon />}
                                                    onClick={() => {
                                                        onInterruptKernelClicked(idx);
                                                    }}
                                                >
                                                    Interrupt
                                                </OverflowMenuDropdownItem>
                                            </DropdownList>
                                        </Dropdown>
                                    </OverflowMenuControl>
                                </OverflowMenu>
                            </DataListAction>,
                        ]}
                    />
                </DataListItemRow>
                {kernel != null && (
                    <DataListContent
                        aria-label={'kernel-' + kernel.kernelId + '-expandable-content'}
                        id={'kernel-' + kernel.kernelId + '-expandable-content'}
                        className="kernel-list-expandable-content"
                        isHidden={!expandedKernels.includes(kernel.kernelId)}
                        hasNoPadding={true}
                    >
                        {kernel != null && expandedKernelContent(kernel)}
                    </DataListContent>
                )}
            </DataListItem>
        );
    };

    const pendingKernelArr = range(0, numKernelsCreating.current);

    // console.log(`Kernels: ${JSON.stringify(kernels)}`);

    return (
        <Card isRounded isFullHeight>
            <CardHeader actions={{ actions: cardHeaderActions, hasNoOffset: false }}>
                <CardTitle>
                    <Title headingLevel="h1" size="xl">
                        Active Kernels
                    </Title>
                </CardTitle>
                <Toolbar
                    hidden={kernels.length == 0}
                    id="content-padding-data-toolbar"
                    usePageInsets
                    clearAllFilters={() => {
                        setStatusSelections([]);
                    }}
                >
                    <ToolbarContent>{ToolbarItems}</ToolbarContent>
                </Toolbar>
            </CardHeader>
            <CardBody>
                {(kernels.length > 0 || pendingKernelArr.length > 0) && (
                    <DataList
                        isCompact
                        aria-label="data list"
                        hidden={kernels.length == 0 && pendingKernelArr.length == 0}
                    >
                        {pendingKernelArr.map((_, idx) => getKernelDataListRow(null, idx))}
                        {filteredKernels.map((kernel, idx) =>
                            getKernelDataListRow(kernel, idx + pendingKernelArr.length),
                        )}
                    </DataList>
                )}
                {kernels.length == 0 && pendingKernelArr.length == 0 && (
                    <Text component={TextVariants.h2}>There are no active kernels.</Text>
                )}
                <CreateKernelsModal
                    isOpen={isConfirmCreateModalOpen}
                    onConfirm={onConfirmCreateKernelClicked}
                    onClose={onCancelCreateKernelClicked}
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
                <Pagination
                    hidden={kernels.length == 0}
                    isDisabled={kernels.length == 0}
                    itemCount={kernels.length}
                    widgetId="kernel-list-pagination"
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
                        // {
                        //     title: '4',
                        //     value: 4,
                        // },
                        // {
                        //     title: '5',
                        //     value: 5,
                        // },
                    ]}
                    onSetPage={onSetPage}
                    onPerPageSelect={onPerPageSelect}
                />
            </CardBody>
        </Card>
    );
};
