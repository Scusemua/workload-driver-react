import { useCallback, useRef, useState } from 'react';
import useSWR from 'swr';
import type { SWRSubscription } from 'swr/subscription';
import { v4 as uuidv4 } from 'uuid';
import { useWebSocket } from 'react-use-websocket/dist/lib/use-websocket';

import { AnsiUp } from 'ansi_up';

const ansi_up = new AnsiUp();

export const useLogs = (podName: string, containerName: string, convertToHtml: boolean) => {
    const logs = useRef<string>('');

    const fetcher = useCallback((req: RequestInfo | URL) => {
        async function get_data(input: RequestInfo | URL) {
            const abortController: AbortController = new AbortController();
            const signal: AbortSignal = abortController.signal;
            const timeout: number = 5000;

            setTimeout(() => {
                abortController.abort(`The request timed-out after ${timeout} milliseconds.`);
            }, timeout);

            try {
                const resp: Response = await fetch(input);

                if (resp.status == 404) {
                    return '';
                }

                const latestLogs: string = await resp.text();
                return latestLogs;
            } catch (e) {
                if (signal.aborted) {
                    console.error(
                        `refresh-kubernetes-logs request for container ${containerName} of Pod ${podName} timed out`,
                    );
                    throw new Error(`The request timed out.`); // Different error.
                } else {
                    console.error(
                        `Failed to refresh logs for container ${containerName} of Pod ${podName} because: ${e}`,
                    );
                    throw e; // Re-throw e.
                }
            }
        }

        let url: RequestInfo | URL = req;
        if (logs.current.length > 0) {
            url += '&sinceSeconds=1000';

            console.log(`Only retrieving new logs for Container ${containerName} of Pod ${podName}`);
        } else {
            console.log(`Retrieving ALL logs for Container ${containerName} of Pod ${podName}`);
        }

        return get_data(url);
    }, []);

    const { data, isLoading, mutate, error } = useSWR(
        `kubernetes/api/v1/namespaces/default/pods/${podName}/log?container=${containerName}`,
        fetcher,
        { refreshInterval: 5000 },
    );

    if (data && data.length > 0) {
        if (convertToHtml) {
            logs.current = logs.current + ansi_up.ansi_to_html(data);
        } else {
            logs.current = logs.current + data;
        }
    }

    return {
        logs: logs.current,
        logsAreRefreshing: isLoading,
        refreshNodes: mutate,
        isError: error,
    };
};
