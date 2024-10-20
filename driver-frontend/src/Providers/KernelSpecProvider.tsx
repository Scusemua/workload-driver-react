import { JupyterKernelSpecWrapper } from '@src/Data';
import { GetPathForFetch } from '@src/Utils/path_utils';
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

        if (response.status !== 200) {
            await Promise.reject(new Error(`HTTP ${response.status} ${response.statusText}`));
        }

        return await response.json();
    } catch (e) {
        if (signal.aborted) {
            console.error('refresh-kernel-specs request timed out.');
            await Promise.reject(new Error(`The request timed out.`)); // Different error.
        } else {
            console.error(`Failed to fetch kernels because: ${e}`);
            await Promise.reject(e); // Re-throw e.
        }
    }
};

export function useKernelSpecs() {
    const { data, error, isLoading } = useSWR(GetPathForFetch('jupyter/api/kernelspecs'), fetcher, { refreshInterval: 5000 });
    const { trigger, isMutating } = useSWRMutation(GetPathForFetch('jupyter/api/kernelspecs'), fetcher);

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
