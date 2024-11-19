import { DistributedJupyterKernel } from '@Data/Kernel';
import { AuthorizationContext } from '@Providers/AuthProvider';
import { RefreshError } from '@Providers/Error';
import { GetPathForFetch } from '@src/Utils/path_utils';
import { GetToastContentWithHeaderAndBody } from '@src/Utils/toast_utils';
import React from 'react';
import { Toast, toast } from 'react-hot-toast';
import useSWR from 'swr';

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
            throw new Error(`request timed out`); // Different error.
        } else {
            console.error(`Failed to fetch kernels because: ${e}`);
            throw e; // Re-throw.
        }
    }

    return response;
};

const fetcher = async (input: RequestInfo | URL | null, forLogging: boolean, throwOnError: boolean) => {
    if (!input) {
        return;
    }

    const response: Response = await baseFetcher(input);

    if (!response.ok) {
        console.error(`Received HTTP ${response.status} ${response.statusText} when retrieving kernels.`);

        if (throwOnError) {
            throw new RefreshError(response);
        } else {
            return;
        }
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
    const { authenticated, setAuthenticated } = React.useContext(AuthorizationContext);
    const { data, mutate, isLoading, error } = useSWR(
        authenticated ? [api_endpoint, forLogging] : null,
        ([url, forLogging]) => fetcher(url, forLogging, false),
        {
            refreshInterval: 5000,
            suspense: false,
            shouldRetryOnError: (err: Error) => {
                // If the error is a RefreshError with status code 401, then don't retry.
                // In all other cases, retry.
                return !(err instanceof RefreshError && (err as RefreshError).statusCode == 401);
            },
            onError: (err: RefreshError) => {
                if (err.statusCode == 401) {
                    setAuthenticated(false);
                    return;
                }

                toast.custom((t: Toast) => {
                    return GetToastContentWithHeaderAndBody(
                        'Automatic refresh of active kernels has failed.',
                        `${err.name}: ${err.message}`,
                        'danger',
                        () => toast.dismiss(t.id),
                    );
                });
            },
        },
    );

    const kernels: DistributedJupyterKernel[] = data || [];
    const kernelsMap: Map<string, DistributedJupyterKernel> = new Map<string, DistributedJupyterKernel>();

    if (kernels.length > 0) {
        kernels.forEach((kernel: DistributedJupyterKernel) => {
            kernelsMap.set(kernel.kernelId, kernel);
        });
    }

    async function refreshKernels() {
        try {
            return await mutate(fetcher(authenticated ? api_endpoint : null, forLogging, true), {
                revalidate: true,
                throwOnError: true,
            });
        } catch (err) {
            if ((err as RefreshError).statusCode == 401) {
                setAuthenticated(false);
            }

            throw err; // Re-throw error.
        }
    }

    return {
        kernels: kernels,
        kernelsMap: kernelsMap,
        kernelsAreLoading: isLoading,
        refreshKernels: refreshKernels,
        isError: error,
        error: error,
    };
}
