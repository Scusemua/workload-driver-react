import React from 'react';
import { Button, Card, CardBody, CardHeader, Title, ToolbarGroup, ToolbarItem, Tooltip } from '@patternfly/react-core';

import { PlusIcon, StopCircleIcon, SyncIcon } from '@patternfly/react-icons';

export interface WorkloadCardProps {
    onLaunchWorkloadClicked: () => void;
}

export const WorkloadCard: React.FunctionComponent<WorkloadCardProps> = (props: WorkloadCardProps) => {
    const [isCardExpanded, setIsCardExpanded] = React.useState(true);

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
                            onClick={() => {}}
                            label="refresh-workload-button"
                            aria-label="refresh-workload-button"
                        >
                            <SyncIcon />
                        </Button>
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
