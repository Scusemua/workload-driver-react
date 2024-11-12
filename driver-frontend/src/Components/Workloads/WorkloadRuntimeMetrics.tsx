import { Flex, FlexItem, Label, Content, ContentVariants, Tooltip } from '@patternfly/react-core';
import {
    CodeIcon,
    CubeIcon,
    MonitoringIcon,
    RunningIcon,
    Stopwatch20Icon,
    StopwatchIcon,
    UserClockIcon,
} from '@patternfly/react-icons';
import { Workload } from '@src/Data';
import React from 'react';

interface IWorkloadRuntimeMetrics {
    workload: Workload;
}

export const WorkloadRuntimeMetrics: React.FunctionComponent<IWorkloadRuntimeMetrics> = (props: IWorkloadRuntimeMetrics) => {
    return (
        <Flex
            className="props.workload-descriptive-icons"
            direction={{ default: 'column' }}
            spaceItems={{ default: 'spaceItemsNone' }}
        >
            <FlexItem>
                <Content component={ContentVariants.small}>
                    <strong>Runtime Metrics</strong>
                </Content>
            </FlexItem>
            <FlexItem>
                <Label>
                    <Flex direction={{ default: 'row' }}>
                        <FlexItem>
                            <Tooltip content={'Number of events processed.'} position="bottom">
                                <React.Fragment>
                                    <MonitoringIcon /> {props.workload.num_events_processed}
                                </React.Fragment>
                            </Tooltip>
                        </FlexItem>
                        <FlexItem>
                            <Tooltip content={'Number of training events completed.'} position="bottom">
                                <React.Fragment>
                                    <CodeIcon /> {props.workload.num_tasks_executed}
                                </React.Fragment>
                            </Tooltip>
                        </FlexItem>
                        {/* We only show the 'time elapsed' icon and field if the time elapsed
                                                                        string is non-empty, which indicates that the props.workload has started. */}
                        {props.workload.time_elapsed_str !== '' && (
                            <FlexItem>
                                <Tooltip content={'Time elapsed since the props.workload began.'} position="bottom">
                                    <React.Fragment>
                                        <StopwatchIcon /> {props.workload.time_elapsed_str}
                                    </React.Fragment>
                                </Tooltip>
                            </FlexItem>
                        )}
                        <FlexItem>
                            <Tooltip content="The current value of the internal props.workload/simulation clock.">
                                <React.Fragment>
                                    <UserClockIcon />{' '}
                                    {props.workload.simulation_clock_time == ''
                                        ? 'N/A'
                                        : props.workload.simulation_clock_time}
                                </React.Fragment>
                            </Tooltip>
                        </FlexItem>
                        <FlexItem>
                            <Tooltip content="The current tick of the props.workload.">
                                <React.Fragment>
                                    <Stopwatch20Icon /> {props.workload.current_tick}
                                </React.Fragment>
                            </Tooltip>
                        </FlexItem>
                        <FlexItem>
                            <Tooltip content="Number of active sessions right now.">
                                <React.Fragment>
                                    <CubeIcon /> {props.workload.num_active_sessions}
                                </React.Fragment>
                            </Tooltip>
                        </FlexItem>
                        <FlexItem>
                            <Tooltip content="Number of actively-training sessions right now.">
                                <React.Fragment>
                                    <RunningIcon /> {props.workload.num_active_trainings}
                                </React.Fragment>
                            </Tooltip>
                        </FlexItem>
                    </Flex>
                </Label>
            </FlexItem>
        </Flex>
    );
};

export default WorkloadRuntimeMetrics;
