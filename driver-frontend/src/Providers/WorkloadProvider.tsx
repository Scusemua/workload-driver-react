import {
    ErrorResponse,
    IsActivelyRunning,
    IsPaused,
    IsPausing,
    PatchedWorkload,
    Workload,
    WorkloadPreset,
    WorkloadResponse,
} from '@Data/Workload';
import { Flex, FlexItem, Text, TextVariants } from '@patternfly/react-core';
import { SpinnerIcon } from '@patternfly/react-icons';
import { AuthorizationContext } from '@Providers/AuthProvider';
import { GetPathForFetch, JoinPaths } from '@src/Utils/path_utils';
import { DefaultDismiss, GetToastContentWithHeaderAndBody } from '@src/Utils/toast_utils';
import { ExportWorkloadToJson } from '@src/Utils/utils';
import jsonmergepatch from 'json-merge-patch';
import React, { createContext, useContext, useRef } from 'react';
import { Toast, toast } from 'react-hot-toast';
import useWebSocket from 'react-use-websocket';
import { WebSocketLike } from 'react-use-websocket/src/lib/types';
import { v4 as uuidv4 } from 'uuid';

const api_endpoint: string = JoinPaths(process.env.PUBLIC_PATH || '/', 'websocket', 'workload');

type WorkloadContextData = {
    pauseWorkload: (workload: Workload) => void;
    toggleDebugLogs: (workloadId: string, enabled: boolean) => void;
    stopAllWorkloads: () => void;
    registerWorkloadFromPreset: (
        workloadName: string,
        selectedPreset: WorkloadPreset,
        workloadSeedString: string,
        debugLoggingEnabled: boolean,
        timescaleAdjustmentFactor: number,
        workloadSessionSamplePercent: number,
    ) => void;
    exportWorkload: (currentLocalWorkload: Workload) => void;
    stopWorkload: (workload: Workload) => void;
    workloads: Workload[];
    sendJsonMessage: (
        msg: string,
        msgId?: string | undefined,
        callback?: (resp?: WorkloadResponse, error?: ErrorResponse) => void,
    ) => string | void;
    registerWorkloadFromTemplate: (
        workloadName: string,
        workloadRegistrationRequest: string,
        messageId?: string,
    ) => void;
    workloadsMap: Map<string, Workload>;
    startWorkload: (workload: Workload) => void;
    refreshWorkloads: () => void;
};

const initialState: WorkloadContextData = {
    pauseWorkload: () => {},
    toggleDebugLogs: () => {},
    stopAllWorkloads: () => {},
    registerWorkloadFromPreset: () => {},
    exportWorkload: () => {},
    stopWorkload: () => {},
    workloads: [],
    sendJsonMessage: () => {},
    registerWorkloadFromTemplate: () => {},
    workloadsMap: new Map<string, Workload>(),
    startWorkload: () => {},
    refreshWorkloads: () => {},
};

const WorkloadContext: React.Context<WorkloadContextData> = createContext(initialState);

function WorkloadProvider({ children }: { children: React.ReactNode }) {
    const { authenticated } = useContext(AuthorizationContext);
    const [workloadsMap, setWorkloadsMap] = React.useState<Map<string, Workload>>(new Map<string, Workload>());
    const [workloads, setWorkloads] = React.useState<Workload[]>([]);

    // Keep track of sent messages by their ID so that we can call the response handler upon receiving a response.
    const callbackMap: React.MutableRefObject<Map<string, (resp?: WorkloadResponse, error?: ErrorResponse) => void>> =
        useRef<Map<string, (resp?: WorkloadResponse, error?: ErrorResponse) => void>>(
            new Map<string, (resp?: WorkloadResponse, error?: ErrorResponse) => void>(),
        );

    // const subscriberSocket = useRef<WebSocket | null>(null);
    const { sendMessage, lastMessage, getWebSocket } = useWebSocket(
        api_endpoint,
        {
            onOpen: () => {
                console.log("Connected to workload websocket. Sending 'subscribe' message now.");
                sendMessage(
                    JSON.stringify({
                        op: 'subscribe',
                        msg_id: uuidv4(),
                    }),
                );
            },
            onError: (event) => {
                console.error(`Workloads Subscriber WebSocket encountered an error: ${JSON.stringify(event)}`);
            },
            onClose: (event) => {
                console.error(`Workloads Subscriber WebSocket closed: ${JSON.stringify(event)}`);
            },
            share: true,
        },
        authenticated,
    );

    /**
     * Send a message to the remote WebSocket.
     * @param msg the JSON-encoded message to send.
     * @param msgId the ID of the message to use as a key for the callback in the callback-response map
     * @param callback the callback to be executed (with the WorkloadResponse as the argument) when the response is received.
     *
     * If an error occurs, then that error will be converted to a string and returned.
     *
     * Returns nothing on success.
     */
    const sendJsonMessageDirectly = React.useCallback(
        (
            msg: string,
            msgId?: string | undefined,
            callback?: (resp?: WorkloadResponse, error?: ErrorResponse) => void,
        ): string | void => {
            if (callbackMap.current && msgId && callback) {
                callbackMap.current.set(msgId, callback);
            }

            try {
                sendMessage(msg);
            } catch (err) {
                console.error(`Failed to send workload-related message via websocket. Error: ${err}`);

                return JSON.stringify(err);
            }
        },
        [sendMessage],
    );

    const handleWebSocketResponse = React.useCallback((workloadResponse?: WorkloadResponse, error?: ErrorResponse) => {
        if (!error && !workloadResponse) {
            return;
        }

        let msg_id: string;
        if (workloadResponse) {
            msg_id = workloadResponse.msg_id;
        } else {
            msg_id = error!.msg_id;
        }

        if (callbackMap.current) {
            const callback = callbackMap.current.get(msg_id);

            if (callback) {
                callback(workloadResponse, error);
            }
        }

        if (workloadResponse === undefined) {
            return;
        }

        if (workloadResponse.op == 'register_workload') {
            toast.custom((t: Toast) =>
                GetToastContentWithHeaderAndBody(
                    'Workload Registered Successfully',
                    `Successfully registered workload "${workloadResponse.new_workloads[0].name}"`,
                    'success',
                    () => toast.dismiss(t.id),
                ),
            );
        }

        const newWorkloads: Workload[] | null | undefined = workloadResponse.new_workloads;
        const modifiedWorkloads: Workload[] | null | undefined = workloadResponse.modified_workloads;
        const deletedWorkloads: Workload[] | null | undefined = workloadResponse.deleted_workloads;
        const patchedWorkloads: PatchedWorkload[] | null | undefined = workloadResponse.patched_workloads;

        setWorkloadsMap((prev: Map<string, Workload>) => {
            const nextData: Map<string, Workload> = new Map<string, Workload>(prev);

            newWorkloads?.forEach((workload: Workload) => {
                if (workload === null || workload === undefined) {
                    return;
                }
                nextData.set(workload.id, workload);
            });
            modifiedWorkloads?.forEach((workload: Workload) => {
                if (workload === null || workload === undefined) {
                    return;
                }
                nextData.set(workload.id, workload);
            });
            deletedWorkloads?.forEach((workload: Workload) => {
                if (workload === null || workload === undefined) {
                    return;
                }
                nextData.delete(workload.id);
            });

            patchedWorkloads?.forEach((patchedWorkload: PatchedWorkload) => {
                const patch = JSON.parse(patchedWorkload.patch);
                const workload: Workload | undefined = nextData.get(patchedWorkload.workloadId);

                if (workload !== null && workload !== undefined) {
                    const mergedWorkload: Workload = jsonmergepatch.apply(workload, patch);
                    nextData.set(patchedWorkload.workloadId, mergedWorkload);
                } else {
                    console.error(
                        `Received patched workload with ID ${patchedWorkload.workloadId}; however, no workload found in previous workload data for that workload...`,
                    );
                    console.error(`Patched data: ${patch}`);
                    console.error('Previous data contains the following keys: ', nextData.keys());
                }
            });

            return nextData;
        });
    }, []);

    React.useEffect(() => {
        if (!lastMessage) {
            return;
        }

        let message: string;
        try {
            message = new TextDecoder('utf-8').decode(lastMessage.data);
        } catch (err) {
            console.error(`Failed to decode Workload-related WebSocket message because: ${err}`);
            return;
        }

        console.log(`Received workload-related WebSocket message:\n${message}`);

        let workloadResponse: WorkloadResponse | undefined = undefined;
        try {
            workloadResponse = JSON.parse(message);
        } catch (err) {
            console.error(`Failed to decode WorkloadResponse: "${message}". Error: ${err}`);
            toast.custom(
                GetToastContentWithHeaderAndBody(
                    'Failed to Decode Workload Response from Workload WebSocket',
                    'See console for details.',
                    'danger',
                    DefaultDismiss,
                ),
            );

            return;
        }

        if (workloadResponse?.status == 'OK') {
            // console.log(`Received valid WorkloadResponse:\n${JSON.stringify(workloadResponse, null, 2)}`);
            return handleWebSocketResponse(workloadResponse, undefined);
        }

        let errorResponse: ErrorResponse;
        try {
            errorResponse = JSON.parse(message);
        } catch (err) {
            console.error(`Failed to decode ErrorResponse: "${message}". Error: ${err}`);
            toast.custom(
                GetToastContentWithHeaderAndBody(
                    'Failed to Decode ErrorResponse from Workload WebSocket',
                    'See console for details.',
                    'danger',
                    DefaultDismiss,
                ),
            );

            return;
        }

        console.error(`Received ErrorResponse for "${errorResponse.op}" workload WebSocket request.`);
        console.error(`ErrorMessage: ${errorResponse.ErrorMessage}`);
        console.error(`Description: ${errorResponse.Description}`);

        if (callbackMap.current) {
            return handleWebSocketResponse(undefined, errorResponse);
        }
    }, [handleWebSocketResponse, lastMessage]);

    React.useEffect(() => {
        setWorkloads(Array.from(workloadsMap.values()));
    }, [workloadsMap]);

    React.useEffect(() => {
        const webSocket: WebSocketLike | null = getWebSocket();

        if (webSocket !== null) {
            if ('binaryType' in webSocket) {
                webSocket.binaryType = 'arraybuffer';
            }
        }
    });

    function refreshWorkloads() {
        sendJsonMessageDirectly(
            JSON.stringify({
                op: 'get_workloads',
            }),
        );
    }

    const startWorkload = (workload: Workload) => {
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
        try {
            sendJsonMessageDirectly(
                JSON.stringify({
                    op: 'start_workload',
                    msg_id: messageId,
                    workload_id: workload.id,
                }),
                messageId,
                (resp?: WorkloadResponse, errResp?: ErrorResponse) => {
                    if (resp !== undefined) {
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
                    } else {
                        toast.custom(
                            (t: Toast) =>
                                GetToastContentWithHeaderAndBody(
                                    'Failed to Start Workload',
                                    [
                                        `Workload "${workload.name}" (ID="${workload.id}") could not be started.`,
                                        <p key={'toast-content-row-2'}>
                                            <b>{'Reason:'}</b> {JSON.stringify(errResp)}
                                        </p>,
                                    ],
                                    'danger',
                                    () => toast.dismiss(t.id),
                                ),
                            { id: toastId },
                        );
                    }
                },
            );
        } catch (err) {
            toast.custom(
                (t: Toast) =>
                    GetToastContentWithHeaderAndBody(
                        'Failed to Start Workload',
                        [
                            `Workload "${workload.name}" (ID="${workload.id}") could not be started.`,
                            <p key={'toast-content-row-2'}>
                                <b>{'Reason:'}</b> {JSON.stringify(err)}
                            </p>,
                        ],
                        'danger',
                        () => toast.dismiss(t.id),
                    ),
                { id: toastId },
            );
        }
    };

    const stopWorkload = (workload: Workload) => {
        const toastId: string = toast.custom(
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
        const sendErrorMessage: string | void = sendJsonMessageDirectly(
            JSON.stringify({
                op: 'stop_workload',
                msg_id: messageId,
                workload_id: workload.id,
            }),
            messageId,
            (workloadResponse?: WorkloadResponse, errorResponse?: ErrorResponse) => {
                if (workloadResponse) {
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
                } else {
                    toast.custom(
                        (t: Toast) =>
                            GetToastContentWithHeaderAndBody(
                                'Failed to Stop Workload',
                                [
                                    `Workload "${workload.name}" (ID="${workload.id}") could not be stopped.`,
                                    <p key={'toast-content-row-2'}>
                                        <b>{'Reason:'}</b> {errorResponse?.ErrorMessage || 'unknown error occurred.'}
                                    </p>,
                                ],
                                'danger',
                                () => toast.dismiss(t.id),
                            ),
                        { id: toastId },
                    );
                }
            },
        );

        if (sendErrorMessage) {
            toast.custom(
                (t: Toast) =>
                    GetToastContentWithHeaderAndBody(
                        'Failed to Stop Workload',
                        [
                            `Workload "${workload.name}" (ID="${workload.id}") could not be stopped.`,
                            <p key={'toast-content-row-2'}>
                                <b>{'Reason:'}</b> {sendErrorMessage}
                            </p>,
                        ],
                        'danger',
                        () => toast.dismiss(t.id),
                    ),
                { id: toastId },
            );
        }
    };

    const stopAllWorkloads = () => {
        toast('Stopping all workload');

        const activeWorkloadsIDs: string[] = [];
        workloads.forEach((workload: Workload) => {
            if (IsActivelyRunning(workload)) {
                activeWorkloadsIDs.push(workload.id);
            }
        });

        const messageId: string = uuidv4();
        sendJsonMessageDirectly(
            JSON.stringify({
                op: 'stop_workloads',
                msg_id: messageId,
                workload_ids: activeWorkloadsIDs,
            }),
        );
    };

    const registerWorkloadFromPreset = (
        workloadName: string,
        selectedPreset: WorkloadPreset,
        workloadSeedString: string,
        debugLoggingEnabled: boolean,
        timescaleAdjustmentFactor: number,
        sessionSamplePercent: number,
    ) => {
        const toastId: string = toast(`Registering preset-based workload ${workloadName} now.`, {
            style: { maxWidth: 650 },
        });

        console.log(`New workload "${workloadName}" registered by user with preset "${selectedPreset.name}"`);

        let workloadSeed = -1;
        if (workloadSeedString != '') {
            workloadSeed = parseInt(workloadSeedString);
        }

        const messageId: string = uuidv4();
        const sendErrorMessage: string | void = sendJsonMessageDirectly(
            JSON.stringify({
                op: 'register_workload',
                msg_id: messageId,
                workload_registration_request: {
                    adjust_gpu_reservations: false,
                    seed: workloadSeed,
                    timescale_adjustment_factor: timescaleAdjustmentFactor,
                    key: selectedPreset.key,
                    name: workloadName,
                    debug_logging: debugLoggingEnabled,
                    type: 'preset',
                    sessions_sample_percentage: sessionSamplePercent,
                    template_file_path: '',
                },
            }),
        );

        if (sendErrorMessage) {
            toast.custom(
                (t: Toast) =>
                    GetToastContentWithHeaderAndBody(
                        'Workload Registration Failed',
                        [
                            `Unable to register workload "${workloadName}" with preset "${selectedPreset.name}" at this time.`,
                            <p key={'toast-content-row-2'}>
                                <b>{'Reason:'}</b> {sendErrorMessage}
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

    const registerWorkloadFromTemplate = (
        workloadName: string,
        workloadRegistrationRequest: string,
        messageId?: string,
    ) => {
        console.log(`Sending WorkloadRegistrationRequest: ${workloadRegistrationRequest}`);

        const toastId: string = toast.custom(
            GetToastContentWithHeaderAndBody(
                `Registering Workload ${workloadName}`,
                `Registering workload ${workloadName} with backend now...`,
                'info',
                DefaultDismiss,
                undefined,
                <SpinnerIcon className={'loading-icon-spin-pulse'} />,
            ),
        );

        const sendErrorMessage: string | void = sendJsonMessageDirectly(
            workloadRegistrationRequest,
            messageId,
            (resp?: WorkloadResponse, errResp?: ErrorResponse) => {
                if (resp !== undefined) {
                    toast.custom(
                        (t: Toast) =>
                            GetToastContentWithHeaderAndBody(
                                `Registered Workload ${workloadName}`,
                                `Workload "${workloadName}" has been registered successfully.`,
                                'success',
                                () => toast.dismiss(t.id),
                            ),
                        { id: toastId, style: { maxWidth: 650 }, duration: 10000 },
                    );
                } else {
                    toast.custom(
                        (t: Toast) =>
                            GetToastContentWithHeaderAndBody(
                                'Failed to Start Workload',
                                [
                                    `Workload "${workloadName}" could not be registered.`,
                                    <p key={'toast-content-row-2'}>
                                        <b>{errResp?.ErrorMessage}</b> {errResp?.Description}
                                    </p>,
                                ],
                                'danger',
                                () => toast.dismiss(t.id),
                            ),
                        { id: toastId, style: { maxWidth: 650 }, duration: 30000 },
                    );
                }
            },
        );

        if (sendErrorMessage) {
            toast.custom(
                GetToastContentWithHeaderAndBody(
                    'Workload Registration Failed',
                    [
                        `Unable to register template-based workload "${workloadName}".`,
                        <p key={'toast-content-row-2'}>
                            <b>{'Reason:'}</b> {sendErrorMessage}
                        </p>,
                    ],
                    'danger',
                    () => toast.dismiss(toastId),
                ),
                { id: toastId },
            );
        }
    };

    const toggleDebugLogs = (workloadId: string, enabled: boolean) => {
        if (enabled) {
            console.log("Enabling debug logging for workload '%s'", workloadId);
        } else {
            console.log("Disabling debug logging for workload '%s'", workloadId);
        }

        const messageId: string = uuidv4();
        const sendErrorMessage: string | void = sendJsonMessageDirectly(
            JSON.stringify({
                op: 'toggle_debug_logs',
                msg_id: messageId,
                workload_id: workloadId,
                enabled: enabled,
            }),
        );

        if (sendErrorMessage !== undefined && sendErrorMessage !== '') {
            toast.custom((t: Toast) =>
                GetToastContentWithHeaderAndBody(
                    `Could Not Toggle Debug Logging for Workload ${workloadId}`,
                    sendErrorMessage,
                    'danger',
                    () => DefaultDismiss(t.id),
                ),
            );
        }
    };

    /**
     * Given the CSV text from the workload, download it as a .CSV file.
     */
    const downloadWorkloadCsv = (csvText: string, targetWorkload: Workload) => {
        const downloadElement: HTMLAnchorElement = document.createElement('a');
        downloadElement.setAttribute('href', 'data:text/csv;charset=utf-8,' + encodeURIComponent(csvText));

        downloadElement.setAttribute('download', `workload_${targetWorkload.id}.csv`);

        downloadElement.style.display = 'none';
        document.body.appendChild(downloadElement);

        downloadElement.click();

        document.body.removeChild(downloadElement);
    };

    /**
     * Retrieve the latest version of the Workload from the backend, including the workload and cluster statistics,
     * and download it as a JSON file.
     */
    const exportWorkload = async (currentLocalWorkload: Workload) => {
        console.log(`Exporting workload ${currentLocalWorkload.name} (ID=${currentLocalWorkload.id}).`);

        const messageId: string = uuidv4();

        // Wait up to 5 seconds before giving up and exporting the local copy instead.
        const timeout = setTimeout(() => {
            console.warn(
                `Could not refresh workload ${currentLocalWorkload.id} after 5 seconds. Exporting local copy.`,
            );
            ExportWorkloadToJson(currentLocalWorkload, `workload_${currentLocalWorkload.id}_local.json`);
        }, 5000);

        const errorMessageFromSending: string | void = sendJsonMessageDirectly(
            JSON.stringify({
                op: 'get_workloads',
                msg_id: messageId,
            }),
            messageId,
            (workloadResponse?: WorkloadResponse, errorResponse?: ErrorResponse) => {
                // First, clear the timeout that we set. We don't need to export the local copy (unless the
                // server didn't return a valid remote copy, but we'll handle that later).
                clearTimeout(timeout);

                if (workloadResponse) {
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
                } else if (errorResponse !== undefined) {
                    toast.custom(
                        GetToastContentWithHeaderAndBody(
                            `Error from Server While Exporting Workload "${currentLocalWorkload.id}"`,
                            [
                                `Will export local copy of workload ${currentLocalWorkload.name} (ID=${currentLocalWorkload.id}) instead.`,
                                errorResponse.ErrorMessage,
                                errorResponse.Description,
                            ],
                            'danger',
                            DefaultDismiss,
                        ),
                    );
                }
            },
        );

        // This would be an error that occurs on sending the WebSocket message.
        if (errorMessageFromSending) {
            clearTimeout(timeout); // Don't need to bother with this; we'll just export the local copy immediately.
            toast.custom(
                GetToastContentWithHeaderAndBody(
                    `Failed to Retrieve Latest Copy of Workload ${currentLocalWorkload.id} from Server`,
                    <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsXs' }}>
                        <FlexItem>
                            <Text>
                                <b>Error</b>: {errorMessageFromSending}
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

        const req: RequestInit = {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
                Authorization: 'Bearer ' + localStorage.getItem('token'),
            },
        };

        const resp: Response = await fetch(
            GetPathForFetch(`api/workload-statistics?workload_id=${currentLocalWorkload.id}`),
            req,
        );

        if (!resp.ok || resp.status != 200) {
            console.error(
                `Received bad response to 'api/workload-statistics' request: HTTP ${resp.status} - ${resp.statusText}.`,
            );
        } else {
            const text: string = await resp.text();

            downloadWorkloadCsv(text, currentLocalWorkload);
        }
    };

    const pauseWorkload = (workload: Workload) => {
        const toastId: string = toast(
            (t: Toast) =>
                GetToastContentWithHeaderAndBody(
                    `Pausing workload ${workload.name} (ID = ${workload.id}).`,
                    [],
                    'info',
                    () => toast.dismiss(t.id),
                ),
            {
                style: { maxWidth: 650 },
            },
        );

        let operation: string;
        if (IsPaused(workload) || IsPausing(workload)) {
            console.log("Resuming workload '%s' (ID=%s)", workload.name, workload.id);
            operation = 'unpause_workload';
        } else {
            console.log("Pausing workload '%s' (ID=%s)", workload.name, workload.id);
            operation = 'pause_workload';
        }

        const messageId: string = uuidv4();
        const sendErrorMessage: string | void = sendJsonMessageDirectly(
            JSON.stringify({
                op: operation,
                msg_id: messageId,
                workload_id: workload.id,
            }),
            messageId,
            (resp?: WorkloadResponse, error?: ErrorResponse) => {
                if (resp) {
                    toast.custom(
                        (t: Toast) =>
                            GetToastContentWithHeaderAndBody(
                                'Workload Paused',
                                `Workload "${workload.name}" (ID="${workload.id}") has been paused successfully.`,
                                'success',
                                () => toast.dismiss(t.id),
                            ),
                        { id: toastId },
                    );
                } else {
                    toast.custom(
                        (t: Toast) =>
                            GetToastContentWithHeaderAndBody(
                                'Failed to Pause Workload',
                                [
                                    `Workload "${workload.name}" (ID="${workload.id}") could not be paused.`,
                                    <p key={'toast-content-row-2'}>
                                        <b>{'Reason:'}</b> {error?.ErrorMessage} {error?.Description}
                                    </p>,
                                ],
                                'danger',
                                () => toast.dismiss(t.id),
                            ),
                        { id: toastId },
                    );
                }
            },
        );

        if (sendErrorMessage) {
            toast.custom(
                (t: Toast) =>
                    GetToastContentWithHeaderAndBody(
                        'Failed to Pause Workload',
                        [
                            `Workload "${workload.name}" (ID="${workload.id}") could not be paused.`,
                            <p key={'toast-content-row-2'}>
                                <b>{'Reason:'}</b> {sendErrorMessage}
                            </p>,
                        ],
                        'danger',
                        () => toast.dismiss(t.id),
                    ),
                { id: toastId },
            );
        }
    };

    return (
        <WorkloadContext.Provider
            value={{
                workloads: workloads,
                workloadsMap: workloadsMap,
                sendJsonMessage: sendJsonMessageDirectly,
                toggleDebugLogs: toggleDebugLogs,
                exportWorkload: exportWorkload,
                pauseWorkload: pauseWorkload,
                registerWorkloadFromPreset: registerWorkloadFromPreset,
                registerWorkloadFromTemplate: registerWorkloadFromTemplate,
                stopAllWorkloads: stopAllWorkloads,
                startWorkload: startWorkload,
                stopWorkload: stopWorkload,
                refreshWorkloads: refreshWorkloads,
            }}
        >
            {children}
        </WorkloadContext.Provider>
    );
}

export { WorkloadContext, WorkloadContextData, WorkloadProvider };
