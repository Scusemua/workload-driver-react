import { KernelManager, ServerConnection, SessionManager } from '@jupyterlab/services';
import { useJupyterAddress } from '@Providers/JupyterAddressProvider';
import { JoinPaths } from '@src/Utils';
import { useCallback, useEffect, useRef } from 'react';
import toast from 'react-hot-toast';

export function useKernelAndSessionManagers() {
    const { jupyterAddress } = useJupyterAddress();

    const kernelManager = useRef<KernelManager | null>(null);
    const sessionManager = useRef<SessionManager | null>(null);

    const initializeKernelManagers = useCallback(async () => {
        if (kernelManager.current === null) {
            const wsUrl: string = `ws://${jupyterAddress}`;
            const jupyterBaseUrl: string = JoinPaths(process.env.PUBLIC_PATH || '/', 'jupyter');

            const kernelSpecManagerOptions: KernelManager.IOptions = {
                serverSettings: ServerConnection.makeSettings({
                    token: '',
                    appendToken: false,
                    baseUrl: jupyterBaseUrl,
                    wsUrl: wsUrl,
                    fetch: fetch,
                }),
            };
            kernelManager.current = new KernelManager(kernelSpecManagerOptions);

            console.log('Waiting for Kernel Manager to be ready.');

            kernelManager.current.connectionFailure.connect((_sender: KernelManager, err: Error) => {
                console.error(
                    'An error has occurred while preparing the Kernel Manager. ' + err.name + ': ' + err.message,
                );

                toast.error(`An error has occurred while preparing the Kernel Manager. ${err.name}: ${err.message}.`);
            });

            await kernelManager.current.ready.then(() => {
                console.log(`Kernel Manager is ready: ${kernelManager.current?.isReady}`);
            });
        }

        if (sessionManager.current === null) {
            const wsUrl: string = `ws://${jupyterAddress}`;
            const jupyterBaseUrl: string = JoinPaths(process.env.PUBLIC_PATH || '/', 'jupyter');

            sessionManager.current = new SessionManager({
                kernelManager: kernelManager.current,
                serverSettings: ServerConnection.makeSettings({
                    token: '',
                    appendToken: false,
                    baseUrl: jupyterBaseUrl,
                    wsUrl: wsUrl,
                    fetch: fetch,
                }),
            });

            await sessionManager.current.ready.then(() => {
                console.log(`Session Manager is ready: ${sessionManager.current?.isReady}`);
            });
        }
    }, [jupyterAddress]);

    useEffect(() => {
        if (jupyterAddress === undefined) {
            return;
        }

        initializeKernelManagers().then(() => {});
    }, [initializeKernelManagers, jupyterAddress]);

    const kernelManagerIsInitializing: boolean = kernelManager.current === null || !kernelManager.current.isReady;
    const sessionManagerIsInitializing: boolean = sessionManager.current === null || !sessionManager.current.isReady;

    return {
        kernelManager: kernelManager.current,
        sessionManager: sessionManager.current,
        kernelManagerIsInitializing: kernelManagerIsInitializing,
        sessionManagerIsInitializing: sessionManagerIsInitializing,
    };
}
