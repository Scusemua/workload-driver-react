import {
    Button,
    Card,
    CardBody,
    Checkbox,
    ClipboardCopy,
    ClipboardCopyVariant,
    Flex,
    FlexItem,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
    ToolbarToggleGroup,
    Tooltip,
} from '@patternfly/react-core';
import { DownloadIcon, EllipsisVIcon } from '@patternfly/react-icons';
import { LogViewer, LogViewerSearch } from '@patternfly/react-log-viewer';
import { Execution, RequestTraceSplitTable } from '@src/Components';
import { DarkModeContext } from '@src/Providers';
import React from 'react';

export interface ExecutionOutputTabContentProps {
    children?: React.ReactNode;
    kernelId?: string;
    replicaId?: number;
    executionId?: string;
    output: string[];
    errorMessage?: string;
    exec?: Execution;
}

export const ExecutionOutputTabContent: React.FunctionComponent<ExecutionOutputTabContentProps> = (
    props: ExecutionOutputTabContentProps,
) => {
    const { darkMode } = React.useContext(DarkModeContext);

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const logViewerRef = React.useRef<React.Ref<any>>();
    const [isOutputTextWrapped, setIsOutputTextWrapped] = React.useState(false);
    const [isOutputFullScreen] = React.useState(false);

    const FooterButton = () => {
        const handleClick = () => {
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-expect-error
            logViewerRef.current?.scrollToBottom();
        };
        return <Button onClick={handleClick}>Jump to the bottom</Button>;
    };

    const onDownloadLogsClick = () => {
        const element = document.createElement('a');
        const dataToDownload: string[] = [props.output.join('\r\n')];
        const file = new Blob(dataToDownload, { type: 'text/plain' });
        element.href = URL.createObjectURL(file);
        element.download = `kernel-${props.kernelId}-${props.replicaId}-${props.executionId}-execution-output.txt`;
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
            </ToolbarGroup>
        </React.Fragment>
    );

    return (
        <Card isCompact isRounded isFlat hidden={props.output.length == 0}>
            <CardBody>
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                    <FlexItem>
                        <LogViewer
                            key={'kernel-execution-output'}
                            ref={logViewerRef}
                            hasLineNumbers={true}
                            data={props.output}
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
                    </FlexItem>
                    <FlexItem hidden={props.errorMessage === undefined}>
                        <Title headingLevel="h3">Error Message</Title>
                        <ClipboardCopy
                            isReadOnly
                            isExpanded
                            hoverTip="Copy"
                            clickTip="Copied"
                            variant={ClipboardCopyVariant.expansion}
                        >
                            {props.errorMessage}
                        </ClipboardCopy>
                    </FlexItem>
                    {props.exec !== undefined && props.exec.requestTraces.length > 0 && (
                        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsXs' }}>
                            <FlexItem>
                                <Title headingLevel="h3">Request Trace</Title>
                            </FlexItem>
                            <FlexItem>
                                <RequestTraceSplitTable
                                    traces={props.exec.requestTraces}
                                    messageId={props.exec.messageId || ''}
                                    receivedReplyAt={props.exec.receivedReplyAt}
                                />
                            </FlexItem>
                        </Flex>
                    )}
                </Flex>
            </CardBody>
        </Card>
    );
};
