import { WorkloadPreset } from '@data/Workload';
import useSWR, { mutate } from 'swr';

const fetcher = (input: RequestInfo | URL) => fetch(input).then((response: Response) => response.json());

const api_endpoint: string = 'api/workload-presets';

export function useWorkloadPresets() {
    const { data, error, isLoading } = useSWR(api_endpoint, fetcher, { refreshInterval: 120000 });

    const workloadPresets: WorkloadPreset[] = data || [];

    return {
        workloadPresets: workloadPresets,
        workloadPresetsAreLoading: isLoading,
        refreshWorkloadPresets: () => {
            mutate(api_endpoint);
        },
        isError: error,
    };
}
