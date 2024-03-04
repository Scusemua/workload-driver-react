import React from 'react';
import { Button, Modal, ModalVariant } from '@patternfly/react-core';

export interface ConfirmationModalProps {
  children?: React.ReactNode;
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: string;
}

export const ConfirmationModal: React.FunctionComponent<ConfirmationModalProps> = (props) => {
  return (
    <Modal
      variant={ModalVariant.small}
      title={props.title}
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
      {props.message}
    </Modal>
  );
};
