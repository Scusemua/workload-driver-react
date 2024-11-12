import { ErrorResponse, PatchedWorkload, Workload, WorkloadResponse } from '@Data/Workload';
import { AuthorizationContext } from '@Providers/AuthProvider';
import { JoinPaths } from '@src/Utils/path_utils';
import { DefaultDismiss, GetToastContentWithHeaderAndBody } from '@src/Utils/toast_utils';
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
    useRef<boolean>(false);

    const callbackMap: React.MutableRefObject<Map<string, (resp: WorkloadResponse) => void>> = useRef<
        Map<string, (resp: WorkloadResponse) => void>
    >(new Map<string, (resp: WorkloadResponse) => void>());

    const handleStandardResponse = (
        next: (
            err?: Error | null | undefined,
            data?: Map<string, Workload> | MutatorCallback<Map<string, Workload>> | undefined,
        ) => void,
        workloadResponse: WorkloadResponse,
    ) => {
        if (callbackMap.current) {
            const callback = callbackMap.current.get(workloadResponse.msg_id);

            if (callback) {
                callback(workloadResponse);
            }
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
                    return handleStandardResponse(next, workloadResponse);
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

                toast.custom(
                    GetToastContentWithHeaderAndBody(
                        `Received ErrorResponse for "${errorResponse.op}" workload WebSocket request`,
                        [errorResponse.Description, errorResponse.ErrorMessage],
                        'danger',
                        DefaultDismiss,
                    ),
                );
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
        callback?: (resp: WorkloadResponse) => void,
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

    return {
        workloads: Array.from(workloadsMap.values()),
        workloadsMap: workloadsMap,
        isError: error,
        refreshWorkloads: refreshWorkloads,
        sendJsonMessage: sendJsonMessage,
    };
};
