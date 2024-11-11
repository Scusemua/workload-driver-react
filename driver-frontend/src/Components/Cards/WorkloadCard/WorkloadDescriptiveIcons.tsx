import { Flex, FlexItem, Label, Switch, Tooltip } from '@patternfly/react-core';
import {
    BlueprintIcon,
    CheckCircleIcon,
    ClockIcon,
    CubeIcon,
    DiceIcon,
    ExclamationTriangleIcon,
    HourglassStartIcon,
    OutlinedCalendarAltIcon,
    SpinnerIcon,
    TimesCircleIcon,
} from '@patternfly/react-icons';
import text from '@patternfly/react-styles/css/utilities/Text/text';
import {
    GetWorkloadStatusTooltip,
    IsWorkloadFinished,
    Workload,
    WORKLOAD_STATE_ERRED,
    WORKLOAD_STATE_FINISHED,
    WORKLOAD_STATE_READY,
    WORKLOAD_STATE_RUNNING,
    WORKLOAD_STATE_TERMINATED,
} from '@src/Data';
import React from 'react';

interface IWorkloadDescriptiveIcons {
    workload: Workload;
    toggleDebugLogs: (workloadId: string, enabled: boolean) => void;
}

export const WorkloadDescriptiveIcons: React.FunctionComponent<IWorkloadDescriptiveIcons> = (
    props: IWorkloadDescriptiveIcons,
) => {
    return (
        <Flex className="props.workload-descriptive-icons" spaceItems={{ default: 'spaceItemsMd' }}>
            <FlexItem>
                <Tooltip content={GetWorkloadStatusTooltip(props.workload)} position="bottom">
                    <React.Fragment>
                        {props.workload.workload_state == WORKLOAD_STATE_READY && (
                            <Label icon={<HourglassStartIcon className={text.infoColor_100} />} color="blue">
                                Ready
                            </Label>
                        )}
                        {props.workload.workload_state == WORKLOAD_STATE_RUNNING && (
                            <Label
                                icon={<SpinnerIcon className={'loading-icon-spin ' + text.successColor_100} />}
                                color="green"
                            >
                                Running
                            </Label>
                            // <React.Fragment>

                            //     Running
                            // </React.Fragment>
                        )}
                        {props.workload.workload_state == WORKLOAD_STATE_FINISHED && (
                            <Label icon={<CheckCircleIcon className={text.successColor_100} />} color="green">
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
                        {props.workload.workload_state == WORKLOAD_STATE_ERRED && (
                            <Label icon={<TimesCircleIcon className={text.dangerColor_100} />} color="red">
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
                        {props.workload.workload_state == WORKLOAD_STATE_TERMINATED && (
                            <Label icon={<ExclamationTriangleIcon className={text.warningColor_100} />} color="orange">
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
            {props.workload.workload_preset && (
                <FlexItem>
                    <Tooltip content={'Workload preset.'} position="bottom">
                        <React.Fragment>
                            <BlueprintIcon /> &quot;
                            {props.workload.workload_preset_name}
                            &quot;
                        </React.Fragment>
                    </Tooltip>
                </FlexItem>
            )}
            {props.workload.workload_template && (
                <FlexItem>
                    <Tooltip content={'Workload template.'} position="bottom">
                        {/* <Label icon={<BlueprintIcon />}>&quot;{props.workload.workload_template.name}&quot;</Label> */}
                        <React.Fragment>
                            <BlueprintIcon /> &quot;
                            {'Workload Template'}&quot;
                        </React.Fragment>
                    </Tooltip>
                </FlexItem>
            )}
            {props.workload.workload_preset && (
                <FlexItem hidden={props.workload.workload_preset.preset_type == 'XML'}>
                    <Tooltip content={'Months of trace data included in the props.workload.'} position="bottom">
                        <React.Fragment>
                            <OutlinedCalendarAltIcon /> {props.workload.workload_preset.months_description}
                        </React.Fragment>
                    </Tooltip>
                </FlexItem>
            )}
            <FlexItem>
                <Tooltip content={'Workload seed.'} position="bottom">
                    {/* <Label icon={<DiceIcon />}>{props.workload.seed}</Label> */}
                    <React.Fragment>
                        <DiceIcon /> {props.workload.seed}
                    </React.Fragment>
                </Tooltip>
            </FlexItem>
            <FlexItem>
                <Tooltip content={'Timescale Adjustment Factor.'} position="bottom">
                    {/* <Label icon={<ClockIcon />}>{props.workload.timescale_adjustment_factor}</Label> */}
                    <React.Fragment>
                        <ClockIcon /> {props.workload.timescale_adjustment_factor}
                    </React.Fragment>
                </Tooltip>
            </FlexItem>
            <FlexItem>
                <Tooltip content={'Total number of Sessions involved in the props.workload.'} position="bottom">
                    <React.Fragment>
                        <CubeIcon /> {props.workload.sessions.length}
                    </React.Fragment>
                </Tooltip>
            </FlexItem>

            <FlexItem align={{ default: 'alignRight' }} alignSelf={{ default: 'alignSelfFlexEnd' }}>
                <Switch
                    id={'props.workload-' + props.workload.id + '-debug-logging-switch'}
                    isDisabled={IsWorkloadFinished(props.workload)}
                    label={'Debug logging'}
                    aria-label="debug-logging-switch"
                    isChecked={props.workload.debug_logging_enabled}
                    ouiaId="DebugLoggingSwitch"
                    onChange={() => {
                        props.toggleDebugLogs(props.workload.id, !props.workload.debug_logging_enabled);
                    }}
                />
            </FlexItem>
        </Flex>
    );
};

export default WorkloadDescriptiveIcons;
