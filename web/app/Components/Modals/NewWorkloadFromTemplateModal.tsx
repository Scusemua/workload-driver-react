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
import { WorkloadPreset } from '@app/Data';
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
        template: string,
        workloadSeed: string,
        debugLoggingEnabled: boolean,
    ) => void;
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
    const [sessionStartTick, setSessionStartTick] = React.useState('4');
    const [sessionStopTick, setSessionStopTick] = React.useState('16');
    const [trainingStartTick, setTrainingStartTick] = React.useState('8');
    const [trainingDurationInTicks, setTrainingDurationInTicks] = React.useState('4');
    const [trainingCpuPercentUtil, setTrainingCpuPercentUtil] = React.useState('10.0');
    const [trainingMemUsageGb, setTrainingMemUsageGb] = React.useState('0.25');

    // const gpuUtilizations = React.useRef<string[]>(["0.0", "0.0", "0.0", "0.0", "0.0", "0.0", "0.0", "0.0"]);
    const [gpuUtilizations, setGpuUtilizations] = React.useState<string[]>(["0.0", "0.0", "0.0", "0.0", "0.0", "0.0", "0.0", "0.0"]);

    const [numberOfGPUs, setNumberOfGPUs] = React.useState<number>(1);

    const defaultWorkloadTitle = React.useRef(uuidv4());
    const defaultSessionId = React.useRef(uuidv4());

    // const { workloadPresets } = useWorkloadPresets();

    const workloadTemplates: string[] = ["1 Session with 1 Training Event"];

    const setGpuUtil = (idx: number, val: string) => {
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

        return false;
    };

    // Called when the 'submit' button is clicked.
    const onSubmitWorkload = () => {
        // If the user left the workload title blank, then use the default workload title, which is a randomly-generated UUID.
        let workloadTitleToSubmit: string = workloadTitle;
        if (workloadTitleToSubmit.length == 0) {
            workloadTitleToSubmit = defaultWorkloadTitle.current;
        }

        props.onConfirm(workloadTitleToSubmit, selectedWorkloadTemplate, workloadSeed, debugLoggingEnabled);

        // Reset all of the fields.
        setSelectedWorkloadTemplate("");
        setWorkloadSeed('');
        setWorkloadTitle('');
        setSessionId('');
        setSessionIdIsValid(false);
        setWorkloadTitleIsValid(false);

        defaultWorkloadTitle.current = uuidv4();
        defaultSessionId.current = uuidv4();
    };

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
                        label="Workload preset:"
                        labelIcon={
                            <Popover
                                aria-label="workload-preset-text-header"
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
                                    aria-describedby="simple-form-workload-preset-01"
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
                            label="workload-preset-dropdown-input-helper"
                            aria-label="workload-preset-dropdown-input-helper"
                        >
                            <HelperText
                                label="workload-preset-dropdown-input-helper"
                                aria-label="workload-preset-dropdown-input-helper"
                            >
                                <HelperTextItem
                                    aria-label="workload-preset-dropdown-input-helper"
                                    label="workload-preset-dropdown-input-helper"
                                >
                                    Select a configuration/data preset for the workload.
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
                                            aria-describedby="simple-form-workload-preset-01"
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
                                <TextInput
                                    isRequired
                                    type="number"
                                    id="session-start-tick-text-input"
                                    name="session-start-tick-text-input"
                                    value={sessionStartTick}
                                    validated={(parseFloat(sessionStartTick) >= 0 && parseFloat(sessionStartTick) < parseFloat(trainingStartTick) && parseFloat(sessionStartTick) < parseFloat(sessionStopTick)) ? 'success' : 'error'}
                                    onChange={(_event, val: string) => setSessionStartTick(val)}
                                >
                                </TextInput>
                            </FormGroup>
                        </GridItem>
                        <GridItem span={3}>
                            <FormGroup label="Training Start Tick">
                                <TextInput
                                    isRequired
                                    type="number"
                                    id="training-start-tick-text-input"
                                    name="training-start-tick-text-input"
                                    value={trainingStartTick}
                                    validated={(parseFloat(trainingStartTick) >= 0 && parseFloat(sessionStartTick) < parseFloat(trainingStartTick) && parseFloat(trainingStartTick) < parseFloat(sessionStopTick) && parseFloat(trainingStartTick) + parseFloat(trainingDurationInTicks) < parseFloat(sessionStopTick)) ? 'success' : 'error'}
                                    onChange={(_event, val: string) => setTrainingStartTick(val)}
                                >
                                </TextInput>
                            </FormGroup>
                        </GridItem>
                        <GridItem span={3}>
                            <FormGroup label="Training Duration (Ticks)">
                                <TextInput
                                    isRequired
                                    type="number"
                                    id="training-duration-ticks-text-input"
                                    name="training-duration-ticks-text-input"
                                    value={trainingDurationInTicks}
                                    validated={(parseFloat(trainingDurationInTicks) >= 0 && parseFloat(trainingStartTick) + parseFloat(trainingDurationInTicks) < parseFloat(sessionStopTick)) ? 'success' : 'error'}
                                    onChange={(_event, val: string) => setTrainingDurationInTicks(val)}
                                >
                                </TextInput>
                            </FormGroup>
                        </GridItem>
                        <GridItem span={3}>
                            <FormGroup label="Session Stop Tick">
                                <TextInput
                                    isRequired
                                    type="number"
                                    id="session-stop-tick-text-input"
                                    name="session-stop-tick-text-input"
                                    value={sessionStopTick}
                                    validated={(parseFloat(sessionStopTick) >= 0 && parseFloat(trainingStartTick) < parseFloat(sessionStopTick) && parseFloat(sessionStartTick) < parseFloat(sessionStopTick) && parseFloat(trainingStartTick) + parseFloat(trainingDurationInTicks) < parseFloat(sessionStopTick)) ? 'success' : 'error'}
                                    onChange={(_event, val: string) => setSessionStopTick(val)}
                                >
                                </TextInput>
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
                                <TextInput
                                    isRequired
                                    type="number"
                                    customIcon={<CpuIcon />}
                                    id="training-cpu-percent-utilt-text-input"
                                    name="training-cpu-percent-utilt-text-input"
                                    value={trainingCpuPercentUtil}
                                    validated={(parseFloat(trainingCpuPercentUtil) >= 0 && parseFloat(trainingCpuPercentUtil) <= 100) ? 'success' : 'error'}
                                    onChange={(_event, val: string) => setTrainingCpuPercentUtil(val)}
                                >
                                </TextInput>
                            </FormGroup>
                        </GridItem>
                        <GridItem span={3}>
                            <FormGroup label="RAM Usage (GB)">
                                <TextInput
                                    isRequired
                                    type="number"
                                    customIcon={<MemoryIcon />}
                                    id="training-mem-usage-gb-text-input"
                                    name="training-mem-usage-gb-text-input"
                                    value={trainingMemUsageGb}
                                    validated={(parseFloat(trainingMemUsageGb) >= 0 && parseFloat(trainingMemUsageGb) <= 128_000) ? 'success' : 'error'}
                                    onChange={(_event, val: string) => setTrainingMemUsageGb(val)}
                                />
                            </FormGroup>
                        </GridItem>
                        <GridItem span={6}>
                            <FormGroup label="Adjust the Number of GPUs">
                                <Grid hasGutter>
                                    <GridItem span={2}>
                                        <Button variant='plain' disabled={numberOfGPUs <= 1} onClick={() => setNumberOfGPUs(Math.max(1, numberOfGPUs - 1))}>Remove GPU <MinusCircleIcon /></Button>
                                    </GridItem>
                                    <GridItem span={2} offset={6}>
                                        <Button variant='plain' disabled={numberOfGPUs >= 8} onClick={() => setNumberOfGPUs(Math.min(8, numberOfGPUs + 1))}>Add GPU <PlusCircleIcon /></Button>
                                    </GridItem>
                                </Grid>
                            </FormGroup>
                        </GridItem>
                        {Array.from({ length: numberOfGPUs }).map((_, idx: number) => {
                            return (
                                <GridItem span={3} rowSpan={1} hidden={numberOfGPUs < idx}>
                                    <FormGroup label={`GPU #${idx} % Utilization`}>
                                        <TextInput
                                            type="number"
                                            customIcon={<GpuIcon />}
                                            id={`gpu${idx}-percent-util-text-input`}
                                            name={`gpu${idx}-percent-util-text-input`}
                                            value={gpuUtilizations[idx]}
                                            onChange={(_event, val: string) => setGpuUtil(idx, val)}
                                            validated={(parseFloat(gpuUtilizations[idx]) >= 0 && parseFloat(gpuUtilizations[idx]) <= 100) ? 'success' : 'error'}
                                        >
                                        </TextInput>
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
