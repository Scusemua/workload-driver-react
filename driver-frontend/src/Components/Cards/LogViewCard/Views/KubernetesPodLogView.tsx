import { LazyLog, ScrollFollow } from '@melloware/react-logviewer';
import React from 'react';
import { v4 as uuidv4 } from 'uuid';

export interface KubernetesPodLogViewProps {
    children?: React.ReactNode;
    podName: string;
    containerName: string;
    logPollIntervalSeconds: number;
    convertToHtml: boolean;
    signal: AbortSignal | undefined;
    height: number;
}

export const KubernetesPodLogView: React.FunctionComponent<KubernetesPodLogViewProps> = (
    props: KubernetesPodLogViewProps,
) => {
    // const [numLogsReceived, setNumLogsReceived] = React.useState<number>(0);
    // const [logs, setLogs] = React.useState<string[]>([]);
    // const [isOutputTextWrapped, setIsOutputTextWrapped] = React.useState(false);

    // const logViewerRef = React.useRef();

    // const { darkMode } = React.useContext(DarkModeContext);

    // const { latestLogMessage } = useLogs(props.podName, props.containerName);

    // const logs = React.useRef<string[]>([]);

    // React.useEffect(() => {
    //     logs.current = [...logs.current, latestLogMessage];
    // }, [latestLogMessage]);

    // const { sendJsonMessage, lastMessage, readyState } = useWebSocket('ws://localhost:8000/logs', {
    //     onOpen: () => {
    //         sendJsonMessage({
    //             op: 'get_logs',
    //             msg_id: uuidv4(),
    //             container: props.podName,
    //             container: props.containerName,
    //             follow: true,
    //         });
    //     },
    //     onClose: () => console.error('Lost connection to backend'),
    //     shouldReconnect: () => true,
    // });

    // React.useEffect(() => {
    //     console.log(`Log websocket: ${readyState}`);
    // }, [readyState]);

    // React.useEffect(() => {
    //     if (lastMessage !== null) {
    //         let reader: FileReader = new FileReader();

    //         reader.onload = () => {
    //             let message: string = reader.result as string;

    //             logs.current = [...logs.current, message];
    //             // setLogs((logs) => [...logs, message]);

    //             setNumLogsReceived((n) => n + 1);
    //         };

    //         reader.readAsText(lastMessage.data);
    //     }
    // }, [lastMessage]);

    // const FooterButton = () => {
    //     const handleClick = () => {
    //         logViewerRef.current?.scrollToBottom();
    //     };
    //     return <Button onClick={handleClick}>Jump to the bottom</Button>;
    // };

    // const onDownloadLogsClick = () => {
    //     const element = document.createElement('a');
    //     const dataToDownload: string[] = [logs.current.join('\r\n')];
    //     const file = new Blob(dataToDownload, { type: 'text/plain' });
    //     element.href = URL.createObjectURL(file);
    //     element.download = `${props.podName}-${props.containerName}-logs.txt`;
    //     document.body.appendChild(element);
    //     element.click();
    //     document.body.removeChild(element);
    // };

    // const leftAlignedOutputToolbarGroup = (
    //     <React.Fragment>
    //         <ToolbarToggleGroup toggleIcon={<EllipsisVIcon />} breakpoint="md">
    //             <ToolbarGroup>
    //                 <ToolbarItem>
    //                     <LogViewerSearch placeholder="Search" minSearchChars={0} />
    //                 </ToolbarItem>
    //                 <ToolbarItem alignSelf="center">
    //                     <Checkbox
    //                         label="Wrap text"
    //                         aria-label="wrap text checkbox"
    //                         isChecked={isOutputTextWrapped}
    //                         id="wrap-text-checkbox"
    //                         onChange={(_event, value) => setIsOutputTextWrapped(value)}
    //                     />
    //                 </ToolbarItem>
    //                 <ToolbarItem>Number of Log Items: {numLogsReceived}</ToolbarItem>
    //             </ToolbarGroup>
    //         </ToolbarToggleGroup>
    //     </React.Fragment>
    // );

    // const rightAlignedOutputToolbarGroup = (
    //     <React.Fragment>
    //         <ToolbarGroup variant="icon-button-group">
    //             <ToolbarItem>
    //                 <Tooltip position="top" content={<div>Download</div>}>
    //                     <Button onClick={onDownloadLogsClick} variant="plain" aria-label="Download current logs">
    //                         <DownloadIcon />
    //                     </Button>
    //                 </Tooltip>
    //             </ToolbarItem>
    //         </ToolbarGroup>
    //     </React.Fragment>
    // );

    return (
        // <React.Fragment>
        //     <LogViewer
        //         ref={logViewerRef}
        //         hasLineNumbers={true}
        //         data={logs.current}
        //         height={props.height}
        //         theme={darkMode ? 'dark' : 'light'}
        //         footer={<FooterButton />}
        //         isTextWrapped={isOutputTextWrapped}
        //         toolbar={
        //             <Toolbar>
        //                 <ToolbarContent>
        //                     <ToolbarGroup align={{ default: 'alignLeft' }}>
        //                         {leftAlignedOutputToolbarGroup}
        //                     </ToolbarGroup>
        //                     <ToolbarGroup align={{ default: 'alignRight' }}>
        //                         {rightAlignedOutputToolbarGroup}
        //                     </ToolbarGroup>
        //                 </ToolbarContent>
        //             </Toolbar>
        //         }
        //     />
        // </React.Fragment>
        <ScrollFollow
            startFollowing={true}
            render={({ follow, onScroll }) => (
                <LazyLog
                    url={'ws://localhost:8000/logs'}
                    enableSearch
                    enableSearchNavigation
                    enableLineNumbers
                    enableMultilineHighlight
                    enableLinks
                    websocket={true}
                    follow={follow}
                    height={props.height}
                    stream={true}
                    onScroll={onScroll}
                    websocketOptions={{
                        onOpen: (_e: Event, socket: WebSocket) => {
                            console.log(
                                `Sending 'get-logs' message for container ${props.containerName} of container ${props.podName}`,
                            );
                            socket.binaryType = 'arraybuffer';
                            socket.send(
                                JSON.stringify({
                                    op: 'get_logs',
                                    msg_id: uuidv4(),
                                    pod: props.podName,
                                    container: props.containerName,
                                    follow: true,
                                }),
                            );
                        },
                        formatMessage: (message: ArrayBuffer) => {
                            return new TextDecoder().decode(message);
                        },
                        onClose: () => {
                            console.debug(
                                `Websocket Log connection closed for container ${props.containerName} of container ${props.podName}`,
                            );
                        },
                    }}
                    extraLines={1}
                    enableHotKeys
                    selectableLines
                />
            )}
        />
    );
};
