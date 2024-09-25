import { DistributedJupyterKernel, JupyterKernelReplica } from '@app/Data';
import { GetHeaderAndBodyForToast } from '@app/utils/toast_utils';
import { CodeEditorComponent } from '@components/CodeEditor';
import { ExecutionOutputTabContent } from '@components/Modals/ExecuteCodeOnKernelModal/ExecutionOutputTabContent';
import { RoundToThreeDecimalPlaces } from '@components/Modals/NewWorkloadFromTemplateModal';
import { KernelManager, ServerConnection } from '@jupyterlab/services';
import { IKernelConnection } from '@jupyterlab/services/lib/kernel/kernel';
import { Language } from '@patternfly/react-code-editor';
import {
    Button,
    Card,
    CardBody,
    Checkbox,
    Flex,
    FlexItem,
    FormSelect,
    FormSelectOption,
    Grid,
    GridItem,
    Modal,
    Tab,
    Tabs,
    TabTitleText,
    Text,
    TextVariants,
    Title,
    Tooltip,
} from '@patternfly/react-core';
import { CheckCircleIcon } from '@patternfly/react-icons';
import React from 'react';
import toast from 'react-hot-toast';
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
    setCode: (newCode: string) => {},
});

export const ExecuteCodeOnKernelModal: React.FunctionComponent<ExecuteCodeOnKernelProps> = (props) => {
    const [code, setCode] = React.useState('');
    const [executionState, setExecutionState] = React.useState('idle');

    const [targetReplicaId, setTargetReplicaId] = React.useState(-1);
    const [forceFailure, setForceFailure] = React.useState(false);
    const [activeExecutionOutputTab, setActiveExecutionOutputTab] = React.useState<string>('');

    const [outputMap, setOutputMap] = React.useState<Map<string, string[]>>(new Map());
    const [execIdToKernelReplicaMap, setExecIdToKernelReplicaMap] = React.useState<
        Map<string, [string, number | undefined]>
    >(new Map());
    const [closedExecutionMap, setClosedExecutionMap] = React.useState<Map<string, boolean>>(new Map());

    const executionOutputTabComponentRef = React.useRef();

    const onExecutionOutputTabSelect = (executionId: string) => {
        setActiveExecutionOutputTab(executionId);
    };

    const onCloseExecutionOutputTab = (_: React.MouseEvent<HTMLElement, MouseEvent>, executionId: string | number) => {
        setOutputMap((prevOutputMap) => {
            const nextOutput = new Map(prevOutputMap);
            nextOutput.delete(executionId as string);
            return nextOutput;
        });
        setClosedExecutionMap((prevClosedExecutionMap) =>
            new Map(prevClosedExecutionMap).set(executionId as string, true),
        );

        // If we're closing the active tab, attempt to select another tab as the active tab.
        if (activeExecutionOutputTab == executionId) {
            for (const [key] of Array.from(outputMap)) {
                if (key != executionId) {
                    console.log(`Setting active tab to ${key}`);
                    setActiveExecutionOutputTab(key);
                    break;
                }
            }
        }
    };

    React.useEffect(() => {
        // Basically, if we don't have an active tab selected, or if the tab we had selected was closed,
        // and we just added a new tab, then set the active tab to the newly-added tab.
        if (outputMap.size >= 1 && (activeExecutionOutputTab === '' || !outputMap.has(activeExecutionOutputTab))) {
            console.log(`Setting active tab to ${outputMap.keys()[0]}`);
            setActiveExecutionOutputTab(outputMap.keys()[0]);
        }
    }, [outputMap]);

    React.useEffect(() => {
        setTargetReplicaId(props.replicaId || -1);
    }, [props.replicaId]);

    const logConsumer = (msg: string, execution_id: string) => {
        console.log(`Appending message to output log for kernel execution: ${msg}`);
        const messages: string[] = msg.trim().split(/\n/);
        console.log(`Appending ${messages.length} message(s) to output log for kerenl execution: ${messages}`);

        setOutputMap((prevOutputMap) => {
            let prevOutput: string[] | undefined = prevOutputMap.get(execution_id);

            // If the user explicitly closed the tab, then we'll just return.
            // If the tab was never explicitly closed, then we're receiving update
            // from the associated execution for the very first time, and so
            // we'll need to add/create an entry in the output map.
            if (prevOutput === undefined) {
                if (!closedExecutionMap.has(execution_id)) {
                    prevOutput = [];
                } else {
                    return new Map(prevOutputMap);
                }
            }

            const nextOutput = [...prevOutput, ...messages];
            return new Map(prevOutputMap.set(execution_id, nextOutput));
        });
    };

    const onSubmit = (action: 'submit' | 'enqueue') => {
        async function runUserCode() {
            const executionId: string = uuidv4();
            const kernelId: string | undefined = props.kernel?.kernelId;

            if (kernelId == undefined) {
                console.error("Couldn't determiner kernel ID of target kernel for code execution...");
                return;
            }

            const kernelSpecManagerOptions: KernelManager.IOptions = {
                serverSettings: ServerConnection.makeSettings({
                    token: '',
                    appendToken: false,
                    baseUrl: '/jupyter',
                    wsUrl: 'ws://localhost:8888/',
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
                        // 'Cache-Control': 'no-cache, no-transform, no-store',
                    },
                    body: JSON.stringify({
                        kernel_id: kernelId,
                    }),
                };

                await fetch('api/yield-next-execute-request', req);
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

            console.log(`Sending 'execute-request' to kernel ${kernelId} for code: '${code}'`);

            const startTime: number = performance.now();
            const future = kernelConnection.requestExecute({ code: code }, undefined, {
                target_replica: targetReplicaId,
                'send-timestamp-unix-milli': Date.now(),
            });

            // Handle iopub messages
            future.onIOPub = (msg) => {
                console.log('Received IOPub message:\n%s\n', JSON.stringify(msg));
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

            if (activeExecutionOutputTab === '' || !outputMap.has(activeExecutionOutputTab)) {
                console.log(`Setting active tab to ${executionId}`);
                setActiveExecutionOutputTab(executionId);
            }

            setExecIdToKernelReplicaMap((prevMap) => new Map(prevMap).set(executionId, [kernelId, props.replicaId]));

            future.onReply = (msg) => {
                console.log(`Received reply for execution request: ${JSON.stringify(msg)}`);
            };

            await toast.promise(
                future.done,
                {
                    success: () => {
                        const latencyMilliseconds: number = performance.now() - startTime;
                        const latencySecRounded: number = RoundToThreeDecimalPlaces(latencyMilliseconds / 1000.0);
                        console.log(`Execution on Kernel ${kernelId} finished after ${latencySecRounded} seconds.`);

                        return GetHeaderAndBodyForToast(
                            `Execution Complete ${Math.random() > 0.5 ? '🔥' : '😍'}`,
                            `Kernel ${kernelId} has finished executing your code after ${latencySecRounded} seconds.`,
                        );
                    },
                    loading: GetHeaderAndBodyForToast(
                        action == 'submit' ? 'Code Submitted 👀' : 'Code Enqueued 👀',
                        action == 'submit'
                            ? `Submitted code for execution to kernel ${kernelId}.`
                            : `Enqueued code for execution with kernel ${kernelId}.`,
                    ),
                    error: (error) => {
                        const latencyMilliseconds: number = performance.now() - startTime;
                        const latencySecRounded: number = RoundToThreeDecimalPlaces(latencyMilliseconds / 1000.0);
                        console.error(
                            `Execution on Kernel ${kernelId} failed to complete after ${latencySecRounded} seconds. Error: ${error}.`,
                        );
                        return GetHeaderAndBodyForToast(
                            '️ Execution Failed ⚠️️️',
                            `Execution on Kernel ${kernelId} failed to complete after ${latencySecRounded} seconds. Error: ${error}.`,
                        );
                    },
                },
                {
                    style: { maxWidth: 750 },
                    duration: 5000,
                },
            );

            // await future.done;
            setExecutionState('done');

            await fetch('api/metrics', {
                method: 'PATCH',
                headers: {
                    'Content-Type': 'application/json',
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
        const val = execIdToKernelReplicaMap.get(execId);
        if (val) {
            return val[0];
        }
        return undefined;
    };

    const getReplicaId = (execId: string) => {
        const val = execIdToKernelReplicaMap.get(execId);
        if (val) {
            return val[1];
        }
        return undefined;
    };

    const getOutput = () => {
        const output = outputMap.get(activeExecutionOutputTab);
        if (output) {
            return output;
        }
        return [];
    };

    const executionOutputArea = (
        <Card isCompact isFlat>
            <CardBody>
                <Tabs
                    hidden={outputMap.size == 0}
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
                    {Array.from(outputMap).map(([execId]) => {
                        return (
                            <Tab
                                id={`execution-output-tab-${execId}`}
                                key={`execution-output-tab-${execId}`}
                                eventKey={execId}
                                aria-label={`${execId} Tab`}
                                title={<TabTitleText>{execId}</TabTitleText>}
                                closeButtonAriaLabel={`Close ${execId} Tab`}
                            />
                        );
                    })}
                </Tabs>
                <ExecutionOutputTabContent
                    output={getOutput()}
                    executionId={activeExecutionOutputTab}
                    kernelId={getKernelId(activeExecutionOutputTab)}
                    replicaId={getReplicaId(activeExecutionOutputTab)}
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
                            console.log('Closing execute code modal.');
                            setExecutionState('idle');
                        }
                    }}
                    isDisabled={code.trim().length == 0}
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
                        isDisabled={executionState === 'idle' || executionState === 'done'}
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
                <FlexItem>
                    <Title headingLevel="h2">Output</Title>
                </FlexItem>
                {executionOutputArea}
            </Flex>
        </Modal>
    );
};
