import WorkloadDescriptiveIcons from '@Cards/WorkloadCard/WorkloadDescriptiveIcons';
import { Button, Flex, FlexItem, PerPageOptions, Text, TextVariants, Tooltip } from '@patternfly/react-core';
import { PauseIcon, PlayIcon, SearchIcon, StopIcon } from '@patternfly/react-icons';
import { CsvFileIcon, TemplateIcon, XmlFileIcon } from '@src/Assets/Icons';
import { WorkloadRuntimeMetrics } from '@src/Components';
import { Workload, WORKLOAD_STATE_READY, WORKLOAD_STATE_RUNNING } from '@src/Data';
import React from 'react';

interface IWorkloadDataListCellProps {
    workload: Workload;
    onPauseWorkloadClicked: (workload: Workload) => void;
    toggleDebugLogs: (workloadId: string, enabled: boolean) => void;
    onSelectWorkload: (event: React.MouseEvent | React.KeyboardEvent, id: string) => void;
    onClickWorkload: (workload: Workload) => void;
    onVisualizeWorkloadClicked: (workload: Workload) => void;
    onStartWorkloadClicked: (workload: Workload) => void;
    onStopWorkloadClicked: (workload: Workload) => void;
    workloadsPerPage?: number;
    selectedWorkloadListId: string;
    perPageOption: PerPageOptions[];
}

export const WorkloadDataListCell: React.FunctionComponent<IWorkloadDataListCellProps> = (
    props: IWorkloadDataListCellProps,
) => {
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
                                                props.onStartWorkloadClicked(props.workload);

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
                                            isDanger
                                            icon={<PauseIcon />}
                                            onClick={(event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
                                                props.onPauseWorkloadClicked(props.workload);

                                                event.stopPropagation();
                                            }}
                                        >
                                            Pause
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
                                                props.onStopWorkloadClicked(props.workload);

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
            <WorkloadDescriptiveIcons workload={props.workload} toggleDebugLogs={props.toggleDebugLogs} />
            <WorkloadRuntimeMetrics workload={props.workload} />
        </Flex>
    );
};

export default WorkloadDataListCell;
