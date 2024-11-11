import { RoundToThreeDecimalPlaces } from '@Components/Modals';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    FlexItem,
} from '@patternfly/react-core';
import {
    BlueprintIcon,
    ClipboardCheckIcon,
    ClockIcon,
    CodeIcon,
    DiceIcon,
    MonitoringIcon,
    Stopwatch20Icon,
    StopwatchIcon,
    UserClockIcon,
} from '@patternfly/react-icons';
import { WorkloadEventTable, WorkloadSessionTable } from '@src/Components';
import { Workload } from '@src/Data';
import { GetToastContentWithHeaderAndBody } from '@src/Utils/toast_utils';
import React from 'react';
import toast from 'react-hot-toast';

interface IWorkloadInspectionViewProps {
    workload: Workload;
}

export const WorkloadInspectionView: React.FunctionComponent<IWorkloadInspectionViewProps> = (
    props: IWorkloadInspectionViewProps,
) => {
    const [currentTick, setCurrentTick] = React.useState<number>(0);
    const tickStartTime = React.useRef<Map<string, number>>(new Map<string, number>());
    const tickDurations = React.useRef<Map<string, number[]>>(new Map<string, number[]>());

    // Map from workload ID to the largest tick for which we've shown a toast notification
    // about the workload being incremented to that tick.
    const showedTickNotifications = React.useRef<Map<string, number>>(new Map<string, number>());

    const shouldShowTickNotification = (workloadId: string, tick: number): boolean => {
        if (
            !showedTickNotifications ||
            !showedTickNotifications.current ||
            !showedTickNotifications.current.has(workloadId)
        ) {
            return false;
        }

        const lastTickNotification: number = showedTickNotifications.current.get(workloadId) || -1;

        return tick > lastTickNotification;
    };

    React.useEffect(() => {
        if (props.workload && props.workload?.current_tick > currentTick) {
            if (!tickStartTime.current.has(props.workload.id)) {
                tickStartTime.current.set(props.workload.id, performance.now());
            } else {
                const tickDuration: number = performance.now() - tickStartTime.current[props.workload.id];

                const durations: number[] | undefined = tickDurations.current.get(props.workload.id) || [];
                durations.push(tickDuration);
                tickDurations.current.set(props.workload.id, durations);
                tickStartTime.current.set(props.workload.id, performance.now());
            }

            setCurrentTick(props.workload.current_tick);

            if (shouldShowTickNotification(props.workload.id, props.workload.current_tick)) {
                toast.custom(
                    (t) =>
                        GetToastContentWithHeaderAndBody(
                            'Tick Incremented',
                            `Workload ${props.workload?.name} has progressed to Tick #${props.workload?.current_tick}.`,
                            'info',
                            () => {
                                toast.dismiss(t.id);
                            },
                        ),
                    { icon: '⏱️', style: { maxWidth: 700 } },
                );

                showedTickNotifications.current.set(props.workload.id, props.workload.current_tick);
            }
        }
    }, [currentTick, props.workload, props.workload?.current_tick]);

    const getTimeElapsedString = () => {
        if (props.workload?.workload_state === undefined || props.workload?.workload_state === '') {
            return 'N/A';
        }

        return props.workload?.time_elapsed_str;
    };

    const getLastTickDuration = () => {
        if (!tickDurations || !tickDurations.current) {
            return 'N/A';
        }

        const durations: number[] | undefined = tickDurations.current.get(props.workload.id);

        if (!durations) {
            return 'N/A';
        }

        return RoundToThreeDecimalPlaces(durations[durations.length - 1]);
    };

    const getAverageTickDuration = () => {
        if (!tickDurations || !tickDurations.current) {
            return 'N/A';
        }

        const durations: number[] | undefined = tickDurations.current.get(props.workload.id);

        if (!durations || durations.length === 0) {
            return 'N/A';
        }

        let sum: number = 0.0;
        durations.forEach((val: number) => {
            sum = sum + val;
        });
        return RoundToThreeDecimalPlaces(sum / durations.length);
    };

    return (
        <Flex direction={{ default: 'column' }}>
            <FlexItem>
                <DescriptionList columnModifier={{ lg: '3Col' }}>
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
                            Seed <DiceIcon />{' '}
                        </DescriptionListTerm>
                        <DescriptionListDescription>{props.workload?.seed}</DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>
                            Time Adjustment Factor <ClockIcon />{' '}
                        </DescriptionListTerm>
                        <DescriptionListDescription>
                            {props.workload?.timescale_adjustment_factor}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>
                            Events Processed <MonitoringIcon />
                        </DescriptionListTerm>
                        <DescriptionListDescription>{props.workload?.num_events_processed}</DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>
                            Training Events Completed <CodeIcon />
                        </DescriptionListTerm>
                        <DescriptionListDescription>{props.workload?.num_tasks_executed}</DescriptionListDescription>
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
                        <DescriptionListDescription>{props.workload?.current_tick}</DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>
                            Last Tick Duration (ms) <ClockIcon />
                        </DescriptionListTerm>
                        <DescriptionListDescription>{getLastTickDuration()}</DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>
                            Average Tick Duration (ms) <ClockIcon />
                        </DescriptionListTerm>
                        <DescriptionListDescription>{getAverageTickDuration()}</DescriptionListDescription>
                    </DescriptionListGroup>
                </DescriptionList>
            </FlexItem>
            <FlexItem>
                <ClipboardCheckIcon /> {<strong>Events Processed:</strong>} {props.workload?.num_events_processed}
            </FlexItem>
            <FlexItem>
                <WorkloadEventTable workload={props.workload} />
            </FlexItem>
            <FlexItem>
                <ClipboardCheckIcon /> {<strong>Sessions:</strong>} {props.workload?.num_sessions_created} /{' '}
                {props.workload?.sessions.length} created, {props.workload?.num_active_trainings} actively training
            </FlexItem>
            <FlexItem>
                <WorkloadSessionTable workload={props.workload} />
            </FlexItem>
        </Flex>
    );
};
