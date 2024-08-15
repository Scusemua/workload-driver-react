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
    Tooltip,
    ValidatedOptions,
    Flex,
    FlexItem,
} from '@patternfly/react-core';

import { v4 as uuidv4 } from 'uuid';
import HelpIcon from '@patternfly/react-icons/dist/esm/icons/help-icon';
import styles from '@patternfly/react-styles/css/components/Form/form';

import { CpuIcon, MemoryIcon, MinusCircleIcon, PlusCircleIcon, SyncIcon } from '@patternfly/react-icons';
import { Session, TrainingEvent, WorkloadPreset, WorkloadTemplate } from '@app/Data';
import { useWorkloadPresets } from '@providers/WorkloadPresetProvider';
import { PlusIcon } from '@patternfly/react-icons';
import { GpuIcon } from '@app/Icons';
import { isArrayOfString } from '@patternfly/react-log-viewer/dist/js/LogViewer/utils/utils';

export interface NewWorkloadFromTemplateModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    onConfirm: (
        workloadTitle: string,
        workloadSeed: string,
        debugLoggingEnabled: boolean,
        workloadTemplate: WorkloadTemplate,
    ) => void;
}

function assertIsNumber(value: number | ''): asserts value is number {
    if (value === '') {
        throw new Error("value is not number");
    }
}

export const NewWorkloadFromTemplateModal: React.FunctionComponent<NewWorkloadFromTemplateModalProps> = (props) => {
    const [workloadTitle, setWorkloadTitle] = React.useState('');
    const [workloadTitleIsValid, setWorkloadTitleIsValid] = React.useState(true);
    const [sessionIdIsValid, setSessionIdIsValid] = React.useState(true);
    const [workloadSeed, setWorkloadSeed] = React.useState('');
    const [workloadSeedIsValid, setWorkloadSeedIsValid] = React.useState(true);
    const [isWorkloadDataDropdownOpen, setIsWorkloadDataDropdownOpen] = React.useState(false);
    const [selectedWorkloadTemplate, setSelectedWorkloadTemplate] = React.useState<string>("1 Session with 1 Training Event");
    const [debugLoggingEnabled, setDebugLoggingEnabled] = React.useState(true);

    const [sessionId, setSessionId] = React.useState('');
    const [sessionStartTick, setSessionStartTick] = React.useState<number | ''>(4);
    const [sessionStopTick, setSessionStopTick] = React.useState<number | ''>(16);
    const [trainingStartTick, setTrainingStartTick] = React.useState<number | ''>(8);
    const [trainingDurationInTicks, setTrainingDurationInTicks] = React.useState<number | ''>(4);
    const [trainingCpuPercentUtil, setTrainingCpuPercentUtil] = React.useState<number | ''>(10.0);
    const [trainingMemUsageGb, setTrainingMemUsageGb] = React.useState<number | ''>(0.25);

    // const gpuUtilizations = React.useRef<string[]>(["0.0", "0.0", "0.0", "0.0", "0.0", "0.0", "0.0", "0.0"]);
    const [gpuUtilizations, setGpuUtilizations] = React.useState<(number | '')[]>([100.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0]);

    const [numberOfGPUs, setNumberOfGPUs] = React.useState<number | ''>(1);
    // const [numberOfGPUsString, setNumberOfGPUsString] = React.useState<string>("1");

    const defaultWorkloadTitle = React.useRef(uuidv4());
    const defaultSessionId = React.useRef(uuidv4());

    // const { workloadPresets } = useWorkloadPresets();

    const workloadTemplates: string[] = ["1 Session with 1 Training Event"];

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

    const handleWorkloadTitleChanged = (_event, title: string) => {
        setWorkloadTitle(title);
        setWorkloadTitleIsValid(title.length >= 0 && title.length <= 36);
    };

    const handleSessionIdChanged = (_event, id: string) => {
        setSessionId(id);
        setSessionIdIsValid(id.length >= 0 && id.length <= 36);
    };

    const handleWorkloadSeedChanged = (_event, seed: string) => {
        const validSeed: boolean = /[0-9]/.test(seed) || seed == '';

        // If it's either the empty string, or we can't even convert the value to a number,
        // then update the state accordingly.
        if (!validSeed || seed == '') {
            setWorkloadSeedIsValid(validSeed);
            setWorkloadSeed('');
            return;
        }

        // Convert to a number.
        const parsed: number = parseInt(seed, 10);

        // If it's a float or something, then just default to no seed.
        if (Number.isNaN(parsed)) {
            setWorkloadSeed('');
            return;
        }

        // If it's greater than the max value, then it is invalid.
        if (parsed > 2147483647 || parsed < 0) {
            setWorkloadSeedIsValid(false);
            setWorkloadSeed(seed);
            return;
        }

        setWorkloadSeed(parsed.toString());
        setWorkloadSeedIsValid(true);
    };

    const onWorkloadDataDropdownToggleClick = () => {
        setIsWorkloadDataDropdownOpen(!isWorkloadDataDropdownOpen);
    };

    const onWorkloadDataDropdownSelect = (
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined,
    ) => {
        // eslint-disable-next-line no-console
        console.log('selected', value);

        console.log(`Value: ${value}`)
        if (value != undefined) {
            setSelectedWorkloadTemplate(workloadTemplates[value]);
        } else {
            setSelectedWorkloadTemplate("");
        }
        setIsWorkloadDataDropdownOpen(false);
    };

    const getWorkloadSeedValidatedState = () => {
        if (!workloadSeedIsValid) {
            return ValidatedOptions.error;
        }

        if (workloadSeed == '') {
            return ValidatedOptions.default;
        }

        return ValidatedOptions.success;
    };

    const isSubmitButtonDisabled = () => {
        if (!workloadTitleIsValid) {
            return true;
        }

        if (setSelectedWorkloadTemplate.length == 0) {
            return true;
        }

        if (!workloadSeedIsValid) {
            return true;
        }

        if (sessionStartTick === '' || trainingStartTick === '' || sessionStopTick === '' || trainingDurationInTicks === '' || trainingMemUsageGb === '' || trainingCpuPercentUtil === '') {
            return true;
        }

        // The following are all the conditions from the `validated` fields of the text inputs.
        if (validateNumberOfGpusInput() !== 'success' || validateTrainingMemoryUsageInput() !== 'success' || validateTrainingCpuInput() !== 'success' || validateTrainingDurationInTicksInput() !== 'success' || validateTrainingStartTickInput() !== 'success' || validateSessionStartStopInput() !== 'success' || validateSessionStartTickInput() !== 'success') {
            return true;
        }

        // This one might be redundant. 
        if (numberOfGPUs === '' || numberOfGPUs < 0 || numberOfGPUs > 8) {
            return true;
        }
        
        const numGPUs: number = (numberOfGPUs || 1);
        for (let i = 0; i < numGPUs; i++) {
            if (validateGpuUtilInput(i) !== 'success') {
                return true;
            }
        }

        return false;
    };

    function assertGpuUtilizationsAreAllNumbers(value: (number | '')[], numGPUs: number): asserts value is number[] {
        for (let i = 0; i < numGPUs; i++) {
            if (validateGpuUtilInput[i] === 'error') {
                console.error(`gpuUtilization[${i}] is not a valid value during submission.`)
                throw new Error(`gpuUtilization[${i}] is not a valid value during submission.`)
            }
        }
    }

    // Called when the 'submit' button is clicked.
    const onSubmitWorkload = () => {
        // If the user left the workload title blank, then use the default workload title, which is a randomly-generated UUID.
        let workloadTitleToSubmit: string = workloadTitle;
        if (workloadTitleToSubmit.length == 0) {
            workloadTitleToSubmit = defaultWorkloadTitle.current;
        }

        assertIsNumber(trainingCpuPercentUtil);
        assertIsNumber(trainingMemUsageGb);
        assertIsNumber(numberOfGPUs);
        assertIsNumber(sessionStartTick);
        assertIsNumber(sessionStopTick);
        assertIsNumber(trainingDurationInTicks);
        assertIsNumber(trainingStartTick);
        assertGpuUtilizationsAreAllNumbers(gpuUtilizations, numberOfGPUs);

        // TOOD: 
        // When we have multiplate templates, we'll add template-specific submission logic
        // to aggregate the information from that template and convert it to a valid
        // workload registration request.

        const trainingEvent: TrainingEvent = {
            sessionId: sessionId,
            trainingId: uuidv4(),
            cpuUtil: trainingCpuPercentUtil,
            memUsageGb: trainingMemUsageGb,
            gpuUtil: gpuUtilizations,
            startTick: trainingStartTick,
            durationInTicks: trainingDurationInTicks,
        }

        const session: Session = {
            id: sessionId,
            maxCPUs: trainingCpuPercentUtil,
            maxMemoryGB: trainingMemUsageGb,
            maxNumGPUs: numberOfGPUs,
            startTick: sessionStartTick,
            stopTick: sessionStopTick,
            trainings: [trainingEvent],
        }

        const sessions: Session[] = [session]

        const workloadTemplate: WorkloadTemplate = {
            name: selectedWorkloadTemplate,
            sessions: sessions,
        }

        // TODO: Create and pass sessions.
        props.onConfirm(workloadTitleToSubmit, workloadSeed, debugLoggingEnabled, workloadTemplate);

        // Reset all of the fields.
        resetSubmissionForm();
    };

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

    const validateGpuUtilInput = (idx: number) => {
        return (gpuUtilizations[idx] !== '' && gpuUtilizations[idx] >= 0 && gpuUtilizations[idx] <= 100) ? 'success' : 'error'
    }

    const resetSubmissionForm = () => {
        setWorkloadTitle('');
        setWorkloadSeed('');
        setWorkloadSeedIsValid(true);
        setIsWorkloadDataDropdownOpen(false);
        setSelectedWorkloadTemplate("");
        setDebugLoggingEnabled(true);

        setSessionId('');
        setSessionStartTick(4);
        setSessionStopTick(16);
        setTrainingStartTick(8);
        setTrainingDurationInTicks(4);
        setTrainingCpuPercentUtil(10.0);
        setTrainingMemUsageGb(0.25);

        setGpuUtilizations([100.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0]);

        setNumberOfGPUs(1);
        // setNumberOfGPUsString("1");

        defaultWorkloadTitle.current = uuidv4();
        defaultSessionId.current = uuidv4();
        setWorkloadTitleIsValid(true);
        setSessionIdIsValid(true);
    }

    return (
        <Modal
            variant={ModalVariant.medium}
            titleIconVariant={'info'}
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
                <Button key="submit" variant="primary" onClick={onSubmitWorkload} isDisabled={isSubmitButtonDisabled()}>
                    Submit
                </Button>,
                <Button key="cancel" variant="link" onClick={props.onClose}>
                    Cancel
                </Button>,
            ]}
        >
            <FormSection title="Select Workload Template" titleElement='h1'>
                <Form>
                    <FormGroup
                        label="Workload template:"
                        labelIcon={
                            <Popover
                                aria-label="workload-template-text-header"
                                headerContent={<div>Workload Preset</div>}
                                bodyContent={
                                    <div>
                                        Select the preprocessed data to use for driving the workload. This largely
                                        determines which subset of trace data will be used to generate the workload.
                                    </div>
                                }
                            >
                                <button
                                    type="button"
                                    aria-label="Select the preprocessed data to use for driving the workload. This largely determines which subset of trace data will be used to generate the workload."
                                    onClick={(e) => e.preventDefault()}
                                    aria-describedby="simple-form-workload-template-01"
                                    className={styles.formGroupLabelHelp}
                                >
                                    <HelpIcon />
                                </button>
                            </Popover>
                        }
                    >
                        <Dropdown
                            aria-label="workload-presetset-dropdown-menu"
                            isScrollable
                            isOpen={isWorkloadDataDropdownOpen}
                            onSelect={onWorkloadDataDropdownSelect}
                            onOpenChange={(isOpen: boolean) => setIsWorkloadDataDropdownOpen(isOpen)}
                            toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                                <MenuToggle
                                    ref={toggleRef}
                                    isFullWidth
                                    onClick={onWorkloadDataDropdownToggleClick}
                                    isExpanded={isWorkloadDataDropdownOpen}
                                >
                                    {selectedWorkloadTemplate}
                                </MenuToggle>
                            )}
                            shouldFocusToggleOnSelect
                        >
                            <DropdownList aria-label="workload-presetset-dropdown-list">
                                <DropdownItem
                                    aria-label={'workload-template-single-session-single-training'}
                                    value={0}
                                    key={"1 Session with 1 Training Event"}
                                    description={"1 Session with 1 Training Event"}
                                >
                                    {"1 Session with 1 Training Event"}
                                </DropdownItem>
                            </DropdownList>
                        </Dropdown>
                        <FormHelperText
                            label="workload-template-dropdown-input-helper"
                            aria-label="workload-template-dropdown-input-helper"
                        >
                            <HelperText
                                label="workload-template-dropdown-input-helper"
                                aria-label="workload-template-dropdown-input-helper"
                            >
                                <HelperTextItem
                                    aria-label="workload-template-dropdown-input-helper"
                                    label="workload-template-dropdown-input-helper"
                                >
                                    Select a template for the workload.
                                </HelperTextItem>
                            </HelperText>
                        </FormHelperText>
                    </FormGroup>
                    <Divider inset={{ 'default': 'insetXl' }} />
                </Form>
            </FormSection>
            <FormSection title="Generic Workload Parameters" titleElement='h1' hidden={selectedWorkloadTemplate == ""}>
                <Form>
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
                                <TextInput
                                    isRequired
                                    label="workload-title-text-input"
                                    aria-label="workload-title-text-input"
                                    type="text"
                                    id="workload-title-text-input"
                                    name="workload-title-text-input"
                                    aria-describedby="workload-title-text-input-helper"
                                    value={workloadTitle}
                                    placeholder={defaultWorkloadTitle.current}
                                    validated={(workloadTitleIsValid && ValidatedOptions.success) || ValidatedOptions.error}
                                    onChange={handleWorkloadTitleChanged}
                                />
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
                        <GridItem span={8}>
                            <FormGroup
                                label="Workload Seed:"
                                labelIcon={
                                    <Popover
                                        aria-label="workload-seed-popover"
                                        headerContent={<div>Workload Title</div>}
                                        bodyContent={
                                            <div>
                                                This is an integer seed for the random number generator used by the workload
                                                generator. You may leave this blank to refrain from seeding the random
                                                number generator. Please note that if you do specify a seed, then the value
                                                must be between 0 and 2,147,483,647.
                                            </div>
                                        }
                                    >
                                        <button
                                            type="button"
                                            aria-label="This is an integer seed (between 0 and 2,147,483,647) for the random number generator used by the workload generator. You may leave this blank to refrain from seeding the random number generator."
                                            onClick={(e) => e.preventDefault()}
                                            aria-describedby="simple-form-workload-seed-01"
                                            className={styles.formGroupLabelHelp}
                                        >
                                            <HelpIcon />
                                        </button>
                                    </Popover>
                                }
                            >
                                <TextInput
                                    isRequired
                                    label="workload-seed-text-input"
                                    aria-label="workload-seed-text-input"
                                    type="number"
                                    id="workload-seed-text-input"
                                    name="workload-seed-text-input"
                                    placeholder="No seed"
                                    value={workloadSeed}
                                    aria-describedby="workload-seed-text-input-helper"
                                    validated={getWorkloadSeedValidatedState()}
                                    onChange={handleWorkloadSeedChanged}
                                />
                                <FormHelperText
                                    label="workload-seed-text-input-helper"
                                    aria-label="workload-seed-text-input-helper"
                                >
                                    <HelperText
                                        label="workload-seed-text-input-helper"
                                        aria-label="workload-seed-text-input-helper"
                                    >
                                        <HelperTextItem
                                            aria-label="workload-seed-text-input-helper"
                                            label="workload-seed-text-input-helper"
                                        >
                                            Provide an optional integer seed for the workload&apos;s
                                            random number generator.
                                        </HelperTextItem>
                                    </HelperText>
                                </FormHelperText>
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
                                            aria-describedby="simple-form-workload-template-01"
                                            className={styles.formGroupLabelHelp}
                                        >
                                            <HelpIcon />
                                        </button>
                                    </Popover>
                                }
                            >
                                <Switch
                                    id="debug-logging-switch"
                                    label="Debug logging enabled"
                                    labelOff="Debug logging disabled"
                                    aria-label="debug-logging-switch"
                                    isChecked={debugLoggingEnabled}
                                    ouiaId="DebugLoggingSwitch"
                                    onChange={(_event: React.FormEvent<HTMLInputElement>, checked: boolean) => {
                                        setDebugLoggingEnabled(checked);
                                    }}
                                />
                            </FormGroup>
                        </GridItem>
                    </Grid>
                    <Divider inset={{ 'default': 'insetXl' }} />
                </Form>
            </FormSection>
            <FormSection title={`General Session Parameters`} titleElement='h1' hidden={selectedWorkloadTemplate != workloadTemplates[0]}>
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
                                    min={0}
                                />
                            </FormGroup>
                        </GridItem>
                    </Grid>
                    <Divider inset={{ 'default': 'inset3xl' }} />
                </Form>
            </FormSection>
            <FormSection title={`Configure Session Resource Utilization`} titleElement='h1' hidden={selectedWorkloadTemplate != workloadTemplates[0]}>
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
                                            inputAriaLabel="num-gpus-input"
                                            minusBtnAriaLabel="minus"
                                            plusBtnAriaLabel="plus"
                                            validated={validateNumberOfGpusInput()}
                                            min={1}
                                            max={8}
                                        />
                                    </GridItem>
                                </Grid>
                            </FormGroup>
                        </GridItem>
                        {Array.from({ length: Math.max(Math.min((numberOfGPUs || 1), 8), 1) }).map((_, idx: number) => {
                            return (
                                <GridItem span={3} rowSpan={1} hidden={(numberOfGPUs || 1) < idx}>
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
        </Modal >
    );
};
