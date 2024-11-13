import { DistributedJupyterKernel, JupyterKernelReplica } from '@Data/Kernel';
import {
    Button,
    Dropdown,
    DropdownList,
    MenuToggle,
    OverflowMenu,
    OverflowMenuContent,
    OverflowMenuControl,
    OverflowMenuDropdownItem,
    OverflowMenuItem,
    Tooltip,
} from '@patternfly/react-core';

import { CodeIcon, EllipsisVIcon, MigrationIcon } from '@patternfly/react-icons';
import { Td, Tr } from '@patternfly/react-table';
import React from 'react';

interface KernelReplicaTableRowProps {
    kernel: DistributedJupyterKernel;
    openMigrationModal: (kernel: DistributedJupyterKernel, replica: JupyterKernelReplica) => void;
    openReplicaDropdownMenu: string;
    onExecuteCodeClicked: (kernel: DistributedJupyterKernel | null, replicaIdx?: number | undefined) => void;
    setOpenReplicaDropdownMenu: (replicaId: string) => void;
    setOpenKernelDropdownMenu: (kernelId: string) => void;
    replica: JupyterKernelReplica;
    replicaIdx: number;
}

export const KernelReplicaTableRow: React.FunctionComponent<KernelReplicaTableRowProps> = (
    props: KernelReplicaTableRowProps,
) => {
    const onToggleOrSelectReplicaDropdown = (replica: JupyterKernelReplica) => {
        const entryId: string = `${replica.kernelId}-${replica.replicaId}`;
        if (props.openReplicaDropdownMenu === entryId) {
            props.setOpenReplicaDropdownMenu('');
        } else {
            props.setOpenReplicaDropdownMenu(entryId);
            props.setOpenKernelDropdownMenu('');
        }
    };

    return (
        <Tr key={props.replica.replicaId}>
            <Td dataLabel="ID">{props.replica.replicaId}</Td>
            <Td dataLabel="Pod" width={25} modifier="truncate">
                {props.replica.podId}
            </Td>
            <Td dataLabel="Node" width={25} modifier="truncate">
                {props.replica.nodeId}
            </Td>
            <Td width={45}>
                <OverflowMenu breakpoint="xl">
                    <OverflowMenuContent>
                        <OverflowMenuItem>
                            <Tooltip
                                exitDelay={20}
                                entryDelay={175}
                                position={'left'}
                                content={
                                    <div>
                                        Execute Python code on replica{' '}
                                        {props.kernel.replicas[props.replicaIdx].replicaId}.
                                    </div>
                                }
                            >
                                <Button
                                    variant={'link'}
                                    icon={<CodeIcon />}
                                    /* Disable the 'Execute' button if we have no replicas, or if we don't have at least 3. */
                                    isDisabled={props.kernel?.replicas === null || props.kernel?.replicas?.length < 3}
                                    onClick={() => props.onExecuteCodeClicked(props.kernel, props.replicaIdx)}
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
                                    isLoading={props.replica.isMigrating}
                                    isDisabled={props.replica.isMigrating}
                                    icon={props.replica.isMigrating ? null : <MigrationIcon />}
                                    onClick={() => {
                                        props.openMigrationModal(props.kernel, props.replica);
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
                                onToggleOrSelectReplicaDropdown(props.replica);
                            }}
                            isOpen={
                                props.openReplicaDropdownMenu === `${props.replica.kernelId}-${props.replica.replicaId}`
                            }
                            toggle={(toggleRef) => (
                                <MenuToggle
                                    ref={toggleRef}
                                    aria-label="Replica dropdown toggle"
                                    variant="plain"
                                    onClick={() => {
                                        onToggleOrSelectReplicaDropdown(props.replica);
                                    }}
                                    isExpanded={
                                        props.openReplicaDropdownMenu ===
                                        `${props.replica.kernelId}-${props.replica.replicaId}`
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
                                    /* Disable the 'Execute' button if we have no replicas, or if we don't have at least 3. */
                                    isDisabled={props.kernel?.replicas === null || props.kernel?.replicas?.length < 3}
                                    icon={<CodeIcon />}
                                    onClick={() => {
                                        props.onExecuteCodeClicked(props.kernel, props.replicaIdx);
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
                                        props.openMigrationModal(props.kernel, props.replica);
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
    );
};
