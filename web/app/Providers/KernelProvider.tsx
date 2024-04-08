import { DistributedJupyterKernel, JupyterKernelReplica } from '@data/Kernel';
import useSWR from 'swr';
import useSWRMutation from 'swr/mutation';
import hash from 'stable-hash';

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

    // const randNumber: number = Math.floor(Math.random() * 1e9);
    // input += `?randNumber=${randNumber}`;

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

    return response;
};

const fetcher = async (input: RequestInfo | URL, forLogging: boolean) => {
    const response: Response = await baseFetcher(input);
    let kernels: DistributedJupyterKernel[] = await response.json();

    if (forLogging) {
        kernels = kernels.map((kernel: DistributedJupyterKernel) =>
            omit(kernel, 'status', 'aggregateBusyStatus', 'kernelSpec', 'kernel'),
        );

        return kernels;
    }

    return kernels;
};

const api_endpoint: string = 'api/get-kernels';

export function useKernels(forLogging: boolean) {
    const { data, error } = useSWR([api_endpoint, forLogging], ([url, forLogging]) => fetcher(url, forLogging), {
        refreshInterval: 5000,
        onError: (error: Error) => {
            console.error(`Automatic refresh of kernels failed because: ${error.message}`);
        },
        // onSuccess: (data, key) => {
        //     console.log(`Refreshed kernels. Key: "${key}".`);
        // },
        // onDiscarded: () => {
        //     console.log('Refreshed kernels.');
        // },
        // compare: (a: any, b: any) => {
        //     console.log('');
        //     console.log('');
        //     console.log('');

        //     if (hash(a) !== hash(b)) {
        //         console.warn('Data IS new!');
        //         console.log(`Old data: ${JSON.stringify(a)}`);
        //         console.log(`New data: ${JSON.stringify(b)}`);
        //         return false;
        //     } else {
        //         console.log('Data is not new.');
        //         return true;
        //     }
        // },
    });
    const { trigger, isMutating } = useSWRMutation([api_endpoint, forLogging], ([url, forLogging]) =>
        fetcher(url, forLogging),
    );

    const kernels: DistributedJupyterKernel[] = data || [];

    return {
        kernels: kernels,
        kernelsAreLoading: isMutating,
        refreshKernels: trigger,
        isError: error,
    };
}
