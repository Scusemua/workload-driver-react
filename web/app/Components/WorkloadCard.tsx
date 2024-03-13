import React from 'react';
import { Button, Card, CardBody, CardHeader, Title, ToolbarGroup, ToolbarItem, Tooltip } from '@patternfly/react-core';

import { PlusIcon, StopCircleIcon, SyncIcon } from '@patternfly/react-icons';

export interface WorkloadCardProps {
    onLaunchWorkloadClicked: () => void;
    refreshWorkloadPresets: (callback: () => void | undefined) => void;
}

export const WorkloadCard: React.FunctionComponent<WorkloadCardProps> = (props: WorkloadCardProps) => {
    const [isCardExpanded, setIsCardExpanded] = React.useState(true);
    const [refreshingWorkloads, setRefreshingWorkloads] = React.useState(false);

    const onCardExpand = () => {
        setIsCardExpanded(!isCardExpanded);
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
                                props.refreshWorkloadPresets(() => {
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
            <CardBody></CardBody>
        </Card>
    );
};
