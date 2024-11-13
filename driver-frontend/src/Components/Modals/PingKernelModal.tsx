import { Button } from '@patternfly/react-core';
import { Modal, ModalVariant } from '@patternfly/react-core/deprecated';
import { AuthorizationContext } from '@Providers/AuthProvider';
import React from 'react';

export interface PingKernelModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    onConfirm: (kernelId: string, socketType: 'control' | 'shell') => void;
    kernelId: string;
}

export const PingKernelModal: React.FunctionComponent<PingKernelModalProps> = (props) => {
    const { authenticated } = React.useContext(AuthorizationContext);

    React.useEffect(() => {
        // Automatically close the modal of we are logged out.
        if (!authenticated) {
            props.onClose();
        }
    }, [props, authenticated]);

    return (
        <Modal
            variant={ModalVariant.medium}
            titleIconVariant={'info'}
            aria-label="Modal to Ping a kernel"
            title={'Select Socket Type for Ping Operation'}
            isOpen={props.isOpen}
            onClose={props.onClose}
            actions={[
                <Button
                    key="control"
                    variant="primary"
                    onClick={() => props.onConfirm(props.kernelId, 'control')}
                    isDisabled={!authenticated}
                >
                    Control
                </Button>,
                <Button
                    key="shell"
                    variant="primary"
                    onClick={() => props.onConfirm(props.kernelId, 'shell')}
                    isDisabled={!authenticated}
                >
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
