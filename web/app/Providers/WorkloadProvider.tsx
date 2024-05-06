import { useRef } from 'react';
import useSWRSubscription from 'swr/subscription';
import type { SWRSubscription } from 'swr/subscription';
import { Workload, WorkloadResponse } from '@data/Workload';
import { v4 as uuidv4 } from 'uuid';
import { MutatorCallback } from 'swr';

const api_endpoint: string = 'ws://localhost:8000/workload';

export const useWorkloads = () => {
    const subscriberSocket = useRef<WebSocket | null>(null);

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
                                    nextData.set(workload.id, workload);
                                });
                                modifiedWorkloads?.forEach((workload: Workload) => {
                                    nextData.set(workload.id, workload);
                                });
                                deletedWorkloads?.forEach((workload: Workload) => {
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
                console.log(`Workloads Subscriber WebSocket encountered error: ${event}`);
            });
        }
    };

    const sendJsonMessage = (msg: string) => {
        if (subscriberSocket.current?.readyState !== WebSocket.OPEN) {
            console.error(
                `Cannot send workload-related message via websocket. Websocket is in state ${subscriberSocket.current?.readyState}`,
            );
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
        setupWebsocket(key, next);
        return () => {};
    };

    const { data, error } = useSWRSubscription(api_endpoint, subscribe);
    const workloadsMap: Map<string, Workload> = data || new Map();

    return {
        workloads: Array.from(workloadsMap.values()),
        isError: error,
        refreshWorkloads: refreshWorkloads,
        sendJsonMessage: sendJsonMessage,
    };
};
