import { KernelReplicaTableRow } from '@Cards/KernelListCard/KernelReplicaTableRow';
import { DistributedJupyterKernel, JupyterKernelReplica } from '@Data/Kernel';
import { Skeleton } from '@patternfly/react-core';

import { BundleIcon, VirtualMachineIcon } from '@patternfly/react-icons';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import React from 'react';

interface KernelReplicaTableProps {
    kernel: DistributedJupyterKernel;
    openMigrationModal: (kernel: DistributedJupyterKernel, replica: JupyterKernelReplica) => void;
    openReplicaDropdownMenu: string;
    onExecuteCodeClicked: (kernel?: DistributedJupyterKernel, replicaIdx?: number | undefined) => void;
    setOpenReplicaDropdownMenu: (replicaId: string) => void;
    setOpenKernelDropdownMenu: (kernelId: string) => void;
}

export const KernelReplicaTable: React.FunctionComponent<KernelReplicaTableProps> = (
    props: KernelReplicaTableProps,
) => {
    const getPendingReplicaRow = (id: number) => {
        return (
            <Tr key={`pending-replica-${id}`}>
                <Td dataLabel="ID">
                    <Skeleton width="100%" />
                </Td>
                <Td dataLabel="Pod/Container" width={25} modifier="truncate">
                    <Skeleton width="100%" />
                </Td>
                <Td dataLabel="Node" width={25} modifier="truncate">
                    <Skeleton width="100%" />
                </Td>
                <Td width={45} />
            </Tr>
        );
    };

    return (
        <Table isStriped aria-label="Pods Table" variant={'compact'} borders={true}>
            <Thead>
                <Tr>
                    <Th aria-label={'kernel-ID'}>ID</Th>
                    <Th aria-label={'kernel-container'}>
                        <BundleIcon />
                        {' Pod/Container'}
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
                        <KernelReplicaTableRow
                            key={`kernel-${props.kernel.kernelId}-replica-table-row-index-${replicaIdx}`}
                            kernel={props.kernel}
                            replica={replica}
                            replicaIdx={replicaIdx}
                            openMigrationModal={props.openMigrationModal}
                            onExecuteCodeClicked={props.onExecuteCodeClicked}
                            setOpenReplicaDropdownMenu={props.setOpenReplicaDropdownMenu}
                            setOpenKernelDropdownMenu={props.setOpenKernelDropdownMenu}
                            openReplicaDropdownMenu={props.openReplicaDropdownMenu}
                        />
                    ))}
            </Tbody>
        </Table>
    );
};
