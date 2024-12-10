import WorkloadTickDurationChart from '@Components/Workloads/WorkloadTickDurationChart';
import {
    Checkbox,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    FlexItem,
    Text,
    Tooltip,
} from '@patternfly/react-core';
import {
    BlueprintIcon,
    ClipboardCheckIcon,
    ClockIcon,
    CodeIcon,
    DiceIcon,
    MonitoringIcon,
    RunningIcon,
    Stopwatch20Icon,
    StopwatchIcon,
    TaskIcon,
    UserClockIcon,
} from '@patternfly/react-icons';
import { GetEventLabel, WorkloadEventTable, WorkloadSessionTable } from '@src/Components';
import { Workload } from '@src/Data';
import { GetToastContentWithHeaderAndBody } from '@src/Utils/toast_utils';
import { numberWithCommas, RoundToTwoDecimalPlaces } from '@Utils/utils';
import { uuidv4 } from 'lib0/random';
import React from 'react';
import toast, { Toast } from 'react-hot-toast';

interface IWorkloadInspectionViewProps {
    workload: Workload;
    showTickDurationChart: boolean;
}

export const WorkloadInspectionView: React.FunctionComponent<IWorkloadInspectionViewProps> = (
    props: IWorkloadInspectionViewProps,
) => {
    const [currentTick, setCurrentTick] = React.useState<number>(0);

    const [tickIdToastId, setTickIdToastId] = React.useState<string | undefined>(undefined);

    // Map from workload ID to the largest tick for which we've shown a toast notification
    // about the workload being incremented to that tick.
    const showedTickNotifications = React.useRef<Map<string, number>>(new Map<string, number>());

    const [showDiscardedEvents, setShowDiscardedEvents] = React.useState<boolean>(false);
    const [showDiscardedSessions, setShowDiscardedSessions] = React.useState<boolean>(false);

    const shouldShowTickNotification = (workloadId: string, tick: number): boolean => {
        if (!showedTickNotifications || !showedTickNotifications.current) {
            return false;
        }

        const lastTickNotification: number = showedTickNotifications.current.get(workloadId) || -1;

        return tick > lastTickNotification;
    };

    // TODO: This will miscount the first tick as being smaller, basically whenever we first open the workload
    //       preview to when the next tick begins, it'll count that block as the duration of the first tick,
    //       which is wrong.
    React.useEffect(() => {
        if (props.workload && props.workload?.statistics.current_tick > currentTick) {
            setCurrentTick(props.workload.statistics.current_tick);

            if (shouldShowTickNotification(props.workload.id, props.workload.statistics.current_tick)) {
                const tick: number = props.workload?.statistics.current_tick;
                const toastId: string = toast.custom(
                    (t: Toast) =>
                        GetToastContentWithHeaderAndBody(
                            'Tick Incremented',
                            `Workload ${props.workload?.name} has progressed to Tick #${tick}.`,
                            'info',
                            () => {
                                toast.dismiss(t.id);
                            },
                        ),
                    { icon: '⏱️', style: { maxWidth: 700 }, duration: 5000, id: tickIdToastId || uuidv4() },
                );

                if (tickIdToastId === undefined) {
                    setTickIdToastId(toastId);
                }

                showedTickNotifications.current.set(props.workload.id, tick);
            }
        }
    }, [currentTick, props.workload, props.workload.statistics.current_tick, tickIdToastId]);

    const getTimeElapsedString = () => {
        if (
            props.workload?.statistics.workload_state === undefined ||
            props.workload?.statistics.workload_state === ''
        ) {
            return 'N/A';
        }

        return props.workload?.statistics.time_elapsed_str;
    };

    // const getLastTickDuration = () => {
    //     if (props.workload.tick_durations_milliseconds.length === 0) {
    //         return 'N/A';
    //     }
    //
    //     return RoundToThreeDecimalPlaces(
    //         props.workload.tick_durations_milliseconds[props.workload.tick_durations_milliseconds.length - 1],
    //     );
    // };

    // const getAverageTickDuration = () => {
    //     if (props.workload.tick_durations_milliseconds.length === 0) {
    //         return 'N/A';
    //     }
    //
    //     return RoundToThreeDecimalPlaces(
    //         props.workload.sum_tick_durations_millis / props.workload.tick_durations_milliseconds.length,
    //     );
    // };

    const getCurrentTickField = () => {
        const totalTicks: number | undefined = props.workload.statistics.total_num_ticks;
        const currentTick: number | undefined = props.workload?.statistics.current_tick;

        if (totalTicks !== undefined && totalTicks > 0) {
            return `${numberWithCommas(currentTick)} / ${numberWithCommas(totalTicks)} (${RoundToTwoDecimalPlaces((currentTick / totalTicks) * 100)}%)`;
        }

        return currentTick;
    };

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsXl' }}>
            <Flex
                direction={{ default: 'row' }}
                spaceItems={{ default: 'spaceItemsNone' }}
                alignItems={{ default: props.showTickDurationChart ? 'alignItemsCenter' : 'alignItemsFlexStart' }}
                justifyContent={{
                    default: props.showTickDurationChart ? 'justifyContentCenter' : 'justifyContentFlexStart',
                }}
            >
                <FlexItem>
                    <DescriptionList columnModifier={{ lg: '3Col' }} displaySize={'lg'}>
                        {props.workload?.workload_preset && (
                            <DescriptionListGroup>
                                <DescriptionListTerm>
                                    Workload Preset <BlueprintIcon />{' '}
                                </DescriptionListTerm>
                                <DescriptionListDescription>
                                    &quot;{props.workload?.workload_preset_name}&quot;
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                        )}
                        {/* {props.workload?.workload_template && <DescriptionListGroup>
                            <DescriptionListTerm>Workload Template <BlueprintIcon /></DescriptionListTerm>
                            <DescriptionListDescription>&quot;{props.workload?.workload_template.name}&quot;</DescriptionListDescription>
                        </DescriptionListGroup>} */}
                        <DescriptionListGroup>
                            <DescriptionListTerm>
                                Seed <DiceIcon />
                            </DescriptionListTerm>
                            <DescriptionListDescription>{props.workload?.seed}</DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>
                                Time Adjustment Factor <ClockIcon />
                            </DescriptionListTerm>
                            <DescriptionListDescription>
                                {props.workload?.timescale_adjustment_factor}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>
                                Sessions Sample Percentage <TaskIcon />
                            </DescriptionListTerm>
                            <DescriptionListDescription>
                                {props.workload.statistics.sessions_sample_percentage || 1.0}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>
                                Events Processed <MonitoringIcon />
                            </DescriptionListTerm>
                            <DescriptionListDescription>
                                {props.workload?.statistics.num_events_processed}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>
                                Training Events Completed <CodeIcon />
                            </DescriptionListTerm>
                            <DescriptionListDescription>
                                {props.workload?.statistics.num_tasks_executed}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>
                                Time Elapsed <StopwatchIcon />
                            </DescriptionListTerm>
                            <DescriptionListDescription>{getTimeElapsedString()}</DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>
                                Workload Clock Time <UserClockIcon />
                            </DescriptionListTerm>
                            <DescriptionListDescription>
                                {props.workload?.simulation_clock_time == ''
                                    ? 'N/A'
                                    : props.workload?.simulation_clock_time}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>
                                Current Tick <Stopwatch20Icon />
                            </DescriptionListTerm>
                            <DescriptionListDescription>{getCurrentTickField()}</DescriptionListDescription>
                        </DescriptionListGroup>
                        {/*<DescriptionListGroup>*/}
                        {/*    <DescriptionListTerm>*/}
                        {/*        Last Tick Duration (ms) <ClockIcon />*/}
                        {/*    </DescriptionListTerm>*/}
                        {/*    <DescriptionListDescription>{getLastTickDuration()}</DescriptionListDescription>*/}
                        {/*</DescriptionListGroup>*/}
                        {/*<DescriptionListGroup>*/}
                        {/*    <DescriptionListTerm>*/}
                        {/*        Average Tick Duration (ms) <ClockIcon />*/}
                        {/*    </DescriptionListTerm>*/}
                        {/*    <DescriptionListDescription>{getAverageTickDuration()}</DescriptionListDescription>*/}
                        {/*</DescriptionListGroup>*/}
                        {/*<DescriptionListGroup>*/}
                        {/*    <DescriptionListTerm>*/}
                        {/*        Next Expected Event <RunningIcon />*/}
                        {/*    </DescriptionListTerm>*/}
                        {/*    <DescriptionListDescription>*/}
                        {/*        <Tooltip*/}
                        {/*            content={`Event will target Session ${props.workload.statistics.next_expected_event_target}`}*/}
                        {/*        >*/}
                        {/*            {props.workload.statistics.next_expected_event_name !== '' ? (*/}
                        {/*                GetEventLabel(props.workload.statistics.next_expected_event_name)*/}
                        {/*            ) : (*/}
                        {/*                <Text component={'p'}>N/A</Text>*/}
                        {/*            )}*/}
                        {/*        </Tooltip>*/}
                        {/*    </DescriptionListDescription>*/}
                        {/*</DescriptionListGroup>*/}
                        {/*<DescriptionListGroup>*/}
                        {/*    <DescriptionListTerm>*/}
                        {/*        Next Expected Event In <ClockIcon />*/}
                        {/*    </DescriptionListTerm>*/}
                        {/*    <DescriptionListDescription>*/}
                        {/*        {numberWithCommas(*/}
                        {/*            props.workload?.statistics.next_event_expected_tick -*/}
                        {/*                props.workload?.statistics.current_tick || 0,*/}
                        {/*        )}{' '}*/}
                        {/*        tick(s)*/}
                        {/*    </DescriptionListDescription>*/}
                        {/*</DescriptionListGroup>*/}
                    </DescriptionList>
                </FlexItem>
                {/*{props.showTickDurationChart && (*/}
                {/*    <FlexItem>*/}
                {/*        <WorkloadTickDurationChart workload={props.workload} />*/}
                {/*    </FlexItem>*/}
                {/*)}*/}
            </Flex>
            <FlexItem>
                <Flex direction={{ default: 'row' }}>
                    <FlexItem align={{ default: 'alignLeft' }}>
                        <ClipboardCheckIcon /> {<strong>Events Processed:</strong>}{' '}
                        {props.workload?.statistics.num_events_processed}
                    </FlexItem>
                    <FlexItem align={{ default: 'alignRight' }}>
                        <Checkbox
                            label="Show Discarded Events"
                            id={'show-discarded-events-checkbox'}
                            isChecked={showDiscardedEvents}
                            onChange={(_event: React.FormEvent<HTMLInputElement>, checked: boolean) =>
                                setShowDiscardedEvents(checked)
                            }
                        />
                    </FlexItem>
                </Flex>
                <WorkloadEventTable workload={props.workload} showDiscardedEvents={showDiscardedEvents} />
            </FlexItem>
            <FlexItem>
                <Flex direction={{ default: 'row' }}>
                    <FlexItem align={{ default: 'alignLeft' }}>
                        <ClipboardCheckIcon /> {<strong>Sessions:</strong>}{' '}
                        {props.workload?.statistics.num_sessions_created} /{' '}
                        {props.workload?.sessions.length - props.workload?.statistics.num_discarded_sessions} (
                        {RoundToTwoDecimalPlaces(
                            100 *
                                (props.workload?.statistics.num_sessions_created /
                                    (props.workload?.sessions.length -
                                        props.workload?.statistics.num_discarded_sessions)),
                        ) + '%'}
                        ) created, {props.workload?.statistics.num_active_trainings} actively training
                    </FlexItem>
                    <FlexItem align={{ default: 'alignRight' }}>
                        <Checkbox
                            label="Show Discarded Sessions"
                            id={'show-discarded-sessions-checkbox'}
                            isChecked={showDiscardedSessions}
                            onChange={(_event: React.FormEvent<HTMLInputElement>, checked: boolean) =>
                                setShowDiscardedSessions(checked)
                            }
                        />
                    </FlexItem>
                </Flex>
                <WorkloadSessionTable workload={props.workload} showDiscardedSessions={showDiscardedSessions} />
            </FlexItem>
        </Flex>
    );
};
