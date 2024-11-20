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
            console.error('refresh-cluster-scheduling-policy request timed out.');
            throw new Error(`The request for the scheduling policy of the cluster has timed out.`); // Different error.
        } else {
            console.error(`Failed to fetch cluster scheduling policy because: ${e}`);
            throw e; // Re-throw e.
        }
    }

    if (!response.ok) {
        const responseBody: string = await response.text();
        console.error(`Refresh cluster scheduling policy (${response.status} ${response.statusText}): ${responseBody}`);
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
            `The request for the scheduling policy of the cluster has timed-out after ${timeout} milliseconds.`,
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
        console.error(`Received HTTP ${response.status} ${response.statusText} when retrieving scheduling policy.`);
        return -1;
    }

    const schedulingPolicy: string = await response.text();

    console.log(`Returning scheduling policy: ${schedulingPolicy}`);
    return schedulingPolicy;
};

const api_endpoint: string = GetPathForFetch('api/scheduling-policy');

export function useClusterSchedulingPolicy() {
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
            console.error(`Automatic refresh of cluster scheduling policy failed because: ${err.message}`);

            if (err instanceof RefreshError && (err as RefreshError).statusCode == 401) {
                setAuthenticated(false);
            }
        },
    });

    return {
        schedulingPolicy: data as string,
        isError: error,
    };
}
