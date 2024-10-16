import { useRef } from 'react';
import useSWRSubscription from 'swr/subscription';
import type { SWRSubscription } from 'swr/subscription';
import { Workload, WorkloadResponse } from '@Data/Workload';
import { v4 as uuidv4 } from 'uuid';
import { MutatorCallback } from 'swr';
import React from 'react';

const api_endpoint: string = 'ws://localhost:8000/logs';

export const useLogs = (container: string) => {
    // const [ logs, setLogs ] = React.useState<string[]>([]);

    const { data, error } = useSWRSubscription(api_endpoint + '/' + container + '/' + container, (key, { next }) => {
        const socket: WebSocket = new WebSocket(api_endpoint);

        socket.addEventListener('open', () => {
            socket.send(
                JSON.stringify({
                    op: 'get_logs',
                    msg_id: uuidv4(),
                    container: container,
                    follow: true,
                }),
            );
        });

        socket.binaryType = 'arraybuffer';
        socket.addEventListener('message', (event) => next(null, new TextDecoder().decode(event.data)));
        return () => socket.close();
    });

    return {
        latestLogMessage: data,

        isError: error,
    };
};
