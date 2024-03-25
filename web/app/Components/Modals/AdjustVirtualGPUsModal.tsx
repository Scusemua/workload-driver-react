import React from 'react';
import { Button, Modal, ModalVariant, TextInputGroup, TextInputGroupMain } from '@patternfly/react-core';
import { KubernetesNode } from '@app/Data';

export interface AdjustVirtualGPUsModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    onConfirm: () => void;
    node: KubernetesNode | null;
    titleIconVariant?: 'success' | 'danger' | 'warning' | 'info';
}

export const AdjustVirtualGPUsModal: React.FunctionComponent<AdjustVirtualGPUsModalProps> = (props) => {
    return (
        <Modal
            variant={ModalVariant.medium}
            titleIconVariant={props.titleIconVariant}
            title={`Adjust vGPUs of Node ${props.node?.NodeId}`}
            isOpen={props.isOpen}
            onClose={props.onClose}
            actions={[
                <Button key="confirm" variant="primary" onClick={props.onConfirm}>
                    Confirm
                </Button>,
                <Button key="cancel" variant="link" onClick={props.onClose}>
                    Cancel
                </Button>,
            ]}
        >
            Current total vGPUs: {props.node?.CapacityVGPUs}
        </Modal>
    );
};
