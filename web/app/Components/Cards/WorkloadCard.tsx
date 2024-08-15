import React from 'react';
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

import text from '@patternfly/react-styles/css/utilities/Text/text';

import {
    BlueprintIcon,
    CheckCircleIcon,
    ClockIcon,
    CodeIcon,
    DiceIcon,
    ExclamationTriangleIcon,
    HourglassStartIcon,
    MonitoringIcon,
    OutlinedCalendarAltIcon,
    PauseIcon,
    PlayIcon,
    PlusIcon,
    SearchIcon,
    SpinnerIcon,
    StopCircleIcon,
    StopIcon,
    TimesCircleIcon,
} from '@patternfly/react-icons';

import {
    Session,
    WORKLOAD_STATE_ERRED,
    WORKLOAD_STATE_FINISHED,
    WORKLOAD_STATE_READY,
    WORKLOAD_STATE_RUNNING,
    WORKLOAD_STATE_TERMINATED,
    Workload,
    WorkloadPreset,
} from '@app/Data/Workload';
import { useWorkloads } from '@providers/WorkloadProvider';
import { HeightFactorContext, WorkloadsHeightFactorContext } from '@app/Dashboard/Dashboard';
import { toast } from 'react-hot-toast';
import { v4 as uuidv4 } from 'uuid';
import { InspectWorkloadModal, NewWorkloadFromTemplateModal, RegisterWorkloadModal } from '../Modals';
import { CsvFileIcon, XmlFileIcon } from '@app/Icons';

export interface WorkloadCardProps {
    workloadsPerPage: number;
}

export const WorkloadCard: React.FunctionComponent<WorkloadCardProps> = (props: WorkloadCardProps) => {
    const [isRegisterWorkloadModalOpen, setIsRegisterWorkloadModalOpen] = React.useState(false);
    const [isRegisterNewWorkloadFromTemplateModalOpen, setIsRegisterNewWorkloadFromTemplateModalOpen] = React.useState(false);
    const [selectedWorkloadListId, setSelectedWorkloadListId] = React.useState('');
    const [page, setPage] = React.useState(1);
    const [perPage, setPerPage] = React.useState(props.workloadsPerPage);

    const [inspectWorkloadModalOpen, setInspectWorkloadModalOpen] = React.useState(false);
    const [workloadBeingInspected, setWorkloadBeingInspected] = React.useState<Workload | null>(null);

    const heightFactorContext: HeightFactorContext = React.useContext(WorkloadsHeightFactorContext);

    const { workloads, sendJsonMessage } = useWorkloads();

    const onCloseInspectWorkloadModal = () => {
        setWorkloadBeingInspected(null);
        setInspectWorkloadModalOpen(false);
    };

    const onSetPage = (_event: React.MouseEvent | React.KeyboardEvent | MouseEvent, newPage: number) => {
        setPage(newPage);
        console.log(
            'onSetPage: Displaying workloads %d through %d.',
            perPage * (newPage - 1),
            perPage * (newPage - 1) + perPage,
        );
    };

    const onConfirmRegisterWorkloadFromTemplate = (
        workloadTitle: string,
        workloadSeed: string,
        debugLoggingEnabled: boolean,
        sessions: Session[],
    ) => {
    }

    const onRegisterWorkloadFromTemplateClicked = () => {
        setIsRegisterNewWorkloadFromTemplateModalOpen(true)
    }

    const onConfirmRegisterWorkload = (
        workloadName: string,
        selectedPreset: WorkloadPreset,
        workloadSeedString: string,
        debugLoggingEnabled: boolean,
    ) => {
        toast('Registering workload now.', {
            icon: '🛈',
        });

        console.log("New workload '%s' registered by user with preset:\n%s", workloadName, selectedPreset.name);
        setIsRegisterWorkloadModalOpen(false);

        let workloadSeed = -1;

        if (workloadSeedString != '') {
            workloadSeed = parseInt(workloadSeedString);
        }

        const messageId: string = uuidv4();
        sendJsonMessage(
            JSON.stringify({
                op: 'register_workload',
                msg_id: messageId,
                workloadRegistrationRequest: {
                    adjust_gpu_reservations: false,
                    seed: workloadSeed,
                    key: selectedPreset.key,
                    name: workloadName,
                    debug_logging: debugLoggingEnabled,
                },
            }),
        );
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

    const onInspectWorkloadClicked = (workload: Workload) => {
        console.log(`Inspecting workload: ${workload.name} (id=${workload.name})`);
        console.log(workload);

        setWorkloadBeingInspected(workload);
        setInspectWorkloadModalOpen(true);
    };

    const onStartWorkloadClicked = (workload: Workload) => {
        toast(() => (
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
                <FlexItem>
                    <b>Starting workload {workload.name}</b>
                </FlexItem>
                <FlexItem>
                    <Text component={TextVariants.small}>
                        <b>Workload ID: </b>
                        {workload.id}
                    </Text>
                </FlexItem>
            </Flex>
        ));

        console.log(`Starting workload '${workload.name}' (ID=${workload.id})`);

        const messageId: string = uuidv4();
        sendJsonMessage(
            JSON.stringify({
                op: 'start_workload',
                msg_id: messageId,
                workload_id: workload.id,
            }),
        );
    };

    const onPauseWorkloadClicked = (workload: Workload) => {
        toast(`Pausing workload ${workload.name} (ID=${workload.id}) now...`);
    };

    const onStopWorkloadClicked = (workload: Workload) => {
        toast(`Stopping workload ${workload.name} (ID = ${workload.id}).`);

        console.log("Stopping workload '%s' (ID=%s)", workload.name, workload.id);

        const messageId: string = uuidv4();
        sendJsonMessage(
            JSON.stringify({
                op: 'stop_workload',
                msg_id: messageId,
                workload_id: workload.id,
            }),
        );
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

    const getWorkloadStatusTooltip = (workload: Workload) => {
        switch (workload.workload_state) {
            case WORKLOAD_STATE_READY:
                return 'The workload has been registered and is ready to begin.';
            case WORKLOAD_STATE_RUNNING:
                return 'The workload is actively-running.';
            case WORKLOAD_STATE_FINISHED:
                return 'The workload has completed successfully.';
            case WORKLOAD_STATE_ERRED:
                return 'The workload has been aborted due to a critical error: ' + workload.error_message;
            case WORKLOAD_STATE_TERMINATED:
                return 'The workload has been explicitly/manually terminated.';
        }

        console.error(
            `Workload ${workload.name} (ID=${workload.id}) is in an unsupported/unknown state: ${workload.workload_state}`,
        );
        return 'The workload is currently in an unknown/unsupported state.';
    };

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
                        <DataList
                            isCompact
                            aria-label="data list"
                            selectedDataListItemId={selectedWorkloadListId}
                            onSelectDataListItem={onSelectWorkload}
                        >
                            {workloads
                                .slice(perPage * (page - 1), perPage * (page - 1) + perPage)
                                .map((workload: Workload, idx: number) => (
                                    <DataListItem key={workload.id} id={workload.id}>
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
                                                                                    {workload.name}
                                                                                </Text>
                                                                            </FlexItem>
                                                                            <FlexItem>
                                                                                <Tooltip
                                                                                    content={`This preset is defined in a ${workload.workload_preset.preset_type} file.`}
                                                                                    position="bottom"
                                                                                >
                                                                                    <React.Fragment>
                                                                                        {workload.workload_preset
                                                                                            .preset_type === 'XML' && (
                                                                                            <XmlFileIcon scale={2.25} />
                                                                                        )}
                                                                                        {workload.workload_preset
                                                                                            .preset_type === 'CSV' && (
                                                                                            <CsvFileIcon scale={2.25} />
                                                                                        )}
                                                                                    </React.Fragment>
                                                                                </Tooltip>
                                                                            </FlexItem>
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
                                                                                spaceItems={{ default: 'spaceItemsXs' }}
                                                                            >
                                                                                <FlexItem>
                                                                                    <Tooltip
                                                                                        content={'Start the workload'}
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
                                                                                        content={'Pause the workload.'}
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
                                                                                        content={'Stop the workload.'}
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
                                                                                                workload.workload_preset
                                                                                                    .preset_type ==
                                                                                                'CSV'
                                                                                            }
                                                                                            variant="link"
                                                                                            icon={<SearchIcon />}
                                                                                            onClick={(
                                                                                                event: React.MouseEvent<
                                                                                                    HTMLButtonElement,
                                                                                                    MouseEvent
                                                                                                >,
                                                                                            ) => {
                                                                                                onInspectWorkloadClicked(
                                                                                                    workload,
                                                                                                );

                                                                                                event.stopPropagation();
                                                                                            }}
                                                                                        >
                                                                                            Inspect
                                                                                        </Button>
                                                                                    </Tooltip>
                                                                                </FlexItem>
                                                                            </Flex>
                                                                        </FlexItem>
                                                                    </Flex>
                                                                </FlexItem>
                                                            </Flex>
                                                            <FlexItem className="workload-descriptive-icons">
                                                                <Flex spaceItems={{ default: 'spaceItemsMd' }}>
                                                                    <FlexItem>
                                                                        <Tooltip
                                                                            content={getWorkloadStatusTooltip(workload)}
                                                                            position="bottom"
                                                                        >
                                                                            <React.Fragment>
                                                                                {workload.workload_state ==
                                                                                    WORKLOAD_STATE_READY && (
                                                                                    <React.Fragment>
                                                                                        <HourglassStartIcon
                                                                                            className={
                                                                                                text.successColor_100
                                                                                            }
                                                                                        />
                                                                                        {' Ready'}
                                                                                    </React.Fragment>
                                                                                )}
                                                                                {workload.workload_state ==
                                                                                    WORKLOAD_STATE_RUNNING && (
                                                                                    <React.Fragment>
                                                                                        <SpinnerIcon
                                                                                            className={
                                                                                                'loading-icon-spin ' +
                                                                                                text.successColor_100
                                                                                            }
                                                                                        />
                                                                                        {' Running'}
                                                                                    </React.Fragment>
                                                                                )}
                                                                                {workload.workload_state ==
                                                                                    WORKLOAD_STATE_FINISHED && (
                                                                                    <React.Fragment>
                                                                                        <CheckCircleIcon
                                                                                            className={
                                                                                                text.successColor_100
                                                                                            }
                                                                                        />
                                                                                        {' Complete'}
                                                                                    </React.Fragment>
                                                                                )}
                                                                                {workload.workload_state ==
                                                                                    WORKLOAD_STATE_ERRED && (
                                                                                    <React.Fragment>
                                                                                        <TimesCircleIcon
                                                                                            className={
                                                                                                text.dangerColor_100
                                                                                            }
                                                                                        />
                                                                                        {' Erred'}
                                                                                    </React.Fragment>
                                                                                )}
                                                                                {workload.workload_state ==
                                                                                    WORKLOAD_STATE_TERMINATED && (
                                                                                    <React.Fragment>
                                                                                        <ExclamationTriangleIcon
                                                                                            className={
                                                                                                text.warningColor_100
                                                                                            }
                                                                                        />
                                                                                        {' Terminated'}
                                                                                    </React.Fragment>
                                                                                )}
                                                                            </React.Fragment>
                                                                        </Tooltip>
                                                                    </FlexItem>
                                                                    <FlexItem>
                                                                        <Tooltip
                                                                            content={'Workload preset.'}
                                                                            position="bottom"
                                                                        >
                                                                            <React.Fragment>
                                                                                <BlueprintIcon /> &quot;
                                                                                {workload.workload_preset_name}&quot;
                                                                            </React.Fragment>
                                                                        </Tooltip>
                                                                    </FlexItem>
                                                                    <FlexItem
                                                                        hidden={
                                                                            workload.workload_preset.preset_type ==
                                                                            'XML'
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
                                                                    <FlexItem>
                                                                        <Tooltip
                                                                            content={'Workload seed.'}
                                                                            position="bottom"
                                                                        >
                                                                            <React.Fragment>
                                                                                <DiceIcon /> {workload.seed}
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
                                                                            label={'Debug logging'}
                                                                            aria-label="debug-logging-switch"
                                                                            isChecked={workload.debug_logging_enabled}
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
                                                                <Flex direction={{ default: 'row' }}>
                                                                    <FlexItem>
                                                                        <Tooltip
                                                                            content={'Number of events processed.'}
                                                                            position="bottom"
                                                                        >
                                                                            <React.Fragment>
                                                                                <MonitoringIcon />{' '}
                                                                                {workload.num_events_processed}
                                                                            </React.Fragment>
                                                                        </Tooltip>
                                                                    </FlexItem>
                                                                    <FlexItem>
                                                                        <Tooltip
                                                                            content={
                                                                                'Number of training sessions completed.'
                                                                            }
                                                                            position="bottom"
                                                                        >
                                                                            <React.Fragment>
                                                                                <CodeIcon />{' '}
                                                                                {workload.num_tasks_executed}
                                                                            </React.Fragment>
                                                                        </Tooltip>
                                                                    </FlexItem>
                                                                    <FlexItem>
                                                                        <Tooltip
                                                                            content={
                                                                                'Time elapsed since the workload began.'
                                                                            }
                                                                            position="bottom"
                                                                        >
                                                                            <React.Fragment>
                                                                                <ClockIcon /> {workload.time_elapsed}
                                                                            </React.Fragment>
                                                                        </Tooltip>
                                                                    </FlexItem>
                                                                </Flex>
                                                            </FlexItem>
                                                        </Flex>
                                                    </DataListCell>,
                                                    // <DataListCell
                                                    //     alignRight={true}
                                                    //     key={'workload-secondary-content-' + idx}
                                                    //     id={'workload-data-list-' + idx}
                                                    //     aria-label="Workload Actions"
                                                    //     aria-labelledby="Workload Actions"
                                                    // >

                                                    // </DataListCell>,
                                                ]}
                                            />
                                        </DataListItemRow>
                                    </DataListItem>
                                ))}
                        </DataList>
                    )}
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
                            // {
                            //     title: '4',
                            //     value: 4,
                            // },
                            // {
                            //     title: '5',
                            //     value: 5,
                            // },
                        ]}
                        onSetPage={onSetPage}
                        onPerPageSelect={onPerPageSelect}
                    />

                    <RegisterWorkloadModal
                        isOpen={isRegisterWorkloadModalOpen}
                        onClose={onCancelStartWorkload}
                        onConfirm={onConfirmRegisterWorkload}
                        onRegisterWorkloadFromTemplateClicked={onRegisterWorkloadFromTemplateClicked}
                    />
                    <NewWorkloadFromTemplateModal
                        isOpen={isRegisterNewWorkloadFromTemplateModalOpen}
                        onClose={onCancelStartWorkloadFromTemplate}
                        onConfirm={onConfirmRegisterWorkloadFromTemplate}
                    />
                </CardBody>
            </Card>

            <InspectWorkloadModal
                isOpen={inspectWorkloadModalOpen}
                workload={workloadBeingInspected}
                onClose={onCloseInspectWorkloadModal}
            />
        </React.Fragment>
    );
};
