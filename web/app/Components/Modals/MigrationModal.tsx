import React from 'react';
import { KubernetesNodeList } from '@app/Components/NodeList';
import { Button, Modal, ModalVariant, Text, TextContent, TextVariants } from '@patternfly/react-core';
import { DistributedJupyterKernel, JupyterKernelReplica } from '@data/Kernel';
import { KubernetesNode } from '@data/Kubernetes';

export interface MigrationModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    onConfirm: (JupyterKernelReplica, DistributedJupyterKernel, string) => void;
    targetKernel?: DistributedJupyterKernel | null;
    targetReplica?: JupyterKernelReplica | null;
    nodes: KubernetesNode[];
    manuallyRefreshNodes: () => void; // Function to manually refresh the nodes.
}

export const MigrationModal: React.FunctionComponent<MigrationModalProps> = (props) => {
    const [targetNodeID, setTargetNodeID] = React.useState('');

    return (
        <Modal
            variant={ModalVariant.large}
            titleIconVariant="info"
            title={'Migrate replica ' + props.targetReplica?.replicaId + ' of kernel ' + props.targetKernel?.kernelId}
            isOpen={props.isOpen}
            onClose={props.onClose}
            actions={[
                <Button
                    key="confirm"
                    variant="primary"
                    onClick={() => props.onConfirm(props.targetReplica, props.targetKernel, targetNodeID)}
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
            <KubernetesNodeList
                nodes={props.nodes}
                manuallyRefreshNodes={props.manuallyRefreshNodes}
                refreshInterval={604800} // Once per week (i.e., never).
                selectable={true}
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
