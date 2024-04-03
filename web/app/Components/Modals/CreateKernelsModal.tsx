import React, { useRef } from 'react';
import {
    Button,
    Divider,
    Form,
    FormGroup,
    FormSection,
    FormSelect,
    FormSelectOption,
    Grid,
    GridItem,
    Modal,
    ModalVariant,
    TextInput,
    TextInputGroup,
    TextInputGroupMain,
    TextInputProps,
} from '@patternfly/react-core';

import { ResourceSpec } from '@app/Data';
import { v4 as uuidv4 } from 'uuid';

export interface CreateKernelsModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    onConfirm: (
        numKernelsToCreate: number,
        kernelIds: string[],
        sessionIds: string[],
        resourceSpecs: ResourceSpec[],
    ) => void;
    defaultInputValue?: string; // Default value for the text input box. Optional; will default to the empty string if none specified.
}

export const CreateKernelsModal: React.FunctionComponent<CreateKernelsModalProps> = (props) => {
    const originalResourceAmountHint: string = '0';

    const [numKernelsText, setNumKernelsText] = React.useState('1');
    const [numKernels, setNumKernels] = React.useState(1);

    const [numKernelsValidated, setNumKernelsValidated] = React.useState<TextInputProps['validated']>('default');
    const [cpusValidated, setCpusValidated] = React.useState<TextInputProps['validated']>('default');
    const [gpusValidated, setGpusValidated] = React.useState<TextInputProps['validated']>('default');
    const [memValidated, setMemValidated] = React.useState<TextInputProps['validated']>('default');

    const [cpus, setCpus] = React.useState<Map<number, string>>(new Map());
    new Map();
    const [memory, setMemory] = React.useState<Map<number, string>>(new Map());
    new Map();
    const [gpus, setGpus] = React.useState<Map<number, string>>(new Map());
    new Map();
    const [kernelIds, setKernelIds] = React.useState<Map<number, string>>(new Map());
    const [sessionIds, setSessionIds] = React.useState<Map<number, string>>(new Map());

    const [currentKernelIndex, setCurrentKernelIndex] = React.useState(0);

    const [cpuHintText, setCpuHintText] = React.useState(originalResourceAmountHint);
    const [memHintText, setMemHintText] = React.useState(originalResourceAmountHint);
    const [gpuHintText, setGpuHintText] = React.useState(originalResourceAmountHint);

    const defaultKernelId = React.useRef<Map<number, string> | null>(null);
    const defaultSessionId = React.useRef<Map<number, string> | null>(null);

    if (defaultKernelId.current === null) {
        defaultKernelId.current = new Map<number, string>();

        for (let i: number = 0; i < 100; i++) {
            defaultKernelId.current.set(i, uuidv4());
        }
    }

    if (defaultSessionId.current === null) {
        defaultSessionId.current = new Map<number, string>();

        for (let i: number = 0; i < 100; i++) {
            defaultSessionId.current.set(i, uuidv4());
        }
    }

    const onChangeCurrentKernelIndex = (_event: React.FormEvent<HTMLSelectElement>, value: string) => {
        setCurrentKernelIndex(Number.parseInt(value));
    };

    const onConfirmClicked = () => {
        const resourceSpecsList: ResourceSpec[] = [];
        const kernelIdList: string[] = [];
        const sessionIdList: string[] = [];

        for (let i = 0; i < numKernels; i++) {
            let cpu: number = Number.parseInt(cpus.get(i) || '0');
            if (Number.isNaN(cpu)) {
                cpu = 0;
            }

            let gpu: number = Number.parseInt(gpus.get(i) || '0');
            if (Number.isNaN(gpu)) {
                gpu = 0;
            }

            let mem: number = Number.parseInt(memory.get(i) || '0');
            if (Number.isNaN(mem)) {
                mem = 0;
            }

            const resourceSpec: ResourceSpec = {
                cpu: cpu,
                memory: mem,
                gpu: gpu,
            };

            resourceSpecsList.push(resourceSpec);
            kernelIdList.push(kernelIds[i] || uuidv4());
            sessionIdList.push(sessionIds[i] || uuidv4());
        }

        props.onConfirm(numKernels, kernelIdList, sessionIdList, resourceSpecsList);
    };

    // Returns true if confirm button should be disabled.
    const isSomeFieldInvalid = () => {
        return (
            cpusValidated == 'error' ||
            cpusValidated == 'warning' ||
            gpusValidated == 'error' ||
            gpusValidated == 'warning' ||
            memValidated == 'error' ||
            memValidated == 'warning' ||
            numKernelsValidated == 'error' ||
            numKernelsValidated == 'warning'
        );
    };

    return (
        <Modal
            variant={ModalVariant.small}
            title={'Create a New Kernel'}
            isOpen={props.isOpen}
            onClose={props.onClose}
            titleIconVariant={'info'}
            actions={[
                <Button key="confirm" variant="primary" onClick={onConfirmClicked} isDisabled={isSomeFieldInvalid()}>
                    Confirm
                </Button>,
                <Button key="cancel" variant="link" onClick={props.onClose}>
                    Cancel
                </Button>,
            ]}
        >
            <Form>
                <FormGroup label="How many kernels would you like to create? (1 - 100)" isRequired>
                    <TextInputGroup>
                        <TextInput
                            id="num-kernels-textinput"
                            aria-label="num-kernels-textinput"
                            placeholder={'1'}
                            value={numKernelsText}
                            validated={numKernelsValidated}
                            type="number"
                            onChange={(_event, value) => {
                                setNumKernelsText(value);

                                if (value == '') {
                                    setNumKernels(1);
                                    setNumKernelsValidated('success');

                                    if (currentKernelIndex >= 1) {
                                        setCurrentKernelIndex(0);
                                    }

                                    return;
                                }

                                const valueAsNumber: number = Number.parseInt(value);

                                if (Number.isNaN(value)) {
                                    setNumKernelsValidated('error');
                                    return;
                                }

                                if (valueAsNumber > 100 || valueAsNumber <= 0) {
                                    setNumKernelsValidated('error');
                                    return;
                                }

                                setNumKernelsValidated('success');
                                setNumKernels(valueAsNumber);

                                if (valueAsNumber <= currentKernelIndex) {
                                    setCurrentKernelIndex(valueAsNumber - 1);
                                }
                            }}
                        />
                    </TextInputGroup>
                </FormGroup>
                <FormGroup label="Which kernel's properties would you like to modify?">
                    <FormSelect
                        isDisabled={isSomeFieldInvalid()}
                        value={currentKernelIndex}
                        onChange={onChangeCurrentKernelIndex}
                        aria-label="Current Kernel Index Select"
                    >
                        {Array.from(Array(numKernels).keys()).map((idx: number) => (
                            <FormSelectOption key={idx} value={idx} label={`Kernel ${idx + 1}`} />
                        ))}
                    </FormSelect>
                </FormGroup>
                <Divider />
                <FormSection title={`Kernel ${currentKernelIndex + 1}`} titleElement="h2">
                    <Grid span={12} hasGutter>
                        <GridItem span={6} rowSpan={1}>
                            <FormGroup label="Kernel ID">
                                <TextInputGroup>
                                    <TextInput
                                        id="kernel-id-text-input"
                                        arial-label="kernel-id-text-input"
                                        placeholder={defaultKernelId.current.get(currentKernelIndex)}
                                        type="text"
                                        maxLength={36}
                                        value={
                                            (kernelIds.get(currentKernelIndex) && kernelIds.get(currentKernelIndex)) ||
                                            ''
                                        }
                                        onChange={(_event: React.FormEvent<HTMLInputElement>, value: string) => {
                                            setKernelIds(new Map(kernelIds).set(currentKernelIndex, value));
                                        }}
                                    />
                                </TextInputGroup>
                            </FormGroup>
                        </GridItem>
                        <GridItem span={6} rowSpan={1}>
                            <FormGroup label="Session ID">
                                <TextInputGroup>
                                    <TextInput
                                        id="session-id-text-input"
                                        arial-label="session-id-text-input"
                                        placeholder={defaultSessionId.current.get(currentKernelIndex)}
                                        type="text"
                                        value={
                                            (sessionIds.get(currentKernelIndex) &&
                                                sessionIds.get(currentKernelIndex)) ||
                                            ''
                                        }
                                        onChange={(_event: React.FormEvent<HTMLInputElement>, value: string) => {
                                            setSessionIds(new Map(sessionIds).set(currentKernelIndex, value));
                                        }}
                                        maxLength={36}
                                    />
                                </TextInputGroup>
                            </FormGroup>
                        </GridItem>
                        <GridItem span={4} rowSpan={1}>
                            <FormGroup label="CPUs (millicpus)">
                                <TextInputGroup>
                                    <TextInput
                                        id="num-cpus-textinput"
                                        aria-label="num-cpus-textinput"
                                        placeholder={cpuHintText}
                                        type={'number'}
                                        validated={cpusValidated}
                                        value={
                                            (cpus.get(currentKernelIndex) &&
                                                cpus.get(currentKernelIndex)?.toString()) ||
                                            ''
                                        }
                                        onChange={(_event, value) => {
                                            setCpus(new Map(cpus.set(currentKernelIndex, value)));

                                            if (value == '') {
                                                setCpusValidated('success');
                                                return;
                                            }

                                            const valueAsNumber: number = Number.parseInt(value);

                                            if (Number.isNaN(value)) {
                                                setCpusValidated('error');
                                                return;
                                            }

                                            if (valueAsNumber > 128 || valueAsNumber <= 0) {
                                                setCpusValidated('error');
                                                return;
                                            }

                                            setCpusValidated('success');
                                        }}
                                    />
                                </TextInputGroup>
                            </FormGroup>
                        </GridItem>
                        <GridItem span={4} rowSpan={1}>
                            <FormGroup label="Memory">
                                <TextInputGroup>
                                    <TextInput
                                        id="amount-mem-textinput"
                                        aria-label="amount-mem-textinput"
                                        type={'number'}
                                        placeholder={memHintText}
                                        validated={memValidated}
                                        value={
                                            (memory.get(currentKernelIndex) !== undefined &&
                                                memory.get(currentKernelIndex)?.toString()) ||
                                            ''
                                        }
                                        onChange={(_event, value) => {
                                            setMemory(new Map(memory.set(currentKernelIndex, value)));

                                            if (value == '') {
                                                setMemValidated('success');
                                                return;
                                            }

                                            const valueAsNumber: number = Number.parseInt(value);

                                            if (Number.isNaN(value)) {
                                                setMemValidated('error');
                                                return;
                                            }

                                            if (valueAsNumber > 128 || valueAsNumber <= 0) {
                                                setMemValidated('error');
                                                return;
                                            }

                                            setMemValidated('success');
                                        }}
                                    />
                                </TextInputGroup>
                            </FormGroup>
                        </GridItem>
                        <GridItem span={4} rowSpan={1}>
                            <FormGroup label="GPUs">
                                <TextInput
                                    id="num-gpus-textinput"
                                    aria-label="num-gpus-textinput"
                                    placeholder={gpuHintText}
                                    type={'number'}
                                    validated={gpusValidated}
                                    value={
                                        (gpus.get(currentKernelIndex) && gpus.get(currentKernelIndex)?.toString()) || ''
                                    }
                                    onChange={(_event, value) => {
                                        setGpus(new Map(gpus).set(currentKernelIndex, value));

                                        if (value == '') {
                                            setGpusValidated('success');
                                            return;
                                        }

                                        const valueAsNumber: number = Number.parseInt(value);

                                        if (Number.isNaN(value)) {
                                            setGpusValidated('error');
                                            return;
                                        }

                                        if (valueAsNumber > 128 || valueAsNumber <= 0) {
                                            setGpusValidated('error');
                                            return;
                                        }

                                        setGpusValidated('success');
                                    }}
                                />
                            </FormGroup>
                        </GridItem>
                    </Grid>
                </FormSection>
            </Form>
        </Modal>
    );
};
