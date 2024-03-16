import React from 'react';
import {
    Button,
    Card,
    CardBody,
    CardExpandableContent,
    CardHeader,
    DataList,
    DataListAction,
    DataListCell,
    DataListContent,
    DataListControl,
    DataListItem,
    DataListItemCells,
    DataListItemRow,
    DataListToggle,
    Flex,
    FlexItem,
    Pagination,
    PaginationVariant,
    Radio,
    Text,
    TextVariants,
    Title,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
} from '@patternfly/react-core';

import {
    BlueprintIcon,
    CheckCircleIcon,
    ClockIcon,
    DiceIcon,
    OutlinedCalendarAltIcon,
    MonitoringIcon,
    SpinnerIcon,
    PendingIcon,
    PlusIcon,
    StopCircleIcon,
    SyncIcon,
    PlayIcon,
    StopIcon,
    ExclamationCircleIcon,
    PowerOffIcon,
    CodeIcon,
} from '@patternfly/react-icons';

import {
    Workload,
    WORKLOAD_STATE_READY,
    WORKLOAD_STATE_RUNNING,
    WORKLOAD_STATE_FINISHED,
    WORKLOAD_STATE_ERRED,
    WORKLOAD_STATE_TERMINATED,
} from '@app/Data/Workload';

export interface WorkloadCardProps {
    onLaunchWorkloadClicked: () => void;
    refreshWorkloads: (callback: () => void | undefined) => void;
    workloads: Workload[];
    onStartWorkloadClicked: (workload: Workload) => void;
    onStopWorkloadClicked: (workload: Workload) => void;
    workloadsPerPage: number;
}

export const WorkloadCard: React.FunctionComponent<WorkloadCardProps> = (props: WorkloadCardProps) => {
    const [isCardExpanded, setIsCardExpanded] = React.useState(true);
    const [refreshingWorkloads, setRefreshingWorkloads] = React.useState(false);
    const [selectedWorkloadListId, setSelectedWorkloadListId] = React.useState('');
    const [page, setPage] = React.useState(1);
    const [perPage, setPerPage] = React.useState(props.workloadsPerPage);

    const onSetPage = (_event: React.MouseEvent | React.KeyboardEvent | MouseEvent, newPage: number) => {
        setPage(newPage);
        console.log(
            'onSetPage: Displaying workloads %d through %d.',
            perPage * (newPage - 1),
            perPage * (newPage - 1) + perPage,
        );
    };

    const onPerPageSelect = (
        _event: React.MouseEvent | React.KeyboardEvent | MouseEvent,
        newPerPage: number,
        newPage: number,
    ) => {
        setPerPage(newPerPage);
        setPage(newPage);
        console.log(
            'onPerPageSelect: Displaying workloads %d through %d.',
            newPerPage * (newPage - 1),
            newPerPage * (newPage - 1) + newPerPage,
        );
    };

    const onCardExpand = () => {
        setIsCardExpanded(!isCardExpanded);
    };

    const onSelectWorkload = (_event: React.MouseEvent | React.KeyboardEvent, id: string) => {
        // Toggle off if it is already selected.
        if (id == selectedWorkloadListId) {
            setSelectedWorkloadListId('');
        } else {
            setSelectedWorkloadListId(id);
        }
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
                            onClick={() => props.onLaunchWorkloadClicked()}
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
                            onClick={() => {}} // () => setIsConfirmDeleteKernelsModalOpen(true)
                        >
                            <StopCircleIcon />
                        </Button>
                    </Tooltip>
                    <Tooltip exitDelay={75} content={<div>Refresh workloads.</div>}>
                        <Button
                            id="refresh-workloads-button"
                            variant="plain"
                            onClick={() => {
                                setRefreshingWorkloads(true);
                                props.refreshWorkloads(() => {
                                    setRefreshingWorkloads(false);
                                });
                            }}
                            label="refresh-workload-button"
                            aria-label="refresh-workload-button"
                            icon={<SyncIcon />}
                            isDisabled={refreshingWorkloads}
                            className={
                                (refreshingWorkloads && 'loading-icon-spin-toggleable') ||
                                'loading-icon-spin-toggleable paused'
                            }
                        />
                    </Tooltip>
                </ToolbarItem>
            </ToolbarGroup>
        </React.Fragment>
    );

    return (
        <Card isRounded isExpanded={isCardExpanded}>
            <CardHeader
                actions={{ actions: cardHeaderActions, hasNoOffset: true }}
                onExpand={onCardExpand}
                toggleButtonProps={{
                    id: 'expand-workloads-button',
                    'aria-label': 'expand-workloads-button',
                    'aria-expanded': isCardExpanded,
                }}
            >
                <Title headingLevel="h1" size="xl">
                    Workloads
                </Title>
            </CardHeader>
            <CardExpandableContent>
                <CardBody>
                    <DataList
                        isCompact
                        aria-label="data list"
                        selectedDataListItemId={selectedWorkloadListId}
                        onSelectDataListItem={onSelectWorkload}
                    >
                        {props.workloads != null &&
                            props.workloads
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
                                                                        <FlexItem>
                                                                            <Text component={TextVariants.h2}>
                                                                                {workload.name}
                                                                            </Text>
                                                                        </FlexItem>
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
                                                                        direction={{ default: 'row' }}
                                                                        spaceItems={{ default: 'spaceItemsXs' }}
                                                                    >
                                                                        <FlexItem>
                                                                            <Tooltip content={'Start the workload'}>
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
                                                                                    onClick={() => {
                                                                                        props.onStartWorkloadClicked(
                                                                                            workload,
                                                                                        );
                                                                                    }}
                                                                                >
                                                                                    Start
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
                                                                                    id={
                                                                                        'stop-workload-' +
                                                                                        idx +
                                                                                        '-button'
                                                                                    }
                                                                                    variant="link"
                                                                                    isDanger
                                                                                    icon={<StopIcon />}
                                                                                    onClick={() => {
                                                                                        props.onStopWorkloadClicked(
                                                                                            workload,
                                                                                        );
                                                                                    }}
                                                                                >
                                                                                    Stop
                                                                                </Button>
                                                                            </Tooltip>
                                                                        </FlexItem>
                                                                    </Flex>
                                                                </FlexItem>
                                                            </Flex>
                                                            <FlexItem className="workload-descriptive-icons">
                                                                <Flex spaceItems={{ default: 'spaceItemsMd' }}>
                                                                    <FlexItem>
                                                                        <Tooltip
                                                                            content={'Workload status/state.'}
                                                                            position="bottom"
                                                                        >
                                                                            <React.Fragment>
                                                                                {workload.workload_state ==
                                                                                    WORKLOAD_STATE_READY && (
                                                                                    <React.Fragment>
                                                                                        <PendingIcon />
                                                                                        {' Ready'}
                                                                                    </React.Fragment>
                                                                                )}
                                                                                {workload.workload_state ==
                                                                                    WORKLOAD_STATE_RUNNING && (
                                                                                    <React.Fragment>
                                                                                        <SpinnerIcon className="loading-icon-spin" />
                                                                                        {' Running'}
                                                                                    </React.Fragment>
                                                                                )}
                                                                                {workload.workload_state ==
                                                                                    WORKLOAD_STATE_FINISHED && (
                                                                                    <React.Fragment>
                                                                                        <CheckCircleIcon />
                                                                                        {' Complete'}
                                                                                    </React.Fragment>
                                                                                )}
                                                                                {workload.workload_state ==
                                                                                    WORKLOAD_STATE_ERRED && (
                                                                                    <React.Fragment>
                                                                                        <ExclamationCircleIcon />
                                                                                        {' Erred'}
                                                                                    </React.Fragment>
                                                                                )}
                                                                                {workload.workload_state ==
                                                                                    WORKLOAD_STATE_TERMINATED && (
                                                                                    <React.Fragment>
                                                                                        <PowerOffIcon />
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
                                                                                <BlueprintIcon /> "
                                                                                {workload.workload_preset_name}"
                                                                            </React.Fragment>
                                                                        </Tooltip>
                                                                    </FlexItem>
                                                                    <FlexItem>
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
                    <Pagination
                        isDisabled={props.workloads.length == 0}
                        itemCount={props.workloads.length}
                        widgetId="bottom-example"
                        perPage={perPage}
                        page={page}
                        variant={PaginationVariant.bottom}
                        perPageOptions={[
                            {
                                title: '1',
                                value: 1,
                            },
                            {
                                title: '2',
                                value: 2,
                            },
                            {
                                title: '3',
                                value: 3,
                            },
                            {
                                title: '4',
                                value: 4,
                            },
                            {
                                title: '5',
                                value: 5,
                            },
                        ]}
                        onSetPage={onSetPage}
                        onPerPageSelect={onPerPageSelect}
                    />
                </CardBody>
            </CardExpandableContent>
        </Card>
    );
};
