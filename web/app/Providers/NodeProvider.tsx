import useSWR from 'swr';
import { KubernetesNode } from '@app/Data';
import useSWRMutation from 'swr/mutation';

const api_endpoint: string = 'api/nodes';

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
            console.error('refresh-kubernetes-nodes request timed out.');
            throw new Error(`The request timed out.`); // Different error.
        } else {
            console.error(`Failed to fetch nodes because: ${e}`);
            throw e; // Re-throw e.
        }
    }

    if (response.status != 200) {
        const responseBody: string = await response.text();
        console.error(`Refresh Nodes Failed (${response.status} ${response.statusText}): ${responseBody}`);
        throw new Error(`${response.status} ${response.statusText}`);
        // throw {
        //     name: `${response.status} ${response.statusText}`,
        //     message: `${response.status} ${response.statusText}: ${responseBody}`,
        // };
    }

    return await response.json();
};

export function useNodes() {
    const { data, error, isLoading, isValidating } = useSWR(api_endpoint, fetcher, { refreshInterval: 600000 });
    const { trigger, isMutating } = useSWRMutation(api_endpoint, fetcher);

    const nodes: KubernetesNode[] = data || [];

    return {
        nodes: nodes,
        nodesAreLoading: isLoading || isValidating || isMutating,
        refreshNodes: trigger,
        isError: error,
    };
}
