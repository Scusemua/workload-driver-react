import { AuthorizationContext } from '@Providers/AuthProvider';
import { RefreshError } from '@Providers/Error';
import { GetPathForFetch } from '@src/Utils/path_utils';
import React from 'react';
import useSWR from 'swr';
import useSWRMutation from 'swr/mutation';

const fetcher = async (input: RequestInfo | URL) => {
    const abortController: AbortController = new AbortController();
    const signal: AbortSignal = abortController.signal;
    const timeout: number = 5000;

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

    // try {
    const response: Response = await fetch(input, init);

    if (!response.ok) {
        const responseBody: string = await response.text();
        console.error(`Refresh Pod Names (${response.status} ${response.statusText}): ${responseBody}`);
        throw new RefreshError(response);
    }

    /* eslint-disable @typescript-eslint/no-explicit-any */
    const responseJson: Record<string, any> = await response.json();
    /* eslint-disable @typescript-eslint/no-explicit-any */
    const podsJson: Record<string, any>[] = responseJson['items'];

    let gatewayPod: string = '';
    let jupyterPod: string = '';
    /* eslint-disable @typescript-eslint/no-explicit-any */
    podsJson.map((container: Record<string, any>) => {
        const podName: string = container['metadata']['name'];
        const containerName: string = container['spec']['containers'][0]['name'];
        console.log(`Discovered Pod ${podName} with Container ${containerName}`);

        if (podName.includes('gateway')) {
            console.log(`Identified Gateway Pod: ${podName}`);
            gatewayPod = podName;
        } else if (podName.includes('jupyter')) {
            console.log(`Identified Jupyter Pod: ${podName}`);
            jupyterPod = podName;
        }
    });

    return {
        gateway: gatewayPod,
        jupyter: jupyterPod,
    };
};

const api_endpoint: string = GetPathForFetch('kubernetes/api/v1/namespaces/default/pods');

export function usePodNames() {
    const { authenticated, setAuthenticated } = React.useContext(AuthorizationContext);

    const { data, error } = useSWR(authenticated ? api_endpoint : null, fetcher, {
        refreshInterval: 600000,
        shouldRetryOnError: (err: Error) => {
            // If the error is a RefreshError with status code 401, then don't retry.
            // In all other cases, retry.
            return !(err instanceof RefreshError && (err as RefreshError).statusCode == 401);
        },
        onError: (error: Error) => {
            console.error(`Automatic refresh of Pod Names failed because: ${error.message}`);

            if (error instanceof RefreshError && (error as RefreshError).statusCode == 401) {
                setAuthenticated(false);
            }
        },
    });
    const { trigger, isMutating } = useSWRMutation(api_endpoint, fetcher);

    return {
        gatewayPod: data?.gateway,
        jupyterPod: data?.jupyter,
        podNamesAreLoading: isMutating,
        refreshPodNames: trigger,
        isError: error,
    };
}
