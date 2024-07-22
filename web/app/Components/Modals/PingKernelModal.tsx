import React from 'react';
import { Button, Modal, ModalVariant } from '@patternfly/react-core';

export interface PingKernelModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    onConfirm: (kernelId: string, socketType: 'control' | 'shell') => void;
    kernelId: string;
}

export const PingKernelModal: React.FunctionComponent<PingKernelModalProps> = (props) => {
    return (
        <Modal
            variant={ModalVariant.medium}
            titleIconVariant={'info'}
            title={"Select Socket Type for Ping Operation"}
            isOpen={props.isOpen}
            onClose={props.onClose}
            actions={[
                <Button key="control" variant="primary" onClick={() => props.onConfirm(props.kernelId, "control")}>
                    Control
                </Button>,
                <Button key="shell" variant="primary" onClick={() => props.onConfirm(props.kernelId, "shell")}>
                    Shell
                </Button>,
                <Button key="cancel" variant="link" onClick={props.onClose}>
                    Cancel
                </Button>,
            ]}
        >
            What socket should be used to ping the replicas of kernel {props.kernelId}?
        </Modal>
    );
};
