import { HeightFactorContext, WorkloadsHeightFactorContext } from '@App/Dashboard';
import {
    InspectWorkloadModal,
    NewWorkloadFromTemplateModal,
    RegisterWorkloadModal,
    VisualizeWorkloadModal,
} from '@Components/Modals';
import {
    Button,
    Card,
    CardBody,
    CardHeader,
    DataList,
    DataListCell,
    DataListItem,
    DataListItemCells,
    DataListItemRow,
    Flex,
    FlexItem,
    Label,
    Pagination,
    PaginationVariant,
    Switch,
    Text,
    TextVariants,
    Title,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
} from '@patternfly/react-core';

import {
    BlueprintIcon,
    CheckCircleIcon,
    ClockIcon,
    CodeIcon,
    CubeIcon,
    DiceIcon,
    ExclamationTriangleIcon,
    HourglassStartIcon,
    MonitoringIcon,
    OutlinedCalendarAltIcon,
    PauseIcon,
    PlayIcon,
    PlusIcon,
    RunningIcon,
    SearchIcon,
    SpinnerIcon,
    StopCircleIcon,
    StopIcon,
    Stopwatch20Icon,
    StopwatchIcon,
    TimesCircleIcon,
    UserClockIcon,
} from '@patternfly/react-icons';

import text from '@patternfly/react-styles/css/utilities/Text/text';
import { useWorkloads } from '@Providers/WorkloadProvider';
import { CsvFileIcon, TemplateIcon, XmlFileIcon } from '@src/Assets/Icons';

import {
    GetWorkloadStatusTooltip,
    IsWorkloadFinished,
    Workload,
    WORKLOAD_STATE_ERRED,
    WORKLOAD_STATE_FINISHED,
    WORKLOAD_STATE_READY,
    WORKLOAD_STATE_RUNNING,
    WORKLOAD_STATE_TERMINATED,
    WorkloadPreset,
} from '@src/Data/Workload';
import { SessionTabsDataProvider } from '@src/Providers';
import { GetToastContentWithHeaderAndBody } from '@src/Utils/toast_utils';
import React, { useEffect } from 'react';
import { Toast, toast } from 'react-hot-toast';
import { v4 as uuidv4 } from 'uuid';

export interface WorkloadCardProps {
    workloadsPerPage: number;
}

export const WorkloadCard: React.FunctionComponent<WorkloadCardProps> = (props: WorkloadCardProps) => {
    const [isRegisterWorkloadModalOpen, setIsRegisterWorkloadModalOpen] = React.useState(false);
    const [isRegisterNewWorkloadFromTemplateModalOpen, setIsRegisterNewWorkloadFromTemplateModalOpen] =
        React.useState(false);
    const [selectedWorkloadListId, setSelectedWorkloadListId] = React.useState('');
    const [page, setPage] = React.useState(1);
    const [perPage, setPerPage] = React.useState(props.workloadsPerPage);

    const [visualizeWorkloadModalOpen, setVisualizeWorkloadModalOpen] = React.useState(false);
    const [workloadBeingVisualized, setWorkloadBeingVisualized] = React.useState<Workload | null>(null);

    const [inspectWorkloadModalOpen, setInspectWorkloadModalOpen] = React.useState(false);
    const [workloadBeingInspected, setWorkloadBeingInspected] = React.useState<Workload | null>(null);

    const heightFactorContext: HeightFactorContext = React.useContext(WorkloadsHeightFactorContext);

    const { workloads, workloadsMap, sendJsonMessage } = useWorkloads();

    useEffect(() => {
        if (workloadBeingInspected !== null && inspectWorkloadModalOpen) {
            const updatedWorkload: Workload | undefined = workloadsMap.get(workloadBeingInspected.id);

            // Ensure the workload is updated in the inspection panel.
            if (updatedWorkload) {
                setWorkloadBeingInspected(updatedWorkload);
            }
        }
    }, [workloadsMap, inspectWorkloadModalOpen, workloadBeingInspected]);

    const onCloseVisualizeWorkloadModal = () => {
        setWorkloadBeingVisualized(null);
        setVisualizeWorkloadModalOpen(false);
    };

    const onCloseInspectWorkloadModal = () => {
        setWorkloadBeingInspected(null);
        setInspectWorkloadModalOpen(false);
    };

    const onClickWorkload = (workload: Workload) => {
        setWorkloadBeingInspected(workload);
        setInspectWorkloadModalOpen(true);
    };

    const onSetPage = (_event: React.MouseEvent | React.KeyboardEvent | MouseEvent, newPage: number) => {
        setPage(newPage);
        console.log(
            'onSetPage: Displaying workloads %d through %d.',
            perPage * (newPage - 1),
            perPage * (newPage - 1) + perPage,
        );
    };

    const onConfirmRegisterWorkloadFromTemplate = (workloadName: string, workloadRegistrationRequest: string) => {
        const toastId: string = toast(`Registering template-based workload now.`, {
            style: { maxWidth: 650 },
        });

        setIsRegisterWorkloadModalOpen(false);
        setIsRegisterNewWorkloadFromTemplateModalOpen(false);
        console.log(`Sending WorkloadRegistrationRequest: ${workloadRegistrationRequest}`);
        const errorMessage: string | void = sendJsonMessage(workloadRegistrationRequest);

        if (errorMessage) {
            toast.custom(
                (t: Toast) =>
                    GetToastContentWithHeaderAndBody(
                        'Workload Registration Failed',
                        [
                            `Unable to register template-based workload "${workloadName}".`,
                            <p key={'toast-content-row-2'}>
                                <b>{'Reason:'}</b> {errorMessage}
                            </p>,
                        ],
                        'danger',
                        () => toast.dismiss(t.id),
                    ),
                { id: toastId },
            );
        } else {
            toast.custom(
                (t: Toast) =>
                    GetToastContentWithHeaderAndBody(
                        'Workload Registered Successfully',
                        `Successfully registered template-based workload "${workloadName}"`,
                        'success',
                        () => toast.dismiss(t.id),
                    ),
                { id: toastId },
            );
        }
    };

    const onRegisterWorkloadFromTemplateClicked = () => {
        setIsRegisterNewWorkloadFromTemplateModalOpen(true);
    };

    const onConfirmRegisterWorkload = (
        workloadName: string,
        selectedPreset: WorkloadPreset,
        workloadSeedString: string,
        debugLoggingEnabled: boolean,
        timescaleAdjustmentFactor: number,
    ) => {
        const toastId: string = toast(`Registering preset-based workload ${workloadName} now.`, {
            style: { maxWidth: 650 },
        });

        console.log(`New workload "${workloadName}" registered by user with preset "${selectedPreset.name}"`);
        setIsRegisterWorkloadModalOpen(false);
        setIsRegisterNewWorkloadFromTemplateModalOpen(false);

        let workloadSeed = -1;
        if (workloadSeedString != '') {
            workloadSeed = parseInt(workloadSeedString);
        }

        const messageId: string = uuidv4();
        const errorMessage: string | void = sendJsonMessage(
            JSON.stringify({
                op: 'register_workload',
                msg_id: messageId,
                workloadRegistrationRequest: {
                    adjust_gpu_reservations: false,
                    seed: workloadSeed,
                    timescale_adjustment_factor: timescaleAdjustmentFactor,
                    key: selectedPreset.key,
                    name: workloadName,
                    debug_logging: debugLoggingEnabled,
                    type: 'preset',
                },
            }),
        );

        if (errorMessage) {
            toast.custom(
                (t: Toast) =>
                    GetToastContentWithHeaderAndBody(
                        'Workload Registration Failed',
                        [
                            `Unable to register workload "${workloadName}" with preset "${selectedPreset.name}" at this time.`,
                            <p key={'toast-content-row-2'}>
                                <b>{'Reason:'}</b> {errorMessage}
                            </p>,
                        ],
                        'danger',
                        () => toast.dismiss(t.id),
                    ),
                { id: toastId },
            );
        } else {
            toast.custom(
                (t: Toast) =>
                    GetToastContentWithHeaderAndBody(
                        `Workload Registered Successfully`,
                        `Successfully registered workload "${workloadName}" with preset "${selectedPreset.name}"`,
                        'success',
                        () => toast.dismiss(t.id),
                    ),
                { id: toastId },
            );
        }
    };

    const onCancelStartWorkload = () => {
        console.log('New workload cancelled by user before starting.');
        setIsRegisterWorkloadModalOpen(false);
    };

    const onCancelStartWorkloadFromTemplate = () => {
        console.log('New workload from template cancelled by user before starting.');
        setIsRegisterNewWorkloadFromTemplateModalOpen(false);
        setIsRegisterWorkloadModalOpen(true);
    };

    const onStopAllWorkloadsClicked = () => {
        toast('Stopping all workload');

        const activeWorkloadsIDs: string[] = [];
        workloads.forEach((workload: Workload) => {
            if (workload.workload_state == WORKLOAD_STATE_RUNNING) {
                activeWorkloadsIDs.push(workload.id);
            }
        });

        const messageId: string = uuidv4();
        sendJsonMessage(
            JSON.stringify({
                op: 'stop_workloads',
                msg_id: messageId,
                workload_ids: activeWorkloadsIDs,
            }),
        );
    };

    const onPerPageSelect = (
        _event: React.MouseEvent | React.KeyboardEvent | MouseEvent,
        newPerPage: number,
        newPage: number,
    ) => {
        setPerPage(newPerPage);
        setPage(newPage);

        heightFactorContext.setHeightFactor(Math.min(workloads.length, newPerPage));
    };

    const onSelectWorkload = (_event: React.MouseEvent | React.KeyboardEvent, id: string) => {
        // Toggle off if it is already selected.
        if (id == selectedWorkloadListId) {
            setSelectedWorkloadListId('');
            console.log("De-selected workload '%s'", id);
        } else {
            setSelectedWorkloadListId(id);
            console.log("Selected workload '%s'", id);
        }
    };

    const toggleDebugLogs = (workloadId: string, enabled: boolean) => {
        if (enabled) {
            console.log("Enabling debug logging for workload '%s'", workloadId);
        } else {
            console.log("Disabling debug logging for workload '%s'", workloadId);
        }

        const messageId: string = uuidv4();
        sendJsonMessage(
            JSON.stringify({
                op: 'toggle_debug_logs',
                msg_id: messageId,
                workload_id: workloadId,
                enabled: enabled,
            }),
        );
    };

    const onVisualizeWorkloadClicked = (workload: Workload) => {
        console.log(`Inspecting workload: ${workload.name} (id=${workload.name})`);
        console.log(workload);

        setWorkloadBeingVisualized(workload);
        setVisualizeWorkloadModalOpen(true);
    };

    const onStartWorkloadClicked = (workload: Workload) => {
        const toastId: string = toast.custom((t: Toast) =>
            GetToastContentWithHeaderAndBody(
                `Starting workload ${workload.name}`,
                [
                    <Text key={`toast-content-start-workload-${workload.id}`} component={TextVariants.small}>
                        <b>Workload ID: </b>
                        {workload.id}
                    </Text>,
                ],
                'info',
                () => toast.dismiss(t.id),
            ),
        );

        console.log(`Starting workload '${workload.name}' (ID=${workload.id})`);

        const messageId: string = uuidv4();
        const errorMessage: string | void = sendJsonMessage(
            JSON.stringify({
                op: 'start_workload',
                msg_id: messageId,
                workload_id: workload.id,
            }),
        );

        if (errorMessage) {
            toast.custom(
                (t: Toast) =>
                    GetToastContentWithHeaderAndBody(
                        'Failed to Start Workload',
                        [
                            `Workload "${workload.name}" (ID="${workload.id}") could not be started.`,
                            <p key={'toast-content-row-2'}>
                                <b>{'Reason:'}</b> {errorMessage}
                            </p>,
                        ],
                        'danger',
                        () => toast.dismiss(t.id),
                    ),
                { id: toastId },
            );
        } else {
            toast.custom(
                (t: Toast) =>
                    GetToastContentWithHeaderAndBody(
                        'Workload Started',
                        `Workload "${workload.name}" (ID="${workload.id}") has been started successfully.`,
                        'success',
                        () => toast.dismiss(t.id),
                    ),
                { id: toastId },
            );
        }
    };

    const onPauseWorkloadClicked = (workload: Workload) => {
        toast(`Pausing workload ${workload.name} (ID=${workload.id}) now...`);
    };

    const onStopWorkloadClicked = (workload: Workload) => {
        const toastId: string = toast(
            (t: Toast) =>
                GetToastContentWithHeaderAndBody(
                    `Stopping workload ${workload.name} (ID = ${workload.id}).`,
                    [],
                    'info',
                    () => toast.dismiss(t.id),
                ),
            {
                style: { maxWidth: 650 },
            },
        );

        console.log("Stopping workload '%s' (ID=%s)", workload.name, workload.id);

        const messageId: string = uuidv4();
        const errorMessage: string | void = sendJsonMessage(
            JSON.stringify({
                op: 'stop_workload',
                msg_id: messageId,
                workload_id: workload.id,
            }),
        );

        if (errorMessage) {
            toast.custom(
                (t: Toast) =>
                    GetToastContentWithHeaderAndBody(
                        'Failed to Stop Workload',
                        [
                            `Workload "${workload.name}" (ID="${workload.id}") could not be stopped.`,
                            <p key={'toast-content-row-2'}>
                                <b>{'Reason:'}</b> {errorMessage}
                            </p>,
                        ],
                        'danger',
                        () => toast.dismiss(t.id),
                    ),
                { id: toastId },
            );
        } else {
            toast.custom(
                (t: Toast) =>
                    GetToastContentWithHeaderAndBody(
                        'Workload Stopped',
                        `Workload "${workload.name}" (ID="${workload.id}") has been stopped successfully.`,
                        'success',
                        () => toast.dismiss(t.id),
                    ),
                { id: toastId },
            );
        }
    };

    const cardHeaderActions = (
        <React.Fragment>
            <ToolbarGroup variant="icon-button-group">
                <ToolbarItem>
                    <Tooltip exitDelay={75} content={<div>Register a new workload.</div>}>
                        <Button
                            label="launch-workload-button"
                            aria-label="launch-workload-button"
                            id="launch-workload-button"
                            variant="plain"
                            onClick={() => {
                                setIsRegisterWorkloadModalOpen(true);
                            }}
                        >
                            <PlusIcon />
                        </Button>
                    </Tooltip>
                    <Tooltip exitDelay={75} content={<div>Stop all running workloads.</div>}>
                        <Button
                            label="stop-workloads-button"
                            aria-label="stop-workloads-button"
                            id="stop-workloads-button"
                            variant="plain"
                            isDanger
                            isDisabled={
                                Object.values(workloads).filter((workload: Workload) => {
                                    return workload.workload_state == WORKLOAD_STATE_RUNNING;
                                }).length == 0
                            }
                            onClick={onStopAllWorkloadsClicked} // () => setIsConfirmDeleteKernelsModalOpen(true)
                        >
                            <StopCircleIcon />
                        </Button>
                    </Tooltip>
                </ToolbarItem>
            </ToolbarGroup>
        </React.Fragment>
    );

    // console.log(`Workloads (${workloads.length}):`);
    // for (let i = 0; i < workloads.length; i++) {
    //   const workload: Workload = workloads[i];
    //
    //   console.log(`\tWorkload #${i}: ${JSON.stringify(workload)}`);
    // }

    return (
        <React.Fragment>
            <Card isRounded isFullHeight id="workload-card">
                <CardHeader actions={{ actions: cardHeaderActions, hasNoOffset: false }}>
                    <Title headingLevel="h1" size="xl">
                        Workloads
                    </Title>
                </CardHeader>
                <CardBody>
                    {workloads.length == 0 && (
                        <Text component={TextVariants.h2}>There are no registered workloads.</Text>
                    )}
                    {workloads.length >= 1 && (
                        <React.Fragment>
                            <DataList
                                isCompact
                                aria-label="data list"
                                selectedDataListItemId={selectedWorkloadListId}
                                onSelectDataListItem={onSelectWorkload}
                            >
                                {workloads
                                    .slice(perPage * (page - 1), perPage * (page - 1) + perPage)
                                    .map((workload: Workload, idx: number) => (
                                        <DataListItem
                                            key={workload.id}
                                            id={workload.id}
                                            onClick={() => {
                                                onClickWorkload(workload);
                                            }}
                                        >
                                            <DataListItemRow>
                                                <DataListItemCells
                                                    dataListCells={[
                                                        <DataListCell
                                                            key={'workload-primary-content-' + idx}
                                                            isFilled={true}
                                                            width={4}
                                                        >
                                                            <Flex
                                                                direction={{ default: 'column' }}
                                                                spaceItems={{ default: 'spaceItemsNone' }}
                                                            >
                                                                <Flex
                                                                    direction={{ default: 'row' }}
                                                                    spaceItems={{ default: 'spaceItemsNone' }}
                                                                >
                                                                    <FlexItem>
                                                                        <Flex
                                                                            direction={{ default: 'column' }}
                                                                            spaceItems={{ default: 'spaceItemsNone' }}
                                                                        >
                                                                            <Flex
                                                                                direction={{ default: 'row' }}
                                                                                spaceItems={{ default: 'spaceItemsMd' }}
                                                                            >
                                                                                <FlexItem>
                                                                                    <Text component={TextVariants.h2}>
                                                                                        <strong>{workload.name}</strong>
                                                                                    </Text>
                                                                                </FlexItem>
                                                                                {workload.workload_preset && (
                                                                                    <FlexItem>
                                                                                        <Tooltip
                                                                                            content={`This preset is defined in a ${workload.workload_preset.preset_type} file.`}
                                                                                            position="bottom"
                                                                                        >
                                                                                            <React.Fragment>
                                                                                                {workload
                                                                                                    .workload_preset
                                                                                                    .preset_type ===
                                                                                                    'XML' && (
                                                                                                    <XmlFileIcon
                                                                                                        scale={2.25}
                                                                                                    />
                                                                                                )}
                                                                                                {workload
                                                                                                    .workload_preset
                                                                                                    .preset_type ===
                                                                                                    'CSV' && (
                                                                                                    <CsvFileIcon
                                                                                                        scale={2.25}
                                                                                                    />
                                                                                                )}
                                                                                            </React.Fragment>
                                                                                        </Tooltip>
                                                                                    </FlexItem>
                                                                                )}
                                                                                {workload.workload_template && (
                                                                                    <FlexItem>
                                                                                        <Tooltip
                                                                                            content={`This workload was created/defined using a template.`}
                                                                                            position="bottom"
                                                                                        >
                                                                                            <React.Fragment>
                                                                                                <TemplateIcon
                                                                                                    scale={3.25}
                                                                                                />
                                                                                            </React.Fragment>
                                                                                        </Tooltip>
                                                                                    </FlexItem>
                                                                                )}
                                                                            </Flex>
                                                                            <FlexItem>
                                                                                <Text component={TextVariants.small}>
                                                                                    <strong>ID: </strong>
                                                                                </Text>
                                                                                <Text component={TextVariants.small}>
                                                                                    {workload.id}
                                                                                </Text>
                                                                            </FlexItem>
                                                                        </Flex>
                                                                    </FlexItem>
                                                                    <FlexItem align={{ default: 'alignRight' }}>
                                                                        <Flex
                                                                            direction={{ default: 'column' }}
                                                                            spaceItems={{ default: 'spaceItemsXs' }}
                                                                        >
                                                                            <FlexItem>
                                                                                <Flex
                                                                                    direction={{ default: 'row' }}
                                                                                    spaceItems={{
                                                                                        default: 'spaceItemsXs',
                                                                                    }}
                                                                                >
                                                                                    <FlexItem>
                                                                                        <Tooltip
                                                                                            content={
                                                                                                'Start the workload'
                                                                                            }
                                                                                        >
                                                                                            <Button
                                                                                                id={
                                                                                                    'start-workload-' +
                                                                                                    idx +
                                                                                                    '-button'
                                                                                                }
                                                                                                isDisabled={
                                                                                                    workload.workload_state !=
                                                                                                    WORKLOAD_STATE_READY
                                                                                                }
                                                                                                variant="link"
                                                                                                icon={<PlayIcon />}
                                                                                                onClick={(
                                                                                                    event: React.MouseEvent<
                                                                                                        HTMLButtonElement,
                                                                                                        MouseEvent
                                                                                                    >,
                                                                                                ) => {
                                                                                                    onStartWorkloadClicked(
                                                                                                        workload,
                                                                                                    );

                                                                                                    event.stopPropagation();
                                                                                                }}
                                                                                            >
                                                                                                Start
                                                                                            </Button>
                                                                                        </Tooltip>
                                                                                    </FlexItem>
                                                                                    <FlexItem>
                                                                                        <Tooltip
                                                                                            content={
                                                                                                'Pause the workload.'
                                                                                            }
                                                                                        >
                                                                                            <Button
                                                                                                isDisabled={
                                                                                                    workload.workload_state !=
                                                                                                    WORKLOAD_STATE_RUNNING
                                                                                                }
                                                                                                id={
                                                                                                    'pause-workload-' +
                                                                                                    idx +
                                                                                                    '-button'
                                                                                                }
                                                                                                variant="link"
                                                                                                isDanger
                                                                                                icon={<PauseIcon />}
                                                                                                onClick={(
                                                                                                    event: React.MouseEvent<
                                                                                                        HTMLButtonElement,
                                                                                                        MouseEvent
                                                                                                    >,
                                                                                                ) => {
                                                                                                    onPauseWorkloadClicked(
                                                                                                        workload,
                                                                                                    );

                                                                                                    event.stopPropagation();
                                                                                                }}
                                                                                            >
                                                                                                Pause
                                                                                            </Button>
                                                                                        </Tooltip>
                                                                                    </FlexItem>
                                                                                    <FlexItem>
                                                                                        <Tooltip
                                                                                            content={
                                                                                                'Stop the workload.'
                                                                                            }
                                                                                        >
                                                                                            <Button
                                                                                                isDisabled={
                                                                                                    workload.workload_state !=
                                                                                                    WORKLOAD_STATE_RUNNING
                                                                                                }
                                                                                                id={
                                                                                                    'stop-workload-' +
                                                                                                    idx +
                                                                                                    '-button'
                                                                                                }
                                                                                                variant="link"
                                                                                                isDanger
                                                                                                icon={<StopIcon />}
                                                                                                onClick={(
                                                                                                    event: React.MouseEvent<
                                                                                                        HTMLButtonElement,
                                                                                                        MouseEvent
                                                                                                    >,
                                                                                                ) => {
                                                                                                    onStopWorkloadClicked(
                                                                                                        workload,
                                                                                                    );

                                                                                                    event.stopPropagation();
                                                                                                }}
                                                                                            >
                                                                                                Stop
                                                                                            </Button>
                                                                                        </Tooltip>
                                                                                    </FlexItem>
                                                                                    {/* The element below is only meant to be visile for preset-based workloads, not template-based workloads. */}
                                                                                    {workload.workload_preset && (
                                                                                        <FlexItem>
                                                                                            <Tooltip
                                                                                                content={
                                                                                                    'Inspect the events of the workload'
                                                                                                }
                                                                                            >
                                                                                                <Button
                                                                                                    id={
                                                                                                        'inspect-workload-' +
                                                                                                        idx +
                                                                                                        '-button'
                                                                                                    }
                                                                                                    isDisabled={
                                                                                                        !workload.workload_preset ||
                                                                                                        workload
                                                                                                            .workload_preset
                                                                                                            .preset_type ==
                                                                                                            'CSV'
                                                                                                    }
                                                                                                    variant="link"
                                                                                                    icon={
                                                                                                        <SearchIcon />
                                                                                                    }
                                                                                                    onClick={(
                                                                                                        event: React.MouseEvent<
                                                                                                            HTMLButtonElement,
                                                                                                            MouseEvent
                                                                                                        >,
                                                                                                    ) => {
                                                                                                        onVisualizeWorkloadClicked(
                                                                                                            workload,
                                                                                                        );

                                                                                                        event.stopPropagation();
                                                                                                    }}
                                                                                                >
                                                                                                    Inspect
                                                                                                </Button>
                                                                                            </Tooltip>
                                                                                        </FlexItem>
                                                                                    )}
                                                                                </Flex>
                                                                            </FlexItem>
                                                                        </Flex>
                                                                    </FlexItem>
                                                                </Flex>
                                                                <FlexItem className="workload-descriptive-icons">
                                                                    <Flex spaceItems={{ default: 'spaceItemsMd' }}>
                                                                        <FlexItem>
                                                                            <Tooltip
                                                                                content={GetWorkloadStatusTooltip(
                                                                                    workload,
                                                                                )}
                                                                                position="bottom"
                                                                            >
                                                                                <React.Fragment>
                                                                                    {workload.workload_state ==
                                                                                        WORKLOAD_STATE_READY && (
                                                                                        <Label
                                                                                            icon={
                                                                                                <HourglassStartIcon
                                                                                                    className={
                                                                                                        text.infoColor_100
                                                                                                    }
                                                                                                />
                                                                                            }
                                                                                            color="blue"
                                                                                        >
                                                                                            Ready
                                                                                        </Label>
                                                                                    )}
                                                                                    {workload.workload_state ==
                                                                                        WORKLOAD_STATE_RUNNING && (
                                                                                        <Label
                                                                                            icon={
                                                                                                <SpinnerIcon
                                                                                                    className={
                                                                                                        'loading-icon-spin ' +
                                                                                                        text.successColor_100
                                                                                                    }
                                                                                                />
                                                                                            }
                                                                                            color="green"
                                                                                        >
                                                                                            Running
                                                                                        </Label>
                                                                                        // <React.Fragment>

                                                                                        //     Running
                                                                                        // </React.Fragment>
                                                                                    )}
                                                                                    {workload.workload_state ==
                                                                                        WORKLOAD_STATE_FINISHED && (
                                                                                        <Label
                                                                                            icon={
                                                                                                <CheckCircleIcon
                                                                                                    className={
                                                                                                        text.successColor_100
                                                                                                    }
                                                                                                />
                                                                                            }
                                                                                            color="green"
                                                                                        >
                                                                                            Complete
                                                                                        </Label>
                                                                                        // <React.Fragment>
                                                                                        //     <CheckCircleIcon
                                                                                        //         className={
                                                                                        //             text.successColor_100
                                                                                        //         }
                                                                                        //     />
                                                                                        //     {' Complete'}
                                                                                        // </React.Fragment>
                                                                                    )}
                                                                                    {workload.workload_state ==
                                                                                        WORKLOAD_STATE_ERRED && (
                                                                                        <Label
                                                                                            icon={
                                                                                                <TimesCircleIcon
                                                                                                    className={
                                                                                                        text.dangerColor_100
                                                                                                    }
                                                                                                />
                                                                                            }
                                                                                            color="red"
                                                                                        >
                                                                                            Erred
                                                                                        </Label>
                                                                                        // <React.Fragment>
                                                                                        //     <TimesCircleIcon
                                                                                        //         className={
                                                                                        //             text.dangerColor_100
                                                                                        //         }
                                                                                        //     />
                                                                                        //     {' Erred'}
                                                                                        // </React.Fragment>
                                                                                    )}
                                                                                    {workload.workload_state ==
                                                                                        WORKLOAD_STATE_TERMINATED && (
                                                                                        <Label
                                                                                            icon={
                                                                                                <ExclamationTriangleIcon
                                                                                                    className={
                                                                                                        text.warningColor_100
                                                                                                    }
                                                                                                />
                                                                                            }
                                                                                            color="orange"
                                                                                        >
                                                                                            Terminated
                                                                                        </Label>
                                                                                        // <React.Fragment>
                                                                                        //     <ExclamationTriangleIcon
                                                                                        //         className={
                                                                                        //             text.warningColor_100
                                                                                        //         }
                                                                                        //     />
                                                                                        //     {' Terminated'}
                                                                                        // </React.Fragment>
                                                                                    )}
                                                                                </React.Fragment>
                                                                            </Tooltip>
                                                                        </FlexItem>
                                                                        {workload.workload_preset && (
                                                                            <FlexItem>
                                                                                <Tooltip
                                                                                    content={'Workload preset.'}
                                                                                    position="bottom"
                                                                                >
                                                                                    <React.Fragment>
                                                                                        <BlueprintIcon /> &quot;
                                                                                        {workload.workload_preset_name}
                                                                                        &quot;
                                                                                    </React.Fragment>
                                                                                </Tooltip>
                                                                            </FlexItem>
                                                                        )}
                                                                        {workload.workload_template && (
                                                                            <FlexItem>
                                                                                <Tooltip
                                                                                    content={'Workload template.'}
                                                                                    position="bottom"
                                                                                >
                                                                                    {/* <Label icon={<BlueprintIcon />}>&quot;{workload.workload_template.name}&quot;</Label> */}
                                                                                    <React.Fragment>
                                                                                        <BlueprintIcon /> &quot;
                                                                                        {'Workload Template'}&quot;
                                                                                    </React.Fragment>
                                                                                </Tooltip>
                                                                            </FlexItem>
                                                                        )}
                                                                        {workload.workload_preset && (
                                                                            <FlexItem
                                                                                hidden={
                                                                                    workload.workload_preset
                                                                                        .preset_type == 'XML'
                                                                                }
                                                                            >
                                                                                <Tooltip
                                                                                    content={
                                                                                        'Months of trace data included in the workload.'
                                                                                    }
                                                                                    position="bottom"
                                                                                >
                                                                                    <React.Fragment>
                                                                                        <OutlinedCalendarAltIcon />{' '}
                                                                                        {
                                                                                            workload.workload_preset
                                                                                                .months_description
                                                                                        }
                                                                                    </React.Fragment>
                                                                                </Tooltip>
                                                                            </FlexItem>
                                                                        )}
                                                                        <FlexItem>
                                                                            <Tooltip
                                                                                content={'Workload seed.'}
                                                                                position="bottom"
                                                                            >
                                                                                {/* <Label icon={<DiceIcon />}>{workload.seed}</Label> */}
                                                                                <React.Fragment>
                                                                                    <DiceIcon /> {workload.seed}
                                                                                </React.Fragment>
                                                                            </Tooltip>
                                                                        </FlexItem>
                                                                        <FlexItem>
                                                                            <Tooltip
                                                                                content={'Timescale Adjustment Factor.'}
                                                                                position="bottom"
                                                                            >
                                                                                {/* <Label icon={<ClockIcon />}>{workload.timescale_adjustment_factor}</Label> */}
                                                                                <React.Fragment>
                                                                                    <ClockIcon />{' '}
                                                                                    {
                                                                                        workload.timescale_adjustment_factor
                                                                                    }
                                                                                </React.Fragment>
                                                                            </Tooltip>
                                                                        </FlexItem>
                                                                        <FlexItem>
                                                                            <Tooltip
                                                                                content={
                                                                                    'Total number of Sessions involved in the workload.'
                                                                                }
                                                                                position="bottom"
                                                                            >
                                                                                <React.Fragment>
                                                                                    <CubeIcon />{' '}
                                                                                    {workload.sessions.length}
                                                                                </React.Fragment>
                                                                            </Tooltip>
                                                                        </FlexItem>

                                                                        <FlexItem
                                                                            align={{ default: 'alignRight' }}
                                                                            alignSelf={{ default: 'alignSelfFlexEnd' }}
                                                                        >
                                                                            <Switch
                                                                                id={
                                                                                    'workload-' +
                                                                                    workload.id +
                                                                                    '-debug-logging-switch'
                                                                                }
                                                                                isDisabled={IsWorkloadFinished(
                                                                                    workload,
                                                                                )}
                                                                                label={'Debug logging'}
                                                                                aria-label="debug-logging-switch"
                                                                                isChecked={
                                                                                    workload.debug_logging_enabled
                                                                                }
                                                                                ouiaId="DebugLoggingSwitch"
                                                                                onChange={() => {
                                                                                    toggleDebugLogs(
                                                                                        workload.id,
                                                                                        !workload.debug_logging_enabled,
                                                                                    );
                                                                                }}
                                                                            />
                                                                        </FlexItem>
                                                                    </Flex>
                                                                </FlexItem>
                                                                <FlexItem className="workload-descriptive-icons">
                                                                    <Flex
                                                                        direction={{ default: 'column' }}
                                                                        spaceItems={{ default: 'spaceItemsNone' }}
                                                                    >
                                                                        <FlexItem>
                                                                            <Text component={TextVariants.small}>
                                                                                <strong>Runtime Metrics</strong>
                                                                            </Text>
                                                                        </FlexItem>
                                                                        <FlexItem>
                                                                            <Label>
                                                                                <Flex direction={{ default: 'row' }}>
                                                                                    <FlexItem>
                                                                                        <Tooltip
                                                                                            content={
                                                                                                'Number of events processed.'
                                                                                            }
                                                                                            position="bottom"
                                                                                        >
                                                                                            <React.Fragment>
                                                                                                <MonitoringIcon />{' '}
                                                                                                {
                                                                                                    workload.num_events_processed
                                                                                                }
                                                                                            </React.Fragment>
                                                                                        </Tooltip>
                                                                                    </FlexItem>
                                                                                    <FlexItem>
                                                                                        <Tooltip
                                                                                            content={
                                                                                                'Number of training events completed.'
                                                                                            }
                                                                                            position="bottom"
                                                                                        >
                                                                                            <React.Fragment>
                                                                                                <CodeIcon />{' '}
                                                                                                {
                                                                                                    workload.num_tasks_executed
                                                                                                }
                                                                                            </React.Fragment>
                                                                                        </Tooltip>
                                                                                    </FlexItem>
                                                                                    {/* We only show the 'time elapsed' icon and field if the time elapsed
                                                                        string is non-empty, which indicates that the workload has started. */}
                                                                                    {workload.time_elapsed_str !==
                                                                                        '' && (
                                                                                        <FlexItem>
                                                                                            <Tooltip
                                                                                                content={
                                                                                                    'Time elapsed since the workload began.'
                                                                                                }
                                                                                                position="bottom"
                                                                                            >
                                                                                                <React.Fragment>
                                                                                                    <StopwatchIcon />{' '}
                                                                                                    {
                                                                                                        workload.time_elapsed_str
                                                                                                    }
                                                                                                </React.Fragment>
                                                                                            </Tooltip>
                                                                                        </FlexItem>
                                                                                    )}
                                                                                    <FlexItem>
                                                                                        <Tooltip content="The current value of the internal workload/simulation clock.">
                                                                                            <React.Fragment>
                                                                                                <UserClockIcon />{' '}
                                                                                                {workload.simulation_clock_time ==
                                                                                                ''
                                                                                                    ? 'N/A'
                                                                                                    : workload.simulation_clock_time}
                                                                                            </React.Fragment>
                                                                                        </Tooltip>
                                                                                    </FlexItem>
                                                                                    <FlexItem>
                                                                                        <Tooltip content="The current tick of the workload.">
                                                                                            <React.Fragment>
                                                                                                <Stopwatch20Icon />{' '}
                                                                                                {workload.current_tick}
                                                                                            </React.Fragment>
                                                                                        </Tooltip>
                                                                                    </FlexItem>
                                                                                    <FlexItem>
                                                                                        <Tooltip content="Number of active sessions right now.">
                                                                                            <React.Fragment>
                                                                                                <CubeIcon />{' '}
                                                                                                {
                                                                                                    workload.num_active_sessions
                                                                                                }
                                                                                            </React.Fragment>
                                                                                        </Tooltip>
                                                                                    </FlexItem>
                                                                                    <FlexItem>
                                                                                        <Tooltip content="Number of actively-training sessions right now.">
                                                                                            <React.Fragment>
                                                                                                <RunningIcon />{' '}
                                                                                                {
                                                                                                    workload.num_active_trainings
                                                                                                }
                                                                                            </React.Fragment>
                                                                                        </Tooltip>
                                                                                    </FlexItem>
                                                                                </Flex>
                                                                            </Label>
                                                                        </FlexItem>
                                                                    </Flex>
                                                                </FlexItem>
                                                            </Flex>
                                                        </DataListCell>,
                                                    ]}
                                                />
                                            </DataListItemRow>
                                        </DataListItem>
                                    ))}
                            </DataList>
                            <Pagination
                                hidden={workloads.length == 0}
                                isDisabled={workloads.length == 0}
                                itemCount={workloads.length}
                                widgetId="workload-list-pagination"
                                perPage={perPage}
                                page={page}
                                variant={PaginationVariant.bottom}
                                perPageOptions={[
                                    {
                                        title: '1 workloads',
                                        value: 1,
                                    },
                                    {
                                        title: '2 workloads',
                                        value: 2,
                                    },
                                    {
                                        title: '3 workloads',
                                        value: 3,
                                    },
                                ]}
                                onSetPage={onSetPage}
                                onPerPageSelect={onPerPageSelect}
                            />
                        </React.Fragment>
                    )}

                    <RegisterWorkloadModal
                        isOpen={isRegisterWorkloadModalOpen}
                        onClose={onCancelStartWorkload}
                        onConfirm={onConfirmRegisterWorkload}
                        onRegisterWorkloadFromTemplateClicked={onRegisterWorkloadFromTemplateClicked}
                    />
                    <SessionTabsDataProvider>
                        <NewWorkloadFromTemplateModal
                            isOpen={isRegisterNewWorkloadFromTemplateModalOpen}
                            onClose={onCancelStartWorkloadFromTemplate}
                            onConfirm={onConfirmRegisterWorkloadFromTemplate}
                        />
                    </SessionTabsDataProvider>
                </CardBody>
            </Card>

            <VisualizeWorkloadModal
                isOpen={visualizeWorkloadModalOpen}
                workload={workloadBeingVisualized}
                onClose={onCloseVisualizeWorkloadModal}
            />

            <InspectWorkloadModal
                isOpen={inspectWorkloadModalOpen}
                workload={workloadBeingInspected}
                onClose={onCloseInspectWorkloadModal}
                onStartClicked={() => {
                    if (workloadBeingInspected) onStartWorkloadClicked(workloadBeingInspected);
                }}
                onStopClicked={() => {
                    if (workloadBeingInspected) onStopWorkloadClicked(workloadBeingInspected);
                }}
            />
        </React.Fragment>
    );
};
