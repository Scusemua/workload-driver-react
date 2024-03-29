import { KernelSpecManager, ServerConnection } from '@jupyterlab/services';
import { ISpecModel } from '@jupyterlab/services/lib/kernelspec/restapi';
import { useEffect, useRef } from 'react';
import useSWR, { mutate } from 'swr';

export function useKernelSpecs() {
    const kernelSpecManager = useRef<KernelSpecManager | null>(null);
    const fetcher = () => {
        if (!kernelSpecManager.current) {
            throw new Error('The KernelSpecManager is still initializing or is otherwise unavailable right now.');
        }
        return kernelSpecManager.current?.refreshSpecs().then(() => {
            const kernelSpecs: { [key: string]: ISpecModel | undefined } =
                kernelSpecManager.current?.specs?.kernelspecs || {};
            return kernelSpecs;
        });
    };

    useEffect(() => {
        async function initializeKernelManagers() {
            if (kernelSpecManager.current === null) {
                const kernelSpecManagerOptions: KernelSpecManager.IOptions = {
                    serverSettings: ServerConnection.makeSettings({
                        token: '',
                        appendToken: false,
                        baseUrl: 'jupyter',
                        fetch: fetch,
                    }),
                };
                kernelSpecManager.current = new KernelSpecManager(kernelSpecManagerOptions);

                console.log('Waiting for kernel spec manager to be ready.');

                kernelSpecManager.current.connectionFailure.connect((_sender: KernelSpecManager, err: Error) => {
                    console.log(
                        '[ERROR] An error has occurred while preparing the Kernel Spec Manager. ' +
                            err.name +
                            ': ' +
                            err.message,
                    );
                });

                await kernelSpecManager.current.ready.then(() => {
                    console.log('Kernel spec manager is ready!');
                });
            }
        }

        initializeKernelManagers();
    }, []);

    const { data, error, isLoading } = useSWR('/', fetcher, { refreshInterval: 5000 });
    const kernelSpecs: { [key: string]: ISpecModel | undefined } = data || {};

    return {
        kernelSpecs: kernelSpecs,
        kernelSpecsAreLoading: isLoading,
        refreshKernelSpecs: () => {
            mutate('/');
        },
        isError: error,
    };
}
