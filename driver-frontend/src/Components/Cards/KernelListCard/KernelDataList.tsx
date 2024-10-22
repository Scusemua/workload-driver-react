import { HeightFactorContext, KernelHeightFactorContext } from '@App/Dashboard';
import { DistributedJupyterKernel, JupyterKernelReplica } from '@Data/Kernel';
import {
    Button,
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
    MenuToggle,
    OverflowMenu,
    OverflowMenuContent,
    OverflowMenuControl,
    OverflowMenuDropdownItem,
    OverflowMenuItem,
    Pagination,
    PaginationVariant,
    Tooltip,
} from '@patternfly/react-core';

import {
    CheckCircleIcon,
    CodeIcon,
    CpuIcon,
    CubesIcon,
    EllipsisVIcon,
    ExclamationTriangleIcon,
    HourglassHalfIcon,
    InfoAltIcon,
    MemoryIcon,
    PauseCircleIcon,
    PauseIcon,
    RebootingIcon,
    SkullIcon,
    SpinnerIcon,
    StopCircleIcon,
    TrashIcon,
} from '@patternfly/react-icons';
import { useKernels } from '@Providers/KernelProvider';
import { GpuIcon, GpuIconAlt2 } from '@src/Assets/Icons';
import { KernelReplicaTable } from '@src/Components';
import { numberArrayFromRange } from '@src/Utils/utils';
import React, { useRef } from 'react';

export interface KernelDataListProps {
    openMigrationModal: (kernel: DistributedJupyterKernel, replica: JupyterKernelReplica) => void;
    kernelsPerPage: number;
    searchValue: string;
    statusSelections: string[];
    onExecuteCodeClicked: (kernel: DistributedJupyterKernel | null, replicaIdx?: number | undefined) => void;
    onPingKernelClicked: (kernel: DistributedJupyterKernel) => void;
    onInterruptKernelClicked: (kernel: DistributedJupyterKernel) => void;
    onTerminateKernelClicked: (kernel: DistributedJupyterKernel) => void;
    onStopTrainingClicked: (kernel: DistributedJupyterKernel) => void;
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

export const KernelDataList: React.FunctionComponent<KernelDataListProps> = (props: KernelDataListProps) => {
    const [expandedKernels, setExpandedKernels] = React.useState<string[]>([]);
    const [selectedKernels, setSelectedKernels] = React.useState<string[]>([]);

    const numKernelsCreating = useRef(0); // Used to display "pending" entries in the kernel list.

    const [page, setPage] = React.useState(1);
    const [perPage, setPerPage] = React.useState(props.kernelsPerPage);
    const { kernels } = useKernels(false);

    const [openReplicaDropdownMenu, setOpenReplicaDropdownMenu] = React.useState<string>('');
    const [openKernelDropdownMenu, setOpenKernelDropdownMenu] = React.useState<string>('');

    const heightFactorContext: HeightFactorContext = React.useContext(KernelHeightFactorContext);

    const toggleExpandedKernel = (kernelId: string) => {
        const index = expandedKernels.indexOf(kernelId);
        const newExpanded =
            index >= 0
                ? [...expandedKernels.slice(0, index), ...expandedKernels.slice(index + 1, expandedKernels.length)]
                : [...expandedKernels, kernelId];
        setExpandedKernels(newExpanded);
    };

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

    const onToggleOrSelectKernelDropdown = (kernel: DistributedJupyterKernel | null) => {
        if (openKernelDropdownMenu === kernel?.kernelId) {
            setOpenKernelDropdownMenu('');
        } else {
            setOpenKernelDropdownMenu(kernel?.kernelId || '');
            setOpenReplicaDropdownMenu('');
        }
    };

    const onFilter = (repo: DistributedJupyterKernel) => {
        // Search name with search value
        let searchValueInput: RegExp;
        try {
            searchValueInput = new RegExp(props.searchValue, 'i');
        } catch (err) {
            searchValueInput = new RegExp(props.searchValue.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'i');
        }
        const matchesSearchValue = repo.kernelId.search(searchValueInput) >= 0;

        // Search status with status selection
        let matchesStatusValue = false;
        props.statusSelections.forEach(function (selectedStatus) {
            const match = repo.status.toLowerCase() === selectedStatus.toLowerCase();
            matchesStatusValue = matchesStatusValue || match;
        });

        return (
            (props.searchValue === '' || matchesSearchValue) &&
            (props.statusSelections.length === 0 || matchesStatusValue)
        );
    };

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
        heightFactorContext.setHeightFactor(Math.min(newPerPage, kernels.length));
    };

    const filteredKernels = kernels.filter(onFilter).slice(perPage * (page - 1), perPage * (page - 1) + perPage);

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
                                            <Tooltip content="Number of replicas">
                                                <CubesIcon />
                                            </Tooltip>
                                            {kernel != null && kernel.numReplicas}
                                            {kernel == null && 'TBD'}
                                        </FlexItem>
                                        <FlexItem>
                                            {kernel != null && kernelStatusIcons[kernel.aggregateBusyStatus]}
                                            {kernel != null && kernel.aggregateBusyStatus}
                                            {kernel == null && kernelStatusIcons['starting']}
                                            {kernel == null && 'starting'}
                                        </FlexItem>
                                        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                                            <FlexItem>
                                                <Tooltip content="millicpus (1/1000th of a CPU core)">
                                                    <CpuIcon className="node-cpu-icon" />
                                                </Tooltip>
                                            </FlexItem>
                                            <FlexItem>
                                                {(kernel != null &&
                                                    kernel.kernelSpec.resourceSpec.cpu != null &&
                                                    kernel.kernelSpec.resourceSpec.cpu / 1000.0) ||
                                                    '0'}
                                            </FlexItem>
                                        </Flex>
                                        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                                            <FlexItem>
                                                <Tooltip content="RAM usage limit in Gigabytes (GB)">
                                                    <MemoryIcon className="node-memory-icon" />
                                                </Tooltip>
                                            </FlexItem>
                                            <FlexItem>
                                                {(kernel != null &&
                                                    kernel.kernelSpec.resourceSpec.memory != null &&
                                                    kernel.kernelSpec.resourceSpec.memory / 1000.0) ||
                                                    '0'}
                                            </FlexItem>
                                        </Flex>
                                        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                                            <FlexItem>
                                                <Tooltip content="GPU resource usage limit">
                                                    <GpuIcon className="node-gpu-icon" />
                                                </Tooltip>
                                            </FlexItem>
                                            <FlexItem>
                                                {(kernel != null &&
                                                    kernel.kernelSpec.resourceSpec.gpu != null &&
                                                    kernel.kernelSpec.resourceSpec.gpu.toFixed(0)) ||
                                                    '0'}
                                            </FlexItem>
                                        </Flex>
                                        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                                            <FlexItem>
                                                <Tooltip content="VRAM resource usage limit">
                                                    <GpuIconAlt2 className="node-gpu-icon" />
                                                </Tooltip>
                                            </FlexItem>
                                            <FlexItem>
                                                {(kernel != null &&
                                                    kernel.kernelSpec.resourceSpec.vram != null &&
                                                    kernel.kernelSpec.resourceSpec.vram.toFixed(0)) ||
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
                                                        position="left"
                                                        content={<div>Execute Python code on this kernel.</div>}
                                                    >
                                                        <Button
                                                            variant={'link'}
                                                            icon={<CodeIcon />}
                                                            /* Disable the 'Execute' button if we have no replicas, or if we don't have at least 3. */
                                                            isDisabled={
                                                                kernel?.replicas === undefined ||
                                                                (kernel !== undefined &&
                                                                    true &&
                                                                    kernel?.replicas?.length < 3)
                                                            }
                                                            onClick={() => props.onExecuteCodeClicked(kernel)}
                                                        >
                                                            Execute
                                                        </Button>
                                                    </Tooltip>
                                                </FlexItem>
                                                <FlexItem>
                                                    <Tooltip
                                                        exitDelay={75}
                                                        entryDelay={250}
                                                        position="right"
                                                        content={<div>Ping kernel.</div>}
                                                    >
                                                        <Button
                                                            variant={'link'}
                                                            icon={<InfoAltIcon />}
                                                            isDisabled={
                                                                kernel == null ||
                                                                false ||
                                                                kernel?.replicas === null ||
                                                                kernel?.replicas?.length < 3
                                                            }
                                                            onClick={() =>
                                                                props.onPingKernelClicked(filteredKernels[idx])
                                                            }
                                                        >
                                                            Ping
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
                                                    <Tooltip
                                                        exitDelay={75}
                                                        entryDelay={250}
                                                        position="right"
                                                        content={<div>Terminate this kernel.</div>}
                                                    >
                                                        <Button
                                                            variant={'link'}
                                                            icon={<TrashIcon />}
                                                            isDanger
                                                            isDisabled={kernel == null}
                                                            onClick={() => props.onTerminateKernelClicked(kernel!)}
                                                        >
                                                            Terminate
                                                        </Button>
                                                    </Tooltip>
                                                </FlexItem>
                                                <FlexItem>
                                                    <Tooltip
                                                        exitDelay={75}
                                                        entryDelay={250}
                                                        position="left"
                                                        content={<div>Interrupt this kernel.</div>}
                                                    >
                                                        <Button
                                                            variant={'link'}
                                                            isDanger
                                                            icon={<PauseIcon />}
                                                            isDisabled={
                                                                kernel == null ||
                                                                kernel?.replicas === null ||
                                                                kernel?.replicas?.length < 3
                                                            }
                                                            onClick={() => props.onInterruptKernelClicked(kernel!)}
                                                        >
                                                            Interrupt
                                                        </Button>
                                                    </Tooltip>
                                                </FlexItem>
                                                <FlexItem>
                                                    <Tooltip
                                                        exitDelay={75}
                                                        entryDelay={250}
                                                        position="left"
                                                        content={<div>Stop training.</div>}
                                                    >
                                                        <Button
                                                            variant={'link'}
                                                            isDanger
                                                            icon={<PauseCircleIcon />}
                                                            isDisabled={
                                                                kernel == null ||
                                                                kernel?.replicas === null ||
                                                                kernel?.replicas?.length < 3
                                                            }
                                                            onClick={() => props.onStopTrainingClicked(kernel!)}
                                                        >
                                                            Stop Training
                                                        </Button>
                                                    </Tooltip>
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
                                                        props.onExecuteCodeClicked(kernel);
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
                                                        props.onPingKernelClicked(kernel!);
                                                    }}
                                                >
                                                    Ping
                                                </OverflowMenuDropdownItem>
                                                ,
                                                <OverflowMenuDropdownItem
                                                    itemId={1}
                                                    key="terminate-kernel-dropdown"
                                                    icon={<TrashIcon />}
                                                    isDisabled={kernel === null}
                                                    isDanger
                                                    onClick={() => props.onTerminateKernelClicked(kernel!)}
                                                >
                                                    Terminate
                                                </OverflowMenuDropdownItem>
                                                ,
                                                <OverflowMenuDropdownItem
                                                    itemId={1}
                                                    key="interrupt-kernel-dropdown"
                                                    isDanger
                                                    icon={<PauseIcon />}
                                                    isDisabled={kernel === null}
                                                    onClick={() => {
                                                        props.onInterruptKernelClicked(kernel!);
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
                        <KernelReplicaTable
                          kernel={kernel}
                          openMigrationModal={props.openMigrationModal}
                          onExecuteCodeClicked={props.onExecuteCodeClicked}
                          setOpenReplicaDropdownMenu={setOpenReplicaDropdownMenu}
                          setOpenKernelDropdownMenu={setOpenKernelDropdownMenu}
                          openReplicaDropdownMenu={openReplicaDropdownMenu}
                        />
                    </DataListContent>
                )}
            </DataListItem>
        );
    };

    const pendingKernelArr = numberArrayFromRange(0, numKernelsCreating.current);

    return (
        <React.Fragment>
            <DataList isCompact aria-label="data list" hidden={kernels.length == 0 && pendingKernelArr.length == 0}>
                {pendingKernelArr.map((_, idx) => getKernelDataListRow(null, idx))}
                {filteredKernels.map((kernel, idx) => getKernelDataListRow(kernel, idx + pendingKernelArr.length))}
            </DataList>
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
                        title: '1 kernels',
                        value: 1,
                    },
                    {
                        title: '2 kernels',
                        value: 2,
                    },
                    {
                        title: '3 kernels',
                        value: 3,
                    },
                ]}
                onSetPage={onSetPage}
                onPerPageSelect={onPerPageSelect}
            />
        </React.Fragment>
    );
};
