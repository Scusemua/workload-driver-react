import React from 'react';
import {
    Button,
    Form,
    FormGroup,
    Grid,
    GridItem,
    Modal,
    ModalVariant,
    TextInput,
    TextInputGroup,
    TextInputGroupMain,
} from '@patternfly/react-core';

import { ResourceSpec } from '@app/Data';

export interface CreateKernelsModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    onConfirm: (input: string, resourceSpec: ResourceSpec) => void;
    defaultInputValue?: string; // Default value for the text input box. Optional; will default to the empty string if none specified.
}

export const CreateKernelsModal: React.FunctionComponent<CreateKernelsModalProps> = (props) => {
    const originalNumKernelsHint: string = '1';
    const originalResourceAmountHint: string = '0';

    const [numKernels, setNumKernels] = React.useState(props.defaultInputValue || '');
    const [cpus, setCpus] = React.useState(props.defaultInputValue || '');
    const [memory, setMemory] = React.useState(props.defaultInputValue || '');
    const [gpus, setGpus] = React.useState(props.defaultInputValue || '');
    const [numKernelsHintText, setNumKernelsHintText] = React.useState(originalNumKernelsHint);
    const [cpuHintText, setCpuHintText] = React.useState(originalResourceAmountHint);
    const [memHintText, setMemHintText] = React.useState(originalResourceAmountHint);
    const [gpuHintText, setGpuHintText] = React.useState(originalResourceAmountHint);

    const onConfirmClicked = () => {
        const resourceSpec: ResourceSpec = {
            cpu: Number.parseInt(cpus),
            memory: Number.parseInt(memory),
            gpu: Number.parseInt(gpus),
        };
        props.onConfirm(numKernels, resourceSpec);
    };

    return (
        <Modal
            variant={ModalVariant.small}
            title={'Create a New Kernel'}
            isOpen={props.isOpen}
            onClose={props.onClose}
            titleIconVariant={'info'}
            actions={[
                <Button key="confirm" variant="primary" onClick={onConfirmClicked}>
                    Confirm
                </Button>,
                <Button key="cancel" variant="link" onClick={props.onClose}>
                    Cancel
                </Button>,
            ]}
        >
            <Form>
                <FormGroup label="How many kernels would you like to create?" isRequired>
                    <TextInputGroup>
                        <TextInputGroupMain
                            id="num-kernels-textinput"
                            aria-label="num-kernels-textinput"
                            hint={numKernelsHintText}
                            value={numKernels}
                            type="number"
                            onChange={(_event, value) => {
                                if (value != '') {
                                    setNumKernelsHintText('');
                                } else {
                                    setNumKernelsHintText(originalNumKernelsHint || '');
                                }

                                setNumKernels(value);
                            }}
                        />
                    </TextInputGroup>
                </FormGroup>
                <Grid hasGutter>
                    <GridItem span={4} rowSpan={1}>
                        <FormGroup label="CPUs? (millicpus)">
                            <TextInputGroup>
                                <TextInputGroupMain
                                    id="num-cpus-textinput"
                                    aria-label="num-cpus-textinput"
                                    hint={cpuHintText}
                                    type={'number'}
                                    value={cpus}
                                    onChange={(_event, value) => {
                                        if (value != '') {
                                            setCpuHintText('');
                                        } else {
                                            setCpuHintText(originalResourceAmountHint || '');
                                        }

                                        setCpus(value);
                                    }}
                                />
                            </TextInputGroup>
                        </FormGroup>
                    </GridItem>
                    <GridItem span={4} rowSpan={1}>
                        <FormGroup label="Memory?">
                            <TextInputGroup>
                                <TextInputGroupMain
                                    id="amount-mem-textinput"
                                    aria-label="amount-mem-textinput"
                                    hint={memHintText}
                                    type={'number'}
                                    value={memory}
                                    onChange={(_event, value) => {
                                        if (value != '') {
                                            setMemHintText('');
                                        } else {
                                            setMemHintText(originalResourceAmountHint || '');
                                        }

                                        setMemory(value);
                                    }}
                                />
                            </TextInputGroup>
                        </FormGroup>
                    </GridItem>
                    <GridItem span={4} rowSpan={1}>
                        <FormGroup label="GPUs?">
                            <TextInput
                                id="num-gpus-textinput"
                                aria-label="num-gpus-textinput"
                                placeholder={gpuHintText}
                                type={'number'}
                                value={gpus}
                                onChange={(_event, value) => {
                                    if (value != '') {
                                        setGpuHintText('');
                                    } else {
                                        setGpuHintText(originalResourceAmountHint || '');
                                    }

                                    setGpus(value);
                                }}
                            />
                        </FormGroup>
                    </GridItem>
                </Grid>
            </Form>
        </Modal>
    );
};
