import { CodeEditorComponent } from '@Components/CodeEditor';
import { Session, TrainingEvent, WorkloadTemplate } from '@src/Data';
import { SessionTabsDataContext } from '@src/Providers';
import {
    CodeContext,
    GetDefaultFormValues,
    NumberOfSessionsDefault,
    NumberOfSessionsMax,
    NumberOfSessionsMin,
    RoundToThreeDecimalPlaces,
    SessionConfigurationForm,
} from '@Components/Modals';
import { Language } from '@patternfly/react-code-editor';
import {
    Button,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    FormHelperText,
    FormSection,
    Grid,
    GridItem,
    HelperText,
    HelperTextItem,
    MultipleFileUpload,
    MultipleFileUploadMain,
    MultipleFileUploadStatus,
    MultipleFileUploadStatusItem,
    NumberInput,
    Popover,
    Switch,
    TextInput,
    Tooltip,
} from '@patternfly/react-core';
import { Modal, ModalVariant } from '@patternfly/react-core/deprecated';
import { DropEvent } from '@patternfly/react-core/src/helpers/typeUtils';
import { CodeIcon, DownloadIcon, PencilAltIcon, SaveAltIcon, TrashAltIcon, UploadIcon } from '@patternfly/react-icons';
import HelpIcon from '@patternfly/react-icons/dist/esm/icons/help-icon';
import styles from '@patternfly/react-styles/css/components/Form/form';
import React from 'react';
import { FileRejection } from 'react-dropzone';

import { Controller, FormProvider, useForm } from 'react-hook-form';
import toast from 'react-hot-toast';

import { v4 as uuidv4 } from 'uuid';
import {
    TimeAdjustmentFactorDefault,
    TimescaleAdjustmentFactorDelta,
    TimescaleAdjustmentFactorMax,
    TimescaleAdjustmentFactorMin,
    WorkloadSeedDefault,
    WorkloadSeedDelta,
    WorkloadSeedMax,
    WorkloadSeedMin,
} from './Constants';

export interface NewWorkloadFromTemplateModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    onConfirm: (workloadName: string, workloadRegistrationRequestJson: string) => void;
}

// Clamp a value between two extremes.
function clamp(value: number, min: number, max: number) {
    return Math.max(Math.min(value, max), min);
}

interface readFile {
    fileName: string;
    data?: string;
    loadResult?: 'danger' | 'success';
    loadError?: DOMException;
}

// Important: this component must be wrapped in a <SessionTabsDataProvider></SessionTabsDataProvider>!
export const NewWorkloadFromTemplateModal: React.FunctionComponent<NewWorkloadFromTemplateModalProps> = (props) => {
    const defaultWorkloadTitle = React.useRef(uuidv4());
    const [jsonModeActive, setJsonModeActive] = React.useState<boolean>(false);
    const [isFailedUploadModalOpen, setFailedUploadModalOpen] = React.useState<boolean>(false);
    const [failedUploadModalTitleText, setFailedUploadModalTitleText] = React.useState<string>('');
    const [failedUploadModalBodyText, setFailedUploadModalBodyText] = React.useState<string>('');
    const [currentFiles, setCurrentFiles] = React.useState<File[]>([]);
    const [readFileData, setReadFileData] = React.useState<readFile[]>([]);
    const [showFileUploadStatus, setShowFileUploadStatus] = React.useState(false);
    const [fileUploadStatusIcon, setFileUploadStatusIcon] = React.useState('inProgress');
    const sessionFormRef = React.useRef<HTMLDivElement>(null);

    // Actively modified by the code editor.
    const [formAsJson, setFormAsJson] = React.useState<string>('');

    const { activeSessionTab, setActiveSessionTab, setSessionTabs, setNewSessionTabNumber } =
        React.useContext(SessionTabsDataContext);

    const form = useForm({
        mode: 'all',
        defaultValues: GetDefaultFormValues(),
    });

    const {
        formState: { isSubmitSuccessful },
    } = form;

    React.useEffect(() => {
        if (isSubmitSuccessful) {
            console.log('Resetting form to default values.');
            form.reset(GetDefaultFormValues());
        }
    }, [form, isSubmitSuccessful]);

    // only show the status component once a file has been uploaded, but keep the status list component itself even if all files are removed
    if (!showFileUploadStatus && currentFiles.length > 0) {
        setShowFileUploadStatus(true);
    }

    // determine the icon that should be shown for the overall status list
    React.useEffect(() => {
        if (readFileData.length < currentFiles.length) {
            setFileUploadStatusIcon('inProgress');
        } else if (readFileData.every((file) => file.loadResult === 'success')) {
            setFileUploadStatusIcon('success');
        } else {
            setFileUploadStatusIcon('danger');
        }
    }, [readFileData, currentFiles]);

    // callback called by the status item when a file encounters an error while being read with the built-in file reader
    const handleReadFail = (error: DOMException, file: File) => {
        setReadFileData((prevReadFiles) => [
            ...prevReadFiles,
            { loadError: error, fileName: file.name, loadResult: 'danger' },
        ]);
    };

    // callback called by the status item when a file is successfully read with the built-in file reader
    const handleReadSuccess = (data: string, file: File) => {
        setReadFileData((prevReadFiles) => [...prevReadFiles, { data, fileName: file.name, loadResult: 'success' }]);
    };

    // remove files from both state arrays based on their name
    const removeFiles = (namesOfFilesToRemove: string[]) => {
        console.log(`Removing file(s) from current files: ${JSON.stringify(namesOfFilesToRemove)}`);
        const newCurrentFiles = currentFiles.filter(
            (currentFile) => !namesOfFilesToRemove.some((fileName) => fileName === currentFile.name),
        );

        setCurrentFiles(newCurrentFiles);

        const newReadFiles = readFileData.filter(
            (readFile) => !namesOfFilesToRemove.some((fileName) => fileName === readFile.fileName),
        );

        setReadFileData(newReadFiles);
    };

    const parseData = (data, space: string | number | undefined = undefined) => {
        const workloadTitle: string = data.workloadTitle;
        const workloadSeedString: string = data.workloadSeed;
        const debugLoggingEnabled: boolean = data.debugLoggingEnabled;
        const timescaleAdjustmentFactor: number = data.timescaleAdjustmentFactor;

        const sessions: Session[] = data.sessions;

        for (let i: number = 0; i < sessions.length; i++) {
            const session: Session = sessions[i];
            const trainings: TrainingEvent[] = session.trainings;

            let max_millicpus: number = -1;
            let max_mem_mb: number = -1;
            let max_num_gpus: number = -1;
            let max_vram_gb: number = -1;
            for (let j: number = 0; j < trainings.length; j++) {
                const training: TrainingEvent = trainings[j];
                training.training_index = j; // Set the training index field.

                if (training.millicpus > max_millicpus) {
                    max_millicpus = training.millicpus;
                }

                if (training.mem_usage_mb > max_mem_mb) {
                    max_mem_mb = training.mem_usage_mb;
                }

                if (training.vram_usage_gb > max_vram_gb) {
                    max_vram_gb = training.vram_usage_gb;
                }

                if (training.gpu_utilizations.length > max_num_gpus) {
                    max_num_gpus = training.gpu_utilizations.length;
                }
            }

            // Construct the resource request and update the session object.
            session.resource_request = {
                cpus: max_millicpus,
                gpus: max_num_gpus,
                memory_mb: max_mem_mb,
                vram: max_vram_gb,
                gpu_type: 'Any_GPU',
            };
        }

        const workloadTemplate: WorkloadTemplate = {
            sessions: data.sessions,
        };

        let workloadSeed: number = 0;
        if (workloadSeedString != '') {
            workloadSeed = parseInt(workloadSeedString);
        }

        const messageId: string = uuidv4();
        return JSON.stringify(
            {
                op: 'register_workload',
                msg_id: messageId,
                workloadRegistrationRequest: {
                    adjust_gpu_reservations: false,
                    seed: workloadSeed,
                    timescale_adjustment_factor: timescaleAdjustmentFactor,
                    key: 'workload_template_key',
                    name: workloadTitle,
                    debug_logging: debugLoggingEnabled,
                    type: 'template',
                    sessions: workloadTemplate.sessions,
                },
            },
            null,
            space,
        );
    };

    const onSubmit = (data: { workloadTitle: string }) => {
        const workloadRegistrationRequest: string = parseData(data);
        console.log(`User submitted workload template data: ${JSON.stringify(data)}`);
        props.onConfirm(data.workloadTitle, workloadRegistrationRequest);
    };

    const getWorkloadNameValidationState = () => {
        const workloadId: string = form.watch('workloadTitle');

        if (workloadId == undefined) {
            return 'default';
        }

        if (workloadId.length >= 1 && workloadId.length <= 36) {
            return 'success';
        }

        return 'error';
    };

    const isWorkloadNameValid = () => {
        const workloadId: string = form.watch('workloadTitle');

        if (workloadId == undefined) {
            // Form hasn't loaded yet.
            return true;
        }

        return workloadId.length >= 1 && workloadId.length <= 36;
    };

    const enableJsonEditorMode = () => {
        const formData = form.getValues();
        // const requestJson: string = parseData(formData, 4);
        const formJson: string = JSON.stringify(formData, null, 4);
        setFormAsJson(formJson);

        setJsonModeActive(true);
    };

    const downloadTemplateAsJson = () => {
        const formData = form.getValues();
        // const formJson: string = parseData(formData, 4);
        const formJson: string = JSON.stringify(formData, null, 4);

        console.log(`Retrieved form data: ${formJson}`);

        const element = document.createElement('a');
        const file = new Blob([formJson], { type: 'text' });
        element.href = URL.createObjectURL(file);
        element.download = `template-${Date.now().toString()}.json`;
        document.body.appendChild(element); // Required for this to work in FireFox
        element.click();
    };

    const applyJsonToForm = (jsonText: string) => {
        console.log('Attempting to apply JSON directly to form.');
        console.log(jsonText);

        const data = JSON.parse(jsonText);

        const sessionTabs: string[] = [];
        for (let i: number = 0; i < data.sessions.length; i++) {
            sessionTabs.push(`Session ${i + 1}`);
        }

        setSessionTabs(sessionTabs);
        setNewSessionTabNumber(data.sessions.length + 1);

        // If the user is currently on a tab that's getting deleted because of the application of the JSON,
        // then we'll switch to the right-most tab.
        if (activeSessionTab > data.sessions.length) {
            setActiveSessionTab(data.sessions.length - 1);
        }

        setJsonModeActive(false);
        form.reset(data);
    };

    const onDiscardJsonChangesButtonClicked = () => {
        setJsonModeActive(false);
    };

    const getSubmitButton = () => {
        if (jsonModeActive) {
            return (
                <Button
                    key="apply-json-to-template-button"
                    variant="primary"
                    onClick={() => applyJsonToForm(formAsJson)}
                    icon={<SaveAltIcon />}
                >
                    Apply Changes to Template
                </Button>
            );
        } else {
            return (
                <Button
                    key="submit-workload-from-template-button"
                    variant="primary"
                    onClick={form.handleSubmit(onSubmit)}
                >
                    Submit Workload
                </Button>
            );
        }
    };

    const getCancelButton = () => {
        if (jsonModeActive) {
            return (
                <Button
                    key="cancel-application-of-json-to-workload-from-template-button"
                    isDanger
                    variant="secondary"
                    onClick={onDiscardJsonChangesButtonClicked}
                >
                    Discard Changes
                </Button>
            );
        } else {
            return (
                <Button
                    key="cancel-submission-of-workload-from-template-button"
                    isDanger
                    variant="secondary"
                    onClick={props.onClose}
                >
                    Cancel
                </Button>
            );
        }
    };

    const onResetFormButtonClicked = () => {
        console.log('Resetting form to default values.');
        form.reset(GetDefaultFormValues());

        setSessionTabs(['Session 1']);
    };

    const getActions = () => {
        if (jsonModeActive) {
            return [getSubmitButton(), getCancelButton()];
        } else {
            return [
                getSubmitButton(),
                <Button
                    key={'switch-to-json-button'}
                    id={'switch-to-json-button'}
                    icon={<CodeIcon />}
                    variant={'primary'}
                    onClick={enableJsonEditorMode}
                >
                    Switch to JSON Editor
                </Button>,
                <Button
                    key={'reset-workload-template-form-button'}
                    id={'reset-workload-template-form-button'}
                    icon={<TrashAltIcon />}
                    variant={'warning'}
                    onClick={onResetFormButtonClicked}
                >
                    Reset Form to Default Values
                </Button>,
                getCancelButton(),
            ];
        }
    };

    const handleFileUploadRejection = (fileRejections: FileRejection[]) => {
        if (fileRejections.length === 1) {
            setFailedUploadModalBodyText(`${fileRejections[0].file.name} is not of an accepted file type.`);
            setFailedUploadModalTitleText('Unsupported File Type');
        } else {
            const rejectedMessages = fileRejections.reduce(
                (acc, fileRejection) => (acc += `${fileRejection.file.name}, `),
                '',
            );
            setFailedUploadModalBodyText(`${rejectedMessages}are not of an accepted file type.`);
            setFailedUploadModalTitleText('Unsupported File Types');
        }

        setFailedUploadModalOpen(true);
    };

    const onFileUploadedNonJsonEditor = (event: DropEvent, uploadedFiles: File[]) => {
        // identify what, if any, files are re-uploads of already uploaded files
        const currentFileNames = currentFiles.map((file) => file.name);
        const reUploads = uploadedFiles.filter((uploadedFile) => currentFileNames.includes(uploadedFile.name));

        /** this promise chain is needed because if the file removal is done at the same time as the file adding react
         * won't realize that the status items for the re-uploaded files needs to be re-rendered */
        Promise.resolve()
            .then(() => removeFiles(reUploads.map((reuploadedFile) => reuploadedFile.name)))
            .then(() => setCurrentFiles((prevFiles) => [...prevFiles, ...uploadedFiles]))
            .then(() => {
                /** this promise chain is needed because if the file removal is done at the same time as the file adding react
                 * won't realize that the status items for the re-uploaded files needs to be re-rendered */
                if (uploadedFiles.length > 1) {
                    console.error(`Too many files uploaded at once (${uploadedFiles.length}).`);
                    setFailedUploadModalBodyText(
                        `Too many files uploaded at once (${uploadedFiles.length}). Please upload a single file.`,
                    );
                    setFailedUploadModalTitleText('Too Many Files Uploaded');
                    setFailedUploadModalOpen(true);
                    return;
                }

                if (uploadedFiles[0].type != 'application/json') {
                    return;
                }

                console.log(`currentFiles: ${JSON.stringify(currentFiles)}`);

                if (uploadedFiles.length == 1) {
                    const reader = new FileReader();
                    const file: File = uploadedFiles[0];

                    reader.onload = (e: ProgressEvent<FileReader>) => {
                        if (e.target === null) {
                            return;
                        }

                        const jsonText: string = e.target.result as string;
                        try {
                            applyJsonToForm(jsonText);
                        } catch (error) {
                            console.error('Error parsing JSON:', error);
                            setFailedUploadModalBodyText(JSON.stringify(error));
                            setFailedUploadModalTitleText('Failed to Parse JSON Template');
                            setFailedUploadModalOpen(true);
                            return;
                        }

                        setFormAsJson(jsonText);

                        if (sessionFormRef && sessionFormRef.current) {
                            sessionFormRef.current.scrollIntoView({ behavior: 'smooth', block: 'start' });
                        }

                        toast.success(`Successfully uploaded and applied JSON template from file "${file.name}"`);
                    };

                    reader.readAsText(file);
                }
            });
    };

    const successfullyReadFileCount = readFileData.filter((fileData) => fileData.loadResult === 'success').length;

    return (
        <FormProvider {...form}>
            <Modal
                variant={ModalVariant.large}
                titleIconVariant={PencilAltIcon}
                aria-label="Modal to create a new workload from a template"
                title={'Create New Workload from Template'}
                isOpen={props.isOpen}
                onClose={props.onClose}
                help={
                    <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsXs' }}>
                        <FlexItem>
                            <Popover
                                headerContent={<div>Creating New Workloads from Templates</div>}
                                bodyContent={
                                    <div>
                                        You can create and register a new workload using a &quot;template&quot;. This
                                        allows for a greater degree of dynamicity in the workload&apos;s execution.
                                        <br />
                                        <br />
                                        Specifically, templates enable you to customize various properties of the
                                        workload, such as the number of sessions, the resource utilization of these
                                        sessions, when the sessions start and stop, and the training events processed by
                                        the workload&apos;s sessions.
                                    </div>
                                }
                            >
                                <Button
                                    icon={<HelpIcon />}
                                    variant="plain"
                                    aria-label="Create New Workload From Template Helper"
                                />
                            </Popover>
                        </FlexItem>
                        {!jsonModeActive && (
                            <FlexItem>
                                <Tooltip
                                    content={'Download the current version of the template to a JSON file.'}
                                    position={'bottom'}
                                >
                                    <Button
                                        icon={<DownloadIcon />}
                                        variant="plain"
                                        aria-label="Download Workload Template (JSON)"
                                        onClick={() => downloadTemplateAsJson()}
                                    />
                                </Tooltip>
                            </FlexItem>
                        )}
                    </Flex>
                }
                actions={getActions()}
            >
                {jsonModeActive && (
                    <CodeContext.Provider value={{ code: formAsJson, setCode: setFormAsJson }}>
                        <CodeEditorComponent
                            showCodeTemplates={false}
                            height={650}
                            language={Language.json}
                            defaultFilename={'template'}
                        />
                    </CodeContext.Provider>
                )}
                {!jsonModeActive && (
                    <React.Fragment>
                        <Form
                            onSubmit={() => {
                                form.clearErrors();
                                form.handleSubmit(onSubmit);
                            }}
                        >
                            <FormSection title="Generic Workload Parameters" titleElement="h1">
                                <div ref={sessionFormRef}>
                                    <Grid hasGutter md={12}>
                                        <GridItem span={12}>
                                            <FormGroup
                                                isRequired
                                                label="Workload name:"
                                                labelInfo="Required length: 1-36 characters"
                                                labelHelp={
                                                    <Popover
                                                        aria-label="workload-title-popover"
                                                        headerContent={<div>Workload Title</div>}
                                                        bodyContent={
                                                            <div>
                                                                This is an identifier (that is not necessarily unique,
                                                                but probably should be) to help you identify the
                                                                specific workload. Please note that the title must be
                                                                between 1 and 36 characters in length.
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
                                                    render={({ field }) => (
                                                        <TextInput
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
                                                        />
                                                    )}
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
                                                            variant={getWorkloadNameValidationState()}
                                                        >
                                                            {isWorkloadNameValid()
                                                                ? ''
                                                                : 'Session ID must be between 1 and 36 characters in length (inclusive).'}
                                                        </HelperTextItem>
                                                    </HelperText>
                                                </FormHelperText>
                                            </FormGroup>
                                        </GridItem>
                                        <GridItem span={3}>
                                            <FormGroup
                                                label={'Verbose Server-Side Log Output'}
                                                labelHelp={
                                                    <Popover
                                                        aria-label="workload-debug-logging-header"
                                                        headerContent={<div>Verbose Server-Side Log Output</div>}
                                                        bodyContent={
                                                            <div>
                                                                Enable or disable server-side debug (i.e., verbose) log
                                                                output from the workload generator and workload driver.
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
                                                    render={({ field }) => (
                                                        <Switch
                                                            id="debug-logging-switch-template"
                                                            label="Debug logging enabled"
                                                            aria-label="debug-logging-switch-template"
                                                            isChecked={field.value === true}
                                                            ouiaId="DebugLoggingSwitchTemplate"
                                                            onChange={(
                                                                _event: React.FormEvent<HTMLInputElement>,
                                                                checked: boolean,
                                                            ) => {
                                                                form.setValue('debugLoggingEnabled', checked);
                                                            }}
                                                        />
                                                    )}
                                                />
                                            </FormGroup>
                                        </GridItem>
                                        <GridItem span={3}>
                                            <FormGroup
                                                label="Workload Seed:"
                                                labelHelp={
                                                    <Popover
                                                        aria-label="workload-seed-popover"
                                                        headerContent={<div>Workload Title</div>}
                                                        bodyContent={
                                                            <div>
                                                                This is an integer seed for the random number generator
                                                                used by the workload generator. Pass a value of 0 to
                                                                refrain from seeding the random generator. Please note
                                                                that if you do specify a seed, then the value must be
                                                                between 0 and 2,147,483,647.
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
                                                    render={({ field }) => (
                                                        <NumberInput
                                                            inputName="workload-seed-number-input"
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
                                                                const curr: number =
                                                                    form.getValues('workloadSeed') || 0;
                                                                let next: number = curr + WorkloadSeedDelta;
                                                                next = clamp(next, WorkloadSeedMin, WorkloadSeedMax);
                                                                form.setValue('workloadSeed', next);
                                                            }}
                                                            onMinus={() => {
                                                                const curr: number =
                                                                    form.getValues('workloadSeed') || 0;
                                                                let next: number = curr - WorkloadSeedDelta;
                                                                next = clamp(next, WorkloadSeedMin, WorkloadSeedMax);
                                                                form.setValue('workloadSeed', next);
                                                            }}
                                                        />
                                                    )}
                                                />
                                            </FormGroup>
                                        </GridItem>
                                        <GridItem span={3}>
                                            <FormGroup
                                                label={'Timescale Adjustment Factor'}
                                                labelHelp={
                                                    <Popover
                                                        aria-label="timescale-adjustment-factor-header"
                                                        headerContent={<div>Timescale Adjustment Factor</div>}
                                                        bodyContent={
                                                            <div>
                                                                This quantity adjusts the timescale at which the trace
                                                                data is replayed. For example, if each tick is 60
                                                                seconds, then setting this value to 1.0 will instruct
                                                                the Workload Driver to simulate each tick for the full
                                                                60 seconds. Alternatively, setting this quantity to 2.0
                                                                will instruct the Workload Driver to spend 120 seconds
                                                                on each tick. Setting the quantity to 0.5 will instruct
                                                                the Workload Driver to spend 30 seconds on each tick.
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
                                                    rules={{
                                                        max: TimescaleAdjustmentFactorMax,
                                                        min: TimescaleAdjustmentFactorMin,
                                                    }}
                                                    render={({ field }) => (
                                                        <NumberInput
                                                            inputName="timescale-adjustment-factor-number-input"
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
                                                                const curr: number = form.getValues(
                                                                    'timescaleAdjustmentFactor',
                                                                ) as number;
                                                                let next: number =
                                                                    curr + TimescaleAdjustmentFactorDelta;

                                                                if (next > TimescaleAdjustmentFactorMax) {
                                                                    next = TimescaleAdjustmentFactorMax;
                                                                }

                                                                next = RoundToThreeDecimalPlaces(next);

                                                                form.setValue(
                                                                    'timescaleAdjustmentFactor',
                                                                    clamp(
                                                                        next,
                                                                        TimescaleAdjustmentFactorMin,
                                                                        TimescaleAdjustmentFactorMax,
                                                                    ),
                                                                );
                                                            }}
                                                            onMinus={() => {
                                                                const curr: number = form.getValues(
                                                                    'timescaleAdjustmentFactor',
                                                                ) as number;
                                                                let next: number =
                                                                    curr - TimescaleAdjustmentFactorDelta;

                                                                // For the timescale adjustment factor, we don't want to decrement it to 0.
                                                                if (next < TimescaleAdjustmentFactorMin) {
                                                                    next = TimescaleAdjustmentFactorMin;
                                                                }

                                                                next = RoundToThreeDecimalPlaces(next);

                                                                form.setValue(
                                                                    'timescaleAdjustmentFactor',
                                                                    clamp(
                                                                        next,
                                                                        TimescaleAdjustmentFactorMin,
                                                                        TimescaleAdjustmentFactorMax,
                                                                    ),
                                                                );
                                                            }}
                                                        />
                                                    )}
                                                />
                                            </FormGroup>
                                        </GridItem>
                                        <GridItem span={3}>
                                            <FormGroup label={'Number of Sessions'}>
                                                <Controller
                                                    name="numberOfSessions"
                                                    control={form.control}
                                                    defaultValue={NumberOfSessionsDefault}
                                                    rules={{ min: NumberOfSessionsMin, max: NumberOfSessionsMax }}
                                                    render={({ field }) => (
                                                        <TextInput
                                                            // inputName='number-of-sessions-in-template-workload-input'
                                                            id="number-of-sessions-in-template-workload-input"
                                                            key={'number-of-sessions-in-template-workload-input'}
                                                            type="number"
                                                            aria-label="Text input for the 'number of sessions'"
                                                            onBlur={field.onBlur}
                                                            onChange={field.onChange}
                                                            name={field.name}
                                                            value={field.value}
                                                            isDisabled={true}
                                                            min={NumberOfSessionsMin}
                                                            max={NumberOfSessionsMax}
                                                        />
                                                    )}
                                                />
                                            </FormGroup>
                                        </GridItem>
                                    </Grid>
                                </div>
                            </FormSection>
                            <SessionConfigurationForm />
                            <FormSection title="Upload JSON Template File" titleElement="h1">
                                <FormGroup hasNoPaddingTop isRequired>
                                    <HelperText>
                                        You may optionally upload a JSON template file. This form will be populated with
                                        the values from the template file.
                                    </HelperText>
                                    <MultipleFileUpload
                                        onFileDrop={onFileUploadedNonJsonEditor}
                                        dropzoneProps={{
                                            accept: {
                                                'application/json': ['.json'],
                                            },
                                            onDropRejected: handleFileUploadRejection,
                                            maxFiles: 1,
                                        }}
                                    >
                                        <MultipleFileUploadMain
                                            titleIcon={<UploadIcon />}
                                            titleText="Drag and drop a Workload Template file here"
                                            titleTextSeparator="or"
                                            infoText="Accepted file types: JSON"
                                        />
                                        <MultipleFileUploadStatus
                                            statusToggleText={`${successfullyReadFileCount} of ${currentFiles.length} files uploaded`}
                                            statusToggleIcon={fileUploadStatusIcon}
                                        >
                                            {currentFiles.map((file) => (
                                                <MultipleFileUploadStatusItem
                                                    file={file}
                                                    key={file.name}
                                                    onClearClick={() => removeFiles([file.name])}
                                                    onReadSuccess={handleReadSuccess}
                                                    onReadFail={handleReadFail}
                                                />
                                            ))}
                                        </MultipleFileUploadStatus>
                                    </MultipleFileUpload>
                                </FormGroup>
                            </FormSection>
                        </Form>
                    </React.Fragment>
                )}
            </Modal>
            <Modal
                isOpen={isFailedUploadModalOpen}
                variant={'small'}
                title={failedUploadModalTitleText}
                titleIconVariant="warning"
                showClose
                aria-label="Failed to parse the uploaded JSON Template"
                onClose={() => setFailedUploadModalOpen(false)}
            >
                {failedUploadModalBodyText}
            </Modal>
        </FormProvider>
    );
};
