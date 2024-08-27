import React from 'react';
import {
    Button,
    Divider,
    Dropdown,
    DropdownItem,
    DropdownList,
    Form,
    FormGroup,
    FormSection,
    FormHelperText,
    Grid,
    GridItem,
    HelperText,
    HelperTextItem,
    MenuToggle,
    MenuToggleElement,
    Modal,
    ModalVariant,
    NumberInput,
    Popover,
    Switch,
    TextInput,
    ValidatedOptions,
    Tabs,
    Tab,
    TabTitleText,
    CardBody,
    Card,
} from '@patternfly/react-core';

import { v4 as uuidv4 } from 'uuid';
import HelpIcon from '@patternfly/react-icons/dist/esm/icons/help-icon';
import styles from '@patternfly/react-styles/css/components/Form/form';

import { ResourceRequest, Session, TrainingEvent, WorkloadTemplate } from '@app/Data';

const SessionStartTickDefault: number = 1;
const SessionStopTickDefault: number = 6;
const TrainingStartTickDefault: number = 2;
const TrainingDurationInTicksDefault: number = 2;
const TrainingCpuPercentUtilDefault: number = 10;
const TrainingMemUsageGbDefault: number = 0.25;
const TimeAdjustmentFactorDefault = 0.1;
const NumberOfGpusDefault: number = 1;

export interface SessionConfigurationFormProps {
    children?: React.ReactNode;
}

export const SessionConfigurationForm: React.FunctionComponent<SessionConfigurationFormProps> = (props) => {
    const [sessionIdIsValid, setSessionIdIsValid] = React.useState(true);
    const [sessionId, setSessionId] = React.useState('');
    const [sessionStartTick, setSessionStartTick] = React.useState<number | ''>(SessionStartTickDefault);
    const [sessionStopTick, setSessionStopTick] = React.useState<number | ''>(SessionStopTickDefault);
    const [trainingStartTick, setTrainingStartTick] = React.useState<number | ''>(TrainingStartTickDefault);
    const [trainingDurationInTicks, setTrainingDurationInTicks] = React.useState<number | ''>(TrainingDurationInTicksDefault);
    const [trainingCpuPercentUtil, setTrainingCpuPercentUtil] = React.useState<number | ''>(TrainingCpuPercentUtilDefault);
    const [trainingMemUsageGb, setTrainingMemUsageGb] = React.useState<number | ''>(TrainingMemUsageGbDefault);
    const [gpuUtilizations, setGpuUtilizations] = React.useState<(number | '')[]>([100.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0]);
    const [numberOfGPUs, setNumberOfGPUs] = React.useState<number | ''>(NumberOfGpusDefault);

    const defaultSessionId = React.useRef(uuidv4());

    const setGpuUtil = (idx: number, val: number | '') => {
        const nextGpuUtilizations = gpuUtilizations.map((v, i) => {
            if (i === idx) {
                // Update the value at the specified index.
                return val;
            } else {
                // The other values do not change.
                return v;
            }
        });
        // gpuUtilizations.current = nextGpuUtilizations;
        setGpuUtilizations(nextGpuUtilizations)
    }

    const handleSessionIdChanged = (_event, id: string) => {
        setSessionId(id);
        setSessionIdIsValid(id.length >= 0 && id.length <= 36);
    };

    const validateGpuUtilInput = (idx: number) => {
        return (gpuUtilizations[idx] !== '' && gpuUtilizations[idx] >= 0 && gpuUtilizations[idx] <= 100) ? 'success' : 'error'
    }

    const validateSessionStartTickInput = () => {
        if (sessionStartTick === '') {
            return 'error';
        }

        if (trainingStartTick === '' || sessionStopTick === '') {
            return 'warning';
        }

        if ((sessionStartTick >= 0 && sessionStartTick < trainingStartTick && sessionStartTick < sessionStopTick)) {
            return 'success';
        }

        return 'error';
    }

    const validateSessionStartStopInput = () => {
        if (sessionStopTick === '') {
            return 'error';
        }

        if (sessionStartTick === '' || trainingStartTick === '' || trainingDurationInTicks === '') {
            return 'warning';
        }

        return (sessionStopTick >= 0 && trainingStartTick < sessionStopTick && sessionStartTick < sessionStopTick && trainingStartTick + trainingDurationInTicks < sessionStopTick) ? 'success' : 'error'
    }

    const validateTrainingStartTickInput = () => {
        if (trainingStartTick === '') {
            return 'error';
        }

        if (sessionStartTick === '' || sessionStopTick === '' || trainingDurationInTicks === '') {
            return 'warning';
        }

        return (trainingStartTick >= 0 && sessionStartTick < trainingStartTick && trainingStartTick < sessionStopTick && trainingStartTick + trainingDurationInTicks < sessionStopTick) ? 'success' : 'error';
    }

    const validateTrainingDurationInTicksInput = () => {
        if (trainingDurationInTicks === '') {
            return 'error';
        }

        if (sessionStartTick === '' || sessionStopTick === '' || trainingStartTick === '') {
            return 'warning';
        }

        return (trainingDurationInTicks >= 0 && trainingStartTick + trainingDurationInTicks < sessionStopTick) ? 'success' : 'error'
    }

    const validateTrainingCpuInput = () => {
        if (trainingCpuPercentUtil === '') {
            return 'error';
        }

        return (trainingCpuPercentUtil >= 0 && trainingCpuPercentUtil <= 100) ? 'success' : 'error'
    }

    const validateTrainingMemoryUsageInput = () => {
        if (trainingMemUsageGb === '') {
            return 'error';
        }

        return (trainingMemUsageGb >= 0 && trainingMemUsageGb <= 128_000) ? 'success' : 'error';
    }

    const validateNumberOfGpusInput = () => {
        if (numberOfGPUs === '') {
            return 'error';
        }

        return (numberOfGPUs !== undefined && numberOfGPUs >= 0 && numberOfGPUs <= 8 && numberOfGPUs >= 0 && numberOfGPUs <= 8) ? 'success' : 'warning';
    }

    function assertGpuUtilizationsAreAllNumbers(value: (number | '')[], numGPUs: number): asserts value is number[] {
        for (let i = 0; i < numGPUs; i++) {
            if (validateGpuUtilInput[i] === 'error') {
                console.error(`gpuUtilization[${i}] is not a valid value during submission.`)
                throw new Error(`gpuUtilization[${i}] is not a valid value during submission.`)
            }
        }
    }

    return (
        <React.Fragment>
            <FormSection title={`General Session Parameters`} titleElement='h1'>
                <Form>
                    <Grid hasGutter md={12}>
                        <GridItem span={12}>
                            <FormGroup
                                label="Session ID:">
                                <TextInput
                                    isRequired
                                    label="session-id-text-input"
                                    aria-label="session-id-text-input"
                                    type="text"
                                    id="session-id-text-input"
                                    name="session-id-text-input"
                                    aria-describedby="session-id-text-input-helper"
                                    value={sessionId}
                                    placeholder={defaultSessionId.current}
                                    validated={(sessionIdIsValid && ValidatedOptions.success) || ValidatedOptions.error}
                                    onChange={handleSessionIdChanged}
                                />
                                <FormHelperText
                                    label="session-id-text-input-helper"
                                    aria-label="session-id-text-input-helper"
                                >
                                    <HelperText
                                        label="session-id-text-input-helper"
                                        aria-label="session-id-text-input-helper"
                                    >
                                        <HelperTextItem
                                            aria-label="session-id-text-input-helper"
                                            label="session-id-text-input-helper"
                                        >
                                            Provide an ID for the session. The session ID must be between 1 and 36 characters (inclusive).
                                        </HelperTextItem>
                                    </HelperText>
                                </FormHelperText>
                            </FormGroup>
                        </GridItem>
                        <GridItem span={3}>
                            <FormGroup label="Session Start Tick">
                                <NumberInput
                                    value={sessionStartTick}
                                    onMinus={() => setSessionStartTick((sessionStartTick || 0) - 1)}
                                    onChange={(event: React.FormEvent<HTMLInputElement>) => {
                                        const value = (event.target as HTMLInputElement).value;
                                        setSessionStartTick(value === '' ? value : +value);
                                    }}
                                    onPlus={() => setSessionStartTick((sessionStartTick || 0) + 1)}
                                    inputName="session-start-tick-input"
                                    inputAriaLabel="session-start-tick-input"
                                    minusBtnAriaLabel="minus"
                                    plusBtnAriaLabel="plus"
                                    validated={validateSessionStartTickInput()}
                                    widthChars={4}
                                    min={0}
                                />
                            </FormGroup>
                        </GridItem>
                        <GridItem span={3}>
                            <FormGroup label="Training Start Tick">
                                <NumberInput
                                    value={trainingStartTick}
                                    onMinus={() => setTrainingStartTick((trainingStartTick || 0) - 1)}
                                    onChange={(event: React.FormEvent<HTMLInputElement>) => {
                                        const value = (event.target as HTMLInputElement).value;
                                        setTrainingStartTick(value === '' ? value : +value);
                                    }}
                                    onPlus={() => setTrainingStartTick((trainingStartTick || 0) + 1)}
                                    inputName="training-start-tick-input"
                                    inputAriaLabel="training-start-tick-input"
                                    minusBtnAriaLabel="minus"
                                    plusBtnAriaLabel="plus"
                                    validated={validateTrainingStartTickInput()}
                                    widthChars={4}
                                    min={0}
                                />
                            </FormGroup>
                        </GridItem>
                        <GridItem span={3}>
                            <FormGroup label="Training Duration (Ticks)">
                                <NumberInput
                                    value={trainingDurationInTicks}
                                    onMinus={() => setTrainingDurationInTicks((trainingDurationInTicks || 0) - 1)}
                                    onChange={(event: React.FormEvent<HTMLInputElement>) => {
                                        const value = (event.target as HTMLInputElement).value;
                                        setTrainingDurationInTicks(value === '' ? value : +value);
                                    }}
                                    onPlus={() => setTrainingDurationInTicks((trainingDurationInTicks || 0) + 1)}
                                    inputName="training-duration-ticks-input"
                                    inputAriaLabel="training-duration-ticks-input"
                                    minusBtnAriaLabel="minus"
                                    plusBtnAriaLabel="plus"
                                    validated={validateTrainingDurationInTicksInput()}
                                    widthChars={4}
                                    min={1}
                                />
                            </FormGroup>
                        </GridItem>
                        <GridItem span={3}>
                            <FormGroup label="Session Stop Tick">
                                <NumberInput
                                    value={sessionStopTick}
                                    onMinus={() => setSessionStopTick((sessionStopTick || 0) - 1)}
                                    onChange={(event: React.FormEvent<HTMLInputElement>) => {
                                        const value = (event.target as HTMLInputElement).value;
                                        setSessionStopTick(value === '' ? value : +value);
                                    }}
                                    onPlus={() => setSessionStopTick((sessionStopTick || 0) + 1)}
                                    inputName="session-stop-tick-input"
                                    inputAriaLabel="session-stop-tick-input"
                                    minusBtnAriaLabel="minus"
                                    plusBtnAriaLabel="plus"
                                    validated={validateSessionStartStopInput()}
                                    widthChars={4}
                                    min={0}
                                />
                            </FormGroup>
                        </GridItem>
                    </Grid>
                    <Divider />
                </Form>
            </FormSection>
            <FormSection title={`Configure Session Resource Utilization`} titleElement='h1'>
                <Form>
                    <Grid hasGutter>
                        <GridItem span={3}>
                            <FormGroup label="CPU % Utilization">
                                <NumberInput
                                    value={trainingCpuPercentUtil}
                                    onMinus={() => setTrainingCpuPercentUtil((trainingCpuPercentUtil || 0) - 1)}
                                    onChange={(event: React.FormEvent<HTMLInputElement>) => {
                                        const value = (event.target as HTMLInputElement).value;
                                        setTrainingCpuPercentUtil(value === '' ? value : +value);
                                    }}
                                    onPlus={() => setTrainingCpuPercentUtil((trainingCpuPercentUtil || 0) + 1)}
                                    inputName="training-cpu-percent-util-input"
                                    inputAriaLabel="training-cpu-percent-util-input"
                                    minusBtnAriaLabel="minus"
                                    plusBtnAriaLabel="plus"
                                    validated={validateTrainingCpuInput()}
                                    widthChars={4}
                                    min={0}
                                    max={100}
                                />
                            </FormGroup>
                        </GridItem>
                        <GridItem span={3}>
                            <FormGroup label="RAM Usage (GB)">
                                <NumberInput
                                    value={trainingMemUsageGb}
                                    onMinus={() => setTrainingMemUsageGb((trainingMemUsageGb || 0) - 0.25)}
                                    onChange={(event: React.FormEvent<HTMLInputElement>) => {
                                        const value = (event.target as HTMLInputElement).value;
                                        setTrainingMemUsageGb(value === '' ? value : +value);
                                    }}
                                    onPlus={() => setTrainingMemUsageGb((trainingMemUsageGb || 0) + 0.25)}
                                    inputName="training-mem-usage-gb-input"
                                    inputAriaLabel="training-mem-usage-gb-input"
                                    minusBtnAriaLabel="minus"
                                    plusBtnAriaLabel="plus"
                                    validated={validateTrainingMemoryUsageInput()}
                                    widthChars={4}
                                    min={0}
                                    max={128_000}
                                />
                            </FormGroup>
                        </GridItem>
                        <GridItem span={6}>
                            <FormGroup label={`Number of GPUs`}>
                                <Grid hasGutter>
                                    <GridItem span={12}>
                                        <NumberInput
                                            value={numberOfGPUs}
                                            onMinus={() => setNumberOfGPUs((numberOfGPUs || 0) - 1)}
                                            onChange={(event: React.FormEvent<HTMLInputElement>) => {
                                                const value = (event.target as HTMLInputElement).value;
                                                setNumberOfGPUs(value === '' ? value : +value);
                                            }}
                                            onPlus={() => setNumberOfGPUs((numberOfGPUs || 0) + 1)}
                                            inputName="num-gpus-input"
                                            key="num-gpus-input"
                                            inputAriaLabel="num-gpus-input"
                                            minusBtnAriaLabel="minus"
                                            plusBtnAriaLabel="plus"
                                            validated={validateNumberOfGpusInput()}
                                            widthChars={1}
                                            min={1}
                                            max={8}
                                        />
                                    </GridItem>
                                </Grid>
                            </FormGroup>
                        </GridItem>
                        {Array.from({ length: Math.max(Math.min((numberOfGPUs || 1), 8), 1) }).map((_, idx: number) => {
                            return (
                                <GridItem key={`gpu-${idx}-util-input-grditem`} span={3} rowSpan={1} hidden={(numberOfGPUs || 1) < idx}>
                                    <FormGroup label={`GPU #${idx} % Utilization`}>
                                        <NumberInput
                                            value={gpuUtilizations[idx]}
                                            onMinus={() => setGpuUtil(idx, (gpuUtilizations[idx] || 0) - 1)}
                                            onChange={(event: React.FormEvent<HTMLInputElement>) => {
                                                const value = (event.target as HTMLInputElement).value;

                                                setGpuUtil(idx, value === '' ? value : +value);
                                            }}
                                            onPlus={() => setGpuUtil(idx, (gpuUtilizations[idx] || 0) + 1)}
                                            inputName={`gpu${idx}-percent-util-input`}
                                            inputAriaLabel={`gpu${idx}-percent-util-input`}
                                            minusBtnAriaLabel="minus"
                                            plusBtnAriaLabel="plus"
                                            validated={validateGpuUtilInput(idx)}
                                            min={0}
                                            max={100}
                                        />
                                    </FormGroup>
                                </GridItem>
                            )
                        })}
                    </Grid>
                </Form>
            </FormSection>
        </React.Fragment>
    )
}