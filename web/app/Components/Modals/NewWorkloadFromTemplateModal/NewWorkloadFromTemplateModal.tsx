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

import { useForm, FormProvider, useFormContext, Controller } from "react-hook-form"

import { ResourceRequest, Session, TrainingEvent, WorkloadTemplate } from '@app/Data';
import { SessionConfigurationForm } from './SessionConfigurationForm';

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

const SessionStartTickDefault: number = 1;
const SessionStopTickDefault: number = 6;
const TrainingStartTickDefault: number = 2;
const TrainingDurationInTicksDefault: number = 2;
const TrainingCpuPercentUtilDefault: number = 10;
const TrainingMemUsageGbDefault: number = 0.25;
const NumberOfGpusDefault: number = 1;

// How much to adjust the timescale adjustment factor when using the 'plus' and 'minus' buttons to adjust the field's value.
const TimescaleAdjustmentFactorDelta: number = 0.1;
const TimescaleAdjustmentFactorMax: number = 10;
const TimescaleAdjustmentFactorMin: number = 1.0e-3;
const TimeAdjustmentFactorDefault: number = 0.1;

// How much to adjust the workload seed when using the 'plus' and 'minus' buttons to adjust the field's value.
const WorkloadSeedDelta: number = 1.0;
const WorkloadSeedMax: number = 2147483647.0;
const WorkloadSeedMin: number = 0.0;
const WorkloadSeedDefault: number = 0.0;

const DefaultGpuUtilizations: number[][] = [[100.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0]];

// Clamp a value between two extremes.
function clamp(value: number, min: number, max: number) {
    return Math.max(Math.min(value, max), min)
}

function roundToThreeDecimalPlaces(num: number) {
    return +(Math.round(num + 'e+3') + 'e-3');
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

export const NewWorkloadFromTemplateModal: React.FunctionComponent<NewWorkloadFromTemplateModalProps> = (props) => {
    const { register, setValue, getValues, handleSubmit, clearErrors, control } = useForm();

    const defaultWorkloadTitle = React.useRef(uuidv4());
    const defaultSessionId = React.useRef(uuidv4());

    const onSubmit = (data) => console.log(data);

    // const [sessions, setSessions] = React.useState<Session[]>([{
    //     id: defaultSessionId.current,
    //     resource_request: {
    //         cpus: TrainingCpuPercentUtilDefault,
    //         mem_gb: TrainingMemUsageGbDefault,
    //         gpu_type: "ANY_GPU",
    //         gpus: NumberOfGpusDefault,
    //     },
    //     start_tick: SessionStartTickDefault,
    //     stop_tick: SessionStopTickDefault,
    //     trainings: [{
    //         sessionId: defaultSessionId.current,
    //         trainingId: uuidv4(),
    //         cpuUtil: TrainingCpuPercentUtilDefault,
    //         memUsageGb: TrainingMemUsageGbDefault,
    //         gpuUtil: [75],
    //         startTick: TrainingStartTickDefault,
    //         durationInTicks: TrainingDurationInTicksDefault,
    //     }],
    //     trainings_completed: 0,
    //     state: "awaiting start",
    //     error_message: "",
    // }]);

    const [activeSessionTab, setActiveSessionTab] = React.useState<number>(0);
    const [sessionTabs, setSessionTabs] = React.useState<string[]>(['Session 1']);
    const [newSessionTabNumber, setNewSessionTabNumber] = React.useState<number>(2);
    const sessionTabComponentRef = React.useRef<any>();
    const firstSessionTabMount = React.useRef<boolean>(true);

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
        if (firstSessionTabMount.current) {
            firstSessionTabMount.current = false;
            return;
        } else {
            const first = sessionTabComponentRef.current?.tabList.current.childNodes[activeSessionTab];
            first && first.firstChild.focus();
        }
    }, [sessionTabs]);

    // const handleWorkloadTitleChanged = (_event, title: string) => {
    //     setWorkloadTitle(title);
    //     setWorkloadTitleIsValid(title.length >= 0 && title.length <= 36);
    // };

    // const handleWorkloadSeedChanged = (_event, seed: string) => {
    //     const validSeed: boolean = /[0-9]/.test(seed) || seed == '';

    //     // If it's either the empty string, or we can't even convert the value to a number,
    //     // then update the state accordingly.
    //     if (!validSeed || seed == '') {
    //         setWorkloadSeedIsValid(validSeed);
    //         setWorkloadSeed('');
    //         return;
    //     }

    //     // Convert to a number.
    //     const parsed: number = parseInt(seed, 10);

    //     // If it's a float or something, then just default to no seed.
    //     if (Number.isNaN(parsed)) {
    //         setWorkloadSeed('');
    //         return;
    //     }

    //     // If it's greater than the max value, then it is invalid.
    //     if (parsed > 2147483647 || parsed < 0) {
    //         setWorkloadSeedIsValid(false);
    //         setWorkloadSeed(seed);
    //         return;
    //     }

    //     setWorkloadSeed(parsed.toString());
    //     setWorkloadSeedIsValid(true);
    // };

    // const onWorkloadDataDropdownToggleClick = () => {
    //     setIsWorkloadDataDropdownOpen(!isWorkloadDataDropdownOpen);
    // };

    // const onWorkloadDataDropdownSelect = (
    //     _event: React.MouseEvent<Element, MouseEvent> | undefined,
    //     value: string | number | undefined,
    // ) => {
    //     // eslint-disable-next-line no-console
    //     console.log('selected', value);

    //     console.log(`Value: ${value}`)
    //     if (value != undefined) {
    //         setSelectedWorkloadTemplate(workloadTemplates[value]);
    //     } else {
    //         setSelectedWorkloadTemplate("");
    //     }
    //     setIsWorkloadDataDropdownOpen(false);
    // };

    // const validateTimescaleAdjustmentFactor = () => {
    //     if (timescaleAdjustmentFactor === '' || Number.isNaN(timescaleAdjustmentFactor)) {
    //         return 'error';
    //     }

    //     return (timescaleAdjustmentFactor <= 0 || timescaleAdjustmentFactor > 10) ? 'error' : 'success';
    // };

    // const getWorkloadSeedValidatedState = () => {
    //     if (!workloadSeedIsValid) {
    //         return ValidatedOptions.error;
    //     }

    //     if (workloadSeed == '') {
    //         return ValidatedOptions.default;
    //     }

    //     return ValidatedOptions.success;
    // };

    // const isSubmitButtonDisabled = () => {
    //     if (!workloadTitleIsValid) {
    //         return true;
    //     }

    //     // if (setSelectedWorkloadTemplate.length == 0) {
    //     //     return true;
    //     // }

    //     if (!workloadSeedIsValid) {
    //         return true;
    //     }

    //     if (sessionStartTick === '' || trainingStartTick === '' || sessionStopTick === '' || trainingDurationInTicks === '' || trainingMemUsageGb === '' || trainingCpuPercentUtil === '') {
    //         return true;
    //     }

    //     // The following are all the conditions from the `validated` fields of the text inputs.
    //     if (validateNumberOfGpusInput() !== 'success' || validateTrainingMemoryUsageInput() !== 'success' || validateTrainingCpuInput() !== 'success' || validateTrainingDurationInTicksInput() !== 'success' || validateTrainingStartTickInput() !== 'success' || validateSessionStartStopInput() !== 'success' || validateSessionStartTickInput() !== 'success') {
    //         return true;
    //     }

    //     for (let i: number = 0; i < sessionTabs.length; i++) {
    //         // This one might be redundant. 
    //         if (numberOfGPUs === '' || numberOfGPUs[i] < 0 || numberOfGPUs[i] > 8) {
    //             return true;
    //         }

    //         const numGPUs: number = (numberOfGPUs[i] || 1);
    //         for (let j = 0; j < numGPUs; j++) {
    //             if (validateGpuUtilInput(i, j) !== 'success') {
    //                 return true;
    //             }
    //         }
    //     }

    //     if (validateTimescaleAdjustmentFactor() == 'error') {
    //         return true;
    //     }

    //     return false;
    // };

    // // We pass 'numGPUs' in directly instead of referencing the state variable so that we don't have to validate its type.
    // function assertGpuUtilizationsAreAllNumbers(gpuUtils: (number[][] | ''), numGPUs: number[]): asserts gpuUtils is number[][] {
    //     if (gpuUtils === '') {
    //         throw new Error("GPU utilizations are invalid.");
    //     }

    //     for (let sessionIdx: number = 0; sessionIdx < sessionTabs.length; sessionIdx++) {
    //         for (let gpuIdx: number = 0; gpuIdx < numGPUs[sessionIdx]; gpuIdx++) {
    //             if (validateGpuUtilInput(sessionIdx, gpuIdx) === 'error') {
    //                 console.error(`gpuUtilization[${sessionIdx}][${gpuIdx}] is not a valid value during submission.`)
    //                 throw new Error(`gpuUtilization[${sessionIdx}][${gpuIdx}] is not a valid value during submission.`)
    //             }
    //         }
    //     }
    // }

    // Called when the 'submit' button is clicked.
    //     const onSubmitWorkload = () => {
    //         // If the user left the workload title blank, then use the default workload title, which is a randomly-generated UUID.
    //         let workloadTitleToSubmit: string = workloadTitle;
    //         if (workloadTitleToSubmit.length == 0) {
    //             workloadTitleToSubmit = defaultWorkloadTitle.current;
    //         }

    //         assertAreNumbers(trainingCpuPercentUtil);
    //         assertAreNumbers(trainingMemUsageGb);
    //         assertAreNumbers(numberOfGPUs);
    //         assertAreNumbers(sessionStartTick);
    //         assertAreNumbers(sessionStopTick);
    //         assertAreNumbers(trainingStartTick);
    //         assertAreNumbers(trainingDurationInTicks);
    //         assertIsNumber(timescaleAdjustmentFactor);
    //         assertGpuUtilizationsAreAllNumbers(gpuUtilizations, numberOfGPUs);

    //         console.debug(`Registering new template-based workload "${workloadTitleToSubmit}":
    // - Training CPU % Util: ${trainingCpuPercentUtil}
    // - Training Memory Usage in GB: : ${trainingMemUsageGb}
    // - Number of GPUs: ${numberOfGPUs}
    // - Session Start Tick: ${sessionStartTick}
    // - Session Stop Tick: ${sessionStopTick}
    // - Training Start Tick: ${trainingStartTick}
    // - Training Duration in Ticks: ${trainingDurationInTicks}
    // - Timescale Adjustment Factor: ${timescaleAdjustmentFactor}
    //             `);

    //         const sessionIdentifier: string = (sessionId.length == 0) ? defaultSessionId.current : sessionId;

    //         // TOOD: 
    //         // When we have multiplate templates, we'll add template-specific submission logic
    //         // to aggregate the information from that template and convert it to a valid
    //         // workload registration request.

    //         let gpuUtilizationsToSubmit: number[] = []
    //         for (let i: number = 0; i < numberOfGPUs; i++) {
    //             // Add only the GPU utilizations for the number of GPUs that the user has configured for the workload.
    //             // If we just passed `gpuUtilizations` directly, then we'd pass all 8 GPU utilizations, which would be wrong.
    //             gpuUtilizationsToSubmit.push(gpuUtilizations[i]);

    //             console.debug(`GPU Utilization of GPU#${i}: ${gpuUtilizations[i]}`)
    //         }

    //         const trainingEvent: TrainingEvent = {
    //             sessionId: sessionIdentifier,
    //             trainingId: uuidv4(),
    //             cpuUtil: trainingCpuPercentUtil,
    //             memUsageGb: trainingMemUsageGb,
    //             gpuUtil: gpuUtilizationsToSubmit,
    //             startTick: trainingStartTick,
    //             durationInTicks: trainingDurationInTicks,
    //         }

    //         const resource_request: ResourceRequest = {
    //             cpus: trainingCpuPercentUtil,
    //             mem_gb: trainingMemUsageGb,
    //             gpus: numberOfGPUs,
    //             gpu_type: "",
    //         }

    //         const session: Session = {
    //             id: sessionIdentifier,
    //             resource_request: resource_request,
    //             start_tick: sessionStartTick,
    //             stop_tick: sessionStopTick,
    //             trainings: [trainingEvent],
    //             trainings_completed: 0,
    //             state: "awaiting start",
    //             error_message: "",
    //         }

    //         const sessions: Session[] = [session]

    //         const workloadTemplate: WorkloadTemplate = {
    //             // name: selectedWorkloadTemplate,
    //             sessions: sessions,
    //         }

    //         console.log(`Submitting workload template: ${JSON.stringify(workloadTemplate)}`)

    //         props.onConfirm(workloadTitleToSubmit, workloadSeed, debugLoggingEnabled, workloadTemplate, timescaleAdjustmentFactor);

    //         // Reset all of the fields.
    //         resetSubmissionForm();
    //     };

    //     const validateSessionStartTickInput = () => {
    //         if (sessionStartTick === '') {
    //             return 'error';
    //         }

    //         if (trainingStartTick === '' || sessionStopTick === '') {
    //             return 'warning';
    //         }

    //         for (let sessionIdx: number = 0; sessionIdx < sessionTabs.length; sessionIdx++) {
    //             if ((sessionStartTick[sessionIdx] < 0 || sessionStartTick[sessionIdx] >= trainingStartTick[sessionIdx] || sessionStartTick[sessionIdx] >= sessionStopTick[sessionIdx])) {
    //                 return 'error';
    //             }
    //         }

    //         return 'success';
    //     }

    //     const validateSessionStartStopInput = () => {
    //         if (sessionStopTick === '') {
    //             return 'error';
    //         }

    //         if (sessionStartTick === '' || trainingStartTick === '' || trainingDurationInTicks === '') {
    //             return 'warning';
    //         }

    //         for (let sessionIdx: number = 0; sessionIdx < sessionTabs.length; sessionIdx++) {
    //             if ((sessionStopTick[sessionIdx] < 0 || trainingStartTick[sessionIdx] >= sessionStopTick[sessionIdx] || sessionStartTick[sessionIdx] >= sessionStopTick[sessionIdx] || trainingStartTick[sessionIdx] + trainingDurationInTicks[sessionIdx] >= sessionStopTick[sessionIdx])) {
    //                 return 'error';
    //             }
    //         }

    //         return 'success'
    //     }

    //     const validateTrainingStartTickInput = () => {
    //         if (trainingStartTick === '') {
    //             return 'error';
    //         }

    //         if (sessionStartTick === '' || sessionStopTick === '' || trainingDurationInTicks === '') {
    //             return 'warning';
    //         }

    //         for (let sessionIdx: number = 0; sessionIdx < sessionTabs.length; sessionIdx++) {
    //             if ((trainingStartTick[sessionIdx] < 0 || sessionStartTick[sessionIdx] >= trainingStartTick[sessionIdx] || trainingStartTick[sessionIdx] >= sessionStopTick[sessionIdx] || trainingStartTick[sessionIdx] + trainingDurationInTicks[sessionIdx] >= sessionStopTick[sessionIdx])) {
    //                 return 'success';
    //             }
    //         }

    //         return 'success';
    //     }

    //     const validateTrainingDurationInTicksInput = () => {
    //         if (trainingDurationInTicks === '') {
    //             return 'error';
    //         }

    //         if (sessionStartTick === '' || sessionStopTick === '' || trainingStartTick === '') {
    //             return 'warning';
    //         }

    //         for (let sessionIdx: number = 0; sessionIdx < sessionTabs.length; sessionIdx++) {
    //             if ((trainingDurationInTicks[sessionIdx] < 0 || trainingStartTick[sessionIdx] + trainingDurationInTicks[sessionIdx] >= sessionStopTick[sessionIdx])) {
    //                 return 'error';
    //             }
    //         }

    //         return 'success';
    //     }

    //     const validateTrainingCpuInput = () => {
    //         if (trainingCpuPercentUtil === '') {
    //             return 'error';
    //         }

    //         for (let sessionIdx: number = 0; sessionIdx < sessionTabs.length; sessionIdx++) {
    //             if ((trainingCpuPercentUtil[sessionIdx] < 0 || trainingCpuPercentUtil[sessionIdx] > 100)) {
    //                 return 'error';
    //             }
    //         }

    //         return 'success';
    //     }

    //     const validateTrainingMemoryUsageInput = () => {
    //         if (trainingMemUsageGb === '') {
    //             return 'error';
    //         }

    //         for (let sessionIdx: number = 0; sessionIdx < sessionTabs.length; sessionIdx++) {
    //             if ((trainingMemUsageGb[sessionIdx] < 0 || trainingMemUsageGb[sessionIdx] > 128_000)) {
    //                 return 'error';
    //             }
    //         }

    //         return 'success';
    //     }

    //     const validateNumberOfGpusInput = () => {
    //         if (numberOfGPUs === '') {
    //             return 'error';
    //         }

    //         if (numberOfGPUs === undefined) return 'warning';

    //         for (let i: number = 0; i < sessionTabs.length; i++) {
    //             if (numberOfGPUs[i] < 0 || numberOfGPUs[i] > 8) return 'warning';
    //         }

    //         return 'success';
    //     }

    //     const validateGpuUtilInput = (outerIndex: number, innerIndex: number) => {
    //         return (gpuUtilizations[outerIndex][innerIndex] >= 0 && gpuUtilizations[outerIndex][innerIndex] <= 100) ? 'success' : 'error'
    //     }

    // const resetSubmissionForm = () => {
    //     setWorkloadTitle('');
    //     setWorkloadSeed('');
    //     setWorkloadSeedIsValid(true);
    //     // setIsWorkloadDataDropdownOpen(false);
    //     // setSelectedWorkloadTemplate("");
    //     setDebugLoggingEnabled(true);

    //     setSessionId('');
    //     setSessionStartTick(SessionStartTickDefault);
    //     setSessionStopTick(SessionStopTickDefault);
    //     setTrainingStartTick(TrainingStartTickDefault);
    //     setTrainingDurationInTicks(TrainingDurationInTicksDefault);
    //     setTrainingCpuPercentUtil(TrainingCpuPercentUtilDefault);
    //     setTrainingMemUsageGb(TrainingMemUsageGbDefault);
    //     setTimescaleAdjustmentFactor(TimeAdjustmentFactorDefault);

    //     setGpuUtilizations(DefaultGpuUtilizations);

    //     setNumberOfGPUs(NumberOfGpusDefault);
    //     // setNumberOfGPUsString("1");

    //     defaultWorkloadTitle.current = uuidv4();
    //     defaultSessionId.current = uuidv4();
    //     setWorkloadTitleIsValid(true);
    //     setSessionIdIsValid(true);
    // }

    return (
        <Modal
            variant={ModalVariant.medium}
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
                <Button key="submit-workload-from-template-button" variant="primary" onClick={handleSubmit(onSubmit)}>
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
                        clearErrors()
                        handleSubmit(onSubmit)
                    }
                }>
                    <Grid hasGutter md={12}>
                        <GridItem span={12}>
                            <FormGroup
                                label="Workload name:"
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
                                    control={control}
                                    rules={{minLength: 1, maxLength: 36, required: true}}
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
                                        validated={getValues("workloadTitle").length >= 1 && getValues("workloadTitle").length <= 36 ? 'success' : 'error'}
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
                                        >
                                            Provide a title to help you identify the workload. The title must be between 1
                                            and 36 characters in length.
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
                                    control={control}
                                    defaultValue={WorkloadSeedDefault}
                                    rules={{ max: WorkloadSeedMax, min: WorkloadSeedMin }}
                                    render={({ field }) => <NumberInput
                                        inputName='workload-seed-number-input'
                                        id="workload-seed-number-input"
                                        type="number"
                                        inputProps={{ innerRef: field.ref }}
                                        min={WorkloadSeedMin}
                                        max={WorkloadSeedMax}
                                        onBlur={field.onBlur}
                                        onChange={field.onChange}
                                        name={field.name}
                                        value={field.value}
                                        widthChars={10}
                                        aria-label="Text input for the 'timescale adjustment factor'"
                                        onPlus={() => {
                                            const curr: number = getValues("workloadSeed") as number;
                                            let next: number = curr + WorkloadSeedDelta;
                                            next = clamp(next, WorkloadSeedMin, WorkloadSeedMax);
                                            next = roundToThreeDecimalPlaces(next);

                                            setValue("workloadSeed", next);
                                        }}
                                        onMinus={() => {
                                            const curr: number = getValues("workloadSeed") as number;
                                            let next: number = curr - WorkloadSeedDelta;
                                            next = clamp(next, WorkloadSeedMin, WorkloadSeedMax);
                                            next = roundToThreeDecimalPlaces(next);

                                            setValue("workloadSeed", next);
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
                                    control={control}
                                    defaultValue={TimeAdjustmentFactorDefault}
                                    rules={{ max: TimescaleAdjustmentFactorMax, min: TimescaleAdjustmentFactorMin }}
                                    render={({ field }) => <NumberInput
                                        inputName='timescale-adjustment-factor-number-input'
                                        id="timescale-adjustment-factor-number-input"
                                        type="number"
                                        inputProps={{ innerRef: field.ref }}
                                        aria-label="Text input for the 'timescale adjustment factor'"
                                        onBlur={field.onBlur}
                                        onChange={field.onChange}
                                        name={field.name}
                                        value={field.value}
                                        min={TimescaleAdjustmentFactorMin}
                                        max={TimescaleAdjustmentFactorMax}
                                        onPlus={() => {
                                            const curr: number = getValues("timescaleAdjustmentFactor") as number;
                                            let next: number = curr + TimescaleAdjustmentFactorDelta;

                                            next = roundToThreeDecimalPlaces(next);

                                            setValue("timescaleAdjustmentFactor", clamp(next, TimescaleAdjustmentFactorMin, TimescaleAdjustmentFactorMax));
                                        }}
                                        onMinus={() => {
                                            const curr: number = getValues("timescaleAdjustmentFactor") as number;
                                            let next: number = curr - TimescaleAdjustmentFactorDelta;

                                            // For the timescale adjustment factor, we don't want to decrement it to 0.
                                            if (next == 0) {
                                                next = TimescaleAdjustmentFactorMin;
                                            }

                                            next = roundToThreeDecimalPlaces(next);

                                            setValue("timescaleAdjustmentFactor", clamp(next, TimescaleAdjustmentFactorMin, TimescaleAdjustmentFactorMax));
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
                                {/* <Switch
                                    id="debug-logging-switch-template"
                                    label="Debug logging enabled"
                                    labelOff="Debug logging disabled"
                                    aria-label="debug-logging-switch-template"
                                    isChecked={debugLoggingEnabled}
                                    ouiaId="DebugLoggingSwitchTemplate"
                                    onChange={(_event: React.FormEvent<HTMLInputElement>, checked: boolean) => {
                                        console.log(`Setting debug logging to ${checked}`)
                                        setDebugLoggingEnabled(checked);
                                    }}
                                /> */}
                            </FormGroup>
                        </GridItem>
                    </Grid>
                    <Divider />
                </Form>
            </FormSection>
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
                    {sessionTabs.map((tabName: string, tabIndex: number) => (
                        <Tab
                            key={tabIndex}
                            eventKey={tabIndex}
                            aria-label={`${tabName} Tab`}
                            title={<TabTitleText>{tabName}</TabTitleText>}
                            closeButtonAriaLabel={`Close ${tabName} Tab`}
                            isCloseDisabled={sessionTabs.length == 1} // Can't close the last session.
                        >
                            <Card isCompact isRounded isFlat>
                                <CardBody>
                                    {/* <SessionConfigurationForm /> */}
                                </CardBody>
                            </Card>
                        </Tab>
                    ))}
                </Tabs>
            </FormSection>
        </Modal >
    );
};
