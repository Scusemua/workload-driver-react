import {
	Button,
	Content,
	ContentVariants
} from '@patternfly/react-core';
import {
	Modal,
	ModalVariant
} from '@patternfly/react-core/deprecated';
import React from 'react';

export interface InformationModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    title: string;
    titleIconVariant?: 'success' | 'danger' | 'warning' | 'info';
    message1?: string;
    message2?: string;
}

export const InformationModal: React.FunctionComponent<InformationModalProps> = (props) => {
    return (
        <Modal
            variant={ModalVariant.small}
            titleIconVariant={props.titleIconVariant}
            title={props.title}
            isOpen={props.isOpen}
            onClose={props.onClose}
            actions={[
                <Button key="dismiss" variant="primary" onClick={props.onClose}>
                    Dismiss
                </Button>,
            ]}
        >
            <Content>
                <Content component={ContentVariants.p}>
                    <b>{props.message1 || ''}</b>
                </Content>
                <Content component={ContentVariants.p}>{props.message2 || ''}</Content>
            </Content>
        </Modal>
    );
};
