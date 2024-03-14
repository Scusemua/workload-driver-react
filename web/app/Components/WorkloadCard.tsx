import React from 'react';
import {
    Button,
    Card,
    CardBody,
    CardHeader,
    DataList,
    DataListCell,
    DataListContent,
    DataListControl,
    DataListItem,
    DataListItemCells,
    DataListItemRow,
    DataListToggle,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Radio,
    Title,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
} from '@patternfly/react-core';

import {
    CheckCircleIcon,
    ClipboardCheckIcon,
    ClockIcon,
    CogIcon,
    SpinnerIcon,
    PlusIcon,
    StopCircleIcon,
    SyncIcon,
} from '@patternfly/react-icons';

import { Workload } from '@app/Data/Workload';

export interface WorkloadCardProps {
    onLaunchWorkloadClicked: () => void;
    refreshWorkloads: (callback: () => void | undefined) => void;
    workloads: Workload[];
    onSelectWorkload?: (workloadId: string) => void;
    selectable?: boolean;
}

export const WorkloadCard: React.FunctionComponent<WorkloadCardProps> = (props: WorkloadCardProps) => {
    const [isCardExpanded, setIsCardExpanded] = React.useState(true);
    const [refreshingWorkloads, setRefreshingWorkloads] = React.useState(false);

    const [expandedWorkloads, setExpandedWorkloads] = React.useState<string[]>([]);
    const [selectedWorkload, setSelectedWorkload] = React.useState('');

    const onCardExpand = () => {
        setIsCardExpanded(!isCardExpanded);
    };

    const toggleExpandedWorkload = (id) => {
        const index = expandedWorkloads.indexOf(id);
        const newExpanded =
            index >= 0
                ? [
                      ...expandedWorkloads.slice(0, index),
                      ...expandedWorkloads.slice(index + 1, expandedWorkloads.length),
                  ]
                : [...expandedWorkloads, id];
        setExpandedWorkloads(newExpanded);
    };

    const cardHeaderActions = (
        <React.Fragment>
            <ToolbarGroup variant="icon-button-group">
                <ToolbarItem>
                    <Tooltip exitDelay={75} content={<div>Create a new kernel.</div>}>
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
                    <Tooltip exitDelay={75} content={<div>Stop selected workloads.</div>}>
                        <Button
                            label="stop-workload-button"
                            aria-label="stop-workload-button"
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
            <CardBody>
                <DataList isCompact aria-label="data list">
                    {props.workloads != null &&
                        props.workloads.map((workload: Workload, idx: number) => (
                            <DataListItem
                                key={workload.id}
                                id={'workload-list-item-' + idx}
                                isExpanded={expandedWorkloads.includes(workload.id)}
                            >
                                <DataListItemRow>
                                    {props.selectable && (
                                        <DataListControl>
                                            <Radio
                                                id={'workload-' + workload.id + '-radio'}
                                                aria-label={'workload-' + workload.id + '-radio'}
                                                aria-labelledby={'workload-' + workload.id + '-radio'}
                                                name={'workload-list-radio-buttons'}
                                                hidden={!props.selectable}
                                                onChange={() => {
                                                    console.log('Selected workload ' + workload.id);
                                                    setSelectedWorkload(workload.id);
                                                    if (props.onSelectWorkload != undefined) {
                                                        props.onSelectWorkload(workload.id);
                                                    }
                                                }}
                                                isChecked={workload.id == selectedWorkload}
                                            />
                                        </DataListControl>
                                    )}
                                    <DataListToggle
                                        onClick={() => toggleExpandedWorkload(workload.id)}
                                        isExpanded={expandedWorkloads.includes(workload.id)}
                                        id={'expand-workload-' + workload.id + '-toggle'}
                                        aria-controls={'expand-workload-' + workload.id + '-toggle'}
                                    />
                                    <DataListItemCells
                                        dataListCells={[
                                            <DataListCell key="primary-content">
                                                <DescriptionList
                                                    className="workload-list-description-list"
                                                    columnModifier={{ lg: '3Col', xl: '3Col' }}
                                                >
                                                    <DescriptionListGroup>
                                                        <DescriptionListTerm>ID</DescriptionListTerm>
                                                        <DescriptionListDescription>
                                                            {workload.id}
                                                        </DescriptionListDescription>
                                                    </DescriptionListGroup>
                                                    <DescriptionListGroup>
                                                        <DescriptionListTerm>Name</DescriptionListTerm>
                                                        <DescriptionListDescription>
                                                            {workload.name}
                                                        </DescriptionListDescription>
                                                    </DescriptionListGroup>
                                                    <DescriptionListGroup>
                                                        <DescriptionListTerm>Status</DescriptionListTerm>
                                                        <DescriptionListDescription>
                                                            {workload.started && (
                                                                <React.Fragment>
                                                                    <SpinnerIcon className="loading-icon-spin" />
                                                                    {' Running'}
                                                                </React.Fragment>
                                                            )}
                                                            {!workload.started && (
                                                                <React.Fragment>
                                                                    <CheckCircleIcon />
                                                                    {' Complete'}
                                                                </React.Fragment>
                                                            )}
                                                        </DescriptionListDescription>
                                                    </DescriptionListGroup>
                                                    <DescriptionListGroup>
                                                        <DescriptionListTerm icon={<CogIcon />}>
                                                            Preset Name
                                                        </DescriptionListTerm>
                                                        <DescriptionListDescription>
                                                            {workload.workload_preset_name}
                                                        </DescriptionListDescription>
                                                    </DescriptionListGroup>
                                                    <DescriptionListGroup>
                                                        <DescriptionListTerm icon={<ClockIcon />}>
                                                            Running Time
                                                        </DescriptionListTerm>
                                                        <DescriptionListDescription>
                                                            {workload.time_elapsed}
                                                        </DescriptionListDescription>
                                                    </DescriptionListGroup>
                                                    <DescriptionListGroup>
                                                        <DescriptionListTerm icon={<ClipboardCheckIcon />}>
                                                            Tasks Executed
                                                        </DescriptionListTerm>
                                                        <DescriptionListDescription>
                                                            {workload.num_tasks_executed}
                                                        </DescriptionListDescription>
                                                    </DescriptionListGroup>
                                                </DescriptionList>
                                            </DataListCell>,
                                        ]}
                                    />
                                </DataListItemRow>
                                <DataListContent
                                    className="workload-list-expandable-content"
                                    aria-label={'workload-' + workload.id + '-expandable-content'}
                                    id={'workload-' + workload.id + '-expandable-content'}
                                    isHidden={!expandedWorkloads.includes(workload.id)}
                                >
                                    {/* {expandedWorkloadContent(workload)} */}
                                </DataListContent>
                            </DataListItem>
                        ))}
                </DataList>
            </CardBody>
        </Card>
    );
};
