import { useRef } from 'react';
import useSWRSubscription from 'swr/subscription';
import type { SWRSubscription } from 'swr/subscription';
import { Workload, WorkloadResponse } from '@data/Workload';
import { v4 as uuidv4 } from 'uuid';
import { useWebSocket } from 'react-use-websocket/dist/lib/use-websocket';

export const useWorkloads = () => {
    const subscriberSocket = useRef<WebSocket | null>(null);

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
                        return respJson;
                    })
                    .then((workloadResponse: WorkloadResponse) =>
                        next(null, (prev: Map<string, Workload> | undefined) => {
                            const newWorkloads: Workload[] | null | undefined = workloadResponse.new_workloads;
                            const modifiedWorkloads: Workload[] | null | undefined =
                                workloadResponse.modified_workloads;
                            const deletedWorkloads: Workload[] | null | undefined = workloadResponse.deleted_workloads;

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
            });
        }
        return () => {};
    };

    const { data, error } = useSWRSubscription('ws://localhost:8000/workload', subscribe);
    const workloadsMap: Map<string, Workload> = data || new Map();

    return {
        workloads: Array.from(workloadsMap.values()),
        isError: error,
        refreshWorkloads: refreshWorkloads,
        sendJsonMessage: sendJsonMessage,
    };
};
