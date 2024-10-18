import { PatchedWorkload, Workload, WorkloadResponse } from '@Data/Workload';
import { AuthorizationContext } from '@Providers/AuthProvider';
import jsonmergepatch from 'json-merge-patch';
import { useContext, useRef } from 'react';
import { MutatorCallback } from 'swr';
import type { SWRSubscription } from 'swr/subscription';
import useSWRSubscription from 'swr/subscription';
import { v4 as uuidv4 } from 'uuid';

const api_endpoint: string = 'ws://localhost:8000/workload';

export const useWorkloads = () => {
    const{ authenticated } = useContext(AuthorizationContext);

    const subscriberSocket = useRef<WebSocket | null>(null);
    useRef<boolean>(false);

    const setupWebsocket = (
        hostname: string,
        next: (
            err?: Error | null | undefined,
            data?: Map<string, Workload> | MutatorCallback<Map<string, Workload>> | undefined,
        ) => void,
    ) => {
        if (subscriberSocket.current == null) {
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

            subscriberSocket.current.addEventListener('message', (event) => {
                try {
                    event.data
                        .text()
                        .then((respText: string) => {
                            const respJson: WorkloadResponse = JSON.parse(respText);
                            // console.log(`Received JSON message from workload websocket: ${respText}`);
                            console.log(respJson);
                            return respJson;
                        })
                        .then((workloadResponse: WorkloadResponse) =>
                            next(null, (prev: Map<string, Workload> | undefined) => {
                                const newWorkloads: Workload[] | null | undefined = workloadResponse.new_workloads;
                                const modifiedWorkloads: Workload[] | null | undefined =
                                    workloadResponse.modified_workloads;
                                const deletedWorkloads: Workload[] | null | undefined =
                                    workloadResponse.deleted_workloads;
                                const patchedWorkloads: PatchedWorkload[] | null | undefined =
                                    workloadResponse.patched_workloads;

                                const nextData: Map<string, Workload> = new Map(prev);

                                // console.log(`NumNewWorkloads: ${newWorkloads?.length}`);
                                // console.log(`NumModifiedWorkloads: ${modifiedWorkloads?.length}`);
                                // console.log(`NumDeletedWorkloads: ${deletedWorkloads?.length}`);
                                // console.log(`NumPatchedWorkloads: ${patchedWorkloads?.length}\n`);

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
                                    // console.log("Processing patched workload: ", JSON.stringify(patchedWorkload))
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
                            }),
                        );
                } catch (err) {
                    const messageData = JSON.parse(event.data);
                    console.log('Received workload-related WebSocket message:');
                    console.log(messageData);
                }
            });

            subscriberSocket.current.addEventListener('close', (event: CloseEvent) => {
                console.error(`Workloads Subscriber WebSocket closed: ${event}`);
            });

            subscriberSocket.current.addEventListener('error', (event: Event) => {
                console.log(`Workloads Subscriber WebSocket encountered error: ${JSON.stringify(event)}`);
            });
        }
    };

    const sendJsonMessage = (msg: string) => {
        if (subscriberSocket.current?.readyState !== WebSocket.OPEN) {
            console.error(
                `Cannot send workload-related message via websocket. Websocket is in state ${subscriberSocket.current?.readyState}`,
            );

            // setupWebsocket(api_endpoint, true, lastNextFunc.current);
            return;
        }

        try {
            subscriberSocket.current?.send(msg);
        } catch (err) {
            console.error(`Failed to send workload-related message via websocket. Error: ${err}`);
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
