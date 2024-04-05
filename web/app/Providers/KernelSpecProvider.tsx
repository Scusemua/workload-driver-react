import { JupyterKernelSpecWrapper } from '@app/Data';
import useSWR from 'swr';
import useSWRMutation from 'swr/mutation';

const fetcher = async (input: RequestInfo | URL) => {
    const abortController: AbortController = new AbortController();
    const signal: AbortSignal = abortController.signal;
    const timeout: number = 10000;

    setTimeout(() => {
        abortController.abort(`The request timed-out after ${timeout} milliseconds.`);
    }, timeout);

    try {
        const response: Response = await fetch(input, {
            signal: signal,
        });
        return await response.json();
    } catch (e) {
        if (signal.aborted) {
            console.error('refresh-kernel-specs request timed out.');
            throw new Error(`The request timed out.`); // Different error.
        } else {
            console.error(`Failed to fetch kernels because: ${e}`);
            throw e; // Re-throw e.
        }
    }
};

export function useKernelSpecs() {
    const { data, error, isLoading } = useSWR('jupyter/api/kernelspecs', fetcher, { refreshInterval: 5000 });
    const { trigger, isMutating } = useSWRMutation('jupyter/api/kernelspecs', fetcher);

    const kernelSpecs: JupyterKernelSpecWrapper[] = [];
    if (data) {
        const kernelSpecsParsed: { [key: string]: JupyterKernelSpecWrapper } = JSON.parse(
            JSON.stringify(data['kernelspecs']),
        );
        Object.keys(kernelSpecsParsed).map((key: string) => {
            kernelSpecs.push(kernelSpecsParsed[key]);
        });
    }

    return {
        kernelSpecs: kernelSpecs,
        kernelSpecsAreLoading: isMutating || isLoading, // We'll use both here since this has weird connection problems and it'd be easier to notice those if we used both.
        refreshKernelSpecs: trigger,
        isError: error,
    };
}
