import { AuthorizationContext } from '@Providers/AuthProvider';
import { RefreshError } from '@Providers/Error';
import { GetPathForFetch } from '@src/Utils/path_utils';
import React from 'react';
import useSWR from 'swr';

interface JupyterAddrResponse {
    jupyter_address: string;
}

const fetcher = async (input: RequestInfo | URL): Promise<JupyterAddrResponse | void> => {
    const init: RequestInit = {
        method: 'GET',
        headers: {
            Authorization: 'Bearer ' + localStorage.getItem('token'),
        },
    };

    const response: Response = await fetch(input, init);

    if (!response.ok) {
        const responseBody: string = await response.text();
        console.error(
            `Refresh JupyterServerAddress failed with (${response.status} ${response.statusText}): ${responseBody}`,
        );
        throw new RefreshError(response);
    }

    const contentType = response.headers.get('content-type');
    if (contentType && contentType.indexOf('application/json') !== -1) {
        const responseJSON: JupyterAddrResponse | Error = await response.json();

        if (!response.ok || response.status != 200 || responseJSON instanceof Error) {
            throw new Error(`HTTP ${response.status} ${response.statusText}: ${JSON.stringify(responseJSON)}`);
        }

        return responseJSON;
    } else {
        const respText: string = await response.text();
        throw new Error(`HTTP ${response.status} ${response.statusText}: ${respText}`);
    }
};

const api_endpoint: string = GetPathForFetch('api/jupyter-address');

export function useJupyterAddress() {
    const { authenticated, setAuthenticated } = React.useContext(AuthorizationContext);

    const { data, isLoading, error } = useSWR(authenticated ? api_endpoint : null, fetcher, {
        refreshInterval: 600000,
        shouldRetryOnError: (err: Error) => {
            // If the error is a RefreshError with status code 401, then don't retry.
            // In all other cases, retry.
            return !(err instanceof RefreshError && (err as RefreshError).statusCode == 401);
        },
        onError: (error: Error) => {
            console.error(`Automatic refresh of JupyterServerAddress failed because: ${error.message}`);

            if (error instanceof RefreshError && (error as RefreshError).statusCode == 401) {
                setAuthenticated(false);
            }
        },
    });

    return {
        jupyterAddress: data?.jupyter_address || undefined,
        isLoading: isLoading,
        isError: error,
    };
}
