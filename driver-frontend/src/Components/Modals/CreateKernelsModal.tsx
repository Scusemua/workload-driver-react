import {
    Button,
    Divider,
    Form,
    FormGroup,
    FormHelperText,
    FormSection,
    FormSelect,
    FormSelectOption,
    Grid,
    GridItem,
    HelperText,
    HelperTextItem,
    Modal,
    ModalVariant,
    TextInput,
    TextInputGroup,
    TextInputProps,
} from '@patternfly/react-core';
import { ExclamationCircleIcon, WarningTriangleIcon } from '@patternfly/react-icons';
import { AuthorizationContext } from '@Providers/AuthProvider';
import { ResourceSpec } from '@src/Data';
import React from 'react';
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

const defaultCPUs: string = '100'; // milli-cpus
const defaultGPUs: string = '1';
const defaultVRAM: string = '1'; // GB
const defaultMemory: string = '1250'; // MB

export const CreateKernelsModal: React.FunctionComponent<CreateKernelsModalProps> = (props) => {
    const [numKernelsText, setNumKernelsText] = React.useState('1');
    const [numKernels, setNumKernels] = React.useState(1);

    const [numKernelsValidated, setNumKernelsValidated] = React.useState<TextInputProps['validated']>('default');
    const [cpusValidated, setCpusValidated] = React.useState<TextInputProps['validated']>('default');
    const [gpusValidated, setGpusValidated] = React.useState<TextInputProps['validated']>('default');
    const [vramValidated, setVramValidated] = React.useState<TextInputProps['validated']>('default');
    const [memValidated, setMemValidated] = React.useState<TextInputProps['validated']>('default');

    const [gpusHelperText, setGpusHelperText] = React.useState<string>('');

    const [cpus, setCpus] = React.useState<Map<number, string>>(new Map());
    const [memory, setMemory] = React.useState<Map<number, string>>(new Map());
    const [gpus, setGpus] = React.useState<Map<number, string>>(new Map());
    const [vram, setVram] = React.useState<Map<number, string>>(new Map());

    const [kernelIds, setKernelIds] = React.useState<Map<number, string>>(new Map());
    const [sessionIds, setSessionIds] = React.useState<Map<number, string>>(new Map());

    const [currentKernelIndex, setCurrentKernelIndex] = React.useState(0);

    const [cpuHintText] = React.useState(defaultCPUs);
    const [memHintText] = React.useState(defaultMemory);
    const [gpuHintText] = React.useState(defaultGPUs);
    const [vramHintText] = React.useState(defaultVRAM);

    const defaultKernelId = React.useRef<Map<number, string> | null>(null);
    const defaultSessionId = React.useRef<Map<number, string> | null>(null);

    const { authenticated } = React.useContext(AuthorizationContext);

    React.useEffect(() => {
        // Automatically close the modal of we are logged out.
        if (!authenticated) {
            props.onClose();
        }
    }, [props, authenticated]);

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
            let cpu: number = Number.parseInt(cpus.get(i) || defaultCPUs);
            if (Number.isNaN(cpu)) {
                cpu = Number.parseInt(defaultCPUs);
            }

            let gpu: number = Number.parseInt(gpus.get(i) || defaultGPUs);
            if (Number.isNaN(gpu)) {
                gpu = Number.parseInt(defaultGPUs);
            }

            let specified_vram: number = Number.parseInt(vram.get(i) || defaultVRAM);
            if (Number.isNaN(vram)) {
                specified_vram = Number.parseInt(defaultVRAM);
            }

            let mem: number = Number.parseInt(memory.get(i) || defaultMemory);
            if (Number.isNaN(mem)) {
                mem = Number.parseInt(defaultMemory);
            }

            const resourceSpec: ResourceSpec = {
                cpu: cpu,
                memory: mem,
                gpu: gpu,
                vram: specified_vram,
            };

            resourceSpecsList.push(resourceSpec);
            kernelIdList.push(kernelIds.get(i) || uuidv4());
            sessionIdList.push(sessionIds.get(i) || uuidv4());
        }

        props.onConfirm(numKernels, kernelIdList, sessionIdList, resourceSpecsList);

        // Reset the form.
        setCpusValidated('default');
        setMemValidated('default');
        setGpusValidated('default');
        setCpus(new Map());
        setMemory(new Map());
        setGpus(new Map());
        setVram(new Map());
        setNumKernels(1);
        setNumKernelsText('1');
        setNumKernelsValidated('default');
        defaultKernelId.current = new Map<number, string>();
        defaultSessionId.current = new Map<number, string>();

        for (let i: number = 0; i < 100; i++) {
            defaultKernelId.current.set(i, uuidv4());
        }

        for (let i: number = 0; i < 100; i++) {
            defaultSessionId.current.set(i, uuidv4());
        }

        setSessionIds(new Map());
        setKernelIds(new Map());

        setCurrentKernelIndex(0);
    };

    // Returns true if the confirm button should be disabled.
    const isSomeFieldInvalid = () => {
        return (
            cpusValidated == 'error' ||
            cpusValidated == 'warning' ||
            gpusValidated == 'error' ||
            memValidated == 'error' ||
            memValidated == 'warning' ||
            numKernelsValidated == 'error' ||
            numKernelsValidated == 'warning'
        );
    };

    return (
        <Modal
            variant={ModalVariant.medium}
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

                                if (valueAsNumber > 128 || valueAsNumber <= 0) {
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
                <Divider className="create-kernel-section-divider" />
                <FormSection
                    title={`Kernel ${currentKernelIndex + 1}`}
                    titleElement="h2"
                    className="create-kernel-properties-section"
                >
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
                                        value={kernelIds.get(currentKernelIndex)?.toLowerCase() || ''}
                                        onChange={(_event: React.FormEvent<HTMLInputElement>, value: string) => {
                                            setKernelIds(
                                                new Map(kernelIds).set(currentKernelIndex, value.toLowerCase()),
                                            );
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
                                        value={sessionIds.get(currentKernelIndex)?.toLowerCase() || ''}
                                        onChange={(_event: React.FormEvent<HTMLInputElement>, value: string) => {
                                            setSessionIds(
                                                new Map(sessionIds).set(currentKernelIndex, value.toLowerCase()),
                                            );
                                        }}
                                        maxLength={36}
                                    />
                                </TextInputGroup>
                            </FormGroup>
                        </GridItem>
                        <GridItem span={3} rowSpan={1}>
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

                                            // 128,000 millicpu is 128 vCPUs.
                                            if (valueAsNumber > 128000 || valueAsNumber <= 0) {
                                                setCpusValidated('error');
                                                return;
                                            }

                                            setCpusValidated('success');
                                        }}
                                    />
                                </TextInputGroup>
                            </FormGroup>
                        </GridItem>
                        <GridItem span={3} rowSpan={1}>
                            <FormGroup label="Memory (MB)">
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

                                            // Probably won't have a node with over 16.3TB of RAM. Can adjust later if necessary.
                                            if (valueAsNumber > 16384 || valueAsNumber <= 0) {
                                                setMemValidated('error');
                                                return;
                                            }

                                            setMemValidated('success');
                                        }}
                                    />
                                </TextInputGroup>
                            </FormGroup>
                        </GridItem>
                        <GridItem span={3} rowSpan={1}>
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
                                            setGpusHelperText('Must be a number');
                                            return;
                                        }

                                        if (valueAsNumber > 8) {
                                            setGpusValidated('warning');
                                            setGpusHelperText('Values greater than 8 may be unsupported');
                                            return;
                                        }

                                        if (valueAsNumber <= 0) {
                                            setGpusValidated('error');
                                            setGpusHelperText('Must be greater than 0');
                                            return;
                                        }

                                        setGpusValidated('success');
                                    }}
                                />
                                <FormHelperText>
                                    <HelperText>
                                        <HelperTextItem
                                            variant={gpusValidated}
                                            {...(gpusValidated === 'error' && {
                                                icon: <ExclamationCircleIcon />,
                                            })}
                                            {...(gpusValidated === 'warning' && {
                                                icon: <WarningTriangleIcon />,
                                            })}
                                        >
                                            {gpusHelperText}
                                        </HelperTextItem>
                                    </HelperText>
                                </FormHelperText>
                            </FormGroup>
                        </GridItem>
                        <GridItem span={3} rowSpan={1}>
                            <FormGroup label="VRAM (GB)">
                                <TextInput
                                    id="vram-gb-textinput"
                                    aria-label="vram-gb-textinput"
                                    placeholder={vramHintText}
                                    type={'number'}
                                    validated={vramValidated}
                                    value={
                                        (vram.get(currentKernelIndex) && vram.get(currentKernelIndex)?.toString()) || ''
                                    }
                                    onChange={(_event, value) => {
                                        setVram(new Map(vram).set(currentKernelIndex, value));

                                        if (value == '') {
                                            setVramValidated('success');
                                            return;
                                        }

                                        const valueAsNumber: number = Number.parseInt(value);

                                        if (Number.isNaN(value)) {
                                            setVramValidated('error');
                                            return;
                                        }

                                        // For now, we assume 40GB is the maximum available on a node.
                                        if (valueAsNumber > 40 || valueAsNumber <= 0) {
                                            setVramValidated('error');
                                            return;
                                        }

                                        setVramValidated('success');
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
