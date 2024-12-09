import { CodeEditorComponent } from '@Components/CodeEditor';
import { SessionConfigurationForm } from '@Components/Modals';
import RemoteStorageDefinitionForm from '@Components/Modals/NewWorkloadFromTemplateModal/RemoteStorage/RemoteStorageDefinitionForm';
import { Language } from '@patternfly/react-code-editor';
import {
    Button,
    Divider,
    Dropdown,
    DropdownItem,
    DropdownList,
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
    MenuToggle,
    MenuToggleElement,
    Modal,
    MultipleFileUpload,
    MultipleFileUploadMain,
    MultipleFileUploadStatus,
    MultipleFileUploadStatusItem,
    NumberInput,
    Popover,
    Switch,
    Text,
    TextInput,
    ValidatedOptions,
} from '@patternfly/react-core';
import { DropEvent } from '@patternfly/react-core/src/helpers/typeUtils';
import { CodeIcon, DownloadIcon, SaveAltIcon, TrashAltIcon, UploadIcon } from '@patternfly/react-icons';
import HelpIcon from '@patternfly/react-icons/dist/esm/icons/help-icon';
import styles from '@patternfly/react-styles/css/components/Form/form';
import { useWorkloadTemplates } from '@Providers/WorkloadTemplatesProvider';
import {
    PreloadedWorkloadTemplate,
    PreloadedWorkloadTemplateWrapper,
    RemoteStorageDefinition,
    Session,
    TrainingEvent,
    WorkloadRegistrationRequest,
    WorkloadRegistrationRequestTemplateWrapper,
    WorkloadRegistrationRequestWrapper,
} from '@src/Data';
import { SessionTabsDataContext } from '@src/Providers';
import { GetPathForFetch, numberWithCommas } from '@src/Utils';
import { RoundToThreeDecimalPlaces } from '@src/Utils/utils';
import {
    GetDefaultFormValues,
    NumberOfSessionsDefault,
    NumberOfSessionsMax,
    NumberOfSessionsMin,
    TimescaleAdjustmentFactorDefault,
    TimescaleAdjustmentFactorDelta,
    TimescaleAdjustmentFactorMax,
    TimescaleAdjustmentFactorMin,
    WorkloadSampleSessionPercentDelta,
    WorkloadSampleSessionPercentMax,
    WorkloadSampleSessionPercentMin,
    WorkloadSeedDefault,
    WorkloadSeedDelta,
    WorkloadSeedMax,
    WorkloadSeedMin,
    WorkloadSessionSamplePercentDefault,
} from '@Workloads/Constants';
import SampleSessionsPopover from '@Workloads/RegistrationForms/SampleSessionsPopover';
import React from 'react';
import { FileRejection } from 'react-dropzone';

import { Controller, FormProvider, useForm } from 'react-hook-form';
import toast from 'react-hot-toast';

import { v4 as uuidv4 } from 'uuid';

export interface IRegisterWorkloadFromTemplateFormProps {
    children?: React.ReactNode;
    onCancel: () => void;
    onConfirm: (workloadName: string, workloadRegistrationRequestJson: string, messageId?: string) => void;
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

export const WorkloadTemplateJsonContext = React.createContext({
    code: '',
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    setCode: (_: string) => {},
});

// Important: this component must be wrapped in a <SessionTabsDataProvider></SessionTabsDataProvider>!
export const RegisterWorkloadFromTemplateForm: React.FunctionComponent<IRegisterWorkloadFromTemplateFormProps> = (
    props: IRegisterWorkloadFromTemplateFormProps,
) => {
    const defaultWorkloadTitle = React.useRef(uuidv4());
    const [jsonModeActive, setJsonModeActive] = React.useState<boolean>(false);
    const [isFailedUploadModalOpen, setFailedUploadModalOpen] = React.useState<boolean>(false);
    const [failedUploadModalTitleText, setFailedUploadModalTitleText] = React.useState<string>('');
    const [failedUploadModalBodyText, setFailedUploadModalBodyText] = React.useState<string>('');
    const [currentFiles, setCurrentFiles] = React.useState<File[]>([]);
    const [readFileData, setReadFileData] = React.useState<readFile[]>([]);
    const [showFileUploadStatus, setShowFileUploadStatus] = React.useState(false);
    const [fileUploadStatusIcon, setFileUploadStatusIcon] = React.useState('inProgress');
    const [isWorkloadDataDropdownOpen, setIsWorkloadDataDropdownOpen] = React.useState<boolean>(false);
    const [selectedPreloadedWorkloadTemplate, setSelectedPreloadedWorkloadTemplate] =
        React.useState<PreloadedWorkloadTemplate | null>(null);
    const sessionFormRef = React.useRef<HTMLDivElement>(null);

    // Actively modified by the code editor.
    const [formAsJson, setFormAsJson] = React.useState<string>('');

    const { preloadedWorkloadTemplates } = useWorkloadTemplates();

    const { activeSessionTab, setActiveSessionTab, setSessionTabs, setNewSessionTabNumber } =
        React.useContext(SessionTabsDataContext);

    const form = useForm({
        mode: 'all',
        defaultValues: GetDefaultFormValues(),
    });

    const {
        formState: { isSubmitSuccessful, isValid },
    } = form;

    React.useEffect(() => {
        if (isValid) {
            console.debug('Workload template is currently valid.');
        } else {
            console.debug('Workload template is NOT valid in its current form.');
        }
    }, [isValid]);

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

    const parseData = (data, space: string | number | undefined = undefined, message_id?: string) => {
        const workloadTitle: string = data.workloadTitle;
        const workloadSeedString: string = data.workloadSeed;
        const debugLoggingEnabled: boolean = data.debugLoggingEnabled;
        const timescaleAdjustmentFactor: number = data.timescaleAdjustmentFactor;
        const sessionsSamplePercentage: number = data.sessionsSamplePercentage;

        const remoteStorageDefinition: RemoteStorageDefinition = data.remoteStorageDefinition;

        let sessions: Session[] = data.sessions;

        // Don't bother parsing if we have a large preloaded template selected.
        // We won't be including any sessions in the request.
        if (!selectedPreloadedWorkloadTemplate || !selectedPreloadedWorkloadTemplate.large) {
            for (let i: number = 0; i < sessions.length; i++) {
                const session: Session = sessions[i];
                const trainings: TrainingEvent[] = session.trainings;

                if (session.num_training_events === 0 && trainings.length > 0) {
                    session.num_training_events = trainings.length;
                }

                let max_millicpus: number = -1;
                let max_mem_mb: number = -1;
                let max_num_gpus: number = -1;
                let max_vram_gb: number = -1;
                for (let j: number = 0; j < trainings.length; j++) {
                    const training: TrainingEvent = trainings[j];
                    training.training_index = j; // Set the training index field.

                    if (training.cpus > max_millicpus) {
                        max_millicpus = training.cpus;
                    }

                    if (training.memory > max_mem_mb) {
                        max_mem_mb = training.memory;
                    }

                    if (training.vram > max_vram_gb) {
                        max_vram_gb = training.vram;
                    }

                    if (training.gpu_utilizations.length > max_num_gpus) {
                        max_num_gpus = training.gpu_utilizations.length;
                    }
                }

                // Construct the resource request and update the session object.
                session.max_resource_request = {
                    cpus: max_millicpus,
                    gpus: max_num_gpus,
                    memory: max_mem_mb,
                    vram: max_vram_gb,
                    gpu_type: 'ANY_GPU',
                };

                session.current_resource_request = {
                    cpus: 0,
                    gpus: 0,
                    memory: 0,
                    vram: 0,
                    gpu_type: 'ANY_GPU',
                };
            }
        }

        let workloadSeed: number = 0;
        if (workloadSeedString != '') {
            workloadSeed = parseInt(workloadSeedString);
        }

        // If we have a large, preloaded template selected, then make sure the sessions are empty
        // so that the server knows to load the template from a file.
        if (selectedPreloadedWorkloadTemplate && selectedPreloadedWorkloadTemplate.large) {
            sessions = [];
        }

        if (!message_id) {
            message_id = uuidv4();
        }

        const request: WorkloadRegistrationRequest = {
            adjust_gpu_reservations: false,
            name: workloadTitle,
            debug_logging: debugLoggingEnabled,
            sessions: sessions,
            template_file_path: selectedPreloadedWorkloadTemplate ? selectedPreloadedWorkloadTemplate.filepath : '',
            type: 'template',
            key: 'workload_template_key',
            seed: workloadSeed,
            timescale_adjustment_factor: timescaleAdjustmentFactor,
            remote_storage_definition: remoteStorageDefinition,
            sessions_sample_percentage: sessionsSamplePercentage,
        };

        console.log(`request: ${JSON.stringify(request, null, '  ')}`);

        const requestWrapper: WorkloadRegistrationRequestWrapper = {
            op: 'register_workload',
            msg_id: message_id,
            workload_registration_request: request,
        };

        return JSON.stringify(requestWrapper, null, space);

        // return JSON.stringify(
        //     {
        //         op: 'register_workload',
        //         msg_id: message_id,
        //         workload_registration_request: {
        //             adjust_gpu_reservations: false,
        //             seed: workloadSeed,
        //             timescale_adjustment_factor: timescaleAdjustmentFactor,
        //             key: 'workload_template_key',
        //             name: workloadTitle,
        //             debug_logging: debugLoggingEnabled,
        //             type: 'template',
        //             sessions: sessions,
        //             remote_storage_definition: remoteStorageDefinition,
        //             sessions_sample_percentage: sessionsSamplePercentage,
        //             template_file_path: selectedPreloadedWorkloadTemplate
        //                 ? selectedPreloadedWorkloadTemplate.filepath
        //                 : '',
        //         },
        //     },
        //     null,
        //     space,
        // );
    };

    const onSubmitTemplate = (data) => {
        const messageId: string = uuidv4();

        let workloadRegistrationRequest: string;
        try {
            workloadRegistrationRequest = parseData(data, undefined, messageId);
        } catch (err) {
            console.error(`Failed to parse template: ${err}`);
            toast.error(`Failed to parse template: ${err}`);
            return;
        }
        console.log(`User submitted workload template data: ${JSON.stringify(workloadRegistrationRequest)}`);
        props.onConfirm(data.workloadTitle, workloadRegistrationRequest, messageId);
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

        for (let i: number = 0; i < data.sessions.length; i++) {
            const session = data.sessions[i];
            form.setValue(`sessions.${i}.num_training_events`, session.num_training_events);
        }
    };

    const onDiscardJsonChangesButtonClicked = () => {
        setJsonModeActive(false);
    };

    const getWorkloadTemplateDropdownDescription = (template: PreloadedWorkloadTemplate) => {
        return (
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
                <FlexItem>
                    <Text component={'h6'}>
                        <b>Number of Sessions: </b> {numberWithCommas(template.num_sessions)}
                    </Text>
                </FlexItem>
                <FlexItem>
                    <Text component={'h6'}>
                        <b>Number of Training Events: </b> {numberWithCommas(template.num_training_events)}
                    </Text>
                </FlexItem>
            </Flex>
        );
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
                    onClick={form.handleSubmit(onSubmitTemplate)}
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
                    onClick={props.onCancel}
                >
                    Cancel
                </Button>
            );
        }
    };

    const onResetFormButtonClicked = () => {
        console.log('Resetting form to default values.');

        form.reset(GetDefaultFormValues());

        setSelectedPreloadedWorkloadTemplate(null);
        setSessionTabs(['Session 1']);
    };

    const getActions = () => {
        if (jsonModeActive) {
            return (
                <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsMd' }}>
                    <FlexItem>{getSubmitButton()}</FlexItem>
                    <FlexItem>{getCancelButton()}</FlexItem>
                </Flex>
            );
        } else {
            return (
                <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsMd' }}>
                    <FlexItem>{getSubmitButton()}</FlexItem>
                    <FlexItem>
                        <Button
                            key={'download-workload-template-as-json-button'}
                            id={'download-workload-template-as-json-button'}
                            icon={<DownloadIcon />}
                            variant={'secondary'}
                            onClick={downloadTemplateAsJson}
                        >
                            Download Template
                        </Button>
                    </FlexItem>
                    <FlexItem>
                        <Button
                            key={'switch-to-json-button'}
                            id={'switch-to-json-button'}
                            icon={<CodeIcon />}
                            variant={'secondary'}
                            onClick={enableJsonEditorMode}
                        >
                            Switch to JSON Editor
                        </Button>
                    </FlexItem>
                    <FlexItem>
                        <Button
                            key={'reset-workload-template-form-button'}
                            id={'reset-workload-template-form-button'}
                            icon={<TrashAltIcon />}
                            variant={'warning'}
                            onClick={onResetFormButtonClicked}
                        >
                            Reset Form to Default Values
                        </Button>
                    </FlexItem>
                    <FlexItem>{getCancelButton()}</FlexItem>
                </Flex>
            );
        }
    };

    const onWorkloadDataDropdownSelect = async (
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined,
    ) => {
        if (value != undefined) {
            const template: PreloadedWorkloadTemplate = preloadedWorkloadTemplates[value];
            setSelectedPreloadedWorkloadTemplate(template);

            try {
                if (template) {
                    const req: RequestInit = {
                        method: 'GET',
                        headers: {
                            'Content-Type': 'application/json',
                            Authorization: 'Bearer ' + localStorage.getItem('token'),
                        },
                    };

                    const returnedTemplateWrapper:
                        | PreloadedWorkloadTemplateWrapper
                        | WorkloadRegistrationRequestTemplateWrapper = await fetch(
                        GetPathForFetch(`api/workload-templates?template=${template.key}`),
                        req,
                    ).then(async (resp: Response) => {
                        console.log(`HTTP ${resp.status} ${resp.statusText}`);
                        return await resp.json();
                    });

                    // Non-large templates are returned directly.
                    if (!template.large) {
                        const templateWrapper: WorkloadRegistrationRequestTemplateWrapper =
                            returnedTemplateWrapper as WorkloadRegistrationRequestTemplateWrapper;
                        console.log(`returnedTemplate: ${JSON.stringify(templateWrapper.template, null, '  ')}`);
                        applyJsonToForm(JSON.stringify(templateWrapper.template));
                    } else {
                        const templateWrapper: PreloadedWorkloadTemplateWrapper =
                            returnedTemplateWrapper as PreloadedWorkloadTemplateWrapper;

                        console.log(
                            `Preloaded Template: ${JSON.stringify(templateWrapper.preloaded_template, null, '  ')}`,
                        );
                    }
                } else {
                    console.warn(`Unknown template with index ${value}`);
                }
            } catch (err) {
                console.error(err);
            }
        } else {
            setSelectedPreloadedWorkloadTemplate(null);
        }
        setIsWorkloadDataDropdownOpen(false);
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

    const onFileUploadedNonJsonEditor = (_event: DropEvent, uploadedFiles: File[]) => {
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

    const jsonEditor = (
        <WorkloadTemplateJsonContext.Provider value={{ code: formAsJson, setCode: setFormAsJson }}>
            <CodeEditorComponent
                showCodeTemplates={false}
                height={650}
                language={Language.json}
                defaultFilename={'template'}
                targetContext={WorkloadTemplateJsonContext}
            />
        </WorkloadTemplateJsonContext.Provider>
    );

    const jsonFileUploadRegion = (
        <FormSection title="Upload JSON Template File" titleElement="h1">
            <FormGroup hasNoPaddingTop isRequired>
                <HelperText>
                    You may optionally upload a JSON template file. This form will be populated with the values from the
                    template file.
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
    );

    const workloadTitleForm = (
        <FormGroup
            isRequired
            label="Workload name:"
            labelInfo="Required length: 1-36 characters"
            labelIcon={
                <Popover
                    aria-label="workload-title-popover"
                    headerContent={<div>Workload Title</div>}
                    bodyContent={
                        <div>
                            This is an identifier (that is not necessarily unique, but probably should be) to help you
                            identify the specific workload. Please note that the title must be between 1 and 36
                            characters in length.
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
            <FormHelperText label="workload-title-text-input-helper" aria-label="workload-title-text-input-helper">
                <HelperText label="workload-title-text-input-helper" aria-label="workload-title-text-input-helper">
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
    );

    const verboseLoggingForm = (
        <FormGroup
            label={'Verbose Server-Side Log Output'}
            labelIcon={
                <Popover
                    aria-label="workload-debug-logging-header"
                    headerContent={<div>Verbose Server-Side Log Output</div>}
                    bodyContent={
                        <div>
                            Enable or disable server-side debug (i.e., verbose) log output from the workload generator
                            and workload driver.
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
                        labelOff="Debug logging disabled"
                        aria-label="debug-logging-switch-template"
                        isChecked={field.value === true}
                        ouiaId="DebugLoggingSwitchTemplate"
                        onChange={(_event: React.FormEvent<HTMLInputElement>, checked: boolean) => {
                            form.setValue('debugLoggingEnabled', checked);
                        }}
                    />
                )}
            />
        </FormGroup>
    );

    const workloadSeedForm = (
        <FormGroup
            label="Workload Seed:"
            labelIcon={
                <Popover
                    aria-label="workload-seed-popover"
                    headerContent={<div>Workload Seed</div>}
                    bodyContent={
                        <div>
                            This is an integer seed for the random number generator used by the workload generator. Pass
                            a value of 0 to refrain from seeding the random generator. Please note that if you do
                            specify a seed, then the value must be between 0 and 2,147,483,647.
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
                        onChange={(event: React.FormEvent<HTMLInputElement>) => {
                            field.onChange(parseFloat((event.target as HTMLInputElement).value));
                        }}
                        name={field.name}
                        value={field.value}
                        widthChars={10}
                        aria-label="Text input for the 'workload seed'"
                        onPlus={() => {
                            const curr: number = form.getValues('workloadSeed') || 0;
                            let next: number = curr + WorkloadSeedDelta;
                            next = clamp(next, WorkloadSeedMin, WorkloadSeedMax);
                            form.setValue('workloadSeed', next);
                        }}
                        onMinus={() => {
                            const curr: number = form.getValues('workloadSeed') || 0;
                            let next: number = curr - WorkloadSeedDelta;
                            next = clamp(next, WorkloadSeedMin, WorkloadSeedMax);
                            form.setValue('workloadSeed', next);
                        }}
                    />
                )}
            />
        </FormGroup>
    );

    const timescaleAdjustmentFactorForm = (
        <FormGroup
            label={'Timescale Adjustment Factor'}
            labelIcon={
                <Popover
                    aria-label="timescale-adjustment-factor-header"
                    headerContent={<div>Timescale Adjustment Factor</div>}
                    bodyContent={
                        <div>
                            This quantity adjusts the timescale at which the trace data is replayed. For example, if
                            each tick is 60 seconds, then setting this value to 1.0 will instruct the Workload Driver to
                            simulate each tick for the full 60 seconds. Alternatively, setting this quantity to 2.0 will
                            instruct the Workload Driver to spend 120 seconds on each tick. Setting the quantity to 0.5
                            will instruct the Workload Driver to spend 30 seconds on each tick.
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
                defaultValue={TimescaleAdjustmentFactorDefault}
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
                        onChange={(event: React.FormEvent<HTMLInputElement>) => {
                            field.onChange(parseFloat((event.target as HTMLInputElement).value));
                        }}
                        name={field.name}
                        value={field.value}
                        min={TimescaleAdjustmentFactorMin}
                        max={TimescaleAdjustmentFactorMax}
                        onPlus={() => {
                            const curr: number = form.getValues('timescaleAdjustmentFactor') as number;
                            let next: number = curr + TimescaleAdjustmentFactorDelta;

                            if (next > TimescaleAdjustmentFactorMax) {
                                next = TimescaleAdjustmentFactorMax;
                            }

                            next = RoundToThreeDecimalPlaces(next);

                            form.setValue(
                                'timescaleAdjustmentFactor',
                                clamp(next, TimescaleAdjustmentFactorMin, TimescaleAdjustmentFactorMax),
                            );
                        }}
                        onMinus={() => {
                            const curr: number = form.getValues('timescaleAdjustmentFactor') as number;
                            let next: number = curr - TimescaleAdjustmentFactorDelta;

                            // For the timescale adjustment factor, we don't want to decrement it to 0.
                            if (next < TimescaleAdjustmentFactorMin) {
                                next = TimescaleAdjustmentFactorMin;
                            }

                            next = RoundToThreeDecimalPlaces(next);

                            form.setValue(
                                'timescaleAdjustmentFactor',
                                clamp(next, TimescaleAdjustmentFactorMin, TimescaleAdjustmentFactorMax),
                            );
                        }}
                    />
                )}
            />
        </FormGroup>
    );

    const sampleSessionsPercentFormGroup = (
        <FormGroup label={'Sample Sessions %'} labelIcon={<SampleSessionsPopover />}>
            <Controller
                name="sessionsSamplePercentage"
                control={form.control}
                defaultValue={WorkloadSessionSamplePercentDefault}
                rules={{ min: WorkloadSampleSessionPercentMin, max: WorkloadSampleSessionPercentMax }}
                render={({ field }) => (
                    <NumberInput
                        inputName="workload-sample-session-number-input"
                        id="workload-sample-session-number-input"
                        type="number"
                        min={0}
                        max={1}
                        onBlur={field.onBlur}
                        onChange={(event: React.FormEvent<HTMLInputElement>) => {
                            field.onChange(parseFloat((event.target as HTMLInputElement).value));
                        }}
                        name={field.name}
                        value={field.value}
                        widthChars={10}
                        aria-label="Text input for the 'workload sample session percent'"
                        onPlus={() => {
                            const curr: number = form.getValues('sessionsSamplePercentage') || 0;
                            let next: number = curr + WorkloadSampleSessionPercentDelta;
                            next = clamp(next, WorkloadSampleSessionPercentMin, WorkloadSampleSessionPercentMax);
                            form.setValue('sessionsSamplePercentage', next);
                        }}
                        onMinus={() => {
                            const curr: number = form.getValues('sessionsSamplePercentage') || 0;
                            let next: number = curr - WorkloadSampleSessionPercentDelta;
                            next = clamp(next, WorkloadSampleSessionPercentMin, WorkloadSampleSessionPercentMax);
                            form.setValue('sessionsSamplePercentage', next);
                        }}
                    />
                )}
            />
        </FormGroup>
    );

    const numTrainingEventsDisplay = (
        <FormGroup label={'Total Number of Training Events'}>
            <TextInput
                // inputName='number-of-sessions-in-template-workload-input'
                id="number-of-training-events-in-template-workload-input"
                key={'number-of-training-events-in-template-workload-input'}
                aria-label="Text display for the 'total number of training events'"
                name={'total-num-training-events'}
                value={
                    selectedPreloadedWorkloadTemplate && selectedPreloadedWorkloadTemplate.large
                        ? numberWithCommas(selectedPreloadedWorkloadTemplate.num_training_events)
                        : 'N/A'
                }
                isDisabled={true}
            />
        </FormGroup>
    );

    const numSessionsDisplay = (
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
                        value={
                            selectedPreloadedWorkloadTemplate && selectedPreloadedWorkloadTemplate.large
                                ? selectedPreloadedWorkloadTemplate.num_sessions
                                : field.value
                        }
                        isDisabled={true}
                        min={NumberOfSessionsMin}
                        max={NumberOfSessionsMax}
                    />
                )}
            />
        </FormGroup>
    );

    const preloadedWorkloadTemplateSection = (
        <FormSection title="Select 'Preloaded' Template" titleElement="h1">
            <FormGroup
                label="Preloaded workload template"
                labelIcon={
                    <Popover
                        aria-label="workload-preset-text-header"
                        headerContent={<div>Workload Preset</div>}
                        bodyContent={
                            <div>
                                Select the preprocessed data to use for driving the workload. This largely determines
                                which subset of trace data will be used to generate the workload.
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
                {preloadedWorkloadTemplates.length == 0 && (
                    <TextInput
                        label="workload-template-set-disabled-text"
                        aria-label="workload-template-set-disabled-text"
                        id="disabled-workload-template-select-text"
                        isDisabled
                        type="text"
                        validated={ValidatedOptions.warning}
                        value="No workload templates available..."
                    />
                )}
                {preloadedWorkloadTemplates.length > 0 && (
                    <Dropdown
                        aria-label="workload-template-set-dropdown-menu"
                        isScrollable
                        isOpen={isWorkloadDataDropdownOpen}
                        maxMenuHeight={'600px'}
                        menuHeight={'600px'}
                        onSelect={onWorkloadDataDropdownSelect}
                        onOpenChange={(isOpen: boolean) => setIsWorkloadDataDropdownOpen(isOpen)}
                        toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                            <MenuToggle
                                ref={toggleRef}
                                isFullWidth
                                onClick={() => setIsWorkloadDataDropdownOpen(!isWorkloadDataDropdownOpen)}
                                isExpanded={isWorkloadDataDropdownOpen}
                            >
                                {selectedPreloadedWorkloadTemplate?.display_name}
                            </MenuToggle>
                        )}
                        shouldFocusToggleOnSelect
                    >
                        <DropdownList aria-label="workload-template-set-dropdown-list">
                            {preloadedWorkloadTemplates.map((template: PreloadedWorkloadTemplate, index: number) => {
                                return (
                                    <DropdownItem
                                        aria-label={'workload-template-set-dropdown-item' + index}
                                        value={index}
                                        key={index}
                                        description={getWorkloadTemplateDropdownDescription(template)}
                                    >
                                        {template.display_name}
                                    </DropdownItem>
                                );
                            })}
                        </DropdownList>
                    </Dropdown>
                )}
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
                            Select a configuration/data template for the workload.
                        </HelperTextItem>
                    </HelperText>
                </FormHelperText>
            </FormGroup>
        </FormSection>
    );

    const sessionsAndTrainingEventsDisplaysVisible = (): boolean => {
        return !!(selectedPreloadedWorkloadTemplate && selectedPreloadedWorkloadTemplate.large);
    };

    const nonJsonForm = (
        <Form
            onSubmit={() => {
                form.clearErrors();
                form.handleSubmit(onSubmitTemplate);
            }}
        >
            <Flex direction={{ default: 'column' }}>
                <Flex direction={{ xl: 'row', default: 'column' }} spaceItems={{ default: 'spaceItemsXl' }}>
                    <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
                        <FlexItem>{preloadedWorkloadTemplateSection}</FlexItem>
                        {selectedPreloadedWorkloadTemplate && (
                            <FlexItem>
                                <Button
                                    id={'clear-selected-preloaded-template'}
                                    disabled={!selectedPreloadedWorkloadTemplate}
                                    onClick={() => {
                                        onResetFormButtonClicked();
                                    }}
                                >
                                    Clear Selection
                                </Button>
                            </FlexItem>
                        )}
                    </Flex>
                    <FlexItem>
                        <FormSection title="Generic Workload Parameters" titleElement="h1">
                            <div ref={sessionFormRef}>
                                <Grid hasGutter md={12}>
                                    <GridItem span={sessionsAndTrainingEventsDisplaysVisible() ? 6 : 12}>
                                        {workloadTitleForm}
                                    </GridItem>
                                    {sessionsAndTrainingEventsDisplaysVisible() && (
                                        <GridItem span={3}>{numSessionsDisplay}</GridItem>
                                    )}
                                    {sessionsAndTrainingEventsDisplaysVisible() && (
                                        <GridItem span={3}>{numTrainingEventsDisplay}</GridItem>
                                    )}
                                    <GridItem span={3}>{verboseLoggingForm}</GridItem>
                                    <GridItem span={3}>{workloadSeedForm}</GridItem>
                                    <GridItem span={3}>{timescaleAdjustmentFactorForm}</GridItem>
                                    <GridItem span={3}>{sampleSessionsPercentFormGroup}</GridItem>
                                </Grid>
                            </div>
                        </FormSection>
                    </FlexItem>
                    <FlexItem>
                        <Divider orientation={{ xl: 'vertical', default: 'horizontal' }} />
                    </FlexItem>
                    <FlexItem>
                        <RemoteStorageDefinitionForm />
                    </FlexItem>
                </Flex>
                {(!selectedPreloadedWorkloadTemplate || !selectedPreloadedWorkloadTemplate.large) && (
                    <FlexItem>
                        <SessionConfigurationForm />
                    </FlexItem>
                )}
                <FlexItem>{jsonFileUploadRegion}</FlexItem>
            </Flex>
        </Form>
    );

    return (
        <FormProvider {...form}>
            <Flex
                direction={{ default: 'column' }}
                alignItems={{ default: 'alignItemsCenter' }}
                alignContent={{ default: 'alignContentCenter' }}
                alignSelf={{ default: 'alignSelfCenter' }}
                justifyContent={{ default: 'justifyContentCenter' }}
            >
                <FlexItem>
                    {jsonModeActive && jsonEditor}
                    {!jsonModeActive && nonJsonForm}
                </FlexItem>
                <FlexItem>{getActions()}</FlexItem>
            </Flex>
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
