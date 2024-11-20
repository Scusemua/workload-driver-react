import KernelInfoBar from '@Cards/KernelListCard/KernelInfoBar';
import { DistributedJupyterKernel } from '@Data/Kernel';
import { DataListAction, DataListCell, DataListItemCells } from '@patternfly/react-core';
import { KernelOverflowMenu } from '@src/Components';
import React from 'react';

interface KernelDataListCellsProps {
    kernel?: DistributedJupyterKernel;
    onExecuteCodeClicked: (kernel?: DistributedJupyterKernel, replicaIdx?: number | undefined) => void;
    onPingKernelClicked: (kernel: DistributedJupyterKernel) => void;
    onInterruptKernelClicked: (kernel: DistributedJupyterKernel) => void;
    onTerminateKernelClicked: (kernel: DistributedJupyterKernel) => void;
    onStopTrainingClicked: (kernel: DistributedJupyterKernel) => void;
    onToggleOrSelectKernelDropdown: (kernel: DistributedJupyterKernel) => void;
    openKernelDropdownMenu: string;
}

export const KernelDataListCells: React.FunctionComponent<KernelDataListCellsProps> = (
    props: KernelDataListCellsProps,
) => {
    return (
        <DataListItemCells
            dataListCells={[
                <DataListCell key="primary-content">
                    <KernelInfoBar kernel={props.kernel} />
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
                    <KernelOverflowMenu
                        kernel={props.kernel}
                        onExecuteCodeClicked={props.onExecuteCodeClicked}
                        onPingKernelClicked={props.onPingKernelClicked}
                        onInterruptKernelClicked={props.onInterruptKernelClicked}
                        onTerminateKernelClicked={props.onTerminateKernelClicked}
                        onStopTrainingClicked={props.onStopTrainingClicked}
                        onToggleOrSelectKernelDropdown={props.onToggleOrSelectKernelDropdown}
                        openKernelDropdownMenu={props.openKernelDropdownMenu}
                    />
                </DataListAction>,
            ]}
        />
    );
};
