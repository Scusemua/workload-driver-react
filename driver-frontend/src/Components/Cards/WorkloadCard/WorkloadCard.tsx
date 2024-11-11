import { WorkloadsDataList } from '@Cards/WorkloadCard/WorkloadsDataList';
import {
    InspectWorkloadModal,
    NewWorkloadFromTemplateModal,
    RegisterWorkloadModal,
    VisualizeWorkloadModal,
} from '@Components/Modals';
import {
  Button,
  Card,
  CardBody,
  CardHeader,
  Flex,
  FlexItem, PerPageOptions,
  Text,
  TextVariants,
  Title,
  ToolbarGroup,
  ToolbarItem,
  Tooltip
} from '@patternfly/react-core';

import { PlusIcon, StopCircleIcon } from '@patternfly/react-icons';

import { useWorkloads } from '@Providers/WorkloadProvider';

import { Workload, WORKLOAD_STATE_RUNNING, WorkloadPreset, WorkloadResponse } from '@src/Data/Workload';
import { SessionTabsDataProvider } from '@src/Providers';
import { DefaultDismiss, GetToastContentWithHeaderAndBody } from '@src/Utils/toast_utils';
import { ExportWorkloadToJson } from '@src/Utils/utils';
import React, { useEffect } from 'react';
import { Toast, toast } from 'react-hot-toast';
import { v4 as uuidv4 } from 'uuid';

export interface WorkloadCardProps {
    workloadsPerPage: number;
    perPageOption: PerPageOptions[];
}

export const WorkloadCard: React.FunctionComponent<WorkloadCardProps> = (props: WorkloadCardProps) => {
    const [isRegisterWorkloadModalOpen, setIsRegisterWorkloadModalOpen] = React.useState(false);
    const [isRegisterNewWorkloadFromTemplateModalOpen, setIsRegisterNewWorkloadFromTemplateModalOpen] =
        React.useState(false);
    const [selectedWorkloadListId, setSelectedWorkloadListId] = React.useState('');

    const [visualizeWorkloadModalOpen, setVisualizeWorkloadModalOpen] = React.useState(false);
    const [workloadBeingVisualized, setWorkloadBeingVisualized] = React.useState<Workload | null>(null);

    const [inspectWorkloadModalOpen, setInspectWorkloadModalOpen] = React.useState(false);
    const [workloadBeingInspected, setWorkloadBeingInspected] = React.useState<Workload | null>(null);

    const { workloads, workloadsMap, sendJsonMessage } = useWorkloads();

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
        setInspectWorkloadModalOpen(true);
    };

    const onConfirmRegisterWorkloadFromTemplate = (workloadName: string, workloadRegistrationRequest: string) => {
        // const toastId: string = toast(`Registering template-based workload now.`, {
        //     style: { maxWidth: 650 },
        // });

        setIsRegisterWorkloadModalOpen(false);
        setIsRegisterNewWorkloadFromTemplateModalOpen(false);
        console.log(`Sending WorkloadRegistrationRequest: ${workloadRegistrationRequest}`);
        const errorMessage: string | void = sendJsonMessage(workloadRegistrationRequest);

        if (errorMessage) {
            toast.custom((t: Toast) =>
                GetToastContentWithHeaderAndBody(
                    'Workload Registration Failed',
                    [
                        `Unable to register template-based workload "${workloadName}".`,
                        <p key={'toast-content-row-2'}>
                            <b>{'Reason:'}</b> {errorMessage}
                        </p>,
                    ],
                    'danger',
                    () => toast.dismiss(t.id),
                ),
            );
        }
    };

    const onRegisterWorkloadFromTemplateClicked = () => {
        setIsRegisterNewWorkloadFromTemplateModalOpen(true);
    };

    const onConfirmRegisterWorkload = (
        workloadName: string,
        selectedPreset: WorkloadPreset,
        workloadSeedString: string,
        debugLoggingEnabled: boolean,
        timescaleAdjustmentFactor: number,
    ) => {
        const toastId: string = toast(`Registering preset-based workload ${workloadName} now.`, {
            style: { maxWidth: 650 },
        });

        console.log(`New workload "${workloadName}" registered by user with preset "${selectedPreset.name}"`);
        setIsRegisterWorkloadModalOpen(false);
        setIsRegisterNewWorkloadFromTemplateModalOpen(false);

        let workloadSeed = -1;
        if (workloadSeedString != '') {
            workloadSeed = parseInt(workloadSeedString);
        }

        const messageId: string = uuidv4();
        const errorMessage: string | void = sendJsonMessage(
            JSON.stringify({
                op: 'register_workload',
                msg_id: messageId,
                workloadRegistrationRequest: {
                    adjust_gpu_reservations: false,
                    seed: workloadSeed,
                    timescale_adjustment_factor: timescaleAdjustmentFactor,
                    key: selectedPreset.key,
                    name: workloadName,
                    debug_logging: debugLoggingEnabled,
                    type: 'preset',
                },
            }),
        );

        if (errorMessage) {
            toast.custom(
                (t: Toast) =>
                    GetToastContentWithHeaderAndBody(
                        'Workload Registration Failed',
                        [
                            `Unable to register workload "${workloadName}" with preset "${selectedPreset.name}" at this time.`,
                            <p key={'toast-content-row-2'}>
                                <b>{'Reason:'}</b> {errorMessage}
                            </p>,
                        ],
                        'danger',
                        () => toast.dismiss(t.id),
                    ),
                { id: toastId },
            );
        } else {
            toast.custom(
                (t: Toast) =>
                    GetToastContentWithHeaderAndBody(
                        `Workload Registered Successfully`,
                        `Successfully registered workload "${workloadName}" with preset "${selectedPreset.name}"`,
                        'success',
                        () => toast.dismiss(t.id),
                    ),
                { id: toastId },
            );
        }
    };

    const onCancelStartWorkload = () => {
        console.log('New workload cancelled by user before starting.');
        setIsRegisterWorkloadModalOpen(false);
    };

    const onCancelStartWorkloadFromTemplate = () => {
        console.log('New workload from template cancelled by user before starting.');
        setIsRegisterNewWorkloadFromTemplateModalOpen(false);
        setIsRegisterWorkloadModalOpen(true);
    };

    const onStopAllWorkloadsClicked = () => {
        toast('Stopping all workload');

        const activeWorkloadsIDs: string[] = [];
        workloads.forEach((workload: Workload) => {
            if (workload.workload_state == WORKLOAD_STATE_RUNNING) {
                activeWorkloadsIDs.push(workload.id);
            }
        });

        const messageId: string = uuidv4();
        sendJsonMessage(
            JSON.stringify({
                op: 'stop_workloads',
                msg_id: messageId,
                workload_ids: activeWorkloadsIDs,
            }),
        );
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

    const toggleDebugLogs = (workloadId: string, enabled: boolean) => {
        if (enabled) {
            console.log("Enabling debug logging for workload '%s'", workloadId);
        } else {
            console.log("Disabling debug logging for workload '%s'", workloadId);
        }

        const messageId: string = uuidv4();
        sendJsonMessage(
            JSON.stringify({
                op: 'toggle_debug_logs',
                msg_id: messageId,
                workload_id: workloadId,
                enabled: enabled,
            }),
        );
    };

    const onVisualizeWorkloadClicked = (workload: Workload) => {
        console.log(`Inspecting workload: ${workload.name} (id=${workload.name})`);
        console.log(workload);

        setWorkloadBeingVisualized(workload);
        setVisualizeWorkloadModalOpen(true);
    };

    const onStartWorkloadClicked = (workload: Workload) => {
        const toastId: string = toast.custom((t: Toast) =>
            GetToastContentWithHeaderAndBody(
                `Starting workload ${workload.name}`,
                [
                    <Text key={`toast-content-start-workload-${workload.id}`} component={TextVariants.small}>
                        <b>Workload ID: </b>
                        {workload.id}
                    </Text>,
                ],
                'info',
                () => toast.dismiss(t.id),
            ),
        );

        console.log(`Starting workload '${workload.name}' (ID=${workload.id})`);

        const messageId: string = uuidv4();
        const errorMessage: string | void = sendJsonMessage(
            JSON.stringify({
                op: 'start_workload',
                msg_id: messageId,
                workload_id: workload.id,
            }),
        );

        if (errorMessage) {
            toast.custom(
                (t: Toast) =>
                    GetToastContentWithHeaderAndBody(
                        'Failed to Start Workload',
                        [
                            `Workload "${workload.name}" (ID="${workload.id}") could not be started.`,
                            <p key={'toast-content-row-2'}>
                                <b>{'Reason:'}</b> {errorMessage}
                            </p>,
                        ],
                        'danger',
                        () => toast.dismiss(t.id),
                    ),
                { id: toastId },
            );
        } else {
            toast.custom(
                (t: Toast) =>
                    GetToastContentWithHeaderAndBody(
                        'Workload Started',
                        `Workload "${workload.name}" (ID="${workload.id}") has been started successfully.`,
                        'success',
                        () => toast.dismiss(t.id),
                    ),
                { id: toastId },
            );
        }
    };

    const onPauseWorkloadClicked = (workload: Workload) => {
        toast(`Pausing workload ${workload.name} (ID=${workload.id}) now...`);
    };

    const onStopWorkloadClicked = (workload: Workload) => {
        const toastId: string = toast(
            (t: Toast) =>
                GetToastContentWithHeaderAndBody(
                    `Stopping workload ${workload.name} (ID = ${workload.id}).`,
                    [],
                    'info',
                    () => toast.dismiss(t.id),
                ),
            {
                style: { maxWidth: 650 },
            },
        );

        console.log("Stopping workload '%s' (ID=%s)", workload.name, workload.id);

        const messageId: string = uuidv4();
        const errorMessage: string | void = sendJsonMessage(
            JSON.stringify({
                op: 'stop_workload',
                msg_id: messageId,
                workload_id: workload.id,
            }),
        );

        if (errorMessage) {
            toast.custom(
                (t: Toast) =>
                    GetToastContentWithHeaderAndBody(
                        'Failed to Stop Workload',
                        [
                            `Workload "${workload.name}" (ID="${workload.id}") could not be stopped.`,
                            <p key={'toast-content-row-2'}>
                                <b>{'Reason:'}</b> {errorMessage}
                            </p>,
                        ],
                        'danger',
                        () => toast.dismiss(t.id),
                    ),
                { id: toastId },
            );
        } else {
            toast.custom(
                (t: Toast) =>
                    GetToastContentWithHeaderAndBody(
                        'Workload Stopped',
                        `Workload "${workload.name}" (ID="${workload.id}") has been stopped successfully.`,
                        'success',
                        () => toast.dismiss(t.id),
                    ),
                { id: toastId },
            );
        }
    };

    /**
     * Retrieve the latest version of the specified workload from the server and then download it as a JSON file.
     */
    const exportWorkloadClicked = (currentLocalWorkload: Workload) => {
        console.log(`Exporting workload ${currentLocalWorkload.name} (ID=${currentLocalWorkload.id}).`);

        const messageId: string = uuidv4();

        // Wait up to 5 seconds before giving up and exporting the local copy instead.
        const timeout = setTimeout(() => {
            console.warn(
                `Could not refresh workload ${currentLocalWorkload.id} after 5 seconds. Exporting local copy.`,
            );
            ExportWorkloadToJson(currentLocalWorkload, `workload_${currentLocalWorkload.id}_local.json`);
        }, 5000);

        const errorMessage: string | void = sendJsonMessage(
            JSON.stringify({
                op: 'get_workloads',
                msg_id: messageId,
            }),
            messageId,
            (workloadResponse: WorkloadResponse) => {
                // First, clear the timeout that we set. We don't need to export the local copy (unless the
                // server didn't return a valid remote copy, but we'll handle that later).
                clearTimeout(timeout);
                console.log(`Resp: ${JSON.stringify(workloadResponse, null, 2)}`);

                if (workloadResponse.modified_workloads.length === 0) {
                    // Server did not return any workloads. We'll just export our local copy...
                    toast.custom(
                        GetToastContentWithHeaderAndBody(
                            `Could Not Find Workload on Server with ID="${currentLocalWorkload.id}"`,
                            `Will export local copy of workload ${currentLocalWorkload.name} (ID=${currentLocalWorkload.id}) instead.`,
                            'danger',
                            DefaultDismiss,
                        ),
                    );
                    ExportWorkloadToJson(currentLocalWorkload, `workload_${currentLocalWorkload.id}_local.json`);
                } else if (workloadResponse.modified_workloads.length > 1) {
                    // The server returned multiple workloads despite us querying for only one ID.
                    // We'll export all the remote workloads as well as the local copy, just to be safe.
                    toast.custom(
                        GetToastContentWithHeaderAndBody(
                            `Server Returned ${workloadResponse.modified_workloads.length} Workloads for Query with WorkloadID="${currentLocalWorkload.id}"`,
                            `Will export local copy of workload ${currentLocalWorkload.name} (ID=${currentLocalWorkload.id}) and all returned remote copies.`,
                            'warning',
                            DefaultDismiss,
                        ),
                    );

                    // Export the local copy of the workload.
                    ExportWorkloadToJson(currentLocalWorkload, `workload_${currentLocalWorkload.id}_local.json`);

                    // Export the multiple remote copies (that we received for some... reason).
                    for (let i = 0; i < workloadResponse.modified_workloads.length; i++) {
                        const remoteWorkload: Workload = workloadResponse.modified_workloads[i];
                        ExportWorkloadToJson(remoteWorkload, `workload_${remoteWorkload.id}_remote_${i}.json`);
                    }
                } else {
                    // The server only returned one remote workload. We'll just export the remote workload.
                    const remoteWorkload: Workload = workloadResponse.modified_workloads[0];
                    ExportWorkloadToJson(remoteWorkload, `workload_${remoteWorkload.id}_remote.json`);
                }
            },
        );

        if (errorMessage) {
            clearTimeout(timeout); // Don't need to bother with this; we'll just export the local copy immediately.
            toast.custom(
                GetToastContentWithHeaderAndBody(
                    `Failed to Retrieve Latest Copy of Workload ${currentLocalWorkload.id} from Server`,
                    <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsXs' }}>
                        <FlexItem>
                            <Text>
                                <b>Error</b>: {errorMessage}
                            </Text>
                        </FlexItem>
                        <FlexItem>
                            <Text>Local copy of workload {currentLocalWorkload.id} will be exported instead.</Text>
                        </FlexItem>
                    </Flex>,
                    'danger',
                    DefaultDismiss,
                ),
            );

            // Export the local copy.
            ExportWorkloadToJson(currentLocalWorkload, `workload_${currentLocalWorkload.id}_local.json`);
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
                            onClick={() => {
                                setIsRegisterWorkloadModalOpen(true);
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
                                    return workload.workload_state == WORKLOAD_STATE_RUNNING;
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
                            onPauseWorkloadClicked={onPauseWorkloadClicked}
                            toggleDebugLogs={toggleDebugLogs}
                            onSelectWorkload={onSelectWorkload}
                            onClickWorkload={onClickWorkload}
                            onVisualizeWorkloadClicked={onVisualizeWorkloadClicked}
                            onStartWorkloadClicked={onStartWorkloadClicked}
                            onStopWorkloadClicked={onStopWorkloadClicked}
                            workloadsPerPage={props.workloadsPerPage}
                            selectedWorkloadListId={selectedWorkloadListId}
                            perPageOption={props.perPageOption}
                        />
                    )}
                </CardBody>
            </Card>

            <RegisterWorkloadModal
                isOpen={isRegisterWorkloadModalOpen}
                onClose={onCancelStartWorkload}
                onConfirm={onConfirmRegisterWorkload}
                onRegisterWorkloadFromTemplateClicked={onRegisterWorkloadFromTemplateClicked}
            />
            <SessionTabsDataProvider>
                <NewWorkloadFromTemplateModal
                    isOpen={isRegisterNewWorkloadFromTemplateModalOpen}
                    onClose={onCancelStartWorkloadFromTemplate}
                    onConfirm={onConfirmRegisterWorkloadFromTemplate}
                />
            </SessionTabsDataProvider>
            <VisualizeWorkloadModal
                isOpen={visualizeWorkloadModalOpen}
                workload={workloadBeingVisualized}
                onClose={onCloseVisualizeWorkloadModal}
            />
            <InspectWorkloadModal
                isOpen={inspectWorkloadModalOpen}
                workload={workloadBeingInspected}
                onClose={onCloseInspectWorkloadModal}
                onStartClicked={() => {
                    if (workloadBeingInspected) onStartWorkloadClicked(workloadBeingInspected);
                }}
                onStopClicked={() => {
                    if (workloadBeingInspected) onStopWorkloadClicked(workloadBeingInspected);
                }}
                onExportClicked={() => {
                    if (workloadBeingInspected) exportWorkloadClicked(workloadBeingInspected);
                }}
            />
        </React.Fragment>
    );
};
