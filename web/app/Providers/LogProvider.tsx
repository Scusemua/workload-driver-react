import React, { useCallback, useRef } from 'react';
import useSWR from 'swr';
import useSWRMutation from 'swr/mutation';
import useSWRSubscription, { SWRSubscription } from 'swr/subscription';

const api_endpoint: string = 'kubernetes/api/v1/namespaces/default/pods';

export function useLogProvider(pod: string, container: string) {
    const abortController: AbortController = new AbortController();
    const signal: AbortSignal = abortController.signal;
    const logs = useRef('');

    const fetcher = async (input: RequestInfo | URL) => {
        const req: RequestInit = {
            method: 'GET',
            headers: {
                'Content-Type': 'text/plain',
                'Transfer-Encoding': 'chunked',
                // 'Cache-Control': 'no-cache, no-transform, no-store',
            },
            signal: signal,
        };

        const randNumber: number = Math.floor(Math.random() * 1e9);
        input += `&randNumber=${randNumber}`;

        const response: Response = await fetch(input, req);

        const reader: ReadableStreamDefaultReader<Uint8Array> | undefined = response.body?.getReader();

        while (true) {
            const response: ReadableStreamReadResult<Uint8Array> | undefined = await reader?.read();

            if (response?.done) {
                return;
            }

            const logsAsString: string = String.fromCharCode.apply(null, response!.value);
            logs.current = logs.current + logsAsString;
        }
    };

    const subscribe: SWRSubscription<string, string, Error> = (key: string, { next }) => {};

    const url: string = `api/logs/pods/${pod}?container=${container}&follow=true`;

    const { data, error } = useSWRSubscription(url, (key, { next }) => {
        dataSource.subscribe(key, (err, data) => {
            next(err, data);
        });
        return abortController.abort;
    });

    return {
        gatewayPod: data?.gateway,
        jupyterPod: data?.jupyter,
        podNamesAreLoading: isMutating,
        refreshPodNames: trigger,
        isError: error,
    };
}
