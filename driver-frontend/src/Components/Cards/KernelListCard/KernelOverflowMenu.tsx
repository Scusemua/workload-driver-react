import { DistributedJupyterKernel } from '@Data/Kernel';
import {
    Button,
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

import { CodeIcon, EllipsisVIcon, InfoAltIcon, PauseCircleIcon, PauseIcon, TrashIcon } from '@patternfly/react-icons';
import React from 'react';

interface IKernelKernelOverflowMenuProps {
    kernel?: DistributedJupyterKernel;
    onExecuteCodeClicked: (kernel?: DistributedJupyterKernel, replicaIdx?: number | undefined) => void;
    onPingKernelClicked: (kernel: DistributedJupyterKernel) => void;
    onInterruptKernelClicked: (kernel: DistributedJupyterKernel) => void;
    onTerminateKernelClicked: (kernel: DistributedJupyterKernel) => void;
    onStopTrainingClicked: (kernel: DistributedJupyterKernel) => void;
    onToggleOrSelectKernelDropdown: (kernel: DistributedJupyterKernel) => void;
    openKernelDropdownMenu: string;
}

export const KernelOverflowMenu: React.FunctionComponent<IKernelKernelOverflowMenuProps> = (
    props: IKernelKernelOverflowMenuProps,
) => {
    return (
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
                                        (props.kernel !== undefined && true && props.kernel?.replicas?.length < 3)
                                    }
                                    onClick={() => props.onExecuteCodeClicked(props.kernel)}
                                >
                                    Execute
                                </Button>
                            </Tooltip>
                        </FlexItem>
                        <FlexItem>
                            <Tooltip exitDelay={75} entryDelay={250} position="right" content={<div>Ping kernel.</div>}>
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
                        if (props.kernel) {
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
    );
};
