import { Button, Flex, FlexItem, Text, TextVariants, Tooltip } from '@patternfly/react-core';
import { PauseIcon, PlayIcon, SearchIcon, StopIcon } from '@patternfly/react-icons';
import { CsvFileIcon, TemplateIcon, XmlFileIcon } from '@src/Assets/Icons';
import WorkloadDescriptiveIcons from '@src/Components/Workloads/WorkloadDescriptiveIcons';
import { WorkloadRuntimeMetrics } from '@src/Components/Workloads/WorkloadRuntimeMetrics';
import { WORKLOAD_STATE_READY, WORKLOAD_STATE_RUNNING, Workload } from '@src/Data';
import { WorkloadContext } from '@src/Providers';
import React from 'react';

interface IWorkloadDataListCellProps {
    workload: Workload;
    onVisualizeWorkloadClicked: (workload: Workload) => void;
}

export const WorkloadDataListCell: React.FunctionComponent<IWorkloadDataListCellProps> = (
    props: IWorkloadDataListCellProps,
) => {
    const {
      pauseWorkload,
      startWorkload,
      stopWorkload,
    } = React.useContext(WorkloadContext);

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
            <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsNone' }}>
                <FlexItem>
                    <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
                        <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsMd' }}>
                            <FlexItem>
                                <Text component={TextVariants.h2}>
                                    <strong>{props.workload.name}</strong>
                                </Text>
                            </FlexItem>
                            {props.workload.workload_preset && (
                                <FlexItem>
                                    <Tooltip
                                        content={`This preset is defined in a ${props.workload.workload_preset.preset_type} file.`}
                                        position="bottom"
                                    >
                                        <React.Fragment>
                                            {props.workload.workload_preset.preset_type === 'XML' && (
                                                <XmlFileIcon scale={2.25} />
                                            )}
                                            {props.workload.workload_preset.preset_type === 'CSV' && (
                                                <CsvFileIcon scale={2.25} />
                                            )}
                                        </React.Fragment>
                                    </Tooltip>
                                </FlexItem>
                            )}
                            {props.workload.workload_template && (
                                <FlexItem>
                                    <Tooltip
                                        content={`This props.workload was created/defined using a template.`}
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
                            <Text component={TextVariants.small}>{props.workload.id}</Text>
                        </FlexItem>
                    </Flex>
                </FlexItem>
                <FlexItem align={{ default: 'alignRight' }}>
                    <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsXs' }}>
                        <FlexItem>
                            <Flex
                                direction={{ default: 'row' }}
                                spaceItems={{
                                    default: 'spaceItemsXs',
                                }}
                            >
                                <FlexItem>
                                    <Tooltip content={'Start the props.workload'}>
                                        <Button
                                            id={`start-props.workload-${props.workload.id}-button`}
                                            isDisabled={props.workload.workload_state != WORKLOAD_STATE_READY}
                                            variant="link"
                                            icon={<PlayIcon />}
                                            onClick={(event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
                                                startWorkload(props.workload);

                                                event.stopPropagation();
                                            }}
                                        >
                                            Start
                                        </Button>
                                    </Tooltip>
                                </FlexItem>
                                <FlexItem>
                                    <Tooltip content={'Pause the props.workload.'}>
                                        <Button
                                            isDisabled={props.workload.workload_state != WORKLOAD_STATE_RUNNING}
                                            id={`pause-props.workload-${props.workload.id}-button`}
                                            variant="link"
                                            isDanger={!props.workload.paused}
                                            icon={props.workload.paused ? <PlayIcon /> : <PauseIcon />}
                                            onClick={(event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
                                                pauseWorkload(props.workload);

                                                event.stopPropagation();
                                            }}
                                        >
                                            {props.workload.paused ? 'Resume' : 'Pause'}
                                        </Button>
                                    </Tooltip>
                                </FlexItem>
                                <FlexItem>
                                    <Tooltip content={'Stop the props.workload.'}>
                                        <Button
                                            isDisabled={props.workload.workload_state != WORKLOAD_STATE_RUNNING}
                                            id={`stop-props.workload-${props.workload.id}-button`}
                                            variant="link"
                                            isDanger
                                            icon={<StopIcon />}
                                            onClick={(event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
                                                stopWorkload(props.workload);

                                                event.stopPropagation();
                                            }}
                                        >
                                            Stop
                                        </Button>
                                    </Tooltip>
                                </FlexItem>
                                {/* The element below is only meant to be visible for preset-based workloads, not template-based workloads. */}
                                {props.workload.workload_preset && (
                                    <FlexItem>
                                        <Tooltip content={'Inspect the events of the props.workload'}>
                                            <Button
                                                id={`inspect-props.workload-${props.workload.id}-button`}
                                                isDisabled={
                                                    !props.workload.workload_preset ||
                                                    props.workload.workload_preset.preset_type == 'CSV'
                                                }
                                                variant="link"
                                                icon={<SearchIcon />}
                                                onClick={(event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
                                                    props.onVisualizeWorkloadClicked(props.workload);

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
            <WorkloadDescriptiveIcons workload={props.workload} />
            <WorkloadRuntimeMetrics workload={props.workload} />
        </Flex>
    );
};

export default WorkloadDataListCell;
