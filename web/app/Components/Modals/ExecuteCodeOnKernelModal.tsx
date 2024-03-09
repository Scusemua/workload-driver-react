import React from 'react';
import {
    Button,
    ClipboardCopyButton,
    CodeBlock,
    CodeBlockAction,
    CodeBlockCode,
    Modal,
    ModalVariant,
    Title,
} from '@patternfly/react-core';

import { CodeEditorComponent } from '@app/Components/CodeEditor';
import { CheckCircleIcon } from '@patternfly/react-icons';

export interface ExecuteCodeOnKernelProps {
    children?: React.ReactNode;
    kernelId: string;
    isOpen: boolean;
    onClose: () => void;
    onSubmit: (code: string, logConsumer: (msg: string) => void) => Promise<void>;
}

export const ExecuteCodeOnKernelModal: React.FunctionComponent<ExecuteCodeOnKernelProps> = (props) => {
    const [code, setCode] = React.useState('');
    const [executionState, setExecutionState] = React.useState('idle');
    const [copied, setCopied] = React.useState(false);
    const output = React.useRef<string[]>([]);

    const clipboardCopyFunc = (_event, text) => {
        navigator.clipboard.writeText(text.toString());
    };

    const onClickCopyToClipboard = (event, text) => {
        clipboardCopyFunc(event, text);
        setCopied(true);
    };

    const logConsumer = (msg: string) => {
        output.current = [...output.current, msg];
    };

    const onSubmit = () => {
        async function runUserCode() {
            await props.onSubmit(code, logConsumer);
            setExecutionState('done');
        }

        runUserCode();
    };

    const onChange = (code) => {
        setCode(code);
    };

    // Reset state, then call user-supplied onClose function.
    const onClose = () => {
        console.log('Closing execute code modal.');
        setExecutionState('idle');
        output.current = [];
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

    return (
        <Modal
            variant={ModalVariant.large}
            title={'Execute Code on Kernel ' + props.kernelId}
            isOpen={props.isOpen}
            onClose={props.onClose}
            actions={[
                <Button
                    key="submit"
                    variant="primary"
                    onClick={() => {
                        if (executionState == 'idle') {
                            console.log('Executing code now.');
                            setExecutionState('busy');
                            onSubmit();
                        } else if (executionState == 'busy') {
                            console.log(
                                'Please wait until the current execution completes before submitting additional code for execution.',
                            );
                        } else {
                            onClose();
                        }
                    }}
                    isLoading={executionState === 'busy'}
                    icon={executionState === 'done' ? <CheckCircleIcon /> : null}
                    spinnerAriaValueText="Loading..."
                >
                    {executionState === 'idle' && 'Execute'}
                    {executionState === 'busy' && 'Executing code'}
                    {executionState === 'done' && 'Complete'}
                </Button>,
                <Button key="cancel" variant="link" onClick={onClose}>
                    Cancel
                </Button>,
            ]}
        >
            Enter the code to be executed below. Once you&apos;re ready, press &apos;Submit&apos; to submit the code to
            the kernel for execution.
            <CodeEditorComponent onChange={onChange} />
            <br />
            <Title headingLevel="h2">Output</Title>
            <CodeBlock actions={outputLogActions}>
                {output.current.map((val, idx) => (
                    <CodeBlockCode key={'log-message-' + idx} id={'log-message-' + idx}>
                        {val}
                    </CodeBlockCode>
                ))}
            </CodeBlock>
        </Modal>
    );
};
