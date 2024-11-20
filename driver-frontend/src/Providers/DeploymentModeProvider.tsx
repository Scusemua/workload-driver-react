import { AuthorizationContext } from '@Providers/AuthProvider';
import { RefreshError } from '@Providers/Error';
import { GetPathForFetch } from '@src/Utils/path_utils';
import React from 'react';
import useSWR from 'swr';

const baseFetcher = async (input: RequestInfo | URL, init: RequestInit) => {
    let response: Response | null = null;
    try {
        response = await fetch(input, init);
    } catch (e) {
        if (init.signal?.aborted) {
            console.error('refresh-cluster-deployment-mode request timed out.');
            throw new Error(`The request for the deployment mode of the cluster has timed out.`); // Different error.
        } else {
            console.error(`Failed to fetch cluster deployment mode because: ${e}`);
            throw e; // Re-throw e.
        }
    }

    if (!response.ok) {
        const responseBody: string = await response.text();
        console.error(`Refresh cluster deployment mode (${response.status} ${response.statusText}): ${responseBody}`);
        throw new RefreshError(response);
    }

    return response;
};

const fetcher = async (input: RequestInfo | URL) => {
    const abortController: AbortController = new AbortController();
    const signal: AbortSignal = abortController.signal;
    const timeout: number = 10000;

    setTimeout(() => {
        abortController.abort(
            `The request for the deployment mode of the cluster has timed-out after ${timeout} milliseconds.`,
        );
    }, timeout);

    const req: RequestInit = {
        method: 'GET',
        headers: {
            Authorization: 'Bearer ' + localStorage.getItem('token'),
        },
        signal: signal,
    };

    const response: Response = await baseFetcher(input, req);

    if (!response.ok) {
        console.error(`Received HTTP ${response.status} ${response.statusText} when retrieving deployment mode.`);
        return -1;
    }

    const deploymentMode: string = await response.text();

    console.log(`Returning deployment mode: ${deploymentMode}`);
    return deploymentMode;
};

const api_endpoint: string = GetPathForFetch('api/deployment-mode');

export function useClusterDeploymentMode() {
    const { authenticated, setAuthenticated } = React.useContext(AuthorizationContext);

    const { data, error } = useSWR(authenticated ? [api_endpoint] : null, ([url]) => fetcher(url), {
        refreshInterval: 120000,
        shouldRetryOnError: (err: Error) => {
            // If the error is a RefreshError with status code 401, then don't retry.
            // In all other cases, retry.
            return !(err instanceof RefreshError && (err as RefreshError).statusCode == 401);
        },
        revalidateOnFocus: true,
        revalidateOnMount: true,
        revalidateOnReconnect: true,
        refreshWhenOffline: true,
        refreshWhenHidden: true,
        onError: (err: Error) => {
            console.error(`Automatic refresh of cluster deployment mode failed because: ${err.message}`);

            if (err instanceof RefreshError && (err as RefreshError).statusCode == 401) {
                setAuthenticated(false);
            }
        },
    });

    return {
        deploymentMode: data as string,
        isError: error,
    };
}
