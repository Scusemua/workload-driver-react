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
                            id="launch-workload-button"
                            variant="plain"
                            onClick={() => props.onLaunchWorkloadClicked()}
                        >
                            <PlusIcon />
                        </Button>
                    </Tooltip>
                    <Tooltip exitDelay={75} content={<div>Stop selected workloads.</div>}>
                        <Button
                            id="stop-workloads-button"
                            variant="plain"
                            onClick={() => {}} // () => setIsConfirmDeleteKernelsModalOpen(true)
                        >
                            <StopCircleIcon />
                        </Button>
                    </Tooltip>
                    <Tooltip exitDelay={75} content={<div>Refresh workloads.</div>}>
                        <Button id="refresh-workloads-button" variant="plain" onClick={() => {}}>
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
                    id: 'toggle-button',
                    'aria-label': 'Actions',
                    'aria-labelledby': 'titleId toggle-button',
                    'aria-expanded': isCardExpanded,
                }}
            >
                <Title headingLevel="h4" size="xl">
                    Workloads
                </Title>
            </CardHeader>
            <CardBody></CardBody>
        </Card>
    );
};
