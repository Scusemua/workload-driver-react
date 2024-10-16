import { WorkloadPreset } from '@Data/Workload';
import useSWR, { mutate } from 'swr';

const fetcher = (input: RequestInfo | URL) => {
    const randNumber: number = Math.floor(Math.random() * 1e9);
    input += `?randNumber=${randNumber}`;

    return fetch(input, {
        // headers: { 'Cache-Control': 'no-cache, no-transform, no-store' }
    }).then((response: Response) => response.json());
};

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
