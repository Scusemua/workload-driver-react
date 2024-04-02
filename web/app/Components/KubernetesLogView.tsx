import React, { useEffect, useRef, useState } from 'react';
import { Divider, Panel, PanelMain, PanelMainBody, Title } from '@patternfly/react-core';
import { LazyLog, ScrollFollow } from '@melloware/react-logviewer';

export interface KubernetesLogViewProps {
    children?: React.ReactNode;
    podName: string;
    containerName: string;
    logPollIntervalSeconds: number;
    convertToHtml: boolean;
}

export const KubernetesLogViewComponent: React.FunctionComponent<KubernetesLogViewProps> = (props) => {
    const url: string = `api/logs/pods/${props.podName}?container=${props.containerName}&follow=true`;

    const alreadyGettingLogs = useRef(false);
    const logs = useRef('');

    useEffect(() => {
        async function get_logs(pod: string, container: string) {
            if (alreadyGettingLogs.current) {
                return;
            }

            alreadyGettingLogs.current = true;

            const req: RequestInit = {
                method: 'GET',
                headers: {
                    'Content-Type': 'text/plain',
                    'Transfer-Encoding': 'chunked',
                    'Cache-Control': 'no-cache, no-transform, no-store',
                },
            };

            const randNumber: number = Math.floor(Math.random() * 1e9); // ?randNumber=${randNumber}
            console.log(`Getting logs for container ${container} of pod ${pod}: ${randNumber}`);
            const response: Response = await fetch(
                `api/logs/pods/${pod}?randNumber=${randNumber}&container=${container}&follow=true`,
                req,
            );

            const reader: ReadableStreamDefaultReader<Uint8Array> | undefined = response.body?.getReader();

            while (true) {
                const response: ReadableStreamReadResult<Uint8Array> | undefined = await reader?.read();

                if (response?.done) {
                    return;
                }

                const logsAsString: string = String.fromCharCode.apply(null, response!.value);
                logs.current = logs.current + logsAsString;
            }
        }

        get_logs(props.podName, props.containerName);
    }, []);

    return (
        <Panel isScrollable variant="bordered">
            <PanelMain maxHeight={'500px'}>
                <PanelMainBody>
                    {/* <Title headingLevel="h1">{`Logs for Container ${props.containerName} of Pod ${props.podName}`}</Title>
                    <Divider /> */}
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
