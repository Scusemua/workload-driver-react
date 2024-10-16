import { NodeList } from '@Cards/NodeListCard/NodeList';
import { DistributedJupyterKernel, JupyterKernelReplica } from '@Data/Kernel';
import { Button, Modal, ModalVariant, Text, TextContent, TextVariants } from '@patternfly/react-core';
import React from 'react';

export interface MigrationModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    onConfirm: (
        replica: JupyterKernelReplica,
        kernel: DistributedJupyterKernel,
        targetNodeId: string,
    ) => void;
    targetKernel?: DistributedJupyterKernel | null;
    targetReplica?: JupyterKernelReplica | null;
}

export const MigrationModal: React.FunctionComponent<MigrationModalProps> = (props) => {
    const [targetNodeID, setTargetNodeID] = React.useState('');

    return (
        <Modal
            variant={ModalVariant.medium}
            titleIconVariant="info"
            aria-label="migration-modal"
            title={'Migrate replica ' + props.targetReplica!.replicaId + ' of kernel ' + props.targetKernel!.kernelId}
            isOpen={props.isOpen}
            onClose={props.onClose}
            actions={[
                <Button
                    key="confirm"
                    variant="primary"
                    onClick={() => props.onConfirm(props.targetReplica!, props.targetKernel!, targetNodeID)}
                >
                    Confirm
                </Button>,
                <Button key="cancel" variant="link" onClick={props.onClose}>
                    Cancel
                </Button>,
            ]}
        >
            <TextContent>
                <Text component={TextVariants.p}>
                    If desired, you may specify a target node. If no target node is specified, then the system will
                    select one automatically.
                </Text>
            </TextContent>
            <br />
            <NodeList
                isDashboardList={false}
                hideAdjustVirtualGPUsButton={true}
                displayNodeToggleSwitch={false}
                nodesPerPage={10}
                selectableViaCheckboxes={true}
                hideControlPlaneNode={true}
                disableRadiosWithKernel={props.targetReplica != null ? props.targetReplica.kernelId : undefined}
                onSelectNode={(nodeId: string) => {
                    setTargetNodeID(nodeId);
                }}
            />
            <br />
            <TextContent>
                <Text component={TextVariants.p} hidden={targetNodeID == ''}>
                    <strong>Selected Node:</strong> {targetNodeID}
                </Text>
                <Text component={TextVariants.p} hidden={targetNodeID != ''}>
                    <strong>Selected Node:</strong> None
                </Text>
            </TextContent>
        </Modal>
    );
};