import { CodeEditorComponent } from '@Components/CodeEditor';
import { ExecutionOutputTabContent } from '@Components/Modals/ExecuteCodeOnKernelModal/ExecutionOutputTabContent';
import { RoundToNDecimalPlaces } from '@Components/Modals/NewWorkloadFromTemplateModal';
import { KernelManager, ServerConnection } from '@jupyterlab/services';
import { IKernelConnection, IShellFuture } from '@jupyterlab/services/lib/kernel/kernel';
import { IExecuteReplyMsg, IExecuteRequestMsg, IIOPubMessage } from '@jupyterlab/services/lib/kernel/messages';
import { Language } from '@patternfly/react-code-editor';
import {
    Alert,
    AlertActionCloseButton,
    Button,
    Card,
    CardBody,
    Checkbox,
    ClipboardCopy,
    Flex,
    FlexItem,
    FormSelect,
    FormSelectOption,
    Grid,
    GridItem,
    Label,
    Modal,
    Tab,
    TabTitleText,
    Tabs,
    Text,
    TextVariants,
    Title,
    Tooltip,
} from '@patternfly/react-core';
import { CheckCircleIcon, SpinnerIcon, TimesCircleIcon, TimesIcon } from '@patternfly/react-icons';
import { AuthorizationContext } from '@Providers/AuthProvider';
import { useJupyterAddress } from '@Providers/JupyterAddressProvider';
import { RequestTraceSplitTable } from '@src/Components';
import { DistributedJupyterKernel, FirstJupyterKernelBuffersFrame, JupyterKernelReplica } from '@src/Data';
import { GetPathForFetch, JoinPaths } from '@src/Utils/path_utils';
import React, { ReactElement } from 'react';
import toast, { Toast } from 'react-hot-toast';
import { v4 as uuidv4 } from 'uuid';

export interface ExecuteCodeOnKernelProps {
    children?: React.ReactNode;
    kernel: DistributedJupyterKernel | null;
    replicaId?: number;
    isOpen: boolean;
    onClose: () => void;
}

export type CodeContext = {
    code: string;
    setCode: (newCode: string) => void;
};

/* eslint-disable-next-line @typescript-eslint/no-unused-vars */
export const CodeContext = React.createContext({
    code: '',
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    setCode: (_: string) => {},
});

// Execution encapsulates the submission of code to be executed on a kernel.
interface Execution {
    // The ID of the kernel to which the code was submitted for execution.
    kernelId: string;
    // The SMR node ID of the replica targeted, if one was explicitly targeted.
    replicaId: number | undefined;
    // The code that was submitted for execution.
    code: string;
    // Unique identifier for the execution.
    executionId: string;
    // The future returned by the IKernelConnection's requestExecute method.
    future: IShellFuture<IExecuteRequestMsg, IExecuteReplyMsg>;
    // Status of the execution. Is it active? Did it succeed? Or did it fail?
    status: 'running' | 'failed' | 'completed';
    // Output from the execution of the code captured from Jupyter ZMQ IOPub messages.
    output: string[];
    // The name of the error that caused the execution to fail (if the execution did fail).
    errorName: string | undefined;
    // The error message that caused the execution to fail (if the execution did fail).
    errorMessage: string | undefined;
}

export const ExecuteCodeOnKernelModal: React.FunctionComponent<ExecuteCodeOnKernelProps> = (props) => {
    const [code, setCode] = React.useState('');
    const [executionState, setExecutionState] = React.useState('idle');

    const [targetReplicaId, setTargetReplicaId] = React.useState(-1);
    const [forceFailure, setForceFailure] = React.useState(false);
    const [activeExecutionOutputTab, setActiveExecutionOutputTab] = React.useState<string>('');

    const [executionMap, setExecutionMap] = React.useState<Map<string, Execution>>(new Map());
    const [, setClosedExecutionMap] = React.useState<Map<string, boolean>>(new Map());

    const { jupyterAddress } = useJupyterAddress();

    const { authenticated } = React.useContext(AuthorizationContext);

    React.useEffect(() => {
        // Automatically close the modal of we are logged out.
        if (!authenticated) {
            props.onClose();
        }
    }, [props, authenticated]);

    const executionOutputTabComponentRef = React.useRef();

    const onExecutionOutputTabSelect = (executionId: string) => {
        setActiveExecutionOutputTab(executionId);
    };

    const onCloseExecutionOutputTab = (_: React.MouseEvent<HTMLElement, MouseEvent>, executionId: string | number) => {
        const execution: Execution | undefined = executionMap.get(executionId as string);
        if (execution === undefined) {
            console.warn(
                `onCloseExecutionOutputTab called with executionId="${executionId}", but no Execution with that ID found in mapping. Mapping contains ${executionMap.size} execution(s).`,
            );
            return;
        }

        if (execution.status == 'running') {
            console.warn(`Cancelling 'running' execution "${executionId}" as its tab is being closed.`);

            try {
                execution.future.dispose();
            } catch (e) {
                console.error(
                    `Exception encountered while cancelling future associated with execution "${executionId}": ${JSON.stringify(e)}`,
                );
            }
        }

        setExecutionMap((prevExecMap) => {
            const nextExecMap = new Map(prevExecMap);
            nextExecMap.delete(executionId as string);
            return nextExecMap;
        });
        setClosedExecutionMap((prevClosedExecutionMap) =>
            new Map(prevClosedExecutionMap).set(executionId as string, true),
        );

        // If we're closing the active tab, attempt to select another tab as the active tab.
        if (activeExecutionOutputTab == executionId) {
            for (const [key] of Array.from(executionMap)) {
                if (key != executionId) {
                    console.log(`Setting active tab to ${key}`);
                    setActiveExecutionOutputTab(key);
                    return;
                }
            }

            // If we get to this point, then there must be no tabs for us to set as the active tab.
            setActiveExecutionOutputTab('');
        }
    };

    React.useEffect(() => {
        // Basically, if we don't have an active tab selected, or if the tab we had selected was closed,
        // and we just added a new tab, then set the active tab to the newly-added tab.
        if (
            executionMap.size >= 1 &&
            (activeExecutionOutputTab === '' || !executionMap.has(activeExecutionOutputTab))
        ) {
            console.log(`Setting active tab to ${executionMap.keys()[0]}`);
            setActiveExecutionOutputTab(executionMap.keys()[0]);
        }
    }, [executionMap, activeExecutionOutputTab]);

    React.useEffect(() => {
        setTargetReplicaId(props.replicaId || -1);
    }, [props.replicaId]);

    const logConsumer = (msg: string, execution_id: string) => {
        // console.log(`Appending message to output log for kernel execution: ${msg}`);
        const messages: string[] = msg.trim().split(/\n/);
        // console.log(`Appending ${messages.length} message(s) to output log for kernel execution: ${messages}`);

        setExecutionMap((prevExecMap) => {
            const exec: Execution | undefined = prevExecMap.get(execution_id);

            // If the user explicitly closed the tab, then we'll just return.
            // If the tab was never explicitly closed, then we're receiving update
            // from the associated execution for the very first time, and so
            // we'll need to add/create an entry in the output map.
            if (exec === undefined) {
                return prevExecMap;
            }

            exec.output = [...exec.output, ...messages];
            return new Map(prevExecMap.set(execution_id, exec));
        });
    };

    /**
     * Extract and return a RequestTrace from the "execute_reply" message.
     * @param response the "execute_reply" message.
     */
    const extractRequestTraceFromResponse = (response: IExecuteReplyMsg): FirstJupyterKernelBuffersFrame | null => {
        const buffers: (ArrayBuffer | ArrayBufferView)[] | undefined = response.buffers;
        if (buffers && buffers.length > 0) {
            console.log('Buffers (from "execute_reply"): have non-zero length.');

            const firstBufferFrame: ArrayBuffer | ArrayBufferView = buffers[0];
            const textDecoder: TextDecoder = new TextDecoder('utf-8');

            let firstBufferFrameAsString: string = '';
            try {
                firstBufferFrameAsString = textDecoder.decode(firstBufferFrame);
            } catch (err) {
                console.error(`Failed to decode (UTF-8) first buffers frame: ${err}`);
                toast.error(`Failed to decode first buffers frame from "execute_reply" message.`);
                return null;
            }

            console.log(`Decoded first buffers frame from "execute_reply" message: ${firstBufferFrameAsString}`);

            try {
                return JSON.parse(firstBufferFrameAsString);
            } catch (err) {
                console.error(
                    `Failed to JSON parse RequestTrace from first buffers frame of "execute_reply" message: ${err}`,
                );
                toast.error(
                    `Failed to JSON parse RequestTrace from first buffers frame of "execute_reply" message: ${err}`,
                );
                return null;
            }
        }

        return null;
    };

    /**
     * Handle an "execute_reply" response to an execution.
     *
     * We update the toast and the tab UI to indicate that the execution has completed,
     * either successfully or with an error.
     *
     * @param response the "execute_reply" response from the kernel.
     * @param executionId the ID of the execution for which we received a response.
     * @param kernelId the ID of the kernel that executed the code for this execution
     * @param latencyMilliseconds the number of milliseconds seconds that elapsed before the execution completed.
     * @param initialRequestTimestamp unix milliseconds (UTC) at which we initially sent the associated "execute_request" message.
     * @param receivedReplyAt unix milliseconds (UTC) at which we received the "execute_reply" message
     * @param toastId the ID of the toast that is being displayed to indicate that the execution is in-progress.
     *
     * @return a boolean indicating whether the execution was successful (true) or if it failed (false).
     */
    const onExecutionResponse = (
        response: IExecuteReplyMsg,
        executionId: string,
        kernelId: string,
        latencyMilliseconds: number,
        initialRequestTimestamp: number,
        receivedReplyAt: number,
        toastId: string,
    ): boolean => {
        console.log(`Received reply for execution ${executionId} future: ${JSON.stringify(response)}`);

        const message_content = response['content'];
        const status: string = message_content['status'];

        // In terms of what we display to the user, if the execution took more than 10 seconds,
        // then we'll convert the units to seconds and display it that way (rounded to 3 decimal places).
        //
        // If the execution took less than 10 seconds (i.e., 1 minute), then we'll display the latency
        // in milliseconds (rounded to 3 decimal places).
        let latencyRounded: number;
        let latencyUnits: string = 'ms'; // Initially, the latency is in milliseconds.
        if (latencyMilliseconds > 10e3) {
            latencyUnits = 'seconds';
            latencyRounded = RoundToNDecimalPlaces(latencyMilliseconds / 1000.0, 3);
        } else {
            latencyRounded = RoundToNDecimalPlaces(latencyMilliseconds, 3);
        }

        // Try to extract a RequestTrace from the first buffers frame of the response.
        const firstBufferFrame: FirstJupyterKernelBuffersFrame | null = extractRequestTraceFromResponse(response);
        if (firstBufferFrame != null) {
            console.log(
                `Extracted RequestTrace from "${response.header.msg_type}" message "${executionId}" from kernel "${kernelId}":\n
                    ${JSON.stringify(firstBufferFrame.request_trace, null, 2)}`,
            );
        }

        if (status == 'ok') {
            setExecutionMap((prevMap) => {
                const exec: Execution | undefined = prevMap.get(executionId);
                if (exec) {
                    exec.status = 'completed';
                    return new Map(prevMap).set(executionId, exec);
                }
                return prevMap;
            });

            console.log(`Execution on Kernel ${kernelId} finished after ${latencyMilliseconds} ms.`);

            toast.custom(
                (t: Toast) => {
                    return (
                        <Alert
                            title={
                                <b>
                                    Execution Complete ✅ ({latencyRounded} {latencyUnits})
                                </b>
                            }
                            variant={'success'}
                            isExpandable
                            timeout={30000}
                            timeoutAnimation={60000}
                            onTimeout={() => toast.dismiss(t.id)}
                            actionClose={<AlertActionCloseButton onClose={() => toast.dismiss(t.id)} />}
                        >
                            {firstBufferFrame !== null && firstBufferFrame.request_trace !== undefined && (
                                <Flex direction={{ default: 'column' }}>
                                    <FlexItem>
                                        <Title headingLevel={'h3'}>Request Trace(s)</Title>
                                    </FlexItem>
                                    <FlexItem>
                                        <RequestTraceSplitTable
                                            receivedReplyAt={receivedReplyAt}
                                            initialRequestSentAt={initialRequestTimestamp}
                                            messageId={response.header.msg_id}
                                            traces={[firstBufferFrame.request_trace]}
                                        />
                                    </FlexItem>
                                </Flex>
                            )}
                            {firstBufferFrame === null ||
                                (firstBufferFrame.request_trace === undefined && (
                                    <p>
                                        Kernel {kernelId} has finished executing your code after {latencyRounded}{' '}
                                        {latencyUnits}
                                    </p>
                                ))}
                        </Alert>
                    );
                },
                {
                    id: toastId,
                    style: { maxWidth: 750 },
                    duration: 5000,
                },
            );

            return true;
        } else {
            const errorName: string = message_content['ename'];
            const errorMessage: string = message_content['evalue'];
            const errorNameAndMessage: string = `${errorName}: ${errorMessage}`;

            setExecutionMap((prevMap) => {
                const exec: Execution | undefined = prevMap.get(executionId);
                if (exec) {
                    exec.status = 'failed';
                    exec.errorName = errorName;
                    exec.errorMessage = errorMessage;
                    return new Map(prevMap).set(executionId, exec);
                }
                return prevMap;
            });

            toast.custom(
                (t) => (
                    <Alert
                        title={<b>{`Execution Failed (${latencyRounded} ${latencyUnits})`}</b>}
                        isExpandable
                        variant={'danger'}
                        timeout={12500}
                        timeoutAnimation={30000}
                        onTimeout={() => toast.dismiss(t.id)}
                        actionClose={<AlertActionCloseButton onClose={() => toast.dismiss(t.id)} />}
                    >
                        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                            <FlexItem>
                                <Text component={TextVariants.p}>
                                    {`Execution on Kernel ${kernelId} failed to complete after ${latencyRounded} ${latencyUnits}.`}
                                </Text>
                            </FlexItem>
                            {/* The error message associated with the failed execution. */}
                            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
                                <FlexItem>
                                    <Title headingLevel={'h3'}>Error Message</Title>
                                </FlexItem>
                                <FlexItem>
                                    <ClipboardCopy isReadOnly hoverTip="Copy" clickTip="Copied">
                                        {errorNameAndMessage}
                                    </ClipboardCopy>
                                </FlexItem>
                            </Flex>
                            {/* We won't display this next part if there's no request trace to display. */}
                            {firstBufferFrame !== null && firstBufferFrame.request_trace !== undefined && (
                                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
                                    <FlexItem>
                                        <Title headingLevel={'h3'}>Request Trace(s)</Title>
                                    </FlexItem>
                                    <FlexItem>
                                        <RequestTraceSplitTable
                                            receivedReplyAt={receivedReplyAt}
                                            initialRequestSentAt={initialRequestTimestamp}
                                            messageId={response.header.msg_id}
                                            traces={[firstBufferFrame.request_trace]}
                                        />
                                    </FlexItem>
                                </Flex>
                            )}
                        </Flex>
                    </Alert>
                ),
                {
                    id: toastId,
                    style: {
                        maxWidth: 750,
                    },
                    duration: 12500,
                },
            );

            return false;
        }
    };

    /**
     * Handle an IO Pub message that we receive while an execution is occurring.
     * @param executionId the ID of the execution associated with the IOPub message.
     * @param msg the IOPub message itself.
     */
    const onExecutionIoPub = (executionId: string, msg: IIOPubMessage) => {
        console.log(`Received IOPub reply for execution ${executionId}: ${JSON.stringify(msg)}`);
        const messageType: string = msg.header.msg_type;
        if (messageType == 'execute_input') {
            // Do nothing.
        } else if (messageType == 'status') {
            logConsumer(
                msg['header']['date'] +
                    ': Execution state changed to ' +
                    JSON.stringify(msg.content['execution_state']) +
                    '\n',
                executionId,
            );
        } else if (messageType == 'stream') {
            if (msg['content']['name'] == 'stderr') {
                logConsumer(msg['header']['date'] + ' <ERROR>: ' + msg.content['text'] + '\n', executionId);
            } else if (msg['content']['name'] == 'stdout') {
                logConsumer(msg['header']['date'] + ': ' + msg.content['text'] + '\n', executionId);
            } else {
                logConsumer(msg['header']['date'] + ': ' + msg.content['text'] + '\n', executionId);
            }
        } else {
            logConsumer(msg['header']['date'] + ': ' + JSON.stringify(msg.content) + '\n', executionId);
        }
    };

    /**
     * Handler for when code is submitted for execution.
     *
     * @param action Indicates whether we're submitting code to an idle kernel or an active kernel. We submit code to
     * an idle kernel. We enqueue code with an active/busy kernel.
     */
    const onSubmit = (action: 'submit' | 'enqueue') => {
        if (!authenticated) {
            return;
        }

        async function runUserCode(): Promise<Execution | undefined> {
            const executionId: string = uuidv4();
            const kernelId: string | undefined = props.kernel?.kernelId;

            if (kernelId == undefined) {
                console.error("Couldn't determiner kernel ID of target kernel for code execution...");
                return undefined;
            }

            const wsUrl: string = `ws://${jupyterAddress}`;
            const jupyterBaseUrl: string = JoinPaths(process.env.PUBLIC_PATH || '/', 'jupyter');

            console.log(`WebSocket URL: ${wsUrl}`);
            const kernelSpecManagerOptions: KernelManager.IOptions = {
                serverSettings: ServerConnection.makeSettings({
                    token: '',
                    appendToken: false,
                    baseUrl: jupyterBaseUrl,
                    wsUrl: wsUrl,
                    fetch: fetch,
                }),
            };
            const kernelManager: KernelManager = new KernelManager(kernelSpecManagerOptions);

            console.log('Waiting for Kernel Manager to be ready.');

            kernelManager.connectionFailure.connect((_sender: KernelManager, err: Error) => {
                console.error(
                    'An error has occurred while preparing the Kernel Manager. ' + err.name + ': ' + err.message,
                );

                toast.error(`An error has occurred while preparing the Kernel Manager. ${err.name}: ${err.message}.`);
            });

            await kernelManager.ready.then(() => {
                console.log('Kernel Manager is ready!');
            });

            if (forceFailure) {
                console.log(
                    `Executing code on kernel ${props.kernel?.kernelId}, but we're forcing a failure:\n${code}`,
                );
                // NOTE: We previously just set the target replica ID to 0, but this doesn't enable us to test a subsequent execution, such as when we're testing migrations in static scheduling.
                // So, we now use a new API that just YIELDs the next request, so that this triggers a migration, and the resubmitted request (after the migration) completes can finish successfully.
                // targetReplicaId = 0; // -1 is used for "auto", while 0 is never used as an actual ID. So, if we specify 0, then the execution will necessarily fail.

                const req: RequestInit = {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        Authorization: 'Bearer ' + localStorage.getItem('token'),
                        // 'Cache-Control': 'no-cache, no-transform, no-store',
                    },
                    body: JSON.stringify({
                        kernel_id: kernelId,
                    }),
                };

                await fetch(GetPathForFetch('api/yield-next-execute-request'), req);
            } else {
                console.log(`Executing code on kernel ${props.kernel?.kernelId}, replica ${targetReplicaId}:\n${code}`);
            }

            const kernelConnection: IKernelConnection = kernelManager.connectTo({
                model: { id: kernelId, name: kernelId },
            });

            kernelConnection.connectionStatusChanged.connect((sender, args) => {
                console.log(
                    `Kernel ${props.kernel?.kernelId} connection status changed. Sender: ${sender}, args: ${args}`,
                );
            });

            kernelConnection.disposed.connect((sender, args) => {
                console.log(
                    `Connection to Kernel ${props.kernel?.kernelId} has been disposed. Sender: ${sender}, args: ${args}`,
                );
            });

            console.log(`Sending 'execute-request' to kernel ${kernelId} for code: '${code}'`);

            const startTime: number = performance.now();
            const initialRequestTimestamp: number = Date.now();
            const future = kernelConnection.requestExecute({ code: code }, undefined, {
                target_replica: targetReplicaId,
                send_timestamp_unix_milli: Date.now(),
            });

            // Handle iopub messages
            future.onIOPub = (msg) => {
                onExecutionIoPub(executionId, msg);
            };

            const execution: Execution = {
                kernelId: kernelId,
                replicaId: props.replicaId,
                code: code,
                future: future,
                executionId: executionId,
                status: 'running',
                output: [],
                errorName: undefined,
                errorMessage: undefined,
            };

            if (activeExecutionOutputTab === '' || !executionMap.has(activeExecutionOutputTab)) {
                console.log(`Setting active tab to ${executionId}`);
                setActiveExecutionOutputTab(executionId);
            }

            setExecutionMap((prevMap) => new Map(prevMap).set(executionId, execution));

            const toastId: string = toast.custom(
                (t: Toast) => {
                    return (
                        <Alert
                            isInline
                            variant={'custom'}
                            title={action == 'submit' ? 'Code Submitted 🚀' : 'Code Enqueued 🚀'}
                            customIcon={<SpinnerIcon className={'loading-icon-spin-pulse'} />}
                            timeoutAnimation={30000}
                            timeout={10000}
                            onTimeout={() => toast.dismiss(t.id)}
                            actionClose={<AlertActionCloseButton onClose={() => toast.dismiss(t.id)} />}
                        >
                            <p>
                                {action == 'submit'
                                    ? `Submitted code for execution to kernel ${kernelId}.`
                                    : `Enqueued code for execution with kernel ${kernelId}.`}
                            </p>
                        </Alert>
                    );
                },
                { style: { maxWidth: 750 }, icon: <SpinnerIcon className={'loading-icon-spin'} /> },
            );

            // For whatever reason, the future returned by the Jupyter API doesn't always resolve?
            // Specifically if we enqueue multiple requests for execution.
            // But onReply is always fired, so we just make our own Promise that resolves once the response message
            // is received. We never reject this promise (even if there's an error), as the error is handled
            // in the future.onReply handler.
            let executionComplete: (value: void | PromiseLike<void>) => void;
            const executionCompletePromise: Promise<void> = new Promise<void>(function (resolve) {
                executionComplete = resolve;
            });

            future.onReply = (response: IExecuteReplyMsg) => {
                const receivedReplyAt: number = Date.now();
                const latencyMilliseconds: number = performance.now() - startTime;

                onExecutionResponse(
                    response,
                    executionId,
                    kernelId,
                    latencyMilliseconds,
                    initialRequestTimestamp,
                    receivedReplyAt,
                    toastId,
                );
                executionComplete();
                future.dispose();
            };

            await executionCompletePromise;

            // await future.done;
            setExecutionState('done');

            await fetch(GetPathForFetch('api/metrics'), {
                method: 'PATCH',
                headers: {
                    'Content-Type': 'application/json',
                    Authorization: 'Bearer ' + localStorage.getItem('token'),
                    // 'Cache-Control': 'no-cache, no-transform, no-store',
                },
                body: JSON.stringify({
                    name: 'distributed_cluster_jupyter_execute_request_e2e_latency_seconds',
                    value: performance.now() - startTime,
                    metadata: {
                        kernel_id: kernelId,
                    },
                }),
            });

            return executionMap.get(executionId);
        }

        runUserCode().then(() => {});
    };

    // Reset state, then call user-supplied onClose function.
    const onClose = () => {
        console.log('Closing execute code modal.');
        setExecutionState('idle');
        props.onClose();
    };

    // Returns the title to use for the Modal depending on whether a specific replica was specified as the target or not.
    const getModalTitle = () => {
        if (props.replicaId) {
            return 'Execute Code on Replica ' + props.replicaId + ' of Kernel ' + props.kernel?.kernelId;
        } else {
            return 'Execute Code on Kernel ' + props.kernel?.kernelId;
        }
    };

    const onTargetReplicaChanged = (_event: React.FormEvent<HTMLSelectElement>, value: string) => {
        const replicaId: number = Number.parseInt(value);
        setTargetReplicaId(replicaId);
        console.log(`Targeting replica ${replicaId}`);
    };

    const getKernelId = (execId: string) => {
        const execution = executionMap.get(execId);
        if (execution) {
            return execution.kernelId;
        }
        return undefined;
    };

    /**
     * Return the kernel ID associated with the given execution, shortened to First_8_Chars_of_Kernel_ID...
     * @param execId the ID of the desired execution
     */
    const getShortenedKernelId = (execId: string) => {
        const execution = executionMap.get(execId);
        if (execution) {
            if (execution.kernelId.length > 20) {
                return execution.kernelId.substring(0, 20) + '...';
            }
            return execution.kernelId;
        }
        return undefined;
    };

    /**
     * Get the replica ID associated with the given execution, if there is a replica ID associated with it.
     * @param execId the ID of the desired execution
     */
    const getReplicaId = (execId: string) => {
        const execution = executionMap.get(execId);
        if (execution) {
            return execution.replicaId;
        }
        return undefined;
    };

    /**
     * Get the output of the active execution
     */
    const getOutputForActiveExecutionTab = () => {
        const exec = executionMap.get(activeExecutionOutputTab);
        if (exec) {
            return exec.output;
        }
        return [];
    };

    /**
     * Return the error message associated with the active execution.
     */
    const getErrorNameAndMessageForActiveExecutionTab = () => {
        const exec = executionMap.get(activeExecutionOutputTab);
        if (exec && exec.errorName && exec.errorMessage) {
            return `${exec.errorName}: ${exec.errorMessage}`;
        }
        return undefined;
    };

    const getExecutionLabel = (exec: Execution) => {
        let color: 'grey' | 'green' | 'red' | 'blue' | 'cyan' | 'orange' | 'purple' | 'gold' | undefined;
        let icon: ReactElement;
        if (exec.status == 'running') {
            color = 'grey';
            icon = <SpinnerIcon className={'loading-icon-spin'} />;
        } else if (exec.status == 'completed') {
            color = 'green';
            icon = <CheckCircleIcon />;
        } else {
            color = 'red';
            icon = <TimesCircleIcon />;
        }

        return (
            <Label color={color} icon={icon}>
                {exec.status}
            </Label>
        );
    };

    const onCloseAllExecutionTabsClicked = () => {
        setActiveExecutionOutputTab('');
        setExecutionMap(new Map()); // Clear this.
    };

    // Note: we're just simulating the tabs here. The tabs don't have any content.
    // We just use the tabs as the UI for selecting which output to view.
    // The tab content is included below the tabs.
    // When I included it explicitly as the tab content, the tab content wouldn't update properly
    // when changing tabs. You'd have to click the "wrap text" button to get it to work.
    const executionOutputArea = (
        <Card isCompact isFlat>
            <CardBody>
                <Tabs
                    hidden={executionMap.size == 0}
                    activeKey={activeExecutionOutputTab}
                    onSelect={(_: React.MouseEvent<HTMLElement, MouseEvent>, eventKey: number | string) => {
                        onExecutionOutputTabSelect(eventKey as string);
                    }}
                    onClose={onCloseExecutionOutputTab}
                    role="region"
                    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                    // @ts-expect-error
                    ref={executionOutputTabComponentRef}
                    aria-label="ExecutionOutput Configuration Tabs"
                >
                    {Array.from(executionMap).map(([execId, exec]) => {
                        return (
                            <Tab
                                id={`execution-output-tab-${execId}`}
                                key={`execution-output-tab-${execId}`}
                                eventKey={execId}
                                aria-label={`${execId} Tab`}
                                title={
                                    <TabTitleText>
                                        <Flex
                                            direction={{ default: 'column' }}
                                            spaceItems={{ default: 'spaceItemsNone' }}
                                        >
                                            <Flex
                                                direction={{ default: 'row' }}
                                                spaceItems={{ default: 'spaceItemsXs' }}
                                            >
                                                <FlexItem align={{ default: 'alignLeft' }}>
                                                    <Text component={'small'}>
                                                        <b>ExecID: </b> {execId.substring(0, 8)}
                                                    </Text>
                                                </FlexItem>
                                                <FlexItem align={{ default: 'alignRight' }}>
                                                    {getExecutionLabel(exec)}
                                                </FlexItem>
                                            </Flex>
                                            <FlexItem
                                                align={{ default: 'alignLeft' }}
                                                alignSelf={{ default: 'alignSelfFlexStart' }}
                                            >
                                                <Text component={'small'}>
                                                    <b>KernelID: </b> {getShortenedKernelId(execId)}
                                                </Text>
                                            </FlexItem>
                                        </Flex>
                                    </TabTitleText>
                                }
                                closeButtonAriaLabel={`Close ${execId} Tab`}
                            />
                        );
                    })}
                </Tabs>
                <ExecutionOutputTabContent
                    output={getOutputForActiveExecutionTab()}
                    executionId={activeExecutionOutputTab}
                    kernelId={getKernelId(activeExecutionOutputTab)}
                    replicaId={getReplicaId(activeExecutionOutputTab)}
                    errorMessage={getErrorNameAndMessageForActiveExecutionTab()}
                />
            </CardBody>
        </Card>
    );

    return (
        <Modal
            // variant={ModalVariant.large}
            width="75%"
            title={getModalTitle()}
            isOpen={props.isOpen}
            onClose={props.onClose}
            actions={[
                <Button
                    key="submit-code-button"
                    variant="primary"
                    onClick={() => {
                        if (executionState == 'idle') {
                            setExecutionState('busy');
                            onSubmit('submit');
                        } else if (executionState == 'busy') {
                            console.log(
                                'Please wait until the current execution completes before submitting additional code for execution.',
                            );
                        } else {
                            setExecutionState('idle');
                        }
                    }}
                    isDisabled={code.trim().length == 0 || !authenticated || jupyterAddress === undefined}
                    isLoading={executionState === 'busy'}
                    icon={executionState === 'done' ? <CheckCircleIcon /> : null}
                    spinnerAriaValueText="Loading..."
                >
                    {executionState === 'idle' && 'Execute'}
                    {executionState === 'busy' && 'Executing code'}
                    {executionState === 'done' && 'Complete'}
                </Button>,
                <Tooltip
                    key="enqueue-button-tooltip"
                    content={
                        'Submit an additional block of code to be executed after the kernel finishes execution its current code submission.'
                    }
                >
                    <Button
                        key="enqueue-code-button"
                        variant={'primary'}
                        onClick={() => onSubmit('enqueue')}
                        isDisabled={
                            executionState === 'idle' ||
                            executionState === 'done' ||
                            !authenticated ||
                            jupyterAddress === undefined
                        }
                    >
                        Enqueue for Execution
                    </Button>
                </Tooltip>,
                <Button
                    key="cancel-code-submission-button"
                    variant="link"
                    onClick={onClose}
                    hidden={executionState === 'done'}
                >
                    Cancel
                </Button>,
            ]}
        >
            <Flex direction={{ default: 'column' }}>
                <FlexItem>
                    <Text component={TextVariants.h3}>
                        Enter the code to be executed below. Once you&apos;re ready, press &apos;Submit&apos; to submit
                        the code to the kernel for execution.
                    </Text>
                </FlexItem>
                <FlexItem>
                    <CodeContext.Provider value={{ code: code, setCode: setCode }}>
                        <CodeEditorComponent
                            showCodeTemplates={true}
                            height={400}
                            language={Language.python}
                            defaultFilename={'code'}
                        />
                    </CodeContext.Provider>
                </FlexItem>
                <FlexItem>
                    <Grid span={6}>
                        <GridItem rowSpan={1} colSpan={1}>
                            <Tooltip content="If checked, then the code execution is guaranteed to fail initially (at the scheduling level). This is useful for testing/debugging.">
                                <Checkbox
                                    label="Force Failure"
                                    id="force-failure-checkbox"
                                    isChecked={forceFailure}
                                    onChange={(_event: React.FormEvent<HTMLInputElement>, checked: boolean) =>
                                        setForceFailure(checked)
                                    }
                                />
                            </Tooltip>
                        </GridItem>
                        <GridItem rowSpan={5} colSpan={1}>
                            <Text component={TextVariants.p}>Target replica:</Text>
                            <Tooltip content="Specify the replica that should execute the code. This will fail (initially) if the target replica does not have enough resources, but may eventually succeed depending on the configured scheduling policy.">
                                <FormSelect
                                    isDisabled={forceFailure}
                                    value={targetReplicaId}
                                    onChange={onTargetReplicaChanged}
                                    aria-label="select-target-replica-menu"
                                    ouiaId="select-target-replica-menu"
                                >
                                    <FormSelectOption key={-1} value={'Auto'} label={'Auto'} />
                                    {props.kernel?.replicas.map((replica: JupyterKernelReplica) => (
                                        <FormSelectOption
                                            key={replica.replicaId}
                                            value={replica.replicaId}
                                            label={`Replica ${replica.replicaId}`}
                                        />
                                    ))}
                                </FormSelect>
                            </Tooltip>
                        </GridItem>
                    </Grid>
                </FlexItem>
                <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsNone' }}>
                    <FlexItem align={{ default: 'alignLeft' }}>
                        <Title headingLevel="h2">Output</Title>
                    </FlexItem>
                    <FlexItem align={{ default: 'alignRight' }}>
                        <Button
                            variant="link"
                            isInline
                            icon={<TimesIcon />}
                            onClick={() => onCloseAllExecutionTabsClicked()}
                        >
                            Close All Tabs
                        </Button>
                    </FlexItem>
                </Flex>
                {executionOutputArea}
            </Flex>
        </Modal>
    );
};
