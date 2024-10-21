import { DistributedJupyterKernel } from '@Data/Kernel';
import { AuthorizationContext } from '@Providers/AuthProvider';
import { GetPathForFetch } from '@src/Utils/path_utils';
import React from 'react';
import useSWR from 'swr';
import useSWRMutation from 'swr/mutation';

function omit(obj, ...props) {
    const result = { ...obj };
    props.forEach(function (prop) {
        delete result[prop];
    });
    return result;
}

const baseFetcher = async (input: RequestInfo | URL) => {
    const abortController: AbortController = new AbortController();
    const signal: AbortSignal = abortController.signal;
    const timeout: number = 10000;

    const init: RequestInit = {
        method: 'GET',
        headers: {
            Authorization: 'Bearer ' + localStorage.getItem('token'),
        },
        signal: signal,
    };

    setTimeout(() => {
        abortController.abort(`The request timed-out after ${timeout} milliseconds.`);
    }, timeout);

    let response: Response | null = null;
    try {
        response = await fetch(input, init);
    } catch (e) {
        if (signal.aborted) {
            console.error('refresh-kernels request timed out.');
            return Promise.reject(new Error(`The request timed out.`)); // Different error.
        } else {
            console.error(`Failed to fetch kernels because: ${e}`);
            return Promise.reject(e); // Re-throw e.
        }
    }

    return response;
};

const fetcher = async (input: RequestInfo | URL, forLogging: boolean) => {
    const response: Response = await baseFetcher(input);

    if (!response.ok) {
        console.error(`Received HTTP ${response.status} ${response.statusText} when retrieving kernels.`);
        throw new Error(`Received HTTP ${response.status} ${response.statusText} when retrieving kernels.`);
    }

    let kernels: DistributedJupyterKernel[] = await response.json();

    if (forLogging) {
        kernels = kernels.map((kernel: DistributedJupyterKernel) =>
            omit(kernel, 'status', 'aggregateBusyStatus', 'kernelSpec', 'kernel'),
        );

        return kernels;
    }

    return kernels;
};

const api_endpoint: string = GetPathForFetch('api/get-kernels');

export function useKernels(forLogging: boolean) {
    const { authenticated } = React.useContext(AuthorizationContext);
    const { data, error } = useSWR(
        authenticated ? [api_endpoint, forLogging] : null,
        ([url, forLogging]) => fetcher(url, forLogging),
        {
            refreshInterval: 5000,
            suspense: false,
            onError: (error: Error) => {
                console.error(`Automatic refresh of kernels failed because: ${error.message}`);
            },
        },
    );
    const { trigger, isMutating } = useSWRMutation([api_endpoint, forLogging], ([url, forLogging]) =>
        fetcher(url, forLogging),
    );

    const kernels: DistributedJupyterKernel[] = data || [];

    return {
        kernels: kernels,
        kernelsAreLoading: isMutating,
        refreshKernels: async () => {
            try {
                const resp: DistributedJupyterKernel[] = await trigger();
                return Promise.resolve(resp);
            } catch (e) {
                console.error(`Trigger failed: ${JSON.stringify(e)}`);
                return Promise.reject(e);
            }
        },
        isError: error,
    };
}
