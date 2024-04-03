import React from 'react';
import { LazyLog, ScrollFollow } from '@melloware/react-logviewer';
import { v4 as uuidv4 } from 'uuid';

export interface KubernetesLogViewProps {
    children?: React.ReactNode;
    podName: string;
    containerName: string;
    logPollIntervalSeconds: number;
    convertToHtml: boolean;
    signal: AbortSignal | undefined;
    height: number;
}

export const KubernetesLogViewComponent: React.FunctionComponent<KubernetesLogViewProps> = (props) => {
    // const logs = useRef('');

    // Just use websockets. Ugh.
    // const { sendMessage, lastMessage } = useWebSocket('ws://localhost:8000/logs');

    // useEffect(() => {
    //     console.log(`Requesting logs for container ${props.containerName} of pod ${props.podName}`);
    //     sendMessage(
    // JSON.stringify({
    //     op: 'get_logs',
    //     msg_id: uuidv4(),
    //     pod: props.podName,
    //     container: props.containerName,
    //     follow: true,
    // }),
    //     );
    // }, [sendMessage, props.podName, props.containerName]);

    // useEffect(() => {
    // async function readFromStream(reader: ReadableStreamDefaultReader<Uint8Array>) {
    //     let response: ReadableStreamReadResult<Uint8Array> = await reader.read();
    //     while (!response?.done) {
    //         const text: string = new TextDecoder().decode(response!.value);
    //         logs.current = logs.current + text;

    //         response = await reader.read();
    //     }
    // }

    // if (lastMessage !== null) {
    //     const data: Blob = lastMessage.data;
    //     const stream: ReadableStream<Uint8Array> = data.stream();
    //     const reader: ReadableStreamDefaultReader<Uint8Array> | undefined = stream.getReader();

    //     if (reader === undefined) {
    //         console.error('Could not get reader for message stream...');
    //         return;
    //     }

    //     readFromStream(reader);
    // }
    // }, [lastMessage]);

    return (
        <ScrollFollow
            startFollowing={true}
            render={({ follow, onScroll }) => (
                <LazyLog
                    // text={logs.current}
                    url={'ws://localhost:8000/logs'}
                    enableSearch
                    enableSearchNavigation
                    websocket={true}
                    follow={follow}
                    height={props.height}
                    stream={true}
                    onScroll={onScroll}
                    websocketOptions={{
                        onOpen: (_e: Event, socket: WebSocket) => {
                            console.log(
                                `Sending 'get-logs' message for container ${props.containerName} of pod ${props.podName}`,
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
                            console.warn(
                                `Websocket Log connection closed for container ${props.containerName} of pod ${props.podName}`,
                            );
                        },
                        onError: (e: Event) => {
                            console.error(
                                `Error encountered by Log Websocket for container ${props.containerName} of pod ${props.podName}: `,
                                e,
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
