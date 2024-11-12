import { NodeListCard } from '@Cards/NodeListCard/NodeListCard';
import { DistributedJupyterKernel, JupyterKernelReplica } from '@Data/Kernel';
import {
	Button,
	Content,
	ContentVariants
} from '@patternfly/react-core';
import {
	Modal,
	ModalVariant
} from '@patternfly/react-core/deprecated';
import { AuthorizationContext } from '@Providers/AuthProvider';
import React from 'react';

export interface MigrationModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    onConfirm: (replica: JupyterKernelReplica, kernel: DistributedJupyterKernel, targetNodeId: string) => void;
    targetKernel?: DistributedJupyterKernel | null;
    targetReplica?: JupyterKernelReplica | null;
}

export const MigrationModal: React.FunctionComponent<MigrationModalProps> = (props) => {
    const [targetNodeID, setTargetNodeID] = React.useState('');

    const { authenticated } = React.useContext(AuthorizationContext);

    React.useEffect(() => {
        // Automatically close the modal of we are logged out.
        if (!authenticated) {
            props.onClose();
        }
    }, [props, authenticated]);

    const onConfirmClicked = () => {
        if (!authenticated) {
            return;
        }

        props.onConfirm(props.targetReplica!, props.targetKernel!, targetNodeID);
    };

    return (
        <Modal
            variant={ModalVariant.medium}
            titleIconVariant="info"
            aria-label="migration-modal"
            title={'Migrate replica ' + props.targetReplica!.replicaId + ' of kernel ' + props.targetKernel!.kernelId}
            isOpen={props.isOpen}
            onClose={props.onClose}
            actions={[
                <Button key="confirm" variant="primary" isDisabled={!authenticated} onClick={() => onConfirmClicked}>
                    Confirm
                </Button>,
                <Button key="cancel" variant="link" onClick={props.onClose}>
                    Cancel
                </Button>,
            ]}
        >
            <Content>
                <Content component={ContentVariants.p}>
                    If desired, you may specify a target node. If no target node is specified, then the system will
                    select one automatically.
                </Content>
            </Content>
            <br />
            <NodeListCard
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
            <Content>
                <Content component={ContentVariants.p} hidden={targetNodeID == ''}>
                    <strong>Selected Node:</strong> {targetNodeID}
                </Content>
                <Content component={ContentVariants.p} hidden={targetNodeID != ''}>
                    <strong>Selected Node:</strong> None
                </Content>
            </Content>
        </Modal>
    );
};
