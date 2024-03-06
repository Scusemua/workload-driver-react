import React from 'react';
import { Button, Modal, ModalVariant, TextInputGroup, TextInputGroupMain } from '@patternfly/react-core';

export interface ConfirmationModalProps {
  children?: React.ReactNode;
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  titleIconVariant?: 'success' | 'danger' | 'warning' | 'info';
  message: string;
}

export const ConfirmationModal: React.FunctionComponent<ConfirmationModalProps> = (props) => {
  return (
    <Modal
      variant={ModalVariant.small}
      titleIconVariant={props.titleIconVariant}
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

export interface ConfirmationWithTextInputModalProps {
  children?: React.ReactNode;
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (input: string) => void;
  title: string;
  message: string;
  titleIconVariant?: 'success' | 'danger' | 'warning' | 'info';
  hint?: string; // Hint text for the text input.
  defaultInputValue?: string; // Default value for the text input box. Optional; will default to the empty string if none specified.
}

export const ConfirmationWithTextInputModal: React.FunctionComponent<ConfirmationWithTextInputModalProps> = (props) => {
  const [textInputValue, setTextInputValue] = React.useState(props.defaultInputValue || '');
  const [hintText, setHintText] = React.useState(props.hint || '');

  const originalHint: string = props.hint || '';

  return (
    <Modal
      variant={ModalVariant.small}
      title={props.title}
      isOpen={props.isOpen}
      onClose={props.onClose}
      titleIconVariant={props.titleIconVariant}
      actions={[
        <Button
          key="confirm"
          variant="primary"
          onClick={() => {
            props.onConfirm(textInputValue);
          }}
        >
          Confirm
        </Button>,
        <Button key="cancel" variant="link" onClick={props.onClose}>
          Cancel
        </Button>,
      ]}
    >
      {props.message}
      <TextInputGroup>
        <TextInputGroupMain
          hint={hintText}
          value={textInputValue}
          onChange={(_event, value) => {
            if (value != '') {
              setHintText('');
            } else {
              setHintText(originalHint || '');
            }

            setTextInputValue(value);
          }}
        />
      </TextInputGroup>
    </Modal>
  );
};
