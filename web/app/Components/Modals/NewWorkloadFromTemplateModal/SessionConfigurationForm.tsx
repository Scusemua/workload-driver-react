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
import { Controller, useFormContext } from 'react-hook-form';

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
    sessionIdentifier: string; // This is not necessarily the session ID field.
}

export const SessionConfigurationForm: React.FunctionComponent<SessionConfigurationFormProps> = (props) => {
    const defaultSessionId = React.useRef(uuidv4());
    const { register, control, setValue, getValues, getFieldState } = useFormContext() // retrieve all hook methods

    const trainingStartTickFieldId: string = `${props.sessionIdentifier}-training-start-tick`;
    const sessionIdFieldId: string = `${props.sessionIdentifier}-session-id`;
    const sessionStartTickFieldId: string = `${props.sessionIdentifier}-session-start-tick`;
    const sessionStopTickFieldId: string = `${props.sessionIdentifier}-session-end-tick`;
    const trainingDurationTicksFieldId: string = `${props.sessionIdentifier}-training-duration-ticks`;
    const trainingCpuPercentUtilFieldId: string = `${props.sessionIdentifier}-cpu-percent-util`;
    const trainingMemUsageGbFieldId: string = `${props.sessionIdentifier}-mem-usage-gb-util`;
    const numGpusFieldId: string = `${props.sessionIdentifier}-num-gpus`;

    const getGpuInputFieldId = (idx: number) => {
        return `${props.sessionIdentifier}-gpu-${idx}-training-util-percent`;
    }

    return (
        <React.Fragment>
            <FormSection title={`General Session Parameters`} titleElement='h1'>
                <Form>
                    <Grid hasGutter md={12}>
                        <GridItem span={12}>
                            <FormGroup
                                label="Session ID:">
                                <Controller
                                    name={sessionIdFieldId}
                                    defaultValue={defaultSessionId.current}
                                    rules={{ minLength: 1, maxLength: 36 }}
                                    render={({ field }) =>
                                        <TextInput
                                            isRequired
                                            label={`${props.sessionIdentifier}-session-id-text-input`}
                                            aria-label={`${props.sessionIdentifier}-session-id-text-input`}
                                            type="text"
                                            id={`${props.sessionIdentifier}-session-id-text-input`}
                                            name={field.name}
                                            value={field.value}
                                            placeholder={defaultSessionId.current}
                                            validated={(getValues(sessionIdFieldId).length >= 0 && getValues(sessionIdFieldId).length <= 0) ? ValidatedOptions.success : ValidatedOptions.error}
                                            onChange={field.onChange}
                                            onBlur={field.onBlur}
                                        />}
                                />
                                <FormHelperText
                                    label={`${props.sessionIdentifier}-session-id-form-helper`}
                                    aria-label={`${props.sessionIdentifier}-session-id-form-helper`}
                                >
                                    <HelperText
                                        label={`${props.sessionIdentifier}-session-id-text-input-helper-text`}
                                        aria-label={`${props.sessionIdentifier}-session-id-text-input-helper-text`}
                                    >
                                        <HelperTextItem
                                            aria-label={`${props.sessionIdentifier}-session-id-text-input-helper-text-item`}
                                            label={`${props.sessionIdentifier}-session-id-text-input-helper-text-item`}
                                        >
                                            Provide an ID for the session. The session ID must be between 1 and 36 characters (inclusive).
                                        </HelperTextItem>
                                    </HelperText>
                                </FormHelperText>
                            </FormGroup>
                        </GridItem>
                        <GridItem span={3}>
                            <FormGroup label="Session Start Tick">
                                <Controller
                                    name={sessionStartTickFieldId}
                                    rules={{ min: 0, max: getValues(sessionStopTickFieldId) }}
                                    render={({ field }) =>
                                        <NumberInput
                                            value={field.value}
                                            onChange={field.onChange}
                                            onBlur={field.onBlur}
                                            name={field.name}
                                            onMinus={() => {
                                                const id: string = sessionStartTickFieldId;
                                                const curr: number = getValues(id);
                                                const next: number = curr - 1;

                                                setValue(id, next);
                                            }}
                                            onPlus={() => {
                                                const id: string = sessionStartTickFieldId;
                                                const curr: number = getValues(id);
                                                const next: number = curr + 1;

                                                setValue(id, next);
                                            }}
                                            inputName={`${props.sessionIdentifier}-session-start-tick-input`}
                                            inputAriaLabel={`${props.sessionIdentifier}-session-start-tick-input`}
                                            minusBtnAriaLabel="minus"
                                            plusBtnAriaLabel="plus"
                                            validated={getFieldState(sessionStartTickFieldId).invalid ? 'error' : 'success'}
                                            widthChars={4}
                                            min={0}
                                        />}
                                />
                            </FormGroup>
                        </GridItem>
                        <GridItem span={3}>
                            <FormGroup label="Training Start Tick">
                                <Controller
                                    name={trainingStartTickFieldId}
                                    rules={{ min: getValues(sessionStartTickFieldId) + 1, max: getValues(sessionStopTickFieldId) - getValues(trainingDurationTicksFieldId) }}
                                    render={({ field }) =>
                                        <NumberInput
                                            value={field.value}
                                            onChange={field.onChange}
                                            onBlur={field.onBlur}
                                            onMinus={() => {
                                                const id: string = trainingStartTickFieldId;
                                                const curr: number = getValues(id);
                                                const next: number = curr - 1;

                                                setValue(id, next);
                                            }}
                                            onPlus={() => {
                                                const id: string = trainingStartTickFieldId;
                                                const curr: number = getValues(id);
                                                const next: number = curr + 1;

                                                setValue(id, next);
                                            }}
                                            name={field.name}
                                            inputName={`${props.sessionIdentifier}-training-start-tick-input`}
                                            inputAriaLabel={`${props.sessionIdentifier}-training-start-tick-input`}
                                            minusBtnAriaLabel="minus"
                                            plusBtnAriaLabel="plus"
                                            validated={getFieldState(trainingStartTickFieldId).invalid ? 'error' : 'success'}
                                            widthChars={4}
                                            min={0}
                                        />}
                                />
                            </FormGroup>
                        </GridItem>
                        <GridItem span={3}>
                            <FormGroup label="Training Duration (Ticks)">
                                <Controller
                                    name={trainingDurationTicksFieldId}
                                    rules={{ min: 1, max: getValues(sessionStopTickFieldId) - getValues(trainingStartTickFieldId) + 1 }}
                                    render={({ field }) =>
                                        <NumberInput
                                            value={field.value}
                                            onChange={field.onChange}
                                            onBlur={field.onBlur}
                                            onMinus={() => {
                                                const id: string = trainingDurationTicksFieldId;
                                                const curr: number = getValues(id);
                                                const next: number = curr - 1;

                                                setValue(id, next);
                                            }}
                                            onPlus={() => {
                                                const id: string = trainingDurationTicksFieldId;
                                                const curr: number = getValues(id);
                                                const next: number = curr + 1;

                                                setValue(id, next);
                                            }}
                                            name={field.name}
                                            inputName={`${props.sessionIdentifier}-training-duration-ticks-input`}
                                            inputAriaLabel={`${props.sessionIdentifier}-training-duration-ticks-input`}
                                            minusBtnAriaLabel="minus"
                                            plusBtnAriaLabel="plus"
                                            validated={getFieldState(trainingDurationTicksFieldId).invalid ? 'error' : 'success'}
                                            widthChars={4}
                                            min={0}
                                        />}
                                />
                            </FormGroup>
                        </GridItem>
                        <GridItem span={3}>
                            <FormGroup label="Session Stop Tick">
                                <Controller
                                    name={sessionStopTickFieldId}
                                    rules={{ min: getValues(sessionStartTickFieldId) }}
                                    render={({ field }) =>
                                        <NumberInput
                                            value={field.value}
                                            onChange={field.onChange}
                                            onBlur={field.onBlur}
                                            onMinus={() => {
                                                const id: string = sessionStopTickFieldId;
                                                const curr: number = getValues(id);
                                                const next: number = curr - 1;

                                                setValue(id, next);
                                            }}
                                            onPlus={() => {
                                                const id: string = sessionStopTickFieldId;
                                                const curr: number = getValues(id);
                                                const next: number = curr + 1;

                                                setValue(id, next);
                                            }}
                                            name={field.name}
                                            inputName={`${props.sessionIdentifier}-session-stop-tick-input`}
                                            inputAriaLabel={`${props.sessionIdentifier}-session-stop-tick-input`}
                                            minusBtnAriaLabel="minus"
                                            plusBtnAriaLabel="plus"
                                            validated={getFieldState(sessionStopTickFieldId).invalid ? 'error' : 'success'}
                                            widthChars={4}
                                            min={0}
                                        />}
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
                                <Controller
                                    name={trainingCpuPercentUtilFieldId}
                                    defaultValue={TrainingCpuPercentUtilDefault}
                                    rules={{ min: 0, max: 100 }}
                                    render={({ field }) =>
                                        <NumberInput
                                            required
                                            onChange={field.onChange}
                                            onBlur={field.onBlur}
                                            value={field.value}
                                            onMinus={() => {
                                                const id: string = trainingCpuPercentUtilFieldId;
                                                const curr: number = getValues(id);
                                                const next: number = curr - 1;

                                                setValue(id, next);
                                            }}
                                            onPlus={() => {
                                                const id: string = trainingCpuPercentUtilFieldId;
                                                const curr: number = getValues(id);
                                                const next: number = curr + 1;

                                                setValue(id, next);
                                            }}
                                            name={field.name}
                                            inputName={`${props.sessionIdentifier}-training-cpu-percent-util-input`}
                                            inputAriaLabel={`${props.sessionIdentifier}-training-cpu-percent-util-input`}
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
                                    name={trainingMemUsageGbFieldId}
                                    rules={{ min: 0, max: 128_000 }}
                                    render={({ field }) =>
                                        <NumberInput
                                            value={field.value}
                                            onChange={field.onChange}
                                            onBlur={field.onBlur}
                                            onMinus={() => {
                                                const id: string = trainingMemUsageGbFieldId;
                                                const curr: number = getValues(id);
                                                const next: number = curr - 0.25;

                                                setValue(id, next);
                                            }}
                                            onPlus={() => {
                                                const id: string = trainingMemUsageGbFieldId;
                                                const curr: number = getValues(id);
                                                const next: number = curr + 0.25;

                                                setValue(id, next);
                                            }}
                                            name={field.name}
                                            inputName={`${props.sessionIdentifier}-training-mem-usage-gb-input`}
                                            inputAriaLabel={`${props.sessionIdentifier}-training-mem-usage-gb-input`}
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
                                            name={numGpusFieldId}
                                            rules={{ min: 0, max: 8 }}
                                            render={({ field }) =>
                                                <NumberInput
                                                    value={field.value}
                                                    onChange={field.onChange}
                                                    onBlur={field.onBlur}
                                                    onMinus={() => {
                                                        const id: string = numGpusFieldId;
                                                        const curr: number = getValues(id);
                                                        const next: number = curr - 0.25;

                                                        setValue(id, next);
                                                    }}
                                                    onPlus={() => {
                                                        const id: string = numGpusFieldId;
                                                        const curr: number = getValues(id);
                                                        const next: number = curr + 0.25;

                                                        setValue(id, next);
                                                    }}
                                                    name={field.name}
                                                    inputName={`${props.sessionIdentifier}-num-gpus-input`}
                                                    key={`${props.sessionIdentifier}-num-gpus-input`}
                                                    inputAriaLabel={`${props.sessionIdentifier}-num-gpus-input`}
                                                    minusBtnAriaLabel="minus"
                                                    plusBtnAriaLabel="plus"
                                                    validated={getFieldState(numGpusFieldId).invalid ? 'error' : 'success'}
                                                    widthChars={4}
                                                    min={0}
                                                />}
                                        />
                                    </GridItem>
                                </Grid>
                            </FormGroup>
                        </GridItem>
                        {Array.from({ length: Math.max(Math.min((getValues(numGpusFieldId) || 1), 8), 1) }).map((_, idx: number) => {
                            return (
                                <GridItem key={`${props.sessionIdentifier}-gpu-${idx}-util-input-grditem`} span={3} rowSpan={1} hidden={(getValues(numGpusFieldId) || 1) < idx}>
                                    <FormGroup label={`GPU #${idx} % Utilization`}>
                                    <Controller
                                            name={getGpuInputFieldId(idx)}
                                            rules={{ min: 0, max: 8 }}
                                            render={({ field }) =>
                                                <NumberInput
                                                    value={field.value}
                                                    onChange={field.onChange}
                                                    onBlur={field.onBlur}
                                                    onMinus={() => {
                                                        const id: string = getGpuInputFieldId(idx);
                                                        const curr: number = getValues(id);
                                                        const next: number = curr - 0.25;

                                                        setValue(id, next);
                                                    }}
                                                    onPlus={() => {
                                                        const id: string = getGpuInputFieldId(idx);
                                                        const curr: number = getValues(id);
                                                        const next: number = curr + 0.25;

                                                        setValue(id, next);
                                                    }}
                                                    name={field.name}
                                                    inputName={`${props.sessionIdentifier}-gpu${idx}-percent-util-input`}
                                                    key={`${props.sessionIdentifier}-gpu${idx}-percent-util-input`}
                                                    inputAriaLabel={`${props.sessionIdentifier}-gpu${idx}-percent-util-input`}
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
                </Form>
            </FormSection>
        </React.Fragment>
    )
}