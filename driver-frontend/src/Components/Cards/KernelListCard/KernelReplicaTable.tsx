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
    Skeleton,
    Tooltip,
} from '@patternfly/react-core';

import { BundleIcon, CodeIcon, EllipsisVIcon, MigrationIcon, VirtualMachineIcon } from '@patternfly/react-icons';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import React from 'react';

interface ExpandedKernelDataListContentProps {
    kernel: DistributedJupyterKernel;
    openMigrationModal: (kernel: DistributedJupyterKernel, replica: JupyterKernelReplica) => void;
    openReplicaDropdownMenu: string;
    onExecuteCodeClicked: (kernel: DistributedJupyterKernel | null, replicaIdx?: number | undefined) => void;
    setOpenReplicaDropdownMenu: (replicaId: string) => void;
    setOpenKernelDropdownMenu: (kernelId: string) => void;
}

export const KernelReplicaTable: React.FunctionComponent<ExpandedKernelDataListContentProps> = (
    props: ExpandedKernelDataListContentProps,
) => {
    const getPendingReplicaRow = (id: number) => {
        return (
            <Tr key={`pending-replica-${id}`}>
                <Td dataLabel="ID">
                    <Skeleton width="100%" />
                </Td>
                <Td dataLabel="Pod" width={25} modifier="truncate">
                    <Skeleton width="100%" />
                </Td>
                <Td dataLabel="Node" width={25} modifier="truncate">
                    <Skeleton width="100%" />
                </Td>
                <Td width={45} />
            </Tr>
        );
    };

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
        <Table isStriped aria-label="Pods Table" variant={'compact'} borders={true}>
            <Thead>
                <Tr>
                    <Th aria-label={'kernel-ID'}>ID</Th>
                    <Th aria-label={'kernel-container'}>
                        <BundleIcon />
                        {' Pod'}
                    </Th>
                    <Th aria-label={'kernel-node'}>
                        <VirtualMachineIcon />
                        {' Node'}
                    </Th>
                    <Th aria-label={'blank'} />
                </Tr>
            </Thead>
            <Tbody>
                {(props.kernel.replicas == undefined || props.kernel.replicas.length == 0) && [
                    getPendingReplicaRow(0),
                    getPendingReplicaRow(1),
                    getPendingReplicaRow(2),
                ]}
                {props.kernel.replicas != undefined &&
                    props.kernel.replicas.map((replica, replicaIdx) => (
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
                                                position={'left'}
                                                content={
                                                    <div>
                                                        Execute Python code on replica{' '}
                                                        {props.kernel.replicas[replicaIdx].replicaId}.
                                                    </div>
                                                }
                                            >
                                                <Button
                                                    variant={'link'}
                                                    icon={<CodeIcon />}
                                                    /* Disable the 'Execute' button if we have no replicas, or if we don't have at least 3. */
                                                    isDisabled={
                                                        props.kernel?.replicas === null ||
                                                        props.kernel?.replicas?.length < 3
                                                    }
                                                    onClick={() => props.onExecuteCodeClicked(props.kernel, replicaIdx)}
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
                                                        props.openMigrationModal(props.kernel, replica);
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
                                                props.openReplicaDropdownMenu ===
                                                `${replica.kernelId}-${replica.replicaId}`
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
                                                        props.openReplicaDropdownMenu ===
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
                                                    /* Disable the 'Execute' button if we have no replicas, or if we don't have at least 3. */
                                                    isDisabled={
                                                        props.kernel?.replicas === null ||
                                                        props.kernel?.replicas?.length < 3
                                                    }
                                                    icon={<CodeIcon />}
                                                    onClick={() => {
                                                        props.onExecuteCodeClicked(props.kernel, replicaIdx);
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
                                                        props.openMigrationModal(props.kernel, replica);
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
};
