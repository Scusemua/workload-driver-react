import { AuthorizationContext } from '@Providers/AuthProvider';
import { RefreshError } from '@Providers/Error';
import { JupyterKernelSpecWrapper } from '@src/Data';
import { GetPathForFetch } from '@src/Utils/path_utils';
import React from 'react';
import useSWR from 'swr';
import useSWRMutation from 'swr/mutation';

const fetcher = async (input: RequestInfo | URL) => {
    const abortController: AbortController = new AbortController();
    const signal: AbortSignal = abortController.signal;
    const timeout: number = 10000;

    setTimeout(() => {
        abortController.abort(`The request timed-out after ${timeout} milliseconds.`);
    }, timeout);

    let response: Response;
    try {
        response = await fetch(input, {
            signal: signal,
        });
    } catch (e) {
        if (signal.aborted) {
            console.error('refresh-kernel-specs request timed out.');
            throw new Error(`The request timed out.`); // Different error.
        } else {
            console.error(`Failed to fetch kernels because: ${e}`);
            throw e; // Re-throw.
        }
    }

    if (response.status !== 200) {
        throw new RefreshError(response);
    }

    return await response.json();
};

export function useKernelSpecs() {
    const { authenticated, setAuthenticated } = React.useContext(AuthorizationContext);

    const { data, error, isLoading } = useSWR(
        authenticated ? GetPathForFetch('/jupyter/api/kernelspecs') : null,
        fetcher,
        {
            refreshInterval: () => {
                if (data) {
                    // If we already have data, then we really don't need to refresh this very often.
                    return 600000; // 10 min.
                } else {
                    return 5000;
                }
            },
            shouldRetryOnError: (err: Error) => {
                // If the error is a RefreshError with status code 401, then don't retry.
                // In all other cases, retry.
                return !(err instanceof RefreshError && (err as RefreshError).statusCode == 401);
            },
            onError: (err: Error) => {
                if (err instanceof RefreshError && (err as RefreshError).statusCode == 401) {
                    setAuthenticated(false);
                }
            },
        },
    );
    const { trigger, isMutating } = useSWRMutation(GetPathForFetch('/jupyter/api/kernelspecs'), fetcher);

    const kernelSpecs: JupyterKernelSpecWrapper[] = [];
    let jsonParseError: boolean = false;
    if (data) {
        try {
            const kernelSpecsParsed: { [key: string]: JupyterKernelSpecWrapper } = JSON.parse(
                JSON.stringify(data['kernelspecs']),
            );
            Object.keys(kernelSpecsParsed).map((key: string) => {
                kernelSpecs.push(kernelSpecsParsed[key]);
            });
        } catch (ex) {
            console.error('Failed to parse kernelspecs: %s', ex);
            jsonParseError = true;
        }
    }

    return {
        kernelSpecs: kernelSpecs,
        kernelSpecsAreLoading: isMutating || isLoading, // We'll use both here since this has weird connection problems and it'd be easier to notice those if we used both.
        refreshKernelSpecs: trigger,
        isError: error || jsonParseError,
    };
}
