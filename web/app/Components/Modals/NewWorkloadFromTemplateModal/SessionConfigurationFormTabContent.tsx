import React from 'react';
import {
    Card,
    CardBody,
    Divider,
    Form,
    FormGroup,
    FormFieldGroup,
    FormFieldGroupExpandable,
    FormFieldGroupHeader,
    FormSection,
    FormHelperText,
    Grid,
    GridItem,
    HelperText,
    HelperTextItem,
    NumberInput,
    TextInput,
    Slider,
    SliderOnChangeEvent,
    FormSelectOption,
    FormSelect,
} from '@patternfly/react-core';

import { v4 as uuidv4 } from 'uuid';

import { Controller, useFieldArray, useFormContext } from 'react-hook-form';

const SessionStartTickDefault: number = 1;
const SessionStopTickDefault: number = 6;
const TrainingStartTickDefault: number = 2;
const TrainingDurationInTicksDefault: number = 2;
const TrainingCpuPercentUtilDefault: number = 10;
const TrainingGpuPercentUtilDefault: number = 50;
const TrainingMemUsageGbDefault: number = 0.25;
const NumberOfGpusDefault: number = 0;

export interface SessionConfigurationFormTabContentProps {
    children?: React.ReactNode;
    tabIndex: number;
    defaultSessionId: string;
}

// TODO: Responsive validation not quite working yet.
export const SessionConfigurationFormTabContent: React.FunctionComponent<SessionConfigurationFormTabContentProps> = (props) => {
    const tabIndex: number = props.tabIndex;
    // const defaultSessionId = React.useRef(uuidv4());
    const { control, setValue, getValues, getFieldState, watch } = useFormContext() // retrieve all hook methods
    const { fields, append, remove } = useFieldArray({ name: `sessions.${tabIndex}.gpu_utilizations`, control });

    const getGpuInputFieldId = (gpuIndex: number) => {
        return `sessions.${tabIndex}.gpu_utilizations.${gpuIndex}.utilization`;
    }

    const sessionIdFieldId: string = `sessions.${tabIndex}.id`;
    const sessionStartTickFieldId: string = `sessions.${tabIndex}.start_tick`;
    const sessionStopTickFieldId: string = `sessions.${tabIndex}.stop_tick`;
    const trainingStartTickFieldId: string = `sessions.${tabIndex}.training_start_tick`;
    const trainingDurationTicksFieldId: string = `sessions.${tabIndex}.training_duration_ticks`;
    const trainingCpuPercentUtilFieldId: string = `sessions.${tabIndex}.cpu_percent_util`;
    const trainingMemUsageGbFieldId: string = `sessions.${tabIndex}.mem_usage_gb_util`;
    const numGpusFieldId: string = `sessions.${tabIndex}.num_gpus`;
    const numTrainingEventsFieldId: string = `sessions.${tabIndex}.num_training_events`;
    const selectedTrainingEventFieldId: string = `sessions.${tabIndex}.selected_training_event`;

    const getSessionIdValidationState = () => {
        const sessionId: string = watch(sessionIdFieldId);

        if (sessionId == undefined || sessionId == null) {
            return 'default';
        }

        if (sessionId.length >= 1 && sessionId.length <= 36) {
            return 'success';
        }

        return 'error';
    }

    const isSessionIdValid = () => {
        const sessionId: string = watch(sessionIdFieldId);

        if (sessionId == undefined || sessionId == null) {
            // Form hasn't loaded yet.
            return true;
        }

        if (sessionId.length >= 1 && sessionId.length <= 36) {
            return true;
        }

        return false;
    }

    return (
        <Card isCompact isRounded isFlat>
            <CardBody>
                <Form>
                    <FormFieldGroup header={< FormFieldGroupHeader
                        titleText={{ text: `Session ${tabIndex + 1} Configuration`, id: `session-${tabIndex}-session-configuration-group` }}
                        titleDescription="Modify the session ID, number of training events, start time, and stop time."
                    />}>
                        <Grid hasGutter md={12}>
                            <GridItem span={6}>
                                <FormGroup
                                    label="Session ID"
                                    labelInfo="Required length: 1-36 characters">
                                    <Controller
                                        control={control}
                                        name={sessionIdFieldId}
                                        defaultValue={props.defaultSessionId}
                                        rules={{ minLength: 1, maxLength: 36, required: true }}
                                        render={({ field }) =>
                                            <TextInput
                                                isRequired
                                                label={`session-${tabIndex}-session-id-text-input`}
                                                aria-label={`session-${tabIndex}-session-id-text-input`}
                                                type="text"
                                                id={`session-${tabIndex}-session-id-text-input`}
                                                name={field.name}
                                                value={field.value}
                                                placeholder={props.defaultSessionId}
                                                validated={getSessionIdValidationState()}
                                                onChange={field.onChange}
                                                onBlur={field.onBlur}
                                            />}
                                    />
                                    <FormHelperText
                                        label={`session-${tabIndex}-session-id-form-helper`}
                                        aria-label={`session-${tabIndex}-session-id-form-helper`}
                                    >
                                        <HelperText
                                            label={`session-${tabIndex}-session-id-text-input-helper-text`}
                                            aria-label={`session-${tabIndex}-session-id-text-input-helper-text`}
                                        >
                                            <HelperTextItem
                                                aria-label={`session-${tabIndex}-session-id-text-input-helper-text-item`}
                                                label={`session-${tabIndex}-session-id-text-input-helper-text-item`}
                                                variant={getSessionIdValidationState()}
                                            >
                                                {isSessionIdValid() ? "" : "Session ID must be between 1 and 36 characters in length (inclusive)."}
                                            </HelperTextItem>
                                        </HelperText>
                                    </FormHelperText>
                                </FormGroup>
                            </GridItem>
                            <GridItem span={1} />
                            <GridItem span={3}>
                                <FormGroup label="Number of Training Events">
                                    <Controller
                                        control={control}
                                        name={numTrainingEventsFieldId}
                                        defaultValue={1}
                                        rules={{ min: 0 }}
                                        render={({ field }) =>
                                            <TextInput
                                                isRequired
                                                label={`session-${tabIndex}-num-training-events-text-input`}
                                                aria-label={`session-${tabIndex}-num-training-events-text-input`}
                                                id={`session-${tabIndex}-num-training-events-text-input`}
                                                name={field.name}
                                                value={field.value}
                                                onBlur={field.onBlur}
                                                onChange={field.onChange}
                                                type='number'
                                                placeholder={'0'}
                                                validated={getFieldState(numTrainingEventsFieldId).invalid ? 'error' : 'success'}
                                            />} />
                                </FormGroup>
                            </GridItem>
                            <GridItem span={3}>
                                <FormGroup label="Session Start Tick">
                                    <Controller
                                        control={control}
                                        name={sessionStartTickFieldId}
                                        defaultValue={SessionStartTickDefault}
                                        rules={{ min: 1, max: watch(sessionStopTickFieldId) as number, required: true }}
                                        render={({ field }) =>
                                            <NumberInput
                                                value={field.value}
                                                onChange={(event) => field.onChange(+(event.target as HTMLInputElement).value)}
                                                onBlur={field.onBlur}
                                                name={field.name}
                                                onMinus={() => {
                                                    const curr: number = getValues(sessionStartTickFieldId) as number;
                                                    const next: number = curr - 1;

                                                    setValue(sessionStartTickFieldId, next);
                                                }}
                                                onPlus={() => {
                                                    const curr: number = getValues(sessionStartTickFieldId) as number;
                                                    const next: number = curr + 1;

                                                    setValue(sessionStartTickFieldId, next);
                                                }}
                                                inputName={`session-${tabIndex}-session-start-tick-input`}
                                                inputAriaLabel={`session-${tabIndex}-session-start-tick-input`}
                                                minusBtnAriaLabel="minus"
                                                plusBtnAriaLabel="plus"
                                                validated={getFieldState(sessionStartTickFieldId).invalid ? 'error' : 'success'}
                                                min={1}
                                            />}
                                    />
                                </FormGroup>
                            </GridItem>
                            <GridItem span={3}>
                                <FormGroup label="Session Stop Tick">
                                    <Controller
                                        control={control}
                                        name={sessionStopTickFieldId}
                                        defaultValue={SessionStopTickDefault}
                                        rules={{ min: watch(sessionStartTickFieldId) as number, required: true }}
                                        render={({ field }) =>
                                            <NumberInput
                                                value={field.value}
                                                onChange={(event) => field.onChange(+(event.target as HTMLInputElement).value)}
                                                onBlur={field.onBlur}
                                                onMinus={() => {
                                                    const id: string = sessionStopTickFieldId;
                                                    const curr: number = getValues(id) as number;
                                                    const next: number = curr - 1;

                                                    setValue(id, next);
                                                }}
                                                onPlus={() => {
                                                    const id: string = sessionStopTickFieldId;
                                                    const curr: number = getValues(id) as number;
                                                    const next: number = curr + 1;

                                                    setValue(id, next);
                                                }}
                                                name={field.name}
                                                inputName={`session-${tabIndex}-session-stop-tick-input`}
                                                inputAriaLabel={`session-${tabIndex}-session-stop-tick-input`}
                                                minusBtnAriaLabel="minus"
                                                plusBtnAriaLabel="plus"
                                                validated={getFieldState(sessionStopTickFieldId).invalid ? 'error' : 'success'}
                                                min={0}
                                            />}
                                    />
                                </FormGroup>
                            </GridItem>
                            <GridItem span={1} />
                            <GridItem span={3}>
                                <FormGroup label="Selected Training Event">
                                    <Controller
                                        control={control}
                                        name={selectedTrainingEventFieldId}
                                        defaultValue={0}
                                        rules={{ min: 0, max: (watch(numTrainingEventsFieldId) as number) - 1 }}
                                        render={({ field }) =>
                                            <FormSelect value={field.value} onChange={field.onChange} aria-label="Training Event Selection Menu" ouiaId="TrainingEventSelectionMenu">
                                                {Array.from({ length: watch(numTrainingEventsFieldId) as number }).map((_, idx: number) => {
                                                    return (<FormSelectOption key={idx} value={idx} label={`Training Event #${idx + 1}`} />)
                                                })}
                                            </FormSelect>
                                        } />
                                </FormGroup>
                            </GridItem>
                        </Grid>
                        <FormFieldGroupExpandable isExpanded toggleAriaLabel={`session-${tabIndex}-training-event-configuration`}
                            header={<FormFieldGroupHeader
                                titleText={{ text: `Training Event #${(watch(selectedTrainingEventFieldId) as number) + 1} Configuration`, id: `session-${tabIndex}-training-resource-configuration-group` }}
                                titleDescription={`Specify the configuration for training event #${(watch(selectedTrainingEventFieldId) as number) + 1} of Session ${tabIndex+1}.`}
                            />}>
                            <Grid hasGutter>
                                <GridItem span={3}>
                                    <FormGroup label="Training Start Tick">
                                        <Controller
                                            control={control}
                                            name={trainingStartTickFieldId}
                                            defaultValue={TrainingStartTickDefault}
                                            rules={{ min: (watch(sessionStartTickFieldId) as number), max: (watch(sessionStopTickFieldId) as number) - (watch(trainingDurationTicksFieldId) as number), required: true }}
                                            render={({ field }) =>
                                                <NumberInput
                                                    value={field.value}
                                                    onChange={(event) => field.onChange(+(event.target as HTMLInputElement).value)}
                                                    onBlur={field.onBlur}
                                                    onMinus={() => {
                                                        const id: string = trainingStartTickFieldId;
                                                        const curr: number = getValues(id) as number;
                                                        const next: number = curr - 1;

                                                        setValue(id, next);
                                                    }}
                                                    onPlus={() => {
                                                        const id: string = trainingStartTickFieldId;
                                                        const curr: number = getValues(id) as number;
                                                        const next: number = curr + 1;

                                                        setValue(id, next);
                                                    }}
                                                    name={field.name}
                                                    inputName={`session-${tabIndex}-training-start-tick-input`}
                                                    inputAriaLabel={`session-${tabIndex}-training-start-tick-input`}
                                                    minusBtnAriaLabel="minus"
                                                    plusBtnAriaLabel="plus"
                                                    validated={(field.value as number < 0 || field.value as number < (watch(sessionStartTickFieldId) as number) || field.value as number > ((watch(sessionStopTickFieldId) as number) - (watch(trainingDurationTicksFieldId) as number))) ? 'error' : 'success'}
                                                    widthChars={4}
                                                    min={(watch(sessionStartTickFieldId) as number)}
                                                />}
                                        />
                                    </FormGroup>
                                </GridItem>
                                <GridItem span={3}>
                                    <FormGroup label="Training Duration (Ticks)">
                                        <Controller
                                            control={control}
                                            name={trainingDurationTicksFieldId}
                                            defaultValue={TrainingDurationInTicksDefault}
                                            rules={{ min: 1, max: (watch(sessionStopTickFieldId) as number) - (watch(trainingStartTickFieldId) as number) + 1, required: true }}
                                            render={({ field }) =>
                                                <NumberInput
                                                    value={field.value}
                                                    onChange={(event) => field.onChange(+(event.target as HTMLInputElement).value)}
                                                    onBlur={field.onBlur}
                                                    onMinus={() => {
                                                        const id: string = trainingDurationTicksFieldId;
                                                        const curr: number = getValues(id) as number;
                                                        const next: number = curr - 1;

                                                        setValue(id, next);
                                                    }}
                                                    onPlus={() => {
                                                        const id: string = trainingDurationTicksFieldId;
                                                        const curr: number = getValues(id) as number;
                                                        const next: number = curr + 1;

                                                        setValue(id, next);
                                                    }}
                                                    name={field.name}
                                                    inputProps={{ innerRef: field.ref }}
                                                    inputName={`session-${tabIndex}-training-duration-ticks-input`}
                                                    inputAriaLabel={`session-${tabIndex}-training-duration-ticks-input`}
                                                    minusBtnAriaLabel="minus"
                                                    plusBtnAriaLabel="plus"
                                                    validated={getFieldState(trainingDurationTicksFieldId).invalid ? 'error' : 'success'}
                                                    widthChars={4}
                                                    min={0}
                                                />}
                                        />
                                    </FormGroup>
                                </GridItem>
                            </Grid>
                            <FormFieldGroup
                                header={<FormFieldGroupHeader
                                    titleText={{ text: 'Training Resource Configuration', id: `session-${tabIndex}-training-resource-configuration-group` }}
                                    titleDescription="Modify the resource configuration of the training event."
                                />}>
                                <Grid hasGutter>
                                    <GridItem span={3}>
                                        <FormGroup label="CPU % Utilization">
                                            <Controller
                                                control={control}
                                                name={trainingCpuPercentUtilFieldId}
                                                defaultValue={TrainingCpuPercentUtilDefault}
                                                rules={{ min: 0, max: 100, required: true }}
                                                render={({ field }) =>
                                                    <NumberInput
                                                        required
                                                        onChange={(event) => field.onChange(+(event.target as HTMLInputElement).value)}
                                                        onBlur={field.onBlur}
                                                        value={field.value}
                                                        onMinus={() => {
                                                            const id: string = trainingCpuPercentUtilFieldId;
                                                            const curr: number = getValues(id) as number;
                                                            const next: number = curr - 1;

                                                            setValue(id, next);
                                                        }}
                                                        onPlus={() => {
                                                            const id: string = trainingCpuPercentUtilFieldId;
                                                            const curr: number = getValues(id) as number;
                                                            const next: number = curr + 1;

                                                            setValue(id, next);
                                                        }}
                                                        name={field.name}
                                                        inputName={`session-${tabIndex}-training-cpu-percent-util-input`}
                                                        inputAriaLabel={`session-${tabIndex}-training-cpu-percent-util-input`}
                                                        minusBtnAriaLabel="minus"
                                                        plusBtnAriaLabel="plus"
                                                        validated={getFieldState(trainingCpuPercentUtilFieldId).invalid ? 'error' : 'success'}
                                                        widthChars={4}
                                                        min={0}
                                                        max={100}
                                                    />} />
                                        </FormGroup>
                                    </GridItem>
                                    <GridItem span={3}>
                                        <FormGroup label="RAM Usage (GB)">
                                            <Controller
                                                control={control}
                                                name={trainingMemUsageGbFieldId}
                                                rules={{ min: 0, max: 128_000, required: true }}
                                                defaultValue={TrainingMemUsageGbDefault}
                                                render={({ field }) =>
                                                    <NumberInput
                                                        value={field.value}
                                                        onChange={(event) => field.onChange(+(event.target as HTMLInputElement).value)}
                                                        onBlur={field.onBlur}
                                                        onMinus={() => {
                                                            const id: string = trainingMemUsageGbFieldId;
                                                            const curr: number = getValues(id) as number;
                                                            const next: number = curr - 0.25;

                                                            setValue(id, next);
                                                        }}
                                                        onPlus={() => {
                                                            const id: string = trainingMemUsageGbFieldId;
                                                            const curr: number = getValues(id) as number;
                                                            const next: number = curr + 0.25;

                                                            setValue(id, next);
                                                        }}
                                                        name={field.name}
                                                        inputName={`session-${tabIndex}-training-mem-usage-gb-input`}
                                                        inputAriaLabel={`session-${tabIndex}-training-mem-usage-gb-input`}
                                                        minusBtnAriaLabel="minus"
                                                        plusBtnAriaLabel="plus"
                                                        validated={getFieldState(trainingMemUsageGbFieldId).invalid ? 'error' : 'success'}
                                                        widthChars={4}
                                                        min={0}
                                                    />}
                                            />
                                        </FormGroup>
                                    </GridItem>
                                    <GridItem span={6}>
                                        <FormGroup label={`Number of GPUs`}>
                                            <Grid hasGutter>
                                                <GridItem span={12}>
                                                    <Controller
                                                        control={control}
                                                        name={numGpusFieldId}
                                                        rules={{ min: 0, max: 8, required: true }}
                                                        defaultValue={NumberOfGpusDefault}
                                                        render={({ field }) =>
                                                            <NumberInput
                                                                value={field.value}
                                                                onChange={(event) => {
                                                                    const newNumberOfGpus: number = +(event.target as HTMLInputElement).value;

                                                                    // update field array when GPUs number changed
                                                                    const numGPUs: number = newNumberOfGpus;
                                                                    const numberOfGpuFields: number = fields.length;
                                                                    if (numGPUs > numberOfGpuFields) {
                                                                        // append GPUs to field array
                                                                        for (let i: number = numberOfGpuFields; i < numGPUs; i++) {
                                                                            append({ utilization: TrainingGpuPercentUtilDefault });
                                                                        }
                                                                    } else {
                                                                        // remove GPUs from field array
                                                                        for (let i: number = numberOfGpuFields; i > numGPUs; i--) {
                                                                            remove(i - 1);
                                                                        }
                                                                    }

                                                                    field.onChange(+(event.target as HTMLInputElement).value);
                                                                }}
                                                                onBlur={field.onBlur}
                                                                onMinus={() => {
                                                                    const curr: number = getValues(numGpusFieldId) as number;
                                                                    let next: number = curr - 1;

                                                                    if (next < 0) {
                                                                        next = 0;
                                                                    }

                                                                    setValue(numGpusFieldId, next);
                                                                    remove(fields.length - 1);
                                                                }}
                                                                onPlus={() => {
                                                                    const curr: number = getValues(numGpusFieldId) as number;
                                                                    let next: number = curr + 1;

                                                                    if (next > 8) {
                                                                        next = 8;
                                                                    }

                                                                    setValue(numGpusFieldId, next);
                                                                    append({ utilization: TrainingGpuPercentUtilDefault });
                                                                }}
                                                                name={field.name}
                                                                inputName={`session-${tabIndex}-num-gpus-input`}
                                                                key={`session-${tabIndex}-num-gpus-input`}
                                                                inputAriaLabel={`session-${tabIndex}-num-gpus-input`}
                                                                minusBtnAriaLabel="minus"
                                                                plusBtnAriaLabel="plus"
                                                                validated={getFieldState(numGpusFieldId).invalid ? 'error' : 'success'}
                                                                widthChars={4}
                                                                min={0}
                                                                max={8}
                                                            />}
                                                    />
                                                </GridItem>
                                            </Grid>
                                        </FormGroup>
                                    </GridItem>
                                    {Array.from({ length: watch(numGpusFieldId) as number }).map((_, idx: number) => {
                                        return (
                                            <GridItem key={`session-${tabIndex}-gpu-${idx}-util-input-grditem`} span={3} rowSpan={1} hidden={(getValues(numGpusFieldId) as number || 1) < idx}>
                                                <FormGroup label={`GPU #${idx} % Utilization`}>
                                                    <Controller
                                                        control={control}
                                                        name={getGpuInputFieldId(idx)}
                                                        defaultValue={TrainingGpuPercentUtilDefault}
                                                        rules={{ min: 0, max: 100, required: true }}
                                                        render={({ field }) =>
                                                            <NumberInput
                                                                value={field.value}
                                                                onChange={(event) => field.onChange(+(event.target as HTMLInputElement).value)}
                                                                onBlur={field.onBlur}
                                                                onMinus={() => {
                                                                    const id: string = getGpuInputFieldId(idx);
                                                                    const curr: number = getValues(id) as number;
                                                                    const next: number = curr - 0.25;

                                                                    setValue(id, next);
                                                                }}
                                                                onPlus={() => {
                                                                    const id: string = getGpuInputFieldId(idx);
                                                                    const curr: number = getValues(id) as number;
                                                                    const next: number = curr + 0.25;

                                                                    setValue(id, next);
                                                                }}
                                                                name={field.name}
                                                                inputName={`session-${tabIndex}-gpu${idx}-percent-util-input`}
                                                                key={`session-${tabIndex}-gpu${idx}-percent-util-input`}
                                                                inputAriaLabel={`session-${tabIndex}-gpu${idx}-percent-util-input`}
                                                                minusBtnAriaLabel="minus"
                                                                plusBtnAriaLabel="plus"
                                                                validated={getFieldState(getGpuInputFieldId(idx)).invalid ? 'error' : 'success'}
                                                                widthChars={4}
                                                                min={0}
                                                            />}
                                                    />
                                                </FormGroup>
                                            </GridItem>
                                        )
                                    })}
                                </Grid>
                            </FormFieldGroup>

                        </FormFieldGroupExpandable>
                    </FormFieldGroup>
                </Form>
            </CardBody>
        </Card >
    )
}