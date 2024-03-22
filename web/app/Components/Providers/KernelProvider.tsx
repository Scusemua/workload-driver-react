import { DistributedJupyterKernel } from '@data/Kernel';
import useSWR, { mutate } from 'swr';
import { useRef, useState } from 'react';
import { v4 as uuidv4 } from 'uuid';

const fetcher = (input: RequestInfo | URL) => fetch(input).then((response: Response) => response.json());

const api_endpoint: string = 'api/get-kernels';

export function useKernels() {
    const { data, error, isLoading } = useSWR(api_endpoint, fetcher, { refreshInterval: 5000 });

    const kernels: DistributedJupyterKernel[] = data || [];

    return {
        kernels: kernels,
        kernelsAreLoading: isLoading,
        refreshKernels: () => {
            mutate(api_endpoint);
        },
        isError: error,
    };
}
