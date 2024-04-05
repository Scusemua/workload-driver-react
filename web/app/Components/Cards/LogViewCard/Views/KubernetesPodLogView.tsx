import React from 'react';
import { LazyLog, ScrollFollow } from '@melloware/react-logviewer';
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
    return (
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
                            console.debug(
                                `Websocket Log connection closed for container ${props.containerName} of pod ${props.podName}`,
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
