import {
    ErrorResponse,
    PatchedWorkload,
    WORKLOAD_STATE_RUNNING,
    Workload,
    WorkloadPreset,
    WorkloadResponse,
} from '@Data/Workload';
import { Flex, FlexItem, Text, TextVariants } from '@patternfly/react-core';
import { AuthorizationContext } from '@Providers/AuthProvider';
import { JoinPaths } from '@src/Utils/path_utils';
import { DefaultDismiss, GetToastContentWithHeaderAndBody } from '@src/Utils/toast_utils';
import { ExportWorkloadToJson } from '@src/Utils/utils';
import jsonmergepatch from 'json-merge-patch';
import React, { useContext, useRef } from 'react';
import { Toast, toast } from 'react-hot-toast';
import { MutatorCallback } from 'swr';
import type { SWRSubscription } from 'swr/subscription';
import useSWRSubscription from 'swr/subscription';
import { v4 as uuidv4 } from 'uuid';

const api_endpoint: string = JoinPaths(process.env.PUBLIC_PATH || '/', 'websocket', 'workload');

export const useWorkloads = () => {
    const { authenticated } = useContext(AuthorizationContext);

    const subscriberSocket = useRef<WebSocket | null>(null);

    // Keep track of sent messages by their ID so that we can call the response handler upon receiving a response.
    const callbackMap: React.MutableRefObject<Map<string, (resp?: WorkloadResponse, error?: ErrorResponse) => void>> =
        useRef<Map<string, (resp?: WorkloadResponse, error?: ErrorResponse) => void>>(
            new Map<string, (resp?: WorkloadResponse, error?: ErrorResponse) => void>(),
        );

    const handleWebSocketResponse = (
        next: (
            err?: Error | null | undefined,
            data?: Map<string, Workload> | MutatorCallback<Map<string, Workload>> | undefined,
        ) => void,
        workloadResponse?: WorkloadResponse,
        error?: ErrorResponse,
    ) => {
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

        next(null, (prev: Map<string, Workload> | undefined) => {
            const newWorkloads: Workload[] | null | undefined = workloadResponse.new_workloads;
            const modifiedWorkloads: Workload[] | null | undefined = workloadResponse.modified_workloads;
            const deletedWorkloads: Workload[] | null | undefined = workloadResponse.deleted_workloads;
            const patchedWorkloads: PatchedWorkload[] | null | undefined = workloadResponse.patched_workloads;

            const nextData: Map<string, Workload> = new Map(prev);

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
                    // console.log(`Patched data: ${JSON.stringify(patch)}`)
                    const mergedWorkload: Workload = jsonmergepatch.apply(workload, patch);
                    // console.log(`Workload after patch: ${JSON.stringify(mergedWorkload)}\n`)
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
    };

    const setupWebsocket = (
        hostname: string,
        next: (
            err?: Error | null | undefined,
            data?: Map<string, Workload> | MutatorCallback<Map<string, Workload>> | undefined,
        ) => void,
    ) => {
        if (subscriberSocket.current == null) {
            console.log(`Attempting to connect Workload WebSocket to hostname "${hostname}"`);
            subscriberSocket.current = new WebSocket(hostname);
            subscriberSocket.current.addEventListener('open', () => {
                console.log("Connected to workload websocket. Sending 'subscribe' message now.");
                subscriberSocket.current?.send(
                    JSON.stringify({
                        op: 'subscribe',
                        msg_id: uuidv4(),
                    }),
                );
            });

            subscriberSocket.current.addEventListener('message', async (event) => {
                const respText: string = await event.data.text();

                let workloadResponse: WorkloadResponse | undefined = undefined;
                try {
                    workloadResponse = JSON.parse(respText);
                } catch (err) {
                    console.error(`Failed to decode WorkloadResponse: "${respText}"`);
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
                    return handleWebSocketResponse(next, workloadResponse, undefined);
                }

                let errorResponse: ErrorResponse;
                try {
                    errorResponse = JSON.parse(respText);
                } catch (err) {
                    console.error(`Failed to decode ErrorResponse: "${respText}"`);
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
                    return handleWebSocketResponse(next, undefined, errorResponse);
                }

                // toast.custom(
                //     GetToastContentWithHeaderAndBody(
                //         `Received ErrorResponse for "${errorResponse.op}" workload WebSocket request`,
                //         [errorResponse.Description, errorResponse.ErrorMessage],
                //         'danger',
                //         DefaultDismiss,
                //     ),
                // );
            });

            subscriberSocket.current.addEventListener('close', (event: CloseEvent) => {
                console.error(`Workloads Subscriber WebSocket closed: ${JSON.stringify(event)}`);
            });

            subscriberSocket.current.addEventListener('error', (event: Event) => {
                console.log(`Workloads Subscriber WebSocket encountered error: ${JSON.stringify(event)}`);
            });
        }
    };

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
    const sendJsonMessage = (
        msg: string,
        msgId?: string | undefined,
        callback?: (resp?: WorkloadResponse, error?: ErrorResponse) => void,
    ): string | void => {
        if (subscriberSocket.current?.readyState !== WebSocket.OPEN) {
            console.error(
                `Cannot send workload-related message via websocket. Websocket is in state ${subscriberSocket.current?.readyState}`,
            );

            return 'WebSocket connection with backend is unavailable';
        }

        if (callbackMap.current && msgId && callback) {
            callbackMap.current.set(msgId, callback);
        }

        try {
            subscriberSocket.current?.send(msg);
        } catch (err) {
            console.error(`Failed to send workload-related message via websocket. Error: ${err}`);

            return JSON.stringify(err);
        }
    };

    function refreshWorkloads() {
        sendJsonMessage(
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
        const sendErrorMessage: string | void = sendJsonMessage(
            JSON.stringify({
                op: 'start_workload',
                msg_id: messageId,
                workload_id: workload.id,
            }),
        );

        if (sendErrorMessage) {
            toast.custom(
                (t: Toast) =>
                    GetToastContentWithHeaderAndBody(
                        'Failed to Start Workload',
                        [
                            `Workload "${workload.name}" (ID="${workload.id}") could not be started.`,
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
                        'Workload Started',
                        `Workload "${workload.name}" (ID="${workload.id}") has been started successfully.`,
                        'success',
                        () => toast.dismiss(t.id),
                    ),
                { id: toastId },
            );
        }
    };

    const stopWorkload = (workload: Workload) => {
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
        const sendErrorMessage: string | void = sendJsonMessage(
            JSON.stringify({
                op: 'stop_workload',
                msg_id: messageId,
                workload_id: workload.id,
            }),
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

    const stopAllWorkloads = () => {
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

    const registerWorkloadFromPreset = (
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

        let workloadSeed = -1;
        if (workloadSeedString != '') {
            workloadSeed = parseInt(workloadSeedString);
        }

        const messageId: string = uuidv4();
        const sendErrorMessage: string | void = sendJsonMessage(
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

    const registerWorkloadFromTemplate = (workloadName: string, workloadRegistrationRequest: string) => {
        console.log(`Sending WorkloadRegistrationRequest: ${workloadRegistrationRequest}`);
        const sendErrorMessage: string | void = sendJsonMessage(workloadRegistrationRequest);

        if (sendErrorMessage) {
            toast.custom((t: Toast) =>
                GetToastContentWithHeaderAndBody(
                    'Workload Registration Failed',
                    [
                        `Unable to register template-based workload "${workloadName}".`,
                        <p key={'toast-content-row-2'}>
                            <b>{'Reason:'}</b> {sendErrorMessage}
                        </p>,
                    ],
                    'danger',
                    () => toast.dismiss(t.id),
                ),
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
        const sendErrorMessage: string | void = sendJsonMessage(
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

    const exportWorkload = (currentLocalWorkload: Workload) => {
        console.log(`Exporting workload ${currentLocalWorkload.name} (ID=${currentLocalWorkload.id}).`);

        const messageId: string = uuidv4();

        // Wait up to 5 seconds before giving up and exporting the local copy instead.
        const timeout = setTimeout(() => {
            console.warn(
                `Could not refresh workload ${currentLocalWorkload.id} after 5 seconds. Exporting local copy.`,
            );
            ExportWorkloadToJson(currentLocalWorkload, `workload_${currentLocalWorkload.id}_local.json`);
        }, 5000);

        const errorMessageFromSending: string | void = sendJsonMessage(
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
        if (workload.paused) {
            console.log("Resuming workload '%s' (ID=%s)", workload.name, workload.id);
            operation = 'unpause_workload';
        } else {
            console.log("Pausing workload '%s' (ID=%s)", workload.name, workload.id);
            operation = 'pause_workload';
        }

        const messageId: string = uuidv4();
        const sendErrorMessage: string | void = sendJsonMessage(
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

    const subscribe: SWRSubscription<string, Map<string, Workload>, Error> = (key: string, { next }) => {
        // Don't establish any WebSocket connections until we've been authenticated...
        if (!authenticated) {
            return null;
        }

        console.log(`Connecting to Websocket server at '${key}'`);
        setupWebsocket(key, next);
        return () => {};
    };

    const { data, error } = useSWRSubscription(api_endpoint, subscribe);
    const workloadsMap: Map<string, Workload> = data || new Map();
    const workloads: Workload[] = Array.from(workloadsMap.values());

    return {
        workloads: workloads,
        workloadsMap: workloadsMap,
        isError: error,
        sendJsonMessage: sendJsonMessage,
        toggleDebugLogs: toggleDebugLogs,
        exportWorkload: exportWorkload,
        pauseWorkload: pauseWorkload,
        registerWorkloadFromPreset: registerWorkloadFromPreset,
        registerWorkloadFromTemplate: registerWorkloadFromTemplate,
        stopAllWorkloads: stopAllWorkloads,
        startWorkload: startWorkload,
        stopWorkload: stopWorkload,
        refreshWorkloads: refreshWorkloads,
    };
};
