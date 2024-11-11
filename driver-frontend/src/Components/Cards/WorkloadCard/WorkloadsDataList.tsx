import { HeightFactorContext, WorkloadsHeightFactorContext } from '@App/Dashboard';
import {
    Button,
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
    RunningIcon,
    SearchIcon,
    SpinnerIcon,
    StopIcon,
    Stopwatch20Icon,
    StopwatchIcon,
    TimesCircleIcon,
    UserClockIcon,
} from '@patternfly/react-icons';

import text from '@patternfly/react-styles/css/utilities/Text/text';
import { CsvFileIcon, TemplateIcon, XmlFileIcon } from '@src/Assets/Icons';

import {
    GetWorkloadStatusTooltip,
    IsWorkloadFinished,
    WORKLOAD_STATE_ERRED,
    WORKLOAD_STATE_FINISHED,
    WORKLOAD_STATE_READY,
    WORKLOAD_STATE_RUNNING,
    WORKLOAD_STATE_TERMINATED,
    Workload,
} from '@src/Data/Workload';
import React from 'react';

export interface IWorkloadsDataListProps {
    workloads: Workload[];
    onPauseWorkloadClicked: (workload: Workload) => void;
    toggleDebugLogs: (workloadId: string, enabled: boolean) => void;
    onSelectWorkload: (event: React.MouseEvent | React.KeyboardEvent, id: string) => void;
    onClickWorkload: (workload: Workload) => void;
    onVisualizeWorkloadClicked: (workload: Workload) => void;
    onStartWorkloadClicked: (workload: Workload) => void;
    onStopWorkloadClicked: (workload: Workload) => void;
    workloadsPerPage?: number;
    selectedWorkloadListId: string;
}

// eslint-disable-next-line prefer-const
let WorkloadsDataList: React.FunctionComponent<IWorkloadsDataListProps> = (props: IWorkloadsDataListProps) => {
    const [page, setPage] = React.useState(1);
    const [perPage, setPerPage] = React.useState(props.workloadsPerPage || 3);

    const heightFactorContext: HeightFactorContext = React.useContext(WorkloadsHeightFactorContext);

    const onPerPageSelect = (
        _event: React.MouseEvent | React.KeyboardEvent | MouseEvent,
        newPerPage: number,
        newPage: number,
    ) => {
        setPerPage(newPerPage);
        setPage(newPage);

        heightFactorContext.setHeightFactor(Math.min(props.workloads.length, newPerPage));
    };

    const onSetPage = (_event: React.MouseEvent | React.KeyboardEvent | MouseEvent, newPage: number) => {
        setPage(newPage);
        // console.log(
        //     'onSetPage: Displaying workloads %d through %d.',
        //     perPage * (newPage - 1),
        //     perPage * (newPage - 1) + perPage,
        // );
    };

    return (
        <React.Fragment>
            <DataList
                isCompact
                aria-label="data list"
                selectedDataListItemId={props.selectedWorkloadListId}
                onSelectDataListItem={props.onSelectWorkload}
            >
                {props.workloads
                    .slice(perPage * (page - 1), perPage * (page - 1) + perPage)
                    .map((workload: Workload, idx: number) => (
                        <DataListItem
                            key={workload.id}
                            id={workload.id}
                            onClick={() => {
                                props.onClickWorkload(workload);
                            }}
                        >
                            <DataListItemRow>
                                <DataListItemCells
                                    dataListCells={[
                                        <DataListCell key={'workload-primary-content-' + idx} isFilled={true} width={4}>
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
                                                                )}
                                                                {workload.workload_template && (
                                                                    <FlexItem>
                                                                        <Tooltip
                                                                            content={`This workload was created/defined using a template.`}
                                                                            position="bottom"
                                                                        >
                                                                            <React.Fragment>
                                                                                <TemplateIcon scale={3.25} />
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
                                                                        <Tooltip content={'Start the workload'}>
                                                                            <Button
                                                                                id={'start-workload-' + idx + '-button'}
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
                                                                                    props.onStartWorkloadClicked(
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
                                                                        <Tooltip content={'Pause the workload.'}>
                                                                            <Button
                                                                                isDisabled={
                                                                                    workload.workload_state !=
                                                                                    WORKLOAD_STATE_RUNNING
                                                                                }
                                                                                id={'pause-workload-' + idx + '-button'}
                                                                                variant="link"
                                                                                isDanger
                                                                                icon={<PauseIcon />}
                                                                                onClick={(
                                                                                    event: React.MouseEvent<
                                                                                        HTMLButtonElement,
                                                                                        MouseEvent
                                                                                    >,
                                                                                ) => {
                                                                                    props.onPauseWorkloadClicked(
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
                                                                        <Tooltip content={'Stop the workload.'}>
                                                                            <Button
                                                                                isDisabled={
                                                                                    workload.workload_state !=
                                                                                    WORKLOAD_STATE_RUNNING
                                                                                }
                                                                                id={'stop-workload-' + idx + '-button'}
                                                                                variant="link"
                                                                                isDanger
                                                                                icon={<StopIcon />}
                                                                                onClick={(
                                                                                    event: React.MouseEvent<
                                                                                        HTMLButtonElement,
                                                                                        MouseEvent
                                                                                    >,
                                                                                ) => {
                                                                                    props.onStopWorkloadClicked(
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
                                                                                        workload.workload_preset
                                                                                            .preset_type == 'CSV'
                                                                                    }
                                                                                    variant="link"
                                                                                    icon={<SearchIcon />}
                                                                                    onClick={(
                                                                                        event: React.MouseEvent<
                                                                                            HTMLButtonElement,
                                                                                            MouseEvent
                                                                                        >,
                                                                                    ) => {
                                                                                        props.onVisualizeWorkloadClicked(
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
                                                                content={GetWorkloadStatusTooltip(workload)}
                                                                position="bottom"
                                                            >
                                                                <React.Fragment>
                                                                    {workload.workload_state ==
                                                                        WORKLOAD_STATE_READY && (
                                                                        <Label
                                                                            icon={
                                                                                <HourglassStartIcon
                                                                                    className={text.infoColor_100}
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
                                                                                    className={text.successColor_100}
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
                                                                                    className={text.dangerColor_100}
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
                                                                                    className={text.warningColor_100}
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
                                                                <Tooltip content={'Workload preset.'} position="bottom">
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
                                                                hidden={workload.workload_preset.preset_type == 'XML'}
                                                            >
                                                                <Tooltip
                                                                    content={
                                                                        'Months of trace data included in the workload.'
                                                                    }
                                                                    position="bottom"
                                                                >
                                                                    <React.Fragment>
                                                                        <OutlinedCalendarAltIcon />{' '}
                                                                        {workload.workload_preset.months_description}
                                                                    </React.Fragment>
                                                                </Tooltip>
                                                            </FlexItem>
                                                        )}
                                                        <FlexItem>
                                                            <Tooltip content={'Workload seed.'} position="bottom">
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
                                                                    <ClockIcon /> {workload.timescale_adjustment_factor}
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
                                                                    <CubeIcon /> {workload.sessions.length}
                                                                </React.Fragment>
                                                            </Tooltip>
                                                        </FlexItem>

                                                        <FlexItem
                                                            align={{ default: 'alignRight' }}
                                                            alignSelf={{ default: 'alignSelfFlexEnd' }}
                                                        >
                                                            <Switch
                                                                id={'workload-' + workload.id + '-debug-logging-switch'}
                                                                isDisabled={IsWorkloadFinished(workload)}
                                                                label={'Debug logging'}
                                                                aria-label="debug-logging-switch"
                                                                isChecked={workload.debug_logging_enabled}
                                                                ouiaId="DebugLoggingSwitch"
                                                                onChange={() => {
                                                                    props.toggleDebugLogs(
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
                                                                                'Number of training events completed.'
                                                                            }
                                                                            position="bottom"
                                                                        >
                                                                            <React.Fragment>
                                                                                <CodeIcon />{' '}
                                                                                {workload.num_tasks_executed}
                                                                            </React.Fragment>
                                                                        </Tooltip>
                                                                    </FlexItem>
                                                                    {/* We only show the 'time elapsed' icon and field if the time elapsed
                                                                        string is non-empty, which indicates that the workload has started. */}
                                                                    {workload.time_elapsed_str !== '' && (
                                                                        <FlexItem>
                                                                            <Tooltip
                                                                                content={
                                                                                    'Time elapsed since the workload began.'
                                                                                }
                                                                                position="bottom"
                                                                            >
                                                                                <React.Fragment>
                                                                                    <StopwatchIcon />{' '}
                                                                                    {workload.time_elapsed_str}
                                                                                </React.Fragment>
                                                                            </Tooltip>
                                                                        </FlexItem>
                                                                    )}
                                                                    <FlexItem>
                                                                        <Tooltip content="The current value of the internal workload/simulation clock.">
                                                                            <React.Fragment>
                                                                                <UserClockIcon />{' '}
                                                                                {workload.simulation_clock_time == ''
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
                                                                                {workload.num_active_sessions}
                                                                            </React.Fragment>
                                                                        </Tooltip>
                                                                    </FlexItem>
                                                                    <FlexItem>
                                                                        <Tooltip content="Number of actively-training sessions right now.">
                                                                            <React.Fragment>
                                                                                <RunningIcon />{' '}
                                                                                {workload.num_active_trainings}
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
                hidden={props.workloads.length == 0}
                isDisabled={props.workloads.length == 0}
                itemCount={props.workloads.length}
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
    );
};

export { WorkloadsDataList };
