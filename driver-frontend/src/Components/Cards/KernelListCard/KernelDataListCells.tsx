import { DistributedJupyterKernel } from '@Data/Kernel';
import {
    Button,
    DataListAction,
    DataListCell,
    DataListItemCells,
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
import { GpuIcon, GpuIconAlt2 } from '@src/Assets/Icons';
import React from 'react';

interface KernelDataListCellsProps {
    kernel: DistributedJupyterKernel | null;
    onExecuteCodeClicked: (kernel: DistributedJupyterKernel | null, replicaIdx?: number | undefined) => void;
    onPingKernelClicked: (kernel: DistributedJupyterKernel) => void;
    onInterruptKernelClicked: (kernel: DistributedJupyterKernel) => void;
    onTerminateKernelClicked: (kernel: DistributedJupyterKernel) => void;
    onStopTrainingClicked: (kernel: DistributedJupyterKernel) => void;
    onToggleOrSelectKernelDropdown: (kernel: DistributedJupyterKernel) => void;
    openKernelDropdownMenu: string;
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

export const KernelDataListCells: React.FunctionComponent<KernelDataListCellsProps> = (
    props: KernelDataListCellsProps,
) => {
    return (
        <DataListItemCells
            dataListCells={[
                <DataListCell key="primary-content">
                    <Flex spaceItems={{ default: 'spaceItemsMd' }} direction={{ default: 'column' }}>
                        <FlexItem>
                            {props.kernel != null && <p>Kernel {props.kernel.kernelId}</p>}
                            {props.kernel == null && <p className="loading">Pending</p>}
                        </FlexItem>
                        <Flex className="kernel-list-stat-icons" spaceItems={{ default: 'spaceItemsMd' }}>
                            <FlexItem>
                                <Tooltip content="Number of replicas">
                                    <CubesIcon />
                                </Tooltip>
                                {props.kernel != null && props.kernel.numReplicas}
                                {props.kernel == null && 'TBD'}
                            </FlexItem>
                            <FlexItem>
                                {props.kernel != null && kernelStatusIcons[props.kernel.aggregateBusyStatus]}
                                {props.kernel != null && props.kernel.aggregateBusyStatus}
                                {props.kernel == null && kernelStatusIcons['starting']}
                                {props.kernel == null && 'starting'}
                            </FlexItem>
                            <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                                <FlexItem>
                                    <Tooltip content="millicpus (1/1000th of a CPU core)">
                                        <CpuIcon className="node-cpu-icon" />
                                    </Tooltip>
                                </FlexItem>
                                <FlexItem>
                                    {(props.kernel != null &&
                                        props.kernel.kernelSpec.resourceSpec.cpu != null &&
                                        props.kernel.kernelSpec.resourceSpec.cpu / 1000.0) ||
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
                                    {(props.kernel != null &&
                                        props.kernel.kernelSpec.resourceSpec.memory != null &&
                                        props.kernel.kernelSpec.resourceSpec.memory / 1000.0) ||
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
                                    {(props.kernel != null &&
                                        props.kernel.kernelSpec.resourceSpec.gpu != null &&
                                        props.kernel.kernelSpec.resourceSpec.gpu.toFixed(0)) ||
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
                                    {(props.kernel != null &&
                                        props.kernel.kernelSpec.resourceSpec.vram != null &&
                                        props.kernel.kernelSpec.resourceSpec.vram.toFixed(0)) ||
                                        '0'}
                                </FlexItem>
                            </Flex>
                        </Flex>
                    </Flex>
                </DataListCell>,
                <DataListAction
                    key={'kernel-' + props.kernel?.kernelId + '-actions'}
                    aria-labelledby={
                        'kernel-data-list-' +
                        props.kernel?.kernelId +
                        ' kernel-data-list-action-item-' +
                        props.kernel?.kernelId
                    }
                    id={'kernel-data-list-' + props.kernel?.kernelId}
                    aria-label="Actions"
                >
                    <OverflowMenu breakpoint="xl">
                        <OverflowMenuContent>
                            <OverflowMenuItem>
                                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
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
                                                    props.kernel?.replicas === undefined ||
                                                    (props.kernel !== undefined &&
                                                        true &&
                                                        props.kernel?.replicas?.length < 3)
                                                }
                                                onClick={() => props.onExecuteCodeClicked(props.kernel)}
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
                                                    props.kernel == null ||
                                                    false ||
                                                    props.kernel?.replicas === null ||
                                                    props.kernel?.replicas?.length < 3
                                                }
                                                onClick={() => props.onPingKernelClicked(props.kernel!)}
                                            >
                                                Ping
                                            </Button>
                                        </Tooltip>
                                    </FlexItem>
                                </Flex>
                            </OverflowMenuItem>
                            <OverflowMenuItem>
                                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
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
                                                isDisabled={props.kernel == null}
                                                onClick={() => props.onTerminateKernelClicked(props.kernel!)}
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
                                                    props.kernel == null ||
                                                    props.kernel?.replicas === null ||
                                                    props.kernel?.replicas?.length < 3
                                                }
                                                onClick={() => props.onInterruptKernelClicked(props.kernel!)}
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
                                                    props.kernel == null ||
                                                    props.kernel?.replicas === null ||
                                                    props.kernel?.replicas?.length < 3
                                                }
                                                onClick={() => props.onStopTrainingClicked(props.kernel!)}
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
                                    if (props.kernel !== null) {
                                        props.onToggleOrSelectKernelDropdown(props.kernel);
                                    }
                                }}
                                isOpen={props.openKernelDropdownMenu === props.kernel?.kernelId}
                                toggle={(toggleRef) => (
                                    <MenuToggle
                                        ref={toggleRef}
                                        aria-label="Kernel dropdown menu"
                                        variant="plain"
                                        isDisabled={props.kernel === null}
                                        onClick={() => {
                                            props.onToggleOrSelectKernelDropdown(props.kernel!);
                                        }}
                                        isExpanded={props.openKernelDropdownMenu === props.kernel?.kernelId}
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
                                            props.onExecuteCodeClicked(props.kernel);
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
                                            props.onPingKernelClicked(props.kernel!);
                                        }}
                                    >
                                        Ping
                                    </OverflowMenuDropdownItem>
                                    ,
                                    <OverflowMenuDropdownItem
                                        itemId={1}
                                        key="terminate-kernel-dropdown"
                                        icon={<TrashIcon />}
                                        isDisabled={props.kernel === null}
                                        isDanger
                                        onClick={() => props.onTerminateKernelClicked(props.kernel!)}
                                    >
                                        Terminate
                                    </OverflowMenuDropdownItem>
                                    ,
                                    <OverflowMenuDropdownItem
                                        itemId={1}
                                        key="interrupt-kernel-dropdown"
                                        isDanger
                                        icon={<PauseIcon />}
                                        isDisabled={props.kernel === null}
                                        onClick={() => {
                                            props.onInterruptKernelClicked(props.kernel!);
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
    );
};
