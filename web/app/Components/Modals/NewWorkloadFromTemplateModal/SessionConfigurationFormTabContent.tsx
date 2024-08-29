import {
  Button,
  Card,
  CardBody,
  Form,
  FormFieldGroup,
  FormFieldGroupExpandable,
  FormFieldGroupHeader,
  FormGroup,
  FormHelperText,
  FormSelect,
  FormSelectOption,
  Grid,
  GridItem,
  HelperText,
  HelperTextItem,
  NumberInput,
  TextInput,
} from '@patternfly/react-core';
import {DiceD6Icon} from "@patternfly/react-icons";
import React from 'react';

import {Controller, useFieldArray, useFormContext, useWatch} from 'react-hook-form';
import {
  DefaultTrainingEventField,
  NumberOfGpusDefault,
  SessionStartTickDefault,
  SessionStopTickDefault,
  TrainingCpuPercentUtilDefault,
  TrainingDurationInTicksDefault,
  TrainingGpuPercentUtilDefault,
  TrainingMemUsageGbDefault,
  TrainingStartTickDefault
} from './Constants';

export interface SessionConfigurationFormTabContentProps {
  children?: React.ReactNode;
  sessionIndex: number;
  defaultSessionId: string;
}

// TODO: Responsive validation not quite working yet.
export const SessionConfigurationFormTabContent: React.FunctionComponent<SessionConfigurationFormTabContentProps> = (props) => {
  const {control, setValue, getValues, getFieldState, watch} = useFormContext() // retrieve all hook methods

  const sessionIndex: number = props.sessionIndex;
  const sessionIdFieldId: string = `sessions.${sessionIndex}.id`;
  const sessionStartTickFieldId: string = `sessions.${sessionIndex}.start_tick`;
  const sessionStopTickFieldId: string = `sessions.${sessionIndex}.stop_tick`;
  const selectedTrainingEventFieldId: string = `sessions.${sessionIndex}.selected_training_event`
  const numTrainingEventsFieldId: string = `sessions.${sessionIndex}.num_training_events`;

  const selectedTrainingEventIndex: number = Number.parseInt(useWatch({control, name: selectedTrainingEventFieldId}));
  const trainingStartTickFieldId: string = `sessions.${sessionIndex}.trainings.${selectedTrainingEventIndex}.start_tick`;
  const trainingDurationTicksFieldId: string = `sessions.${sessionIndex}.trainings.${selectedTrainingEventIndex}.duration_in_ticks`;
  const trainingCpuPercentUtilFieldId: string = `sessions.${sessionIndex}.trainings.${selectedTrainingEventIndex}.cpu_util`;
  const trainingMemUsageGbFieldId: string = `sessions.${sessionIndex}.trainings.${selectedTrainingEventIndex}.mem_usage_gb`;
  const numGpusFieldId: string = `sessions.${sessionIndex}.trainings.${selectedTrainingEventIndex}.num_gpus`;

  const {
    fields: trainingEventFields,
    append: appendTrainingEvent,
    remove: removeTrainingEvent
  } = useFieldArray({name: `sessions.${sessionIndex}.trainings`, control});
  const {
    fields: gpuUtilizationFields,
    append: appendGpuUtilization,
    remove: removeGpuUtilization
  } = useFieldArray({
    name: `sessions.${sessionIndex}.trainings.${selectedTrainingEventIndex}.gpu_utilizations`,
    control
  });

  const getGpuInputFieldId = (gpuIndex: number) => {
    return `sessions.${sessionIndex}.trainings.${selectedTrainingEventIndex}.gpu_utilizations.${gpuIndex}.utilization`;
  };

  const getSessionIdValidationState = () => {
    const sessionId: string = watch(sessionIdFieldId);

    if (sessionId == undefined) {
      return 'default';
    }

    if (sessionId.length >= 1 && sessionId.length <= 36) {
      return 'success';
    }

    return 'error';
  }

  const isSessionIdValid = () => {
    const sessionId: string = watch(sessionIdFieldId);

    if (sessionId == undefined) {
      // Form hasn't loaded yet.
      return true;
    }

    return sessionId.length >= 1 && sessionId.length <= 36;
  }

  const validateTrainingStartTick = (value: string | number) => {
    if ((value as number) < 0 || (value as number) < (watch(sessionStartTickFieldId) as number) || (value as number) > ((watch(sessionStopTickFieldId) as number) - (watch(trainingDurationTicksFieldId) as number))) {
      return 'error';
    }

    return 'success';
  }

  return (
    <Card isCompact isRounded isFlat>
      <CardBody>
        <Form>
          <FormFieldGroup header={< FormFieldGroupHeader
            titleText={{
              text: `Session ${sessionIndex + 1} Configuration`,
              id: `session-${sessionIndex}-session-configuration-group`
            }}
            titleDescription="Modify the session ID, number of training events, start time, and stop time."
          />}>
            <Grid hasGutter md={12}>
              <GridItem span={6} key={`session-${sessionIndex}-session-id-grid-item`}>
                <FormGroup
                  label="Session ID"
                  labelInfo="Required length: 1-36 characters">
                  <Controller
                    control={control}
                    name={sessionIdFieldId}
                    defaultValue={props.defaultSessionId}
                    rules={{minLength: 1, maxLength: 36, required: true}}
                    render={({field}) =>
                      <TextInput
                        isRequired
                        label={`session-${sessionIndex}-session-id-text-input`}
                        aria-label={`session-${sessionIndex}-session-id-text-input`}
                        type="text"
                        id={`session-${sessionIndex}-session-id-text-input`}
                        name={field.name}
                        value={field.value}
                        placeholder={props.defaultSessionId}
                        validated={getSessionIdValidationState()}
                        onChange={field.onChange}
                        onBlur={field.onBlur}
                      />}
                  />
                  <FormHelperText
                    label={`session-${sessionIndex}-session-id-form-helper`}
                    aria-label={`session-${sessionIndex}-session-id-form-helper`}
                  >
                    <HelperText
                      label={`session-${sessionIndex}-session-id-text-input-helper-text`}
                      aria-label={`session-${sessionIndex}-session-id-text-input-helper-text`}
                    >
                      <HelperTextItem
                        aria-label={`session-${sessionIndex}-session-id-text-input-helper-text-item`}
                        label={`session-${sessionIndex}-session-id-text-input-helper-text-item`}
                        variant={getSessionIdValidationState()}
                      >
                        {isSessionIdValid() ? "" : "Session ID must be between 1 and 36 characters in length (inclusive)."}
                      </HelperTextItem>
                    </HelperText>
                  </FormHelperText>
                </FormGroup>
              </GridItem>
              <GridItem span={3} key={`session-${sessionIndex}-session-start-tick-grid-item`}>
                <FormGroup label="Session Start Tick">
                  <Controller
                    control={control}
                    name={sessionStartTickFieldId}
                    defaultValue={SessionStartTickDefault}
                    rules={{min: 1, max: watch(sessionStopTickFieldId) as number, required: true}}
                    render={({field}) =>
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
                        inputName={`session-${sessionIndex}-session-start-tick-input`}
                        inputAriaLabel={`session-${sessionIndex}-session-start-tick-input`}
                        minusBtnAriaLabel="minus"
                        plusBtnAriaLabel="plus"
                        validated={getFieldState(sessionStartTickFieldId).invalid ? 'error' : 'success'}
                        min={1}
                      />}
                  />
                </FormGroup>
              </GridItem>
              <GridItem span={3} key={`session-${sessionIndex}-session-stop-tick-grid-item`}>
                <FormGroup label="Session Stop Tick">
                  <Controller
                    control={control}
                    name={sessionStopTickFieldId}
                    defaultValue={SessionStopTickDefault}
                    rules={{min: watch(sessionStartTickFieldId) as number, required: true}}
                    render={({field}) =>
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
                        inputName={`session-${sessionIndex}-session-stop-tick-input`}
                        inputAriaLabel={`session-${sessionIndex}-session-stop-tick-input`}
                        minusBtnAriaLabel="minus"
                        plusBtnAriaLabel="plus"
                        validated={getFieldState(sessionStopTickFieldId).invalid ? 'error' : 'success'}
                        min={0}
                      />}
                  />
                </FormGroup>
              </GridItem>
              <GridItem span={3} key={`session-${sessionIndex}-num-training-events-grid-item`}>
                <FormGroup label="Number of Training Events">
                  <Controller
                    control={control}
                    name={numTrainingEventsFieldId}
                    defaultValue={1}
                    rules={{min: 0}}
                    render={({field}) =>
                      <TextInput
                        isRequired
                        label={`session-${sessionIndex}-num-training-events-text-input`}
                        aria-label={`session-${sessionIndex}-num-training-events-text-input`}
                        id={`session-${sessionIndex}-num-training-events-text-input`}
                        name={field.name}
                        value={field.value}
                        onBlur={field.onBlur}
                        onChange={(event: React.FormEvent<HTMLInputElement>) => {
                          let newNumTrainingEvents: number = +(event.target as HTMLInputElement).value;

                          if (newNumTrainingEvents < 0) {
                            newNumTrainingEvents = 0;
                          }

                          // update field array when GPUs number changed
                          const numTrainingEventFields: number = trainingEventFields.length;
                          if (newNumTrainingEvents > numTrainingEventFields) {
                            // Append GPUs to field array
                            for (let i: number = numTrainingEventFields; i < newNumTrainingEvents; i++) {
                              appendTrainingEvent({
                                ...DefaultTrainingEventField
                              });
                            }
                          } else {
                            // Remove GPUs from field array
                            for (let i: number = numTrainingEventFields; i > newNumTrainingEvents; i--) {
                              removeTrainingEvent(i - 1);
                            }
                          }

                          field.onChange(newNumTrainingEvents);
                        }}
                        type='number'
                        placeholder={'0'}
                        validated={getFieldState(numTrainingEventsFieldId).invalid ? 'error' : 'success'}
                      />}/>
                </FormGroup>
              </GridItem>
              <GridItem span={3} key={`session-${sessionIndex}-selected-training-item-grid-item`}>
                <FormGroup label="Selected Training Event">
                  <Controller
                    control={control}
                    name={selectedTrainingEventFieldId}
                    defaultValue={0}
                    rules={{min: 0, max: (watch(numTrainingEventsFieldId) as number) - 1}}
                    render={({field}) =>
                      <FormSelect value={field.value} onChange={field.onChange}
                                  aria-label="Training Event Selection Menu" ouiaId="TrainingEventSelectionMenu">
                        {Array.from({length: watch(numTrainingEventsFieldId) as number}).map((_, idx: number) => {
                          return (<FormSelectOption key={idx} value={idx} label={`Training Event #${idx + 1}`}/>)
                        })}
                      </FormSelect>
                    }/>
                </FormGroup>
              </GridItem>
            </Grid>
            <FormFieldGroupExpandable isExpanded
                                      toggleAriaLabel={`session-${sessionIndex}-training-event-configuration`}
                                      header={<FormFieldGroupHeader
                                        titleText={{
                                          text: `Training Event #${selectedTrainingEventIndex + 1} Configuration`,
                                          id: `session-${sessionIndex}-training-resource-configuration-group`
                                        }}
                                        titleDescription={`Specify the configuration for training event #${selectedTrainingEventIndex + 1} of Session ${sessionIndex + 1}.`}
                                      />}>
              <Grid hasGutter>
                <GridItem span={3}
                          key={`session-${sessionIndex}-training${selectedTrainingEventIndex}-training-start-grid-item`}>
                  <FormGroup label="Training Start Tick">
                    <Controller
                      control={control}
                      name={trainingStartTickFieldId}
                      defaultValue={TrainingStartTickDefault}
                      rules={{
                        min: (watch(sessionStartTickFieldId) as number),
                        max: (watch(sessionStopTickFieldId) as number) - (watch(trainingDurationTicksFieldId) as number),
                        required: true
                      }}
                      render={({field}) =>
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
                          id={`session-${sessionIndex}-training${selectedTrainingEventIndex}-start-tick-input`}
                          inputName={`session-${sessionIndex}-training${selectedTrainingEventIndex}-start-tick-input`}
                          inputAriaLabel={`session-${sessionIndex}-training${selectedTrainingEventIndex}-start-tick-input`}
                          minusBtnAriaLabel="minus"
                          plusBtnAriaLabel="plus"
                          validated={validateTrainingStartTick(field.value)}
                          widthChars={4}
                          min={(watch(sessionStartTickFieldId) as number)}
                        />}
                    />
                  </FormGroup>
                </GridItem>
                <GridItem span={3}
                          key={`session-${sessionIndex}-training${selectedTrainingEventIndex}-training-duration-grid-item`}>
                  <FormGroup label="Training Duration (Ticks)">
                    <Controller
                      control={control}
                      name={trainingDurationTicksFieldId}
                      defaultValue={TrainingDurationInTicksDefault}
                      rules={{
                        min: 1,
                        max: (watch(sessionStopTickFieldId) as number) - (watch(trainingStartTickFieldId) as number) + 1,
                        required: true
                      }}
                      render={({field}) =>
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
                          id={`session-${sessionIndex}-training${selectedTrainingEventIndex}-duration-ticks-input`}
                          inputName={`session-${sessionIndex}-training${selectedTrainingEventIndex}-duration-ticks-input`}
                          inputAriaLabel={`session-${sessionIndex}-training${selectedTrainingEventIndex}-duration-ticks-input`}
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
                  titleText={{
                    text: 'Training Resource Configuration',
                    id: `session-${sessionIndex}-training-resource-configuration-group`
                  }}
                  titleDescription="Modify the resource configuration of the training event."
                />}>
                <Grid hasGutter>
                  <GridItem span={3}
                            key={`session-${sessionIndex}-training${selectedTrainingEventIndex}-cpu-util-grid-item`}>
                    <FormGroup label="CPU % Utilization">
                      <Controller
                        control={control}
                        name={trainingCpuPercentUtilFieldId}
                        defaultValue={TrainingCpuPercentUtilDefault}
                        rules={{min: 0, max: 100, required: true}}
                        render={({field}) =>
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
                            id={`session-${sessionIndex}-training${selectedTrainingEventIndex}-cpu-percent-util-input`}
                            inputName={`session-${sessionIndex}-training${selectedTrainingEventIndex}-cpu-percent-util-input`}
                            inputAriaLabel={`session-${sessionIndex}-training${selectedTrainingEventIndex}-cpu-percent-util-input`}
                            minusBtnAriaLabel="minus"
                            plusBtnAriaLabel="plus"
                            validated={getFieldState(trainingCpuPercentUtilFieldId).invalid ? 'error' : 'success'}
                            widthChars={4}
                            min={0}
                            max={100}
                          />}/>
                    </FormGroup>
                  </GridItem>
                  <GridItem span={3}
                            key={`session-${sessionIndex}-training${selectedTrainingEventIndex}-ram-usage-grid-item`}>
                    <FormGroup label="RAM Usage (GB)">
                      <Controller
                        control={control}
                        name={trainingMemUsageGbFieldId}
                        rules={{min: 0, max: 128_000, required: true}}
                        defaultValue={TrainingMemUsageGbDefault}
                        render={({field}) =>
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
                            id={`session-${sessionIndex}-training${selectedTrainingEventIndex}-mem-usage-gb-input`}
                            inputName={`session-${sessionIndex}-training${selectedTrainingEventIndex}-mem-usage-gb-input`}
                            inputAriaLabel={`session-${sessionIndex}-training${selectedTrainingEventIndex}-mem-usage-gb-input`}
                            minusBtnAriaLabel="minus"
                            plusBtnAriaLabel="plus"
                            validated={getFieldState(trainingMemUsageGbFieldId).invalid ? 'error' : 'success'}
                            widthChars={4}
                            min={0}
                          />}
                      />
                    </FormGroup>
                  </GridItem>
                  <GridItem span={3}
                            key={`session-${sessionIndex}-training${selectedTrainingEventIndex}-num-gpus-grid-item`}>
                    <FormGroup label={`Number of GPUs`}>
                      <Controller
                        control={control}
                        name={numGpusFieldId}
                        rules={{min: 0, max: 8, required: true}}
                        defaultValue={NumberOfGpusDefault}
                        render={({field}) =>
                          <NumberInput
                            value={field.value}
                            onChange={(event) => {
                              const newNumberOfGpus: number = +(event.target as HTMLInputElement).value;

                              // update field array when GPUs number changed
                              const numberOfGpuFields: number = gpuUtilizationFields.length;
                              if (newNumberOfGpus > numberOfGpuFields) {
                                // Append GPUs to field array
                                for (let i: number = numberOfGpuFields; i < newNumberOfGpus; i++) {
                                  appendGpuUtilization({utilization: TrainingGpuPercentUtilDefault});
                                }
                              } else {
                                // Remove GPUs from field array
                                for (let i: number = numberOfGpuFields; i > newNumberOfGpus; i--) {
                                  removeGpuUtilization(i - 1);
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
                              removeGpuUtilization(gpuUtilizationFields.length - 1);
                            }}
                            onPlus={() => {
                              const curr: number = getValues(numGpusFieldId) as number;
                              let next: number = curr + 1;

                              if (next > 8) {
                                next = 8;
                              }

                              setValue(numGpusFieldId, next);
                              appendGpuUtilization({utilization: TrainingGpuPercentUtilDefault});
                            }}
                            name={field.name}
                            id={`session-${sessionIndex}-training${selectedTrainingEventIndex}-num-gpus-input`}
                            inputName={`session-${sessionIndex}-training${selectedTrainingEventIndex}-num-gpus-input`}
                            key={`session-${sessionIndex}-training${selectedTrainingEventIndex}-num-gpus-input`}
                            inputAriaLabel={`session-${sessionIndex}-num-gpus-input`}
                            minusBtnAriaLabel="minus"
                            plusBtnAriaLabel="plus"
                            validated={getFieldState(numGpusFieldId).invalid ? 'error' : 'success'}
                            widthChars={4}
                            min={0}
                            max={8}
                          />}
                      />
                    </FormGroup>
                  </GridItem>
                  <GridItem span={3} key={`session-${sessionIndex}-training${selectedTrainingEventIndex}-randomize-resources-grid-item`}>
                    <FormGroup label={`Randomize Resources`}>
                      <Button id={`session-${sessionIndex}-training${selectedTrainingEventIndex}-randomize-resources-button`} name={`session-${sessionIndex}-training${selectedTrainingEventIndex}-randomize-resources-button`} icon={<DiceD6Icon/>}/>
                    </FormGroup>
                  </GridItem>
                  {Array.from({length: watch(numGpusFieldId) as number}).map((_, idx: number) => {
                    return (
                      <GridItem
                        key={`session-${sessionIndex}-training${selectedTrainingEventIndex}-gpu-${idx}-util-input-grditem`}
                        span={3} rowSpan={1} hidden={(watch(numGpusFieldId) as number || 1) < idx}>
                        <FormGroup label={`GPU #${idx} % Utilization`}>
                          <Controller
                            control={control}
                            name={getGpuInputFieldId(idx)}
                            defaultValue={TrainingGpuPercentUtilDefault}
                            rules={{min: 0, max: 100, required: true}}
                            render={({field}) =>
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
                                id={`session-${sessionIndex}-training${selectedTrainingEventIndex}-gpu${idx}-percent-util-input`}
                                inputName={`session-${sessionIndex}-training${selectedTrainingEventIndex}-gpu${idx}-percent-util-input`}
                                key={`session-${sessionIndex}-training${selectedTrainingEventIndex}-gpu${idx}-percent-util-input`}
                                inputAriaLabel={`session-${sessionIndex}-training${selectedTrainingEventIndex}-gpu${idx}-percent-util-input`}
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
    </Card>
  )
}
