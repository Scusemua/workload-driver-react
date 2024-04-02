import { DistributedJupyterKernel } from '@data/Kernel';
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

    try {
        const response: Response = await fetch(input, {
            signal: signal,
            headers: { 'Cache-Control': 'no-cache, no-transform, no-store' },
        });
        return await response.json();
    } catch (e) {
        if (signal.aborted) {
            console.error('refresh-kernels request timed out.');
            throw new Error(`The request timed out.`); // Different error.
        } else {
            console.error(`Failed to fetch kernels because: ${e}`);
            throw e; // Re-throw e.
        }
    }
};

const api_endpoint: string = 'api/get-kernels';

export function useKernels() {
    const { data, error } = useSWR(api_endpoint, fetcher, {
        refreshInterval: 5000,
        onError: (error: Error) => {
            console.error(`Automatic refresh of kernels failed because: ${error.message}`);
        },
    });
    const { trigger, isMutating } = useSWRMutation(api_endpoint, fetcher);

    const kernels: DistributedJupyterKernel[] = data || [];

    return {
        kernels: kernels,
        kernelsAreLoading: isMutating,
        refreshKernels: trigger,
        isError: error,
    };
}
