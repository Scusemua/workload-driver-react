import React, { useEffect, useRef } from 'react';
import { Panel, PanelMain, PanelMainBody } from '@patternfly/react-core';
import { LazyLog } from '@melloware/react-logviewer';
import { useWebSocket } from 'react-use-websocket/dist/lib/use-websocket';
import { v4 as uuidv4 } from 'uuid';

export interface KubernetesLogViewProps {
    children?: React.ReactNode;
    podName: string;
    containerName: string;
    logPollIntervalSeconds: number;
    convertToHtml: boolean;
    signal: AbortSignal | undefined;
}

export const KubernetesLogViewComponent: React.FunctionComponent<KubernetesLogViewProps> = (props) => {
    // const url: string = `api/logs/pods/${props.podName}?container=${props.containerName}&follow=true`;

    const logs = useRef('');

    // Just use websockets. Ugh.
    const { sendMessage, lastMessage } = useWebSocket('ws://localhost:8000/logs');

    useEffect(() => {
        console.log(`Requesting logs for container ${props.containerName} of pod ${props.podName}`);
        sendMessage(
            JSON.stringify({
                op: 'get_logs',
                msg_id: uuidv4(),
                pod: props.podName,
                container: props.containerName,
                follow: true,
            }),
        );
    }, [sendMessage, props.podName, props.containerName]);

    useEffect(() => {
        async function readFromStream(reader: ReadableStreamDefaultReader<Uint8Array>) {
            let response: ReadableStreamReadResult<Uint8Array> = await reader.read();
            while (!response?.done) {
                const text: string = new TextDecoder().decode(response!.value);
                logs.current = logs.current + text;

                response = await reader.read();
            }
        }

        if (lastMessage !== null) {
            const data: Blob = lastMessage.data;
            const stream: ReadableStream<Uint8Array> = data.stream();
            const reader: ReadableStreamDefaultReader<Uint8Array> | undefined = stream.getReader();

            if (reader === undefined) {
                console.error('Could not get reader for message stream...');
                return;
            }

            readFromStream(reader);
        }
    }, [lastMessage]);

    return (
        <Panel isScrollable variant="bordered">
            <PanelMain maxHeight={'500px'}>
                <PanelMainBody>
                    <LazyLog
                        text={logs.current}
                        enableSearch
                        enableSearchNavigation
                        follow={true}
                        extraLines={1}
                        enableHotKeys
                        selectableLines
                        height={400}
                    />
                </PanelMainBody>
            </PanelMain>
        </Panel>
    );
};
