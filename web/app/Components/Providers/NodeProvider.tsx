import useSWR, { mutate } from 'swr';
import { KubernetesNode } from '@app/Data';

const fetcher = (input: RequestInfo | URL) => fetch(input).then((response: Response) => response.json());

const api_endpoint: string = 'api/nodes';

export function useNodes() {
    const { data, error, isLoading } = useSWR(api_endpoint, fetcher, { refreshInterval: 600000 });

    const nodes: KubernetesNode[] = data || [];

    return {
        nodes: nodes,
        nodesAreLoading: isLoading,
        refreshNodes: () => {
            mutate(api_endpoint);
        },
        isError: error,
    };
}
