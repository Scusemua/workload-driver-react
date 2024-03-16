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
    onSelectWorkload?: (workloadId: string) => void;
    onStartWorkloadClicked: (workload: Workload) => void;
    onStopWorkloadClicked: (workload: Workload) => void;
}

export const WorkloadCard: React.FunctionComponent<WorkloadCardProps> = (props: WorkloadCardProps) => {
    const [isCardExpanded, setIsCardExpanded] = React.useState(true);
    const [refreshingWorkloads, setRefreshingWorkloads] = React.useState(false);
    const [selectedWorkloadListId, setSelectedWorkloadListId] = React.useState('');

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
                            props.workloads.map((workload: Workload, idx: number) => (
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
                                                        <FlexItem>
                                                            <Text component={TextVariants.h2}>{workload.name}</Text>
                                                        </FlexItem>
                                                        <FlexItem>
                                                            <Text component={TextVariants.small}>
                                                                <strong>ID: </strong>
                                                            </Text>
                                                            <Text component={TextVariants.small}>{workload.id}</Text>
                                                        </FlexItem>
                                                        <FlexItem>
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
                                                                <FlexItem>
                                                                    <Tooltip
                                                                        content={'Number of tasks executed.'}
                                                                        position="bottom"
                                                                    >
                                                                        <React.Fragment>
                                                                            <MonitoringIcon />{' '}
                                                                            {workload.num_tasks_executed}
                                                                        </React.Fragment>
                                                                    </Tooltip>
                                                                </FlexItem>
                                                                {workload.workload_state != WORKLOAD_STATE_READY && (
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
                                                                )}
                                                            </Flex>
                                                        </FlexItem>
                                                    </Flex>
                                                </DataListCell>,
                                                <DataListCell
                                                    alignRight={true}
                                                    width={2}
                                                    key={'workload-secondary-content-' + idx}
                                                    id={'workload-data-list-' + idx}
                                                    aria-label="Workload Actions"
                                                    aria-labelledby="Workload Actions"
                                                >
                                                    <Flex
                                                        direction={{ default: 'row' }}
                                                        spaceItems={{ default: 'spaceItemsXs' }}
                                                    >
                                                        <FlexItem>
                                                            <Tooltip content={'Start the workload'}>
                                                                <Button
                                                                    isDisabled={
                                                                        workload.workload_state != WORKLOAD_STATE_READY
                                                                    }
                                                                    variant="link"
                                                                    icon={<PlayIcon />}
                                                                    onClick={() => {
                                                                        props.onStartWorkloadClicked(workload);
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
                                                                    variant="link"
                                                                    isDanger
                                                                    icon={<StopIcon />}
                                                                    onClick={() => {
                                                                        props.onStopWorkloadClicked(workload);
                                                                    }}
                                                                >
                                                                    Stop
                                                                </Button>
                                                            </Tooltip>
                                                        </FlexItem>
                                                    </Flex>
                                                </DataListCell>,
                                            ]}
                                        />
                                    </DataListItemRow>
                                </DataListItem>
                            ))}
                    </DataList>
                </CardBody>
            </CardExpandableContent>
        </Card>
    );
};
