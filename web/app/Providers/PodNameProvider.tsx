import React from 'react';
import useSWR from 'swr';
import useSWRMutation from 'swr/mutation';

const fetcher = async (input: RequestInfo | URL) => {
    const abortController: AbortController = new AbortController();
    const signal: AbortSignal = abortController.signal;
    const timeout: number = 5000;

    setTimeout(() => {
        abortController.abort(`The request timed-out after ${timeout} milliseconds.`);
    }, timeout);

    try {
        const response: Response = await fetch(input, {
            signal: signal,
            // headers: { 'Cache-Control': 'no-cache, no-transform, no-store' },
        });
        const responseJson: Record<string, any> = await response.json();
        const podsJson: Record<string, any>[] = responseJson['items'];

        let gatewayPod: string = '';
        let jupyterPod: string = '';
        podsJson.map((pod: Record<string, any>) => {
            const podName: string = pod['metadata']['name'];
            const containerName: string = pod['spec']['containers'][0]['name'];
            console.log(`Discovered Pod ${podName} with Container ${containerName}`);

            if (podName.includes('gateway')) {
                console.log(`Identified Gateway Pod: ${podName}`);
                gatewayPod = podName;
            } else if (podName.includes('jupyter')) {
                console.log(`Identified Jupyter Pod: ${podName}`);
                jupyterPod = podName;
            }
        });

        return {
            gateway: gatewayPod,
            jupyter: jupyterPod,
        };
    } catch (e) {
        if (signal.aborted) {
            console.error('refresh-pod-names request timed out.');
            throw new Error(`The request timed out.`); // Different error.
        } else {
            console.error(`Failed to fetch Pod Names because: ${e}`);
            throw e; // Re-throw e.
        }
    }
};

const api_endpoint: string = 'kubernetes/api/v1/namespaces/default/pods';

export function usePodNames() {
    const { data, error } = useSWR(api_endpoint, fetcher, {
        refreshInterval: 600000,
        onError: (error: Error) => {
            console.error(`Automatic refresh of Pod Names failed because: ${error.message}`);
        },
    });
    const { trigger, isMutating } = useSWRMutation(api_endpoint, fetcher);

    return {
        gatewayPod: data?.gateway,
        jupyterPod: data?.jupyter,
        podNamesAreLoading: isMutating,
        refreshPodNames: trigger,
        isError: error,
    };
}
