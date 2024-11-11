import { HeightFactorContext, WorkloadsHeightFactorContext } from '@App/Dashboard';
import WorkloadDescriptiveIcons from '@Cards/WorkloadCard/WorkloadDescriptiveIcons';

import { WorkloadRuntimeMetrics } from '@Cards/WorkloadCard/WorkloadRuntimeMetrics';
import {
    Button,
    DataList,
    DataListCell,
    DataListItem,
    DataListItemCells,
    DataListItemRow,
    Flex,
    FlexItem,
    Pagination,
    PaginationVariant,
    Text,
    TextVariants,
    Tooltip,
} from '@patternfly/react-core';

import { PauseIcon, PlayIcon, SearchIcon, StopIcon } from '@patternfly/react-icons';

import { CsvFileIcon, TemplateIcon, XmlFileIcon } from '@src/Assets/Icons';

import { WORKLOAD_STATE_READY, WORKLOAD_STATE_RUNNING, Workload } from '@src/Data/Workload';
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
                                                <WorkloadDescriptiveIcons
                                                    workload={workload}
                                                    toggleDebugLogs={props.toggleDebugLogs}
                                                />
                                                <WorkloadRuntimeMetrics workload={workload} />
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
