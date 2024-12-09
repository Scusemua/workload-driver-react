import { InspectWorkloadModal, NewWorkloadFromTemplateModal, VisualizeWorkloadModal } from '@Components/Modals';
import { WorkloadsDataList } from '@Components/Workloads/WorkloadsDataList';
import {
    Button,
    Card,
    CardBody,
    CardHeader,
    PerPageOptions,
    Text,
    TextVariants,
    Title,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
} from '@patternfly/react-core';

import { PlusIcon, StopCircleIcon } from '@patternfly/react-icons';
import useNavigation from '@Providers/NavigationProvider';

import { WorkloadContext } from '@Providers/WorkloadProvider';

import { IsInProgress, Workload } from '@src/Data/Workload';
import React, { useEffect } from 'react';

export interface WorkloadCardProps {
    workloadsPerPage: number;
    perPageOption: PerPageOptions[];
    inspectInModal: boolean;
    useCreationModal: boolean;
}

export const WorkloadCard: React.FunctionComponent<WorkloadCardProps> = (props: WorkloadCardProps) => {
    // const [isRegisterWorkloadModalOpen, setIsRegisterWorkloadModalOpen] = React.useState(false);
    const [isRegisterNewWorkloadFromTemplateModalOpen, setIsRegisterNewWorkloadFromTemplateModalOpen] =
        React.useState(false);
    const [selectedWorkloadListId, setSelectedWorkloadListId] = React.useState('');

    const [visualizeWorkloadModalOpen, setVisualizeWorkloadModalOpen] = React.useState(false);
    const [workloadBeingVisualized, setWorkloadBeingVisualized] = React.useState<Workload | null>(null);

    const [inspectWorkloadModalOpen, setInspectWorkloadModalOpen] = React.useState(false);
    const [workloadBeingInspected, setWorkloadBeingInspected] = React.useState<Workload | null>(null);

    const { workloads, workloadsMap, registerWorkloadFromPreset, registerWorkloadFromTemplate, stopAllWorkloads } =
        React.useContext(WorkloadContext);

    const { navigate } = useNavigation();

    useEffect(() => {
        if (workloadBeingInspected !== null && inspectWorkloadModalOpen) {
            const updatedWorkload: Workload | undefined = workloadsMap.get(workloadBeingInspected.id);

            // Ensure the workload is updated in the inspection panel.
            if (updatedWorkload) {
                setWorkloadBeingInspected(updatedWorkload);
            }
        }
    }, [workloadsMap, inspectWorkloadModalOpen, workloadBeingInspected]);

    const onCloseVisualizeWorkloadModal = () => {
        setWorkloadBeingVisualized(null);
        setVisualizeWorkloadModalOpen(false);
    };

    const onCloseInspectWorkloadModal = () => {
        setWorkloadBeingInspected(null);
        setInspectWorkloadModalOpen(false);
    };

    const onClickWorkload = (workload: Workload) => {
        setWorkloadBeingInspected(workload);

        if (props.inspectInModal) {
            setInspectWorkloadModalOpen(true);
        } else {
            navigate('/workload/' + workload.id);
        }
    };

    const onConfirmRegisterWorkloadFromTemplate = (
        workloadName: string,
        workloadRegistrationRequest: string,
        messageId?: string,
    ) => {
        // setIsRegisterWorkloadModalOpen(false);
        setIsRegisterNewWorkloadFromTemplateModalOpen(false);
        registerWorkloadFromTemplate(workloadName, workloadRegistrationRequest, messageId);
    };

    // const onRegisterWorkloadFromTemplateClicked = () => {
    //     setIsRegisterNewWorkloadFromTemplateModalOpen(true);
    // };

    // const onConfirmRegisterWorkload = (
    //     workloadName: string,
    //     selectedPreset: WorkloadPreset,
    //     workloadSeedString: string,
    //     debugLoggingEnabled: boolean,
    //     timescaleAdjustmentFactor: number,
    //     workloadSessionSamplePercent: number,
    // ) => {
    //     // setIsRegisterWorkloadModalOpen(false);
    //     setIsRegisterNewWorkloadFromTemplateModalOpen(false);
    //
    //     registerWorkloadFromPreset(
    //         workloadName,
    //         selectedPreset,
    //         workloadSeedString,
    //         debugLoggingEnabled,
    //         timescaleAdjustmentFactor,
    //         workloadSessionSamplePercent,
    //     );
    // };

    // const onCancelStartWorkload = () => {
    //     console.log('New workload cancelled by user before starting.');
    //     setIsRegisterWorkloadModalOpen(false);
    // };

    const onCancelStartWorkloadFromTemplate = () => {
        console.log('New workload from template cancelled by user before starting.');
        setIsRegisterNewWorkloadFromTemplateModalOpen(false);
        // setIsRegisterWorkloadModalOpen(true);
    };

    const onStopAllWorkloadsClicked = () => {
        stopAllWorkloads();
    };

    const onSelectWorkload = (_event: React.MouseEvent | React.KeyboardEvent, id: string) => {
        // Toggle off if it is already selected.
        if (id == selectedWorkloadListId) {
            setSelectedWorkloadListId('');
            console.log("De-selected workload '%s'", id);
        } else {
            setSelectedWorkloadListId(id);
            console.log("Selected workload '%s'", id);
        }
    };

    const onVisualizeWorkloadClicked = (workload: Workload) => {
        console.log(`Inspecting workload: ${workload.name} (id=${workload.name})`);
        console.log(workload);

        setWorkloadBeingVisualized(workload);
        setVisualizeWorkloadModalOpen(true);
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
                            onClick={() => {
                                if (props.useCreationModal) {
                                    // setIsRegisterWorkloadModalOpen(true);
                                    setIsRegisterNewWorkloadFromTemplateModalOpen(true);
                                } else {
                                    navigate('/register_workload');
                                }
                            }}
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
                            isDanger
                            isDisabled={
                                Object.values(workloads).filter((workload: Workload) => {
                                    return IsInProgress(workload);
                                }).length == 0
                            }
                            onClick={onStopAllWorkloadsClicked} // () => setIsConfirmDeleteKernelsModalOpen(true)
                        >
                            <StopCircleIcon />
                        </Button>
                    </Tooltip>
                </ToolbarItem>
            </ToolbarGroup>
        </React.Fragment>
    );

    return (
        <React.Fragment>
            <Card isRounded isFullHeight id="workload-card">
                <CardHeader actions={{ actions: cardHeaderActions, hasNoOffset: false }}>
                    <Title headingLevel="h1" size="xl">
                        Workloads
                    </Title>
                </CardHeader>
                <CardBody>
                    {workloads.length == 0 && (
                        <Text component={TextVariants.h2}>There are no registered workloads.</Text>
                    )}
                    {workloads.length >= 1 && (
                        <WorkloadsDataList
                            workloads={workloads}
                            onSelectWorkload={onSelectWorkload}
                            onClickWorkload={onClickWorkload}
                            onVisualizeWorkloadClicked={onVisualizeWorkloadClicked}
                            workloadsPerPage={props.workloadsPerPage}
                            selectedWorkloadListId={selectedWorkloadListId}
                            perPageOption={props.perPageOption}
                        />
                    )}
                </CardBody>
            </Card>
            {/*<RegisterWorkloadModal*/}
            {/*    isOpen={isRegisterWorkloadModalOpen}*/}
            {/*    onClose={onCancelStartWorkload}*/}
            {/*    onConfirm={onConfirmRegisterWorkload}*/}
            {/*    onRegisterWorkloadFromTemplateClicked={onRegisterWorkloadFromTemplateClicked}*/}
            {/*/>*/}
            <NewWorkloadFromTemplateModal
                isOpen={isRegisterNewWorkloadFromTemplateModalOpen}
                onClose={onCancelStartWorkloadFromTemplate}
                onConfirm={onConfirmRegisterWorkloadFromTemplate}
            />
            <VisualizeWorkloadModal
                isOpen={visualizeWorkloadModalOpen}
                workload={workloadBeingVisualized}
                onClose={onCloseVisualizeWorkloadModal}
            />
            {workloadBeingInspected !== null && (
                <InspectWorkloadModal
                    isOpen={inspectWorkloadModalOpen}
                    workload={workloadBeingInspected}
                    onClose={onCloseInspectWorkloadModal}
                />
            )}
        </React.Fragment>
    );
};
