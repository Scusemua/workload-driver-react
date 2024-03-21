import { useRef } from 'react';
import useSWRSubscription from 'swr/subscription';
import type { SWRSubscription } from 'swr/subscription';
import { Workload, WorkloadResponse } from '@data/Workload';
import { v4 as uuidv4 } from 'uuid';
import { useWebSocket } from 'react-use-websocket/dist/lib/use-websocket';

export const useWorkloads = () => {
    console.log('useWorkloads() has been called!');

    const subscriberSocket = useRef<WebSocket | null>(null);

    // function sendJsonMessage(data) {
    //     if (subscriberSocket.current) {
    //         subscriberSocket.current.send(JSON.stringify(data));
    //     }
    // }

    const { sendJsonMessage } = useWebSocket<Record<string, unknown>>('ws://localhost:8000/workload', {
        share: false,
        shouldReconnect: () => true,
    });

    function refreshWorkloads() {
        sendJsonMessage(
            JSON.stringify({
                op: 'get_workloads',
            }),
        );
    }

    const subscribe: SWRSubscription<string, Map<string, Workload>, Error> = (key: string, { next }) => {
        console.log(`Connecting to Websocket server at '${key}'`);

        if (subscriberSocket.current == null) {
            subscriberSocket.current = new WebSocket(key);
            subscriberSocket.current.addEventListener('open', () => {
                console.log('Connected to websocket. Sending message now.');
                subscriberSocket.current?.send(
                    JSON.stringify({
                        op: 'subscribe',
                        msg_id: uuidv4(),
                    }),
                );
            });

            subscriberSocket.current.addEventListener('message', (event) => {
                event.data
                    .text()
                    .then((respText) => {
                        const respJson: WorkloadResponse = JSON.parse(respText);
                        console.log('Received WebsocketMessage: ', respJson);
                        return respJson;
                    })
                    .then((workloadResponse: WorkloadResponse) =>
                        next(null, (prev: Map<string, Workload> | undefined) => {
                            console.log(`Previous data: ${JSON.stringify(prev)}`);
                            console.log(`Next data: ${JSON.stringify(workloadResponse)}`);

                            const newWorkloads: Workload[] | null | undefined = workloadResponse.new_workloads;
                            const modifiedWorkloads: Workload[] | null | undefined =
                                workloadResponse.modified_workloads;
                            const deletedWorkloads: Workload[] | null | undefined = workloadResponse.deleted_workloads;

                            const nextData: Map<string, Workload> = new Map(prev);

                            newWorkloads?.forEach((workload: Workload) => {
                                console.log(`Found new workload: ${JSON.stringify(workload)}`);
                                nextData.set(workload.id, workload);
                            });
                            modifiedWorkloads?.forEach((workload: Workload) => {
                                console.log(`Found modified workload: ${JSON.stringify(workload)}`);
                                nextData.set(workload.id, workload);
                            });
                            deletedWorkloads?.forEach((workload: Workload) => {
                                console.log(`Found deleted workload: ${JSON.stringify(workload)}`);
                                nextData.delete(workload.id);
                            });

                            console.log(`Next data has this many elements: ${nextData.size}`);
                            console.log(`Returning data: ${nextData}`);

                            return nextData;
                        }),
                    );
            });
        }
        return () => {};
    };

    const { data, error } = useSWRSubscription('ws://localhost:8000/workload', subscribe);
    const workloadsMap: Map<string, Workload> = data || new Map();
    console.log(`Data is a map with this many elements: ${workloadsMap.size}`);

    return {
        workloads: Array.from(workloadsMap.values()),
        isError: error,
        refreshWorkloads: refreshWorkloads,
        sendJsonMessage: sendJsonMessage,
    };
};
