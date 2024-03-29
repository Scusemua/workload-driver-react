import useSWR, { mutate } from 'swr';
import { KubernetesNode } from '@app/Data';
import useSWRMutation from 'swr/mutation';

const api_endpoint: string = 'api/nodes';

const fetcher = async (input: RequestInfo | URL) => {
    const abortController: AbortController = new AbortController();
    const signal: AbortSignal = abortController.signal;
    const timeout: number = 5000;

    setTimeout(() => {
        abortController.abort(`The request timed-out after ${timeout} milliseconds.`);
    }, timeout);

    try {
        const response: Response = await fetch(input, { signal: signal });
        return await response.json();
    } catch (e) {
        if (signal.aborted) {
            console.error(`Request timed out.`);
            throw new Error(`The request timed out.`); // Different error.
        } else {
            console.error(`Failed to fetch nodes because: ${e}`);
            throw e; // Re-throw e.
        }
    }
    // .then((response: Response) => response.json());
};

export function useNodes() {
    const { data, error } = useSWR(api_endpoint, fetcher, { refreshInterval: 600000 });
    const { trigger, isMutating } = useSWRMutation(api_endpoint, fetcher);

    const nodes: KubernetesNode[] = data || [];

    return {
        nodes: nodes,
        nodesAreLoading: isMutating,
        refreshNodes: trigger,
        isError: error,
    };
}
