import { Flex, FlexItem, Switch, Tooltip } from '@patternfly/react-core';
import { BlueprintIcon, ClockIcon, CubeIcon, DiceIcon, OutlinedCalendarAltIcon } from '@patternfly/react-icons';
import { GetWorkloadStatusLabel, GetWorkloadStatusTooltip, IsWorkloadFinished, Workload } from '@src/Data';
import { WorkloadContext } from '@src/Providers';
import React from 'react';

interface IWorkloadDescriptiveIcons {
    workload: Workload;
}

export const WorkloadDescriptiveIcons: React.FunctionComponent<IWorkloadDescriptiveIcons> = (
    props: IWorkloadDescriptiveIcons,
) => {
    const { toggleDebugLogs } = React.useContext(WorkloadContext);

    return (
        <Flex className="props.workload-descriptive-icons" spaceItems={{ default: 'spaceItemsMd' }}>
            <FlexItem>
                <Tooltip content={GetWorkloadStatusTooltip(props.workload)} position="bottom">
                    <React.Fragment>{GetWorkloadStatusLabel(props.workload)}</React.Fragment>
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
                    onClick={(e) => {
                        e.stopPropagation();
                        e.preventDefault();
                    }}
                    onChange={() => {
                        toggleDebugLogs(props.workload.id, !props.workload.debug_logging_enabled);
                    }}
                />
            </FlexItem>
        </Flex>
    );
};

export default WorkloadDescriptiveIcons;
