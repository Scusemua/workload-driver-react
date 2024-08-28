import React from 'react';
import {
    Button,
    Divider,
    Form,
    FormGroup,
    FormSection,
    FormHelperText,
    Grid,
    GridItem,
    HelperText,
    HelperTextItem,
    Modal,
    ModalVariant,
    NumberInput,
    Popover,
    Switch,
    TextInput,
} from '@patternfly/react-core';

import { v4 as uuidv4 } from 'uuid';
import HelpIcon from '@patternfly/react-icons/dist/esm/icons/help-icon';
import styles from '@patternfly/react-styles/css/components/Form/form';

import { useForm, FormProvider, Controller } from "react-hook-form"

import { ResourceRequest, Session, TrainingEvent, WorkloadTemplate } from '@app/Data';
import { SessionConfigurationForm } from './SessionConfigurationForm';
import { DefaultSessionFieldValue, TimeAdjustmentFactorDefault, TimescaleAdjustmentFactorDelta, TimescaleAdjustmentFactorMax, TimescaleAdjustmentFactorMin, WorkloadSeedDefault, WorkloadSeedDelta, WorkloadSeedMax, WorkloadSeedMin } from './Constants';

export interface NewWorkloadFromTemplateModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    onConfirm: (
        workloadTitle: string,
        workloadSeed: string,
        debugLoggingEnabled: boolean,
        workloadTemplate: WorkloadTemplate,
        timescaleAdjustmentFactor: number,
    ) => void;
}

// Clamp a value between two extremes.
function clamp(value: number, min: number, max: number) {
    return Math.max(Math.min(value, max), min)
}

function roundToThreeDecimalPlaces(num: number) {
    return +(Math.round(Number.parseFloat(num.toString() + 'e+3')).toString() + 'e-3');
}

function assertIsNumber(value: number | ''): asserts value is number {
    if (value === '') {
        throw new Error("value is not number");
    }
}

function assertAreNumbers(values: number[] | ''): asserts values is number[] {
    if (values === '') {
        throw new Error("value is not number");
    }
}

// TODO: Responsive validation not quite working yet.
// TODO: Re-implement onSubmit.
export const NewWorkloadFromTemplateModal: React.FunctionComponent<NewWorkloadFromTemplateModalProps> = (props) => {
    const defaultWorkloadTitle = React.useRef(uuidv4());
    const form = useForm({
        mode: 'all',
        defaultValues: {
            "workloadTitle": defaultWorkloadTitle.current,
            "workloadSeed": WorkloadSeedDefault,
            "timescaleAdjustmentFactor": TimeAdjustmentFactorDefault,
            "debugLoggingEnabled": true,
            "sessions": [
                DefaultSessionFieldValue
            ]
        }
    });

    const onSubmit = (data) => {
        const workloadTitle: string = data.workloadTitle;
        const workloadSeed: string = data.workloadSeed;
        const debugLoggingEnabled: boolean = data.debugLoggingEnabled;
        const timescaleAdjustmentFactor: number = data.timescaleAdjustmentFactor;

        const sessions: Session[] = data.sessions;

        for (let i: number = 0; i < sessions.length; i++) {
            const session: Session = sessions[i];
            const trainings: TrainingEvent[] = session.trainings;

            let max_cpu: number = -1;
            let max_mem: number = -1;
            let max_num_gpus: number = -1;
            for (let j: number = 0; j < trainings.length; j++) {
                const training: TrainingEvent = trainings[j];
                training.training_index = j; // Set the training index field.

                if (training.cpu_util > max_cpu) {
                    max_cpu = training.cpu_util;
                }

                if (training.mem_usage_gb > max_mem) {
                    max_mem = training.mem_usage_gb;
                }

                if (training.gpu_utilizations.length > max_num_gpus) {
                    max_num_gpus = training.gpu_utilizations.length;
                }
            }

            // Construct the resource request.
            const resource_request: ResourceRequest = {
                cpus: max_cpu,
                gpus: max_num_gpus,
                mem_gb: max_mem,
                gpu_type: "Any_GPU"
            }

            // Update the session object.
            session.resource_request = resource_request;
        }

        const workloadTemplate: WorkloadTemplate = {
            sessions: data.sessions
        }
        
        console.log(`User submitted workload template data: ${JSON.stringify(data)}`);
        props.onConfirm(workloadTitle, workloadSeed, debugLoggingEnabled, workloadTemplate, timescaleAdjustmentFactor);
        form.reset();
    };

    const getWorkloadNameValidationState = () => {
        const workloadId: string = form.watch("workloadTitle"); 

        if (workloadId == undefined || workloadId == null) {
            return 'default';
        }

        if (workloadId.length >= 1 && workloadId.length <= 36) {
            return 'success';
        }

        return 'error';
    }

    const isWorkloadNameValid = () => {
        const workloadId: string = form.watch("workloadTitle"); 

        if (workloadId == undefined || workloadId == null) {
            // Form hasn't loaded yet.
            return true;
        }

        if (workloadId.length >= 1 && workloadId.length <= 36) {
            return true;
        }

        return false;
    }

    return (
        <FormProvider {...form}>
            <Modal
                variant={ModalVariant.large}
                titleIconVariant={'info'}
                aria-label="Modal to create a new workload from a template"
                title={'Create New Workload from Template'}
                isOpen={props.isOpen}
                onClose={props.onClose}
                help={
                    <Popover
                        headerContent={<div>Creating New Workloads from Templates</div>}
                        bodyContent={
                            <div>
                                You can create and register a new workload using a "template". This allows for a creater degree of dynamicity in the workload's execution.
                                <br />
                                <br />
                                Specifically, templates enable you to customize various properties of the workload, such as the number of sessions, the resource utilization of these sessions,
                                when the sessions start and stop, and the training events processed by the workload's sessions.
                            </div>
                        }
                    >
                        <Button variant="plain" aria-label="Create New Workload From Template Helper">
                            <HelpIcon />
                        </Button>
                    </Popover>
                }
                actions={[
                    <Button key="submit-workload-from-template-button" variant="primary" onClick={form.handleSubmit(onSubmit)}>
                        Submit
                    </Button>,
                    <Button key="cancel-submission-of-workload-from-template-button" variant="link" onClick={props.onClose}>
                        Cancel
                    </Button>,
                ]}
            >
                <FormSection title="Generic Workload Parameters" titleElement='h1'>
                    <Form onSubmit={
                        () => {
                            form.clearErrors()
                            form.handleSubmit(onSubmit)
                        }
                    }>
                        <Grid hasGutter md={12}>
                            <GridItem span={12}>
                                <FormGroup
                                    label="Workload name:"
                                    labelInfo="Required length: 1-36 characters"
                                    labelIcon={
                                        <Popover
                                            aria-label="workload-title-popover"
                                            headerContent={<div>Workload Title</div>}
                                            bodyContent={
                                                <div>
                                                    This is an identifier (that is not necessarily unique, but probably should
                                                    be) to help you identify the specific workload. Please note that the title
                                                    must be between 1 and 36 characters in length.
                                                </div>
                                            }
                                        >
                                            <button
                                                type="button"
                                                aria-label="This is an identifier (that is not necessarily unique, but probably should be) to help you identify the specific workload."
                                                onClick={(e) => e.preventDefault()}
                                                aria-describedby="simple-form-workload-name-01"
                                                className={styles.formGroupLabelHelp}
                                            >
                                                <HelpIcon />
                                            </button>
                                        </Popover>
                                    }
                                >
                                    <Controller
                                        name="workloadTitle"
                                        control={form.control}
                                        rules={{ minLength: 1, maxLength: 36, required: true }}
                                        defaultValue={defaultWorkloadTitle.current}
                                        render={({ field }) => <TextInput
                                            isRequired
                                            onChange={field.onChange}
                                            onBlur={field.onBlur}
                                            value={field.value}
                                            label="workload-title-text-input"
                                            aria-label="workload-title-text-input"
                                            type="text"
                                            id="workload-title-text-input"
                                            aria-describedby="workload-title-text-input-helper"
                                            placeholder={defaultWorkloadTitle.current}
                                            validated={getWorkloadNameValidationState()}
                                        />} />
                                    <FormHelperText
                                        label="workload-title-text-input-helper"
                                        aria-label="workload-title-text-input-helper"
                                    >
                                        <HelperText
                                            label="workload-title-text-input-helper"
                                            aria-label="workload-title-text-input-helper"
                                        >
                                            <HelperTextItem
                                                aria-label="workload-title-text-input-helper"
                                                label="workload-title-text-input-helper"
                                                variant={getWorkloadNameValidationState()}
                                            >
                                                {isWorkloadNameValid() ? "" : "Session ID must be between 1 and 36 characters in length (inclusive)."}
                                            </HelperTextItem>
                                        </HelperText>
                                    </FormHelperText>
                                </FormGroup>
                            </GridItem>
                            <GridItem span={4}>
                                <FormGroup
                                    label="Workload Seed:"
                                    labelIcon={
                                        <Popover
                                            aria-label="workload-seed-popover"
                                            headerContent={<div>Workload Title</div>}
                                            bodyContent={
                                                <div>
                                                    This is an integer seed for the random number generator used by the workload
                                                    generator. Pass a value of 0 to refrain from seeding the random generator.
                                                    Please note that if you do specify a seed, then the value must be between 0 and 2,147,483,647.
                                                </div>
                                            }
                                        >
                                            <button
                                                type="button"
                                                aria-label="This is an integer seed (between 0 and 2,147,483,647) for the random number generator used by the workload generator. Pass a value of 0 to refrain from seeding the random generator."
                                                onClick={(e) => e.preventDefault()}
                                                aria-describedby="simple-form-workload-seed-01"
                                                className={styles.formGroupLabelHelp}
                                            >
                                                <HelpIcon />
                                            </button>
                                        </Popover>
                                    }
                                >
                                    <Controller
                                        name="workloadSeed"
                                        control={form.control}
                                        defaultValue={WorkloadSeedDefault}
                                        rules={{ max: WorkloadSeedMax, min: WorkloadSeedMin }}
                                        render={({ field }) => <NumberInput
                                            inputName='workload-seed-number-input'
                                            id="workload-seed-number-input"
                                            type="number"
                                            min={WorkloadSeedMin}
                                            max={WorkloadSeedMax}
                                            onBlur={field.onBlur}
                                            onChange={field.onChange}
                                            name={field.name}
                                            value={field.value}
                                            widthChars={10}
                                            aria-label="Text input for the 'timescale adjustment factor'"
                                            onPlus={() => {
                                                const curr: number = form.getValues("workloadSeed") || 0;
                                                let next: number = curr + WorkloadSeedDelta;
                                                next = clamp(next, WorkloadSeedMin, WorkloadSeedMax);
                                                form.setValue("workloadSeed", next);
                                            }}
                                            onMinus={() => {
                                                const curr: number = form.getValues("workloadSeed") || 0;
                                                let next: number = curr - WorkloadSeedDelta;
                                                next = clamp(next, WorkloadSeedMin, WorkloadSeedMax);
                                                form.setValue("workloadSeed", next);
                                            }}
                                        />}
                                    />
                                </FormGroup>
                            </GridItem>
                            <GridItem span={4}>
                                <FormGroup
                                    label={'Timescale Adjustment Factor'}
                                    labelIcon={
                                        <Popover
                                            aria-label="timescale-adjustment-factor-header"
                                            headerContent={<div>Timescale Adjustment Factor</div>}
                                            bodyContent={
                                                <div>
                                                    This quantity adjusts the timescale at which the trace data is replayed.
                                                    For example, if each tick is 60 seconds, then setting this value to 1.0 will instruct
                                                    the Workload Driver to simulate each tick for the full 60 seconds.
                                                    Alternatively, setting this quantity to 2.0 will instruct the Workload Driver to spend 120 seconds on each tick.
                                                    Setting the quantity to 0.5 will instruct the Workload Driver to spend 30 seconds on each tick.
                                                </div>
                                            }
                                        >
                                            <button
                                                type="button"
                                                aria-label="Set the Timescale Adjustment Factor."
                                                onClick={(e) => e.preventDefault()}
                                                className={styles.formGroupLabelHelp}
                                            >
                                                <HelpIcon />
                                            </button>
                                        </Popover>
                                    }
                                >
                                    <Controller
                                        name="timescaleAdjustmentFactor"
                                        control={form.control}
                                        defaultValue={TimeAdjustmentFactorDefault}
                                        rules={{ max: TimescaleAdjustmentFactorMax, min: TimescaleAdjustmentFactorMin }}
                                        render={({ field }) => <NumberInput
                                            inputName='timescale-adjustment-factor-number-input'
                                            id="timescale-adjustment-factor-number-input"
                                            type="number"
                                            aria-label="Text input for the 'timescale adjustment factor'"
                                            onBlur={field.onBlur}
                                            onChange={field.onChange}
                                            name={field.name}
                                            value={field.value}
                                            min={TimescaleAdjustmentFactorMin}
                                            max={TimescaleAdjustmentFactorMax}
                                            onPlus={() => {
                                                const curr: number = form.getValues("timescaleAdjustmentFactor") as number;
                                                let next: number = curr + TimescaleAdjustmentFactorDelta;

                                                next = roundToThreeDecimalPlaces(next);

                                                form.setValue("timescaleAdjustmentFactor", clamp(next, TimescaleAdjustmentFactorMin, TimescaleAdjustmentFactorMax));
                                            }}
                                            onMinus={() => {
                                                const curr: number = form.getValues("timescaleAdjustmentFactor") as number;
                                                let next: number = curr - TimescaleAdjustmentFactorDelta;

                                                // For the timescale adjustment factor, we don't want to decrement it to 0.
                                                if (next == 0) {
                                                    next = TimescaleAdjustmentFactorMin;
                                                }

                                                next = roundToThreeDecimalPlaces(next);

                                                form.setValue("timescaleAdjustmentFactor", clamp(next, TimescaleAdjustmentFactorMin, TimescaleAdjustmentFactorMax));
                                            }}
                                        />}
                                    />
                                </FormGroup>
                            </GridItem>
                            <GridItem span={4}>
                                <FormGroup
                                    label={'Verbose Server-Side Log Output'}
                                    labelIcon={
                                        <Popover
                                            aria-label="workload-debug-logging-header"
                                            headerContent={<div>Verbose Server-Side Log Output</div>}
                                            bodyContent={
                                                <div>
                                                    Enable or disable server-side debug (i.e., verbose) log output from the
                                                    workload generator and workload driver.
                                                </div>
                                            }
                                        >
                                            <button
                                                type="button"
                                                aria-label="Select the preprocessed data to use for driving the workload. This largely determines which subset of trace data will be used to generate the workload."
                                                onClick={(e) => e.preventDefault()}
                                                className={styles.formGroupLabelHelp}
                                            >
                                                <HelpIcon />
                                            </button>
                                        </Popover>
                                    }
                                >
                                    <Controller
                                        name="debugLoggingEnabled"
                                        control={form.control}
                                        defaultValue={true}
                                        render={({ field }) => <Switch
                                            id="debug-logging-switch-template"
                                            label="Debug logging enabled"
                                            labelOff="Debug logging disabled"
                                            aria-label="debug-logging-switch-template"
                                            isChecked={field.value === true}
                                            ouiaId="DebugLoggingSwitchTemplate"
                                            onChange={(_event: React.FormEvent<HTMLInputElement>, checked: boolean) => {
                                                form.setValue("debugLoggingEnabled", checked);
                                            }}
                                        />
                                        }
                                    />
                                </FormGroup>
                            </GridItem>
                        </Grid>
                        <Divider />
                    </Form>
                </FormSection>
                <SessionConfigurationForm/>
            </Modal >
        </FormProvider>
    );
};
