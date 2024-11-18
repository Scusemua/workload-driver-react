import {
    ConfirmationModal,
    CreateKernelsModal,
    ExecuteCodeOnKernelModal,
    InformationModal,
    PingKernelModal,
    RoundToNDecimalPlaces,
    RoundToThreeDecimalPlaces,
} from '@Components/Modals';
import { RequestTraceSplitTable } from '@Components/Tables';
import { DistributedJupyterKernel, JupyterKernelReplica, ResourceSpec } from '@Data/Kernel';
import { PongResponse } from '@Data/Message';

import { KernelManager, ServerConnection, SessionManager } from '@jupyterlab/services';
import { IKernelConnection } from '@jupyterlab/services/lib/kernel/kernel';
import { IModel as ISessionModel, ISessionConnection } from '@jupyterlab/services/lib/session/session';
import {
    Alert,
    AlertActionCloseButton,
    Button,
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Flex,
    FlexItem,
    InputGroup,
    InputGroupItem,
    PerPageOptions,
    SearchInput,
    Text,
    TextVariants,
    Title,
    ToolbarGroup,
    ToolbarItem,
    ToolbarToggleGroup,
    Tooltip,
} from '@patternfly/react-core';

import { FilterIcon, PlusIcon, SpinnerIcon, SyncIcon, TrashIcon } from '@patternfly/react-icons';
import { ExecutionOutputTabsDataProvider } from '@Providers/ExecutionOutputTabsDataProvider';
import { useJupyterAddress } from '@Providers/JupyterAddressProvider';
import { useKernels } from '@Providers/KernelProvider';
import { KernelDataList } from '@src/Components';
import { useNodes } from '@src/Providers';
import { GetPathForFetch, JoinPaths } from '@src/Utils/path_utils';
import { DefaultDismiss, GetToastContentWithHeaderAndBody, ToastPromise, ToastRefresh } from '@src/Utils/toast_utils';
import { numberArrayFromRange } from '@src/Utils/utils';
import React, { useCallback, useEffect, useReducer, useRef } from 'react';

import toast, { Toast } from 'react-hot-toast';

export interface KernelListProps {
    openMigrationModal: (kernel: DistributedJupyterKernel, replica: JupyterKernelReplica) => void;
    kernelsPerPage: number;
    perPageOption: PerPageOptions[];
}

export const KernelListCard: React.FunctionComponent<KernelListProps> = (props: KernelListProps) => {
    const [searchValue, setSearchValue] = React.useState('');
    const [statusSelections] = React.useState<string[]>([]);
    const [isConfirmCreateModalOpen, setIsConfirmCreateModalOpen] = React.useState(false);
    const [isConfirmDeleteKernelsModalOpen, setIsConfirmDeleteKernelsModalOpen] = React.useState(false);
    const [isConfirmDeleteKernelModalOpen, setIsConfirmDeleteKernelModalOpen] = React.useState(false);
    const [isErrorModalOpen, setIsErrorModalOpen] = React.useState(false);
    const [isPingKernelModalOpen, setIsPingKernelModalOpen] = React.useState(false);
    const [targetIdPingKernel, setTargetIdPingKernel] = React.useState<string>('');
    const [errorMessage, setErrorMessage] = React.useState('');
    const [errorMessagePreamble, setErrorMessagePreamble] = React.useState('');
    const [isExecuteCodeModalOpen, setIsExecuteCodeModalOpen] = React.useState(false);
    const [executeCodeKernel, setExecuteCodeKernel] = React.useState<DistributedJupyterKernel | null>(null);
    const [executeCodeKernelReplica, setExecuteCodeKernelReplica] = React.useState<JupyterKernelReplica | null>(null);
    const [selectedKernels, setSelectedKernels] = React.useState<string[]>([]);
    const [kernelToDelete, setKernelToDelete] = React.useState<string>('');
    const { kernels, kernelsAreLoading, refreshKernels } = useKernels(false);
    const { refreshNodes } = useNodes();
    const { jupyterAddress } = useJupyterAddress();

    const [, forceUpdate] = useReducer((x) => x + 1, 0);

    const kernelIdSet = useRef<Set<string>>(new Set()); // Keep track of kernels we've seen before.
    const numKernelsCreating = useRef(0); // Used to display "pending" entries in the kernel list.
    const kernelManager = useRef<KernelManager | null>(null);
    const sessionManager = useRef<SessionManager | null>(null);

    // If there are any new kernels, decrement `numKernelsCreating`.
    React.useEffect(() => {
        kernels.forEach((kernel: DistributedJupyterKernel) => {
            if (kernel === null || kernel === undefined) {
                return;
            }

            if (!kernelIdSet.current.has(kernel.kernelId)) {
                kernelIdSet.current.add(kernel.kernelId);
                numKernelsCreating.current -= 1;

                if (numKernelsCreating.current < 0) {
                    // TODO: Need to keep track of how many kernels we're actually waiting on.
                    // If we're not waiting on any kernels, then we shouldn't try to decrement 'numKernelsCreating'.
                    console.warn(`Tried to decrement 'numKernelsCreating' below 0 (kernelID: ${kernel.kernelId})...`);
                    numKernelsCreating.current = 0;
                }
            }
        });
    }, [kernels]);

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
                console.log('Kernel Manager is ready!');
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
                console.log('Session Manager is ready!');
            });
        }
    }, [jupyterAddress]);

    useEffect(() => {
        if (jupyterAddress === undefined) {
            return;
        }

        initializeKernelManagers().then(() => {});
    }, [initializeKernelManagers, jupyterAddress]);

    const onSearchChange = (value: string) => {
        setSearchValue(value);
    };

    const onCancelCreateKernelClicked = () => {
        setIsConfirmCreateModalOpen(false);
    };

    const onCancelDeleteKernelClicked = () => {
        setIsConfirmDeleteKernelModalOpen(false);
    };

    const onCancelDeleteKernelsClicked = () => {
        setIsConfirmDeleteKernelsModalOpen(false);
    };

    const onCancelExecuteCodeClicked = () => {
        setIsExecuteCodeModalOpen(false);
        setExecuteCodeKernel(null);
        setExecuteCodeKernelReplica(null);
    };

    const onCancelPingKernelClicked = () => {
        setIsPingKernelModalOpen(false);
        setTargetIdPingKernel('');
    };

    const onConfirmPingKernelClicked = (kernelId: string, socketType: 'control' | 'shell') => {
        setIsPingKernelModalOpen(false);
        setTargetIdPingKernel('');
        console.log('User is pinging kernel ' + kernelId);

        const req: RequestInit = {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                Authorization: 'Bearer ' + localStorage.getItem('token'),
            },
            body: JSON.stringify({
                socketType: socketType,
                kernelId: kernelId,
            }),
        };

        const toastId: string = toast.custom(
            (t) => {
                return (
                    <Alert
                        title={<b>Pinging kernel {kernelId} now...</b>}
                        variant={'custom'}
                        customIcon={<SpinnerIcon className={'loading-icon-spin-pulse'} />}
                        timeout={false}
                        actionClose={<AlertActionCloseButton onClose={() => toast.dismiss(t.id)} />}
                    />
                );
            },
            {
                style: {
                    maxWidth: 750,
                },
                icon: <SpinnerIcon className={'loading-icon-spin-pulse'} />,
            },
        );

        const startTime: number = performance.now();
        const initialRequestTimestamp: number = Date.now();
        fetch(GetPathForFetch('api/ping-kernel'), req)
            .catch((err: Error) => {
                toast.custom(
                    () =>
                        GetToastContentWithHeaderAndBody(
                            `Failed to ping one or more replicas of kernel ${kernelId}.`,
                            err.message,
                            'danger',
                            () => {
                                toast.dismiss(toastId);
                            },
                        ),
                    { id: toastId, style: { maxWidth: 750 } },
                );
            })
            .then(async (resp: Response | void) => {
                if (!resp) {
                    console.error('No response from ping-kernel.');
                    return;
                }

                if (resp.status != 200 || !resp.ok) {
                    const response = await resp.json();
                    toast.custom(
                        () =>
                            GetToastContentWithHeaderAndBody(
                                `Failed to ping one or more replicas of kernel ${kernelId}.`,
                                `${JSON.stringify(response)}`,
                                'danger',
                                () => {
                                    toast.dismiss(toastId);
                                },
                            ),
                        { id: toastId, style: { maxWidth: 750 } },
                    );
                } else {
                    const response: PongResponse = await resp.json();
                    const receivedReplyAt: number = Date.now();
                    const latencyMilliseconds: number = RoundToNDecimalPlaces(performance.now() - startTime, 6);

                    console.log('All Request Traces:');
                    console.log(JSON.stringify(response.requestTraces, null, 2));

                    toast.custom(
                        <Alert
                            isExpandable
                            variant={'success'}
                            title={`Pinged kernel ${response.id} via its ${socketType} channel (${latencyMilliseconds} ms)`}
                            timeoutAnimation={30000}
                            timeout={15000}
                            onTimeout={() => toast.dismiss(toastId)}
                            actionClose={<AlertActionCloseButton onClose={() => toast.dismiss(toastId)} />}
                        >
                            {response.requestTraces.length > 0 && (
                                <Flex direction={{ default: 'column' }}>
                                    <FlexItem>
                                        <Title headingLevel={'h3'}>Request Trace(s)</Title>
                                    </FlexItem>
                                    <FlexItem>
                                        <RequestTraceSplitTable
                                            receivedReplyAt={receivedReplyAt}
                                            initialRequestSentAt={initialRequestTimestamp}
                                            messageId={response.msg}
                                            traces={response.requestTraces}
                                        />
                                    </FlexItem>
                                </Flex>
                            )}
                        </Alert>,
                        { id: toastId },
                    );
                }
            });
    };

    const onExecuteCodeClicked = (kernel: DistributedJupyterKernel | null, replicaIdx?: number | undefined) => {
        if (kernel == null) {
            return;
        }

        // If we clicked the 'Execute' button associated with a specific replica, then set the state for that replica.
        if (replicaIdx !== undefined) {
            // Need to use "!== undefined" because a `replicaIdx` of 0 will be coerced to false if by itself.
            console.log(
                'Will be executing code on replica %d of kernel %s.',
                kernel.replicas[replicaIdx].replicaId,
                kernel.kernelId,
            );
            setExecuteCodeKernelReplica(kernel.replicas[replicaIdx]);
        } else {
            setExecuteCodeKernelReplica(null);
        }

        setExecuteCodeKernel(kernel);
        setIsExecuteCodeModalOpen(true);
    };

    function onPingKernelClicked(kernel: DistributedJupyterKernel) {
        setIsPingKernelModalOpen(true);
        setTargetIdPingKernel(kernel.kernelId);
    }

    const onInterruptKernelClicked = (kernel: DistributedJupyterKernel) => {
        async function interrupt_kernel(kernelId: string) {
            if (!kernelManager.current || !kernelManager.current.isReady) {
                console.error(
                    `KernelManager is NOT ready... will try to initialize the KernelManager before proceeding.`,
                );
                await initializeKernelManagers();

                if (!kernelManager.current || !kernelManager.current.isReady) {
                    toast.error('Cannot establish connection with Jupyter Server.');

                    return;
                }
            }

            console.log(`Connecting to kernel ${kernelId} (so we can interrupt it) now...`);

            const kernelConnection: IKernelConnection = kernelManager.current!.connectTo({
                model: { id: kernelId, name: kernelId },
            });

            console.log(`Connected to kernel ${kernelId}. Attempting to interrupt kernel now...`);

            await kernelConnection.interrupt();

            console.log(`Interrupted kernel ${kernelId}.`);
        }

        const kernelId: string | undefined = kernel.kernelId;
        interrupt_kernel(kernelId).then(() => {});
    };

    const onStopTrainingClicked = (kernel: DistributedJupyterKernel) => {
        const kernelId: string | undefined = kernel.kernelId;

        if (kernelId === undefined) {
            console.error('Undefined kernel specified for interrupt target...');
            return;
        }

        console.log('User is interrupting kernel ' + kernelId);

        const req: RequestInit = {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                Authorization: 'Bearer ' + localStorage.getItem('token'),
                // 'Cache-Control': 'no-cache, no-transform, no-store',
            },
            body: JSON.stringify({
                session_id: '',
                kernel_id: kernelId,
            }),
        };

        toast
            .promise(fetch(GetPathForFetch('api/stop-training'), req), {
                loading: <b>Interrupting kernel {kernelId} now...</b>,
                success: (resp: Response) => {
                    if (!resp.ok || resp.status != 200) {
                        console.error(`Failed to interrupt kernel ${kernelId}.`);
                        throw new Error(`HTTP ${resp.status}: ${resp.statusText}`);
                    }
                    console.log(`Successfully interrupted kernel ${kernelId}.`);
                    return (
                        <b>
                            Successfully interrupted kernel {kernelId} (HTTP {resp.status}: {resp.statusText}).
                        </b>
                    );
                },
                error: (reason: Error) =>
                    GetToastContentWithHeaderAndBody(
                        `Failed to interrupt kernel ${kernelId}.`,
                        `<b>Reason:</b> ${reason.message}`,
                        'danger',
                        () => {},
                    ),
            })
            .then(() => {});
    };

    async function startKernel(kernelId: string, sessionId: string, resourceSpec: ResourceSpec) {
        numKernelsCreating.current = numKernelsCreating.current + 1;

        const startTime: DOMHighResTimeStamp = performance.now();

        console.log(
            `Starting kernel ${kernelId} (sessionId=${sessionId}) now. ResourceSpec: ${JSON.stringify(resourceSpec)}`,
        );

        console.log(`Starting new 'distributed' kernel for user ${sessionId} with clientID=${sessionId}.`);
        console.log(`Creating new Jupyter Session ${sessionId} now...`);

        if (!sessionManager.current || !sessionManager.current.isReady) {
            console.error(
                `SessionManager is NOT ready... will try to initialize the SessionManager before proceeding.`,
            );
            await initializeKernelManagers();
            console.warn(`Trying again...`);

            if (!sessionManager.current || !sessionManager.current.isReady) {
                toast.error('Cannot establish connection with Jupyter Server.');
                numKernelsCreating.current -= 1;
                return;
            }
        }

        console.log(`sessionManager.current.isReady: ${sessionManager.current.isReady}`);

        const req: RequestInit = {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                // 'Cache-Control': 'no-cache, no-transform, no-store',
            },
            body: JSON.stringify({
                id: sessionId,
                kernel: {
                    name: 'distributed',
                    id: kernelId,
                },
                name: sessionId,
                path: sessionId,
                type: 'notebook',
                resource_spec: resourceSpec,
            }),
        };

        async function start_session(): Promise<ISessionModel> {
            const response: Response = await fetch(GetPathForFetch('jupyter/api/sessions'), req);

            if (response.status != 201) {
                numKernelsCreating.current -= 1;
                const responseText: string = await response.text();
                let err: Error | null;
                try {
                    const responseJson = JSON.parse(responseText);
                    console.error(
                        `Failed to create new Session. Received (${response.status} ${response.statusText}): ${responseJson.message}`,
                    );
                    err = {
                        name: `${response.status} ${response.statusText}`,
                        message: `${response.status} ${response.statusText}: ${responseJson.message}`,
                        stack: new Error().stack,
                    };
                } catch (e) {
                    console.log(e);
                    console.error(
                        `Failed to create new Session. Received (${response.status} ${response.statusText}): ${responseText}`,
                    );
                    err = {
                        name: `${response.status} ${response.statusText}`,
                        message: `${response.status} ${response.statusText}: ${responseText}`,
                        stack: new Error().stack,
                    };
                }

                throw err;
            }

            return await response.json();
        }

        const sessionModel: ISessionModel | null = await ToastPromise<ISessionModel>(
            start_session,
            (t: Toast) =>
                GetToastContentWithHeaderAndBody(
                    'Creating New Jupyter Kernel',
                    undefined,
                    'info',
                    DefaultDismiss(t.id),
                    false,
                    <SpinnerIcon className={'loading-icon-spin-pulse'} />,
                ),
            (t: Toast, _: ISessionModel, latencyMilliseconds: number) => {
                const latencySeconds: number = RoundToThreeDecimalPlaces(latencyMilliseconds / 1000.0);
                return GetToastContentWithHeaderAndBody(
                    'Successfully Created New Jupyter Kernel',
                    `Successfully created and launched new Jupyter kernel in ${latencySeconds} seconds.`,
                    'success',
                    DefaultDismiss(t.id),
                    8500,
                );
            },
            (t: Toast, e: Error) =>
                GetToastContentWithHeaderAndBody(
                    'Failed to start new Jupyter Session and Jupyter Kernel.',
                    e.message,
                    'danger',
                    DefaultDismiss(t.id),
                    15000,
                ),
        );

        // const sessionModel: ISessionModel = await toast.promise(
        //     start_session(),
        //     {
        //         loading: <b>Creating new Jupyter kernel now...</b>,
        //         success: () => {
        //             return (
        //                 <b>{`Successfully launched new Jupyter kernel in ${RoundToThreeDecimalPlaces((performance.now() - startTime) / 1000.0)} seconds.`}</b>
        //             );
        //         },
        //         error: (reason: Error) =>
        //             GetToastContentWithHeaderAndBody(
        //                 'Failed to start new Jupyter Session and Jupyter Kernel.',
        //                 reason.message,
        //                 'danger',
        //                 () => {},
        //             ),
        //     },
        //     { style: { maxWidth: 650 } },
        // );

        if (!sessionModel) {
            return;
        }

        await refreshNodes(false);

        const session: ISessionConnection = sessionManager.current.connectTo({
            model: sessionModel,
            kernelConnectionOptions: {
                handleComms: true,
            },
            username: sessionId,
            clientId: sessionId,
        });

        if (session === null) {
            console.error(`Failed to connect to Jupyter session ${sessionId}.`);
            toast.error(`Failed to connect to Jupyter session ${sessionId}.`);
            return;
        }

        console.log(
            `Successfully created new Jupyter Session. ClientID=${sessionId}, SessionID=${session.id}, SessionName=${session.name},
            SessionKernelClientID=${session.kernel?.clientId}, SessionKernelName=${session.kernel?.name}, SessionKernelID=${session.kernel?.id}.`,
        );

        if (session.kernel === null) {
            toast.custom((t: Toast) =>
                GetToastContentWithHeaderAndBody(
                    `Kernel for newly-created Session ${session.id} is null...`,
                    null,
                    'danger',
                    DefaultDismiss(t.id),
                ),
            );
            return;
        }
        const kernel: IKernelConnection = session.kernel!;

        const timeElapsedMilliseconds: number = performance.now() - startTime;
        const timeElapsedSecRounded: number = RoundToThreeDecimalPlaces(timeElapsedMilliseconds / 1000.0);
        console.log(`Successfully launched kernel ${kernel.id} in ${timeElapsedSecRounded} seconds.`);

        // Register a callback for when the kernel changes state.
        kernel.statusChanged.connect((_, status) => {
            console.log(`New Kernel Status Update: ${status}`);
        });

        await fetch(GetPathForFetch('api/metrics'), {
            method: 'PATCH',
            headers: {
                'Content-Type': 'application/json',
                Authorization: 'Bearer ' + localStorage.getItem('token'),
                // 'Cache-Control': 'no-cache, no-transform, no-store',
            },
            body: JSON.stringify({
                name: 'distributed_cluster_jupyter_session_creation_latency_seconds',
                value: timeElapsedMilliseconds,
                metadata: {
                    kernel_id: kernel.id,
                },
            }),
        });

        await refreshKernels().catch((err: Error) => console.log(`Kernel refresh failed: ${err}`));
    }

    const onConfirmDeleteKernelsClicked = async (kernelIds: string[]) => {
        // Close the confirmation dialogue.
        setIsConfirmDeleteKernelsModalOpen(false);
        setIsConfirmDeleteKernelModalOpen(false);

        // Create a new kernel.
        if (!kernelManager.current) {
            console.error('Kernel Manager is not available. Will try to connect...');
            initializeKernelManagers().then(() => {});
            return;
        }

        /**
         * Delete the specified kernel.
         *
         * @param kernelId The ID of the kernel to be deleted.
         * @param toastId ID of associated Toast notification
         */
        async function delete_kernel(kernelId: string, toastId?: string) {
            console.log('Deleting Kernel ' + kernelId + ' now.');
            const startTime: number = performance.now();

            const req: RequestInit = {
                method: 'DELETE',
            };

            let resp: Response;
            try {
                resp = await fetch(`jupyter/api/kernels/${kernelId}`, req);
            } catch (err) {
                toast.custom(
                    GetToastContentWithHeaderAndBody(
                        `Failed to Delete Kernel ${kernelId}`,
                        [`Error: ${err}`],
                        'danger',
                        DefaultDismiss,
                    ),
                    { id: toastId },
                );
                return;
            }

            if (resp.ok && resp.status == 204) {
                console.log(`Successfully deleted kernel ${kernelId}`);
                toast.custom(
                    GetToastContentWithHeaderAndBody(
                        `Successfully Deleted Kernel ${kernelId}`,
                        null,
                        'danger',
                        DefaultDismiss,
                    ),
                    { id: toastId },
                );
            } else {
                console.error(
                    `Received HTTP ${resp.status} ${resp.statusText} when trying to delete kernel ${kernelId}.`,
                );

                const respText: string = await resp.text();

                toast.custom(
                    GetToastContentWithHeaderAndBody(
                        'Failed to Delete Kernel',
                        [`Failed to delete kernel ${kernelId}`, `HTTP ${resp.status} ${resp.statusText}: ${respText}`],
                        'danger',
                        DefaultDismiss,
                    ),
                    { id: toastId },
                );

                return;
            }

            await fetch(GetPathForFetch('api/metrics'), {
                method: 'PATCH',
                headers: {
                    'Content-Type': 'application/json',
                    Authorization: 'Bearer ' + localStorage.getItem('token'),
                    // 'Cache-Control': 'no-cache, no-transform, no-store',
                },
                body: JSON.stringify({
                    name: 'distributed_cluster_jupyter_session_termination_latency_seconds',
                    value: performance.now() - startTime,
                    metadata: {
                        kernel_id: kernelId,
                    },
                }),
            });
        }

        for (let i: number = 0; i < kernelIds.length; i++) {
            const kernelId: string = kernelIds[i];
            const toastId: string = toast.custom(
                GetToastContentWithHeaderAndBody(
                    'Deleting Kernel',
                    `Deleting kernel ${kernelId}`,
                    'info',
                    DefaultDismiss,
                ),
            );
            await delete_kernel(kernelId, toastId);
        }

        setSelectedKernels([]);
        setKernelToDelete('');
    };

    const onSelectKernel = (kernelId: string) => {
        const item = kernelId as string;

        if (selectedKernels.includes(item)) {
            setSelectedKernels(selectedKernels.filter((id) => id !== item));
        } else {
            setSelectedKernels([...selectedKernels, item]);
        }
    };

    const onConfirmCreateKernelClicked = (
        numKernelsToCreate: number,
        kernelIds: string[],
        sessionIds: string[],
        resourceSpecs: ResourceSpec[],
    ) => {
        console.log(`Creating ${numKernelsToCreate} new Kernel(s).`);

        // Close the confirmation dialogue.
        setIsConfirmCreateModalOpen(false);

        // Create a new kernel.
        if (!kernelManager.current) {
            console.error('Kernel Manager is not available. Will try to connect...');
            initializeKernelManagers().then(() => {});
            return;
        } else if (!kernelManager.current.isReady) {
            console.warn("Kernel Manager isn't ready yet!");
            toast.error("Kernel Manager isn't ready yet.");
            return;
        }

        if (!sessionManager.current) {
            console.error('Session Manager is not available. Will try to connect...');
            initializeKernelManagers().then(() => {});
            return;
        } else if (!sessionManager.current.isReady) {
            console.warn("Session Manager isn't ready yet!");
            toast.error("Cannot create kernel: Session Manager isn't ready yet. Please try again in a few seconds.", {
                style: { maxWidth: 750 },
            });
            return;
        }

        console.log("We're now creating %d kernel(s).", numKernelsToCreate);
        forceUpdate();

        let errorOccurred = false;
        for (let i = 0; i < numKernelsToCreate; i++) {
            if (errorOccurred) break;

            console.log(
                `Creating kernel ${i + 1} / ${numKernelsToCreate} now. KernelID: ${kernelIds[i]}, SessionID: ${
                    sessionIds[i]
                }, ResourceSpec: ${JSON.stringify(resourceSpecs[i])}`,
            );

            // Create a new kernel.
            startKernel(kernelIds[i], sessionIds[i], resourceSpecs[i]).catch((error) => {
                console.error('Error while trying to start a new kernel:\n' + JSON.stringify(error));
                setErrorMessagePreamble(`An error occurred while trying to start a new kernel: ${error.name}`);
                setErrorMessage(`${error.message}`);
                setIsErrorModalOpen(true);
                errorOccurred = true;
            });
        }
    };

    // Set up status single select
    const [isStatusMenuOpen, setIsStatusMenuOpen] = React.useState<boolean>(false);
    const statusToggleRef = React.useRef<HTMLButtonElement>(null);
    const statusMenuRef = React.useRef<HTMLDivElement>(null);
    // const statusContainerRef = React.useRef<HTMLDivElement>(null);

    useEffect(() => {
        const handleStatusClickOutside = (event: MouseEvent) => {
            if (isStatusMenuOpen && !statusMenuRef.current?.contains(event.target as Node)) {
                setIsStatusMenuOpen(false);
            }
        };

        const handleStatusMenuKeys = (event: KeyboardEvent) => {
            if (isStatusMenuOpen && statusMenuRef.current?.contains(event.target as Node)) {
                if (event.key === 'Escape' || event.key === 'Tab') {
                    setIsStatusMenuOpen(!isStatusMenuOpen);
                    statusToggleRef.current?.focus();
                }
            }
        };

        window.addEventListener('keydown', handleStatusMenuKeys);
        window.addEventListener('click', handleStatusClickOutside);
        return () => {
            window.removeEventListener('keydown', handleStatusMenuKeys);
            window.removeEventListener('click', handleStatusClickOutside);
        };
    }, [isStatusMenuOpen, statusMenuRef]);

    const cardHeaderActions = (
        <React.Fragment>
            <ToolbarToggleGroup className="kernel-list-card-actions" toggleIcon={<FilterIcon />} breakpoint="md">
                <Flex
                    alignSelf={{ default: 'alignSelfFlexEnd' }}
                    alignItems={{ default: 'alignItemsFlexEnd' }}
                    spaceItems={{ default: 'spaceItemsSm' }}
                >
                    <ToolbarItem>
                        <InputGroup>
                            <InputGroupItem isFill>
                                <SearchInput
                                    placeholder="Filter by kernel name"
                                    value={searchValue}
                                    onChange={(_event, value) => onSearchChange(value)}
                                    onClear={() => onSearchChange('')}
                                />
                            </InputGroupItem>
                        </InputGroup>
                    </ToolbarItem>
                </Flex>
            </ToolbarToggleGroup>
            <ToolbarGroup variant="icon-button-group">
                <ToolbarItem>
                    <Tooltip exitDelay={75} content={<div>Create a new kernel.</div>}>
                        <Button
                            label="create-kernels-button"
                            aria-label="create-kernels-button"
                            id="create-kernel-button"
                            variant="plain"
                            onClick={() => setIsConfirmCreateModalOpen(!isConfirmCreateModalOpen)}
                        >
                            <PlusIcon />
                        </Button>
                    </Tooltip>
                    <Tooltip exitDelay={75} content={<div>Terminate selected kernels.</div>}>
                        <Button
                            label="delete-kernels-button"
                            aria-label="delete-kernels-button"
                            id="delete-kernels-button"
                            variant="plain"
                            isDanger
                            isDisabled={kernels.length == 0 || selectedKernels.length == 0}
                            onClick={() => setIsConfirmDeleteKernelsModalOpen(true)}
                        >
                            <TrashIcon />
                        </Button>
                    </Tooltip>
                    <Tooltip exitDelay={75} content={<div>Refresh kernels.</div>}>
                        <Button
                            label="refresh-kernels-button"
                            aria-label="refresh-kernels-button"
                            id="refresh-kernels-button"
                            variant="plain"
                            isDisabled={kernelsAreLoading}
                            className={
                                (kernelsAreLoading && 'loading-icon-spin-toggleable') ||
                                'loading-icon-spin-toggleable paused'
                            }
                            onClick={() => {
                                ToastRefresh(
                                    refreshKernels,
                                    'Refreshing kernels...',
                                    'Failed to refresh kernels',
                                    'Refreshed kernels',
                                );
                            }}
                        >
                            <SyncIcon />
                        </Button>
                    </Tooltip>
                </ToolbarItem>
            </ToolbarGroup>
        </React.Fragment>
    );

    const onTerminateKernelClicked = (kernel: DistributedJupyterKernel | null) => {
        if (kernel == null) {
            return;
        }

        // We're trying to delete a specific kernel.
        setKernelToDelete(kernel.kernelId);
        setIsConfirmDeleteKernelModalOpen(true);
    };

    const pendingKernelArr = numberArrayFromRange(0, numKernelsCreating.current);

    return (
        <Card isRounded isFullHeight id="kernel-list-card">
            <CardHeader actions={{ actions: cardHeaderActions, hasNoOffset: false }}>
                <CardTitle>
                    <Title headingLevel="h1" size="xl">
                        Active Kernels
                    </Title>
                </CardTitle>
            </CardHeader>
            <CardBody>
                <KernelDataList
                    openMigrationModal={props.openMigrationModal}
                    kernelsPerPage={props.kernelsPerPage}
                    perPageOption={props.perPageOption}
                    searchValue={searchValue}
                    statusSelections={statusSelections}
                    onExecuteCodeClicked={onExecuteCodeClicked}
                    onPingKernelClicked={onPingKernelClicked}
                    onInterruptKernelClicked={onInterruptKernelClicked}
                    onTerminateKernelClicked={onTerminateKernelClicked}
                    onStopTrainingClicked={onStopTrainingClicked}
                    onSelectKernel={onSelectKernel}
                    selectedKernels={selectedKernels}
                />
                {kernels.length == 0 && pendingKernelArr.length == 0 && (
                    <Text component={TextVariants.h2}>There are no active kernels.</Text>
                )}
                <CreateKernelsModal
                    isOpen={isConfirmCreateModalOpen}
                    onConfirm={onConfirmCreateKernelClicked}
                    onClose={onCancelCreateKernelClicked}
                />
                <ConfirmationModal
                    isOpen={isConfirmDeleteKernelsModalOpen}
                    onConfirm={() => onConfirmDeleteKernelsClicked(selectedKernels)}
                    onClose={onCancelDeleteKernelsClicked}
                    title={'Terminate Selected Kernels'}
                    message={"Are you sure you'd like to delete the specified kernel(s)?"}
                />
                <ConfirmationModal
                    isOpen={isConfirmDeleteKernelModalOpen}
                    onConfirm={() => onConfirmDeleteKernelsClicked([kernelToDelete])}
                    onClose={onCancelDeleteKernelClicked}
                    title={'Terminate Kernel'}
                    message={"Are you sure you'd like to delete the specified kernel?"}
                />
                <ExecutionOutputTabsDataProvider>
                    <ExecuteCodeOnKernelModal
                        kernel={executeCodeKernel}
                        replicaId={executeCodeKernelReplica?.replicaId}
                        isOpen={isExecuteCodeModalOpen}
                        onClose={onCancelExecuteCodeClicked}
                    />
                </ExecutionOutputTabsDataProvider>
                <InformationModal
                    isOpen={isErrorModalOpen}
                    onClose={() => {
                        setIsErrorModalOpen(false);
                        setErrorMessage('');
                        setErrorMessagePreamble('');
                    }}
                    title="An Error has Occurred"
                    titleIconVariant="danger"
                    message1={errorMessagePreamble}
                    message2={errorMessage}
                />
                <PingKernelModal
                    isOpen={isPingKernelModalOpen}
                    onClose={onCancelPingKernelClicked}
                    onConfirm={onConfirmPingKernelClicked}
                    kernelId={targetIdPingKernel}
                />
            </CardBody>
        </Card>
    );
};
