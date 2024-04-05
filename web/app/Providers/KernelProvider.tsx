import { DistributedJupyterKernel, JupyterKernelReplica } from '@data/Kernel';
import useSWR from 'swr';
import useSWRMutation from 'swr/mutation';

const fetcher = async (input: RequestInfo | URL) => {
    const abortController: AbortController = new AbortController();
    const signal: AbortSignal = abortController.signal;
    const timeout: number = 10000;

    const randNumber: number = Math.floor(Math.random() * 1e9);
    input += `?randNumber=${randNumber}`;

    setTimeout(() => {
        abortController.abort(`The request timed-out after ${timeout} milliseconds.`);
    }, timeout);

    let response: Response | null = null;
    try {
        response = await fetch(input, {
            signal: signal,
        });
    } catch (e) {
        if (signal.aborted) {
            console.error('refresh-kernels request timed out.');
            throw new Error(`The request timed out.`); // Different error.
        } else {
            console.error(`Failed to fetch kernels because: ${e}`);
            throw e; // Re-throw e.
        }
    }

    if (response.status != 200) {
        const responseBody: string = await response.text();
        console.error(`Refresh Nodes Failed (${response.status} ${response.statusText}): ${responseBody}`);
        throw {
            name: `${response.status} ${response.statusText}`,
            message: `${response.status} ${response.statusText}: ${responseBody}`,
        };
    }

    return await response.json();
};

const api_endpoint: string = 'api/get-kernels';

export function useKernelsNamesOnly() {
    const { data, error } = useSWR(api_endpoint, fetcher, {
        refreshInterval: 5000,
        onError: (error: Error) => {
            console.error(`Automatic refresh of kernels failed because: ${error.message}`);
        },
    });
    const { trigger, isMutating } = useSWRMutation(api_endpoint, fetcher);

    const kernels: Pick<DistributedJupyterKernel<JupyterKernelReplica>, 'kernelId' | 'numReplicas' | 'replicas'>[] =
        data || [];

    return {
        kernels: kernels,
        kernelsAreLoading: isMutating,
        refreshKernels: trigger,
        isError: error,
    };
}

export function useKernels() {
    const { data, error } = useSWR(api_endpoint, fetcher, {
        refreshInterval: 5000,
        onError: (error: Error) => {
            console.error(`Automatic refresh of kernels failed because: ${error.message}`);
        },
    });
    const { trigger, isMutating } = useSWRMutation(api_endpoint, fetcher);

    const kernels: DistributedJupyterKernel<JupyterKernelReplica>[] = data || [];

    return {
        kernels: kernels,
        kernelsAreLoading: isMutating,
        refreshKernels: trigger,
        isError: error,
    };
}
