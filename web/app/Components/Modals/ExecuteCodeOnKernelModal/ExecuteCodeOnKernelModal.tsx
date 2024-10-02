import { DistributedJupyterKernel, JupyterKernelReplica } from '@app/Data';
import {
    GetToastContentWithHeaderAndBody,
    GetToastContentWithHeaderAndBodyAndDismissButton,
} from '@app/utils/toast_utils';
import { CodeEditorComponent } from '@components/CodeEditor';
import { ExecutionOutputTabContent } from '@components/Modals/ExecuteCodeOnKernelModal/ExecutionOutputTabContent';
import { RoundToThreeDecimalPlaces } from '@components/Modals/NewWorkloadFromTemplateModal';
import { KernelManager, ServerConnection } from '@jupyterlab/services';
import { IKernelConnection, IShellFuture } from '@jupyterlab/services/lib/kernel/kernel';
import { IExecuteReplyMsg, IExecuteRequestMsg } from '@jupyterlab/services/lib/kernel/messages';
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
    Label,
    Modal,
    Tab,
    Tabs,
    TabTitleText,
    Text,
    TextVariants,
    Title,
    Tooltip,
} from '@patternfly/react-core';
import { CheckCircleIcon, SpinnerIcon, TimesCircleIcon, TimesIcon } from '@patternfly/react-icons';
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
    setCode: (newCode: string) => {},
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
}

export const ExecuteCodeOnKernelModal: React.FunctionComponent<ExecuteCodeOnKernelProps> = (props) => {
    const [code, setCode] = React.useState('');
    const [executionState, setExecutionState] = React.useState('idle');

    const [targetReplicaId, setTargetReplicaId] = React.useState(-1);
    const [forceFailure, setForceFailure] = React.useState(false);
    const [activeExecutionOutputTab, setActiveExecutionOutputTab] = React.useState<string>('');

    const [executionMap, setExecutionMap] = React.useState<Map<string, Execution>>(new Map());
    const [, setClosedExecutionMap] = React.useState<Map<string, boolean>>(new Map());

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
    }, [executionMap]);

    React.useEffect(() => {
        setTargetReplicaId(props.replicaId || -1);
    }, [props.replicaId]);

    const logConsumer = (msg: string, execution_id: string) => {
        console.log(`Appending message to output log for kernel execution: ${msg}`);
        const messages: string[] = msg.trim().split(/\n/);
        console.log(`Appending ${messages.length} message(s) to output log for kerenl execution: ${messages}`);

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

    const onSubmit = (action: 'submit' | 'enqueue') => {
        async function runUserCode(): Promise<Execution | undefined> {
            const executionId: string = uuidv4();
            const kernelId: string | undefined = props.kernel?.kernelId;

            if (kernelId == undefined) {
                console.error("Couldn't determiner kernel ID of target kernel for code execution...");
                return undefined;
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

            kernelConnection.disposed.connect((sender, args) => {
                console.log(
                    `Connection to Kernel ${props.kernel?.kernelId} has been disposed. Sender: ${sender}, args: ${args}`,
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

            const execution: Execution = {
                kernelId: kernelId,
                replicaId: props.replicaId,
                code: code,
                future: future,
                executionId: executionId,
                status: 'running',
                output: [],
            };

            if (activeExecutionOutputTab === '' || !executionMap.has(activeExecutionOutputTab)) {
                console.log(`Setting active tab to ${executionId}`);
                setActiveExecutionOutputTab(executionId);
            }

            setExecutionMap((prevMap) => new Map(prevMap).set(executionId, execution));

            future.onReply = (msg) => {
                console.log(`Received reply for execution request: ${JSON.stringify(msg)}`);
            };

            const toastId: string = toast.loading(
                (t: Toast) => {
                    return GetToastContentWithHeaderAndBodyAndDismissButton(
                        action == 'submit' ? 'Code Submitted 🚀' : 'Code Enqueued 🚀',
                        action == 'submit'
                            ? `Submitted code for execution to kernel ${kernelId}.`
                            : `Enqueued code for execution with kernel ${kernelId}.`,
                        t.id,
                    );
                },
                { style: { maxWidth: 750 } },
            );

            await future.done
                .catch((error: Error) => {
                    const latencyMilliseconds: number = performance.now() - startTime;
                    const latencySecRounded: number = RoundToThreeDecimalPlaces(latencyMilliseconds / 1000.0);
                    console.error(
                        `Execution on Kernel ${kernelId} failed to complete after ${latencySecRounded} seconds. Error: ${error}.`,
                    );

                    setExecutionMap((prevMap) => {
                        const exec: Execution | undefined = prevMap.get(executionId);
                        if (exec) {
                            exec.status = 'failed';
                            return new Map(prevMap).set(executionId, exec);
                        }
                        return prevMap;
                    });

                    toast.error(
                        GetToastContentWithHeaderAndBody(
                            '️ Execution Failed ⚠️️️',
                            `Execution on Kernel ${kernelId} failed to complete after ${latencySecRounded} seconds. Error: ${error}.`,
                        ),
                        {
                            id: toastId,
                            style: { maxWidth: 750 },
                            duration: 12500,
                        },
                    );

                    future.dispose();
                })
                .then(() => {
                    const latencyMilliseconds: number = performance.now() - startTime;
                    const latencySecRounded: number = RoundToThreeDecimalPlaces(latencyMilliseconds / 1000.0);
                    console.log(`Execution on Kernel ${kernelId} finished after ${latencySecRounded} seconds.`);

                    setExecutionMap((prevMap) => {
                        const exec: Execution | undefined = prevMap.get(executionId);
                        if (exec) {
                            exec.status = 'completed';
                            return new Map(prevMap).set(executionId, exec);
                        }
                        return prevMap;
                    });

                    const successIcon: string = Math.random() > 0.5 ? '✅' : '✅';

                    toast.success(
                        (t: Toast) => {
                            return GetToastContentWithHeaderAndBodyAndDismissButton(
                                `Execution Complete ${successIcon}`,
                                `Kernel ${kernelId} has finished executing your code after ${latencySecRounded} seconds.`,
                                t.id,
                            );
                        },
                        {
                            id: toastId,
                            style: { maxWidth: 750 },
                            duration: 5000,
                        },
                    );

                    future.dispose();
                });

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
                                                <FlexItem alignSelf={{ default: 'alignSelfFlexEnd' }}>
                                                    <Text component={'small'}>
                                                        <b>ExecID: </b> {execId.substring(0, 8)}
                                                    </Text>
                                                </FlexItem>
                                                <FlexItem>{getExecutionLabel(exec)}</FlexItem>
                                            </Flex>
                                            <FlexItem alignSelf={{ default: 'alignSelfFlexStart' }}>
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
