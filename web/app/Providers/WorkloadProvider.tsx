import { useRef } from 'react';
import useSWRSubscription from 'swr/subscription';
import type { SWRSubscription } from 'swr/subscription';
import { Workload, WorkloadResponse } from '@data/Workload';
import { v4 as uuidv4 } from 'uuid';
import { MutatorCallback } from 'swr';

const api_endpoint: string = 'ws://localhost:8000/workload';

export const useWorkloads = () => {
    const subscriberSocket = useRef<WebSocket | null>(null);
    const lastNextFunc = useRef<(
        err?: Error | null | undefined,
        data?: Map<string, Workload> | MutatorCallback<Map<string, Workload>> | undefined,
    ) => void>();

    const setupWebsocket = (
        hostname: string,
        // forceRecreate: boolean,
        next: ((
            err?: Error | null | undefined,
            data?: Map<string, Workload> | MutatorCallback<Map<string, Workload>> | undefined,
        ) => void),
    ) => {
        if (subscriberSocket.current == null) {
            // We'll use this when we reconnect if we're disconnected.
            // if (next !== undefined && next !== null) {
            //     lastNextFunc.current = next; // Cache the next function so we can reuse it.
            // } else if (lastNextFunc.current !== undefined && lastNextFunc.current !== null) {
            //     console.debug("Used cached `next` function in Workload Websocket.");
            //     next = lastNextFunc.current;
            // } else {
            //     console.error("`next` parameter is null/undefined when setting up Workload Websocket, and we have no cached previous `next` function to fallback to...");
            // }

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
                        .then((respText) => {
                            const respJson: WorkloadResponse = JSON.parse(respText);
                            console.log('Received JSON message from workload websocket:');
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
        console.log(`Connecting to Websocket server at '${key}'`);
        // setupWebsocket(key, false, next);
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
