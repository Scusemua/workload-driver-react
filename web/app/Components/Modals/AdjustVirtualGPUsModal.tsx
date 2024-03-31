import React from 'react';
import { Button, Form, FormGroup, Modal, ModalVariant, TextInput, ValidatedOptions } from '@patternfly/react-core';
import { KubernetesNode } from '@app/Data';
import { CheckCircleIcon } from '@patternfly/react-icons';
import { useNodes } from '@app/Providers';

export interface AdjustVirtualGPUsModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    onConfirm: (value: number) => Promise<void>;
    node: KubernetesNode | null;
    titleIconVariant?: 'success' | 'danger' | 'warning' | 'info';
}

export const AdjustVirtualGPUsModal: React.FunctionComponent<AdjustVirtualGPUsModalProps> = (props) => {
    const [inputValidated, setInputValidated] = React.useState(true);
    const [adjustmentState, setAdjustmentState] = React.useState('idle');
    const [adjustedGPUs, setAdjustedGPUs] = React.useState('');

    const { refreshNodes } = useNodes();

    const handleAdjustedGPUsChanged = (_event, vgpus: string) => {
        const validValue: boolean = /[0-9]/.test(vgpus) || vgpus == '';

        // If it is the empty string, then we'll default to the current value, which will ultimately do nothing.
        if (vgpus == '') {
            setAdjustedGPUs('');
            setInputValidated(true);
            return;
        }

        // If we can't even convert the value to a number, then update the state accordingly.
        if (!validValue) {
            setInputValidated(false);
            setAdjustedGPUs('');
            return;
        }

        // Convert to a number.
        const parsed: number = parseInt(vgpus, 10);

        // If it's a float or something, then just default to no seed.
        if (Number.isNaN(parsed)) {
            setInputValidated(false);
            setAdjustedGPUs('');
            return;
        }

        // If it's greater than the max value, then it is invalid.
        if (parsed > 2147483647 || parsed < 0) {
            setInputValidated(false);
            setAdjustedGPUs(vgpus); // Leave the string unchanged.
            return;
        }

        setAdjustedGPUs(parsed.toString());
        setInputValidated(true);
    };

    const onCloseclicked = () => {
        if (adjustmentState === 'applied') {
            setAdjustmentState('idle');
        }
        if (adjustmentState === 'applied' || adjustmentState === 'idle') {
            setAdjustedGPUs('');
        }
        props.onClose();
    };

    const onConfirmClicked = () => {
        if (!props.node) {
            console.error(`Cannot determine target node of adjust-vgpus operation...`);
            return;
        }

        if (adjustmentState === 'applied') {
            setAdjustmentState('idle');
            props.onClose();
            return;
        }

        // The default value is the current number of vGPUs.
        let value = props.node?.CapacityVGPUs;
        if (adjustedGPUs != '') {
            value = parseInt(adjustedGPUs, 10);
        }

        setAdjustmentState('processing');
        props.onConfirm(value).then(() => {
            // Update/refresh the nodes since we know one of their virtual GPU resources changed.
            setTimeout(() => {
                refreshNodes();
                setAdjustmentState('applied');
                console.log(`Completed vGPU change of node ${props.node?.NodeId}`);
            }, 5000);
        });
    };

    return (
        <Modal
            variant={ModalVariant.medium}
            titleIconVariant={props.titleIconVariant}
            title={`Adjust vGPUs of Node ${props.node?.NodeId}`}
            isOpen={props.isOpen}
            onClose={props.onClose}
            actions={[
                <Button
                    key="confirm-adjusted-vgpus"
                    variant="primary"
                    onClick={onConfirmClicked}
                    isDisabled={!inputValidated}
                    isLoading={adjustmentState == 'processing'}
                    icon={adjustmentState === 'applied' ? <CheckCircleIcon /> : null}
                >
                    {adjustmentState === 'idle' && 'Confirm'}
                    {adjustmentState === 'processing' && 'Applying...'}
                    {adjustmentState === 'applied' && 'Done'}
                </Button>,
                <Button key="cancel-adjusted-vgpus" variant="link" onClick={onCloseclicked}>
                    Cancel
                </Button>,
            ]}
        >
            <Form>
                <FormGroup label={`New vGPUs value? (Current total vGPUs: ${props.node?.CapacityVGPUs})`}>
                    <TextInput
                        id="adjusted-vgpus-value"
                        aria-label="adjusted-vgpus-value"
                        type="number"
                        isDisabled={adjustmentState !== 'idle'}
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