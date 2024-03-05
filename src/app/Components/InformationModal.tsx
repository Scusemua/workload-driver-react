import React from 'react';
import { Button, Modal, ModalVariant, Text, TextContent, TextVariants } from '@patternfly/react-core';

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
      <TextContent>
        <Text component={TextVariants.p}>{props.message1 || ''}</Text>
        <Text component={TextVariants.p}>{props.message2 || ''}</Text>
      </TextContent>
    </Modal>
  );
};
