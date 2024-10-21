import { RoundToTwoDecimalPlaces } from '@Components/Modals';
import { AuthorizationContext } from '@Providers/AuthProvider';
import { ClusterNode } from '@src/Data';
import { GetPathForFetch } from '@src/Utils/path_utils';
import { GetToastContentWithHeaderAndBody } from '@src/Utils/toast_utils';
import React from 'react';
import { toast } from 'react-hot-toast';
import useSWR from 'swr';
import useSWRMutation, { TriggerWithoutArgs } from 'swr/mutation';

const api_endpoint: string = GetPathForFetch('api/nodes');

const fetcher = async (input: RequestInfo | URL) => {
    const abortController: AbortController = new AbortController();
    const signal: AbortSignal = abortController.signal;
    const timeout: number = 10000;

    const randNumber: number = Math.floor(Math.random() * 1e9);
    input += `?randNumber=${randNumber}`;

    setTimeout(() => {
        abortController.abort(`The request timed-out after ${timeout} milliseconds.`);
    }, timeout);

    const init: RequestInit = {
        method: 'GET',
        headers: {
            Authorization: 'Bearer ' + localStorage.getItem('token'),
        },
        signal: signal,
    };

    let response: Response | null = null;
    try {
        response = await fetch(input, init);
    } catch (e) {
        if (signal.aborted) {
            console.error('refresh-kubernetes-nodes request timed out.');
            throw new Error(`The request timed out.`); // Different error.
        } else {
            console.error(`Failed to fetch nodes because: ${e}`);
            throw e; // Re-throw e.
        }
    }

    if (!response.ok) {
        const responseBody: string = await response.text();
        console.error(`Refresh Nodes Failed (${response.status} ${response.statusText}): ${responseBody}`);
        throw new Error(`${response.status} ${response.statusText}`);
    }

    return await response.json();
};

function getManualRefreshTrigger(trigger: TriggerWithoutArgs<any, any, string, never>): (showToast?: boolean) => void {
    return async (showToast: boolean = true) => {
        console.log('Manually refreshing nodes now.');

        if (!showToast) {
            await trigger().catch((error: Error) => {
                console.error(`Failed to refresh nodes because: ${error.message}`);
            });
            return;
        }

        const st: number = performance.now();
        const toastId: string = toast.loading(() => <b>Refreshing nodes...</b>);
        await trigger()
            .catch((error: Error) => {
                toast.custom(
                    GetToastContentWithHeaderAndBody('Could not refresh nodes.', error.message, 'danger', () => {
                        toast.dismiss(toastId);
                    }),
                    {
                        id: toastId,
                        duration: 10000,
                        style: { maxWidth: 600 },
                    },
                );
            })
            .then(() =>
                toast.custom(
                    GetToastContentWithHeaderAndBody(
                        'Refreshed nodes.',
                        `Time elapsed: ${RoundToTwoDecimalPlaces(performance.now() - st)} ms`,
                        'success',
                        () => {
                            toast.dismiss(toastId);
                        },
                    ),
                    { id: toastId, duration: 7500, style: { maxWidth: 600 } },
                ),
            );
    };
}

export function useNodes() {
    const { authenticated } = React.useContext(AuthorizationContext);
    const { data, error, isLoading, isValidating } = useSWR(authenticated ? api_endpoint : null, fetcher, { refreshInterval: 600000 });
    const { trigger, isMutating } = useSWRMutation(api_endpoint, fetcher);

    const nodes: ClusterNode[] = data || [];

    // if (nodes.length > 0) {
    //     console.log(`Received ${nodes.length} ClusterNode(s) from server:`);
    //     console.log(JSON.stringify(nodes, null, 2));
    // } else {
    //     console.warn('Received 0 ClusterNodes from server...');
    // }

    return {
        nodes: nodes,
        nodesAreLoading: isMutating || isLoading || isValidating,
        refreshNodes: getManualRefreshTrigger(trigger),
        isError: error,
    };
}
