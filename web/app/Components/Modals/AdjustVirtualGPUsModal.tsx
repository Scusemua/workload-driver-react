import React from 'react';
import { Button, Form, FormGroup, Modal, ModalVariant, TextInput, ValidatedOptions } from '@patternfly/react-core';
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
    const [inputValidated, setInputValidated] = React.useState(true);
    const [adjustedGPUs, setAdjustedGPUs] = React.useState('');

    const handleAdjustedGPUsChanged = (_event, vgpus: string) => {
        const validValue: boolean = /[0-9]/.test(vgpus) || vgpus == '';

        // If it's either the empty string, or we can't even convert the value to a number,
        // then update the state accordingly.
        if (!validValue || vgpus == '') {
            setInputValidated(validValue);
            setAdjustedGPUs('');
            return;
        }

        // Convert to a number.
        const parsed: number = parseInt(vgpus, 10);

        // If it's a float or something, then just default to no seed.
        if (Number.isNaN(parsed)) {
            setAdjustedGPUs('');
            return;
        }

        // If it's greater than the max value, then it is invalid.
        if (parsed > 2147483647 || parsed < 0) {
            setInputValidated(false);
            setAdjustedGPUs(vgpus);
            return;
        }

        setAdjustedGPUs(parsed.toString());
        setInputValidated(true);
    };

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
            <Form>
                <FormGroup label={`New vGPUs value? (Current total vGPUs: ${props.node?.CapacityVGPUs})`}>
                    <TextInput
                        type="number"
                        value={adjustedGPUs}
                        onChange={handleAdjustedGPUsChanged}
                        validated={(inputValidated && ValidatedOptions.success) || ValidatedOptions.error}
                        placeholder={`${props.node?.CapacityVGPUs}`}
                    />
                </FormGroup>
            </Form>
        </Modal>
    );
};
