import {CodeEditorComponent} from '@app/Components/CodeEditor';
import {DistributedJupyterKernel, JupyterKernelReplica} from '@app/Data';
import {DarkModeContext} from '@app/Providers/DarkModeProvider';
import {KernelManager, ServerConnection} from '@jupyterlab/services';
import {IKernelConnection} from '@jupyterlab/services/lib/kernel/kernel';
import {Language} from "@patternfly/react-code-editor";
import {
  Button,
  Checkbox,
  ClipboardCopyButton,
  CodeBlockAction,
  Flex,
  FlexItem,
  FormSelect,
  FormSelectOption,
  Grid,
  GridItem,
  Modal,
  Text,
  TextVariants,
  Title,
  Toolbar,
  ToolbarContent,
  ToolbarGroup,
  ToolbarItem,
  ToolbarToggleGroup,
  Tooltip,
} from '@patternfly/react-core';
import {CheckCircleIcon, DownloadIcon, EllipsisVIcon} from '@patternfly/react-icons';
import {LogViewer, LogViewerSearch} from '@patternfly/react-log-viewer';
import React from 'react';

export interface ExecuteCodeOnKernelProps {
    children?: React.ReactNode;
    kernel: DistributedJupyterKernel | null;
    replicaId?: number;
    isOpen: boolean;
    onClose: () => void;
    onSubmit: (
        code: string,
        targetReplicaId: number,
        forceFailure: boolean,
        logConsumer: (msg: string) => void,
    ) => Promise<void>;
}

export type CodeContext = {
    code: string;
    setCode: (newCode: string) => void;
};

/* eslint-disable-next-line @typescript-eslint/no-unused-vars */
export const CodeContext = React.createContext({ code: '', setCode: (newCode: string) => {} });

export const ExecuteCodeOnKernelModal: React.FunctionComponent<ExecuteCodeOnKernelProps> = (props) => {
    const [code, setCode] = React.useState('');
    const [executionState, setExecutionState] = React.useState('idle');
    const [copied, setCopied] = React.useState(false);
    const [targetReplicaId, setTargetReplicaId] = React.useState(-1);
    const [forceFailure, setForceFailure] = React.useState(false);
    const [isOutputTextWrapped, setIsOutputTextWrapped] = React.useState(false);
    const [isOutputFullScreen, setIsOutputFullScreen] = React.useState(false);
    const logViewerRef = React.useRef<any>();

    const { darkMode, toggleDarkMode } = React.useContext(DarkModeContext);

    const [output, setOutput] = React.useState<string[]>([]);

    React.useEffect(() => {
        setTargetReplicaId(props.replicaId || -1);
    }, [props.replicaId]);

    const clipboardCopyFunc = (_event, text) => {
        navigator.clipboard.writeText(text.toString());
    };

    const onClickCopyToClipboard = (event, text) => {
        clipboardCopyFunc(event, text);
        setCopied(true);
    };

    const logConsumer = (msg: string) => {
        console.log(`Appending message to output log for kerenl execution: ${msg}`);
        const messages: string[] = msg.trim().split(/\n/);
        console.log(`Appending ${messages.length} message(s) to output log for kerenl execution: ${messages}`);
        setOutput((output) => [...output, ...messages]);
    };

    React.useEffect(() => {
        console.log(`There are now ${output.length} entries in the output log.`);
    }, [output]);

    const onSubmit = () => {
        async function runUserCode() {
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
            let kernelManager = new KernelManager(kernelSpecManagerOptions);

            console.log('Waiting for Kernel Manager to be ready.');

            kernelManager.connectionFailure.connect((_sender: KernelManager, err: Error) => {
                console.error(
                    'An error has occurred while preparing the Kernel Manager. ' + err.name + ': ' + err.message,
                );
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

            console.log(`Sending 'execute-request' to kernel ${kernelId} for code: '${code}'`);

            const future = kernelConnection.requestExecute({ code: code }, undefined, {
                target_replica: targetReplicaId,
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
                    );
                } else if (messageType == 'stream') {
                    if (msg['content']['name'] == 'stderr') {
                        logConsumer(msg['header']['date'] + ' <ERROR>: ' + msg.content['text'] + '\n');
                    } else if (msg['content']['name'] == 'stdout') {
                        logConsumer(msg['header']['date'] + ': ' + msg.content['text'] + '\n');
                    } else {
                        logConsumer(msg['header']['date'] + ': ' + msg.content['text'] + '\n');
                    }
                } else {
                    logConsumer(msg['header']['date'] + ': ' + JSON.stringify(msg.content) + '\n');
                }
            };

            future.onReply = (msg) => {
                console.log(`Received reply for execution request: ${JSON.stringify(msg)}`);
            };

            await future.done;
            console.log('Execution on Kernel ' + kernelId + ' is done.');
            setExecutionState('done');
        }

        runUserCode();
    };

    // Reset state, then call user-supplied onClose function.
    const onClose = () => {
        console.log('Closing execute code modal.');
        setExecutionState('idle');
        setOutput([]);
        props.onClose();
    };

    const outputLogActions = (
        <React.Fragment>
            <CodeBlockAction>
                <ClipboardCopyButton
                    id="basic-copy-button"
                    textId="code-content"
                    aria-label="Copy to clipboard"
                    onClick={(e) => onClickCopyToClipboard(e, code)}
                    exitDelay={copied ? 1500 : 600}
                    maxWidth="110px"
                    variant="plain"
                    onTooltipHidden={() => setCopied(false)}
                >
                    {copied ? 'Successfully copied to clipboard!' : 'Copy to clipboard'}
                </ClipboardCopyButton>
            </CodeBlockAction>
        </React.Fragment>
    );

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

    const FooterButton = () => {
        const handleClick = () => {
            logViewerRef.current?.scrollToBottom();
        };
        return <Button onClick={handleClick}>Jump to the bottom</Button>;
    };

    // Buggy.
    // const onExpandLogsClick = (_event) => {
    //     const element = document.querySelector('#kernel-execution-output');

    //     if (!isOutputFullScreen) {
    //         if (element?.requestFullscreen) {
    //             element.requestFullscreen();
    //         } else if (element?.mozRequestFullScreen) {
    //             element?.mozRequestFullScreen();
    //         } else if (element?.webkitRequestFullScreen) {
    //             element?.webkitRequestFullScreen(Element.ALLOW_KEYBOARD_INPUT);
    //         }
    //         setIsOutputFullScreen(true);
    //     } else {
    //         if (document.exitFullscreen) {
    //             document.exitFullscreen();
    //         } else if (document?.webkitExitFullscreen) {
    //             /* Safari */
    //             document.webkitExitFullscreen();
    //         } else if (document?.msExitFullscreen) {
    //             /* IE11 */
    //             document?.msExitFullscreen();
    //         }
    //         setIsOutputFullScreen(false);
    //     }
    // };

    const onDownloadLogsClick = () => {
        const element = document.createElement('a');
        const dataToDownload: string[] = [output.join('\r\n')];
        const file = new Blob(dataToDownload, { type: 'text/plain' });
        element.href = URL.createObjectURL(file);
        element.download = `kernel-${props.kernel?.kernelId}-${props.replicaId}-execution-output.txt`;
        document.body.appendChild(element);
        element.click();
        document.body.removeChild(element);
    };

    const leftAlignedOutputToolbarGroup = (
        <React.Fragment>
            <ToolbarToggleGroup toggleIcon={<EllipsisVIcon />} breakpoint="md">
                <ToolbarGroup>
                    <ToolbarItem>
                        <LogViewerSearch placeholder="Search" minSearchChars={0} />
                    </ToolbarItem>
                    <ToolbarItem alignSelf="center">
                        <Checkbox
                            label="Wrap text"
                            aria-label="wrap text checkbox"
                            isChecked={isOutputTextWrapped}
                            id="wrap-text-checkbox"
                            onChange={(_event, value) => setIsOutputTextWrapped(value)}
                        />
                    </ToolbarItem>
                </ToolbarGroup>
            </ToolbarToggleGroup>
        </React.Fragment>
    );

    const rightAlignedOutputToolbarGroup = (
        <React.Fragment>
            <ToolbarGroup variant="icon-button-group">
                <ToolbarItem>
                    <Tooltip position="top" content={<div>Download</div>}>
                        <Button onClick={onDownloadLogsClick} variant="plain" aria-label="Download current logs">
                            <DownloadIcon />
                        </Button>
                    </Tooltip>
                </ToolbarItem>
                {/* <ToolbarItem>
                    <Tooltip position="top" content={<div>Expand</div>}>
                        <Button onClick={onExpandLogsClick} variant="plain" aria-label="View log viewer in full screen">
                            <ExpandIcon />
                        </Button>
                    </Tooltip>
                </ToolbarItem> */}
            </ToolbarGroup>
        </React.Fragment>
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
                    key="submit"
                    variant="primary"
                    onClick={() => {
                        if (executionState == 'idle') {
                            setExecutionState('busy');
                            onSubmit();
                        } else if (executionState == 'busy') {
                            console.log(
                                'Please wait until the current execution completes before submitting additional code for execution.',
                            );
                        } else {
                            console.log('Closing execute code modal.');
                            setExecutionState('idle');
                            setOutput([]);
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
                <Button key="cancel" variant="link" onClick={onClose} hidden={executionState === 'done'}>
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
                        <CodeEditorComponent showCodeTemplates={true} height={400} language={Language.python}/>
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
                <FlexItem>
                    <LogViewer
                        // id={'kernel-execution-output'}
                        ref={logViewerRef}
                        hasLineNumbers={true}
                        data={output}
                        theme={darkMode ? 'dark' : 'light'}
                        height={isOutputFullScreen ? '100%' : 300}
                        footer={<FooterButton />}
                        isTextWrapped={isOutputTextWrapped}
                        toolbar={
                            <Toolbar>
                                <ToolbarContent>
                                    <ToolbarGroup align={{ default: 'alignLeft' }}>
                                        {leftAlignedOutputToolbarGroup}
                                    </ToolbarGroup>
                                    <ToolbarGroup align={{ default: 'alignRight' }}>
                                        {rightAlignedOutputToolbarGroup}
                                    </ToolbarGroup>
                                </ToolbarContent>
                            </Toolbar>
                        }
                    />
                    {/* <CodeBlock actions={outputLogActions}>
                        {output.map((val, idx) => (
                            <CodeBlockCode key={'log-message-' + idx} id={'log-message-' + idx}>
                                {val}
                            </CodeBlockCode>
                        ))}
                    </CodeBlock> */}
                </FlexItem>
            </Flex>
        </Modal>
    );
};
