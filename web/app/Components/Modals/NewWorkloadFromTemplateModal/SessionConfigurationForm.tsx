import React from 'react';
import {
    Card,
    CardBody,
    Divider,
    Form,
    FormGroup,
    FormSection,
    FormHelperText,
    Grid,
    GridItem,
    HelperText,
    HelperTextItem,
    NumberInput,
    TextInput,
    ValidatedOptions,
    Tabs,
    Tab,
    TabTitleText,
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
const NumberOfGpusDefault: number = 1;

export interface SessionConfigurationFormProps {
    children?: React.ReactNode;
}

// TODO: Responsive validation not quite working yet.
export const SessionConfigurationForm: React.FunctionComponent<SessionConfigurationFormProps> = (props) => {
    const defaultSessionId = React.useRef(uuidv4());
    const { control, setValue, getValues, getFieldState, watch, formState, formState: { errors } } = useFormContext() // retrieve all hook methods
    const { fields, append, remove } = useFieldArray({ name: "sessions", control });

    const [activeSessionTab, setActiveSessionTab] = React.useState<number>(0);
    const [sessionTabs, setSessionTabs] = React.useState<string[]>(['Session 1']);
    const [newSessionTabNumber, setNewSessionTabNumber] = React.useState<number>(2);
    const sessionTabComponentRef = React.useRef<any>();
    const firstSessionTabMount = React.useRef<boolean>(true);

    const getGpuInputFieldId = (sessionIndex: number, gpuIndex: number) => {
        return `sessions.${sessionIndex}.gpu_${gpuIndex}_training_util_percent`;
    }

    React.useEffect(() => {
        console.log(errors);
    }, [errors]);

    const onSessionTabSelect = (
        tabIndex: number
    ) => {
        setActiveSessionTab(tabIndex);
    };

    const onCloseSessionTab = (event: any, tabIndex: string | number) => {
        const tabIndexNum = tabIndex as number;
        let nextTabIndex = activeSessionTab;
        if (tabIndexNum < activeSessionTab) {
            // if a preceding tab is closing, keep focus on the new index of the current tab
            nextTabIndex = activeSessionTab - 1 > 0 ? activeSessionTab - 1 : 0;
        } else if (activeSessionTab === sessionTabs.length - 1) {
            // if the closing tab is the last tab, focus the preceding tab
            nextTabIndex = sessionTabs.length - 2 > 0 ? sessionTabs.length - 2 : 0;
        }
        setActiveSessionTab(nextTabIndex);
        setSessionTabs(sessionTabs.filter((tab, index) => index !== tabIndex));
    };

    const onAddSessionTab = () => {
        setSessionTabs([...sessionTabs, `Session ${newSessionTabNumber}`]);
        setActiveSessionTab(sessionTabs.length);
        setNewSessionTabNumber(newSessionTabNumber + 1);
    };

    React.useEffect(() => {
        const newVal: number = sessionTabs.length;
        const oldVal: number = fields.length;
        console.log(`Old (fields.length): ${oldVal}, New (sessionTabs.length): ${newVal}`);
        if (newVal > oldVal) {
            // Append sessions to field array
            for (let i = oldVal; i < newVal; i++) {
                console.log(`Adding new session field. fields.length pre-add: ${fields.length}. i: ${i}, oldVal: ${oldVal}, newVal: ${newVal}.`)
                append({});
                console.log(`Added new session field. fields.length post-add: ${fields.length}. i: ${i}, oldVal: ${oldVal}, newVal: ${newVal}.`)
            }
        } else {
            // Remove sessions from field array
            for (let i = oldVal; i > newVal; i--) {
                console.log(`Removing session field. fields.length pre-removal: ${fields.length}`)
                remove(i - 1);
                console.log(`Removed session field. fields.length post-removal: ${fields.length}`)
            }
        }

        if (firstSessionTabMount.current) {
            firstSessionTabMount.current = false;
            return;
        } else {
            const first = sessionTabComponentRef.current?.tabList.current.childNodes[activeSessionTab];
            first && first.firstChild.focus();
        }
    }, [sessionTabs]);


    return (
        <FormSection title={`Workload Sessions (${sessionTabs.length})`} titleElement='h1' >
            <Tabs
                isFilled
                activeKey={activeSessionTab}
                onSelect={(event: React.MouseEvent<HTMLElement, MouseEvent>, eventKey: number | string) => { onSessionTabSelect(eventKey as number) }}
                isBox={true}
                onClose={onCloseSessionTab}
                onAdd={onAddSessionTab}
                addButtonAriaLabel='Add Additional Session to Workload'
                role='region'
                ref={sessionTabComponentRef}
                aria-label="Session Configuration Tabs"
            >
                {sessionTabs.map((tabName: string, tabIndex: number) => {
                    const trainingStartTickFieldId: string = `sessions.${tabIndex}.training_start_tick`;
                    const sessionIdFieldId: string = `sessions.${tabIndex}.session_id`;
                    const sessionStartTickFieldId: string = `sessions.${tabIndex}.session_start_tick`;
                    const sessionStopTickFieldId: string = `sessions.${tabIndex}.session_end_tick`;
                    const trainingDurationTicksFieldId: string = `sessions.${tabIndex}.training_duration_ticks`;
                    const trainingCpuPercentUtilFieldId: string = `sessions.${tabIndex}.cpu_percent_util`;
                    const trainingMemUsageGbFieldId: string = `sessions.${tabIndex}.mem_usage_gb_util`;
                    const numGpusFieldId: string = `sessions.${tabIndex}.num_gpus`;

                    return (<Tab
                        key={tabIndex}
                        eventKey={tabIndex}
                        aria-label={`${tabName} Tab`}
                        title={<TabTitleText>{tabName}</TabTitleText>}
                        closeButtonAriaLabel={`Close ${tabName} Tab`}
                        isCloseDisabled={sessionTabs.length == 1} // Can't close the last session.
                    >
                        <Card isCompact isRounded isFlat>
                            <CardBody>
                                <FormSection title={`General Session Parameters`} titleElement='h1'>
                                    <Form>
                                        <Grid hasGutter md={12}>
                                            <GridItem span={12}>
                                                <FormGroup
                                                    label="Session ID:">
                                                    <Controller
                                                        control={control}
                                                        name={sessionIdFieldId}
                                                        defaultValue={defaultSessionId.current}
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
                                                                placeholder={defaultSessionId.current}
                                                                // validated={(watch(sessionIdFieldId).length >= 1 && watch(sessionIdFieldId).length <= 36) ? ValidatedOptions.success : ValidatedOptions.error}
                                                                onChange={(event) => field.onChange}
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
                                                                widthChars={4}
                                                                min={1}
                                                            />}
                                                    />
                                                </FormGroup>
                                            </GridItem>
                                            <GridItem span={3}>
                                                <FormGroup label="Training Start Tick">
                                                    <Controller
                                                        control={control}
                                                        name={trainingStartTickFieldId}
                                                        defaultValue={TrainingStartTickDefault}
                                                        rules={{ min: (watch(sessionStartTickFieldId) as number) + 1, max: (watch(sessionStopTickFieldId) as number) - (watch(trainingDurationTicksFieldId) as number), required: true }}
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
                                                                validated={getFieldState(trainingStartTickFieldId).invalid ? 'error' : 'success'}
                                                                widthChars={4}
                                                                min={(watch(sessionStartTickFieldId) as number) + 1}
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
                                                                        onChange={(event) => field.onChange(+(event.target as HTMLInputElement).value)}
                                                                        onBlur={field.onBlur}
                                                                        onMinus={() => {
                                                                            const curr: number = getValues(numGpusFieldId) as number;
                                                                            const next: number = curr - 1;

                                                                            setValue(numGpusFieldId, next);
                                                                        }}
                                                                        onPlus={() => {
                                                                            const curr: number = getValues(numGpusFieldId) as number;
                                                                            const next: number = curr + 1;

                                                                            setValue(numGpusFieldId, next);
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
                                                                    />}
                                                            />
                                                        </GridItem>
                                                    </Grid>
                                                </FormGroup>
                                            </GridItem>
                                            {Array.from({ length: Math.max(Math.min((watch(numGpusFieldId) as number), 8), 1) }).map((_, idx: number) => {
                                                return (
                                                    <GridItem key={`session-${tabIndex}-gpu-${idx}-util-input-grditem`} span={3} rowSpan={1} hidden={(getValues(numGpusFieldId) as number || 1) < idx}>
                                                        <FormGroup label={`GPU #${idx} % Utilization`}>
                                                            <Controller
                                                                control={control}
                                                                name={getGpuInputFieldId(tabIndex, idx)}
                                                                defaultValue={TrainingGpuPercentUtilDefault}
                                                                rules={{ min: 0, max: 100, required: true }}
                                                                render={({ field }) =>
                                                                    <NumberInput
                                                                        value={field.value}
                                                                        onChange={(event) => field.onChange(+(event.target as HTMLInputElement).value)}
                                                                        onBlur={field.onBlur}
                                                                        onMinus={() => {
                                                                            const id: string = getGpuInputFieldId(tabIndex, idx);
                                                                            const curr: number = getValues(id) as number;
                                                                            const next: number = curr - 0.25;

                                                                            setValue(id, next);
                                                                        }}
                                                                        onPlus={() => {
                                                                            const id: string = getGpuInputFieldId(tabIndex, idx);
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
                                                                        validated={getFieldState(getGpuInputFieldId(tabIndex, idx)).invalid ? 'error' : 'success'}
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
                            </CardBody>
                        </Card>
                    </Tab>)
                })}
            </Tabs>
        </FormSection>
    )
}