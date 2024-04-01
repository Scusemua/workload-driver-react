import React, { useEffect, useRef } from 'react';
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

    // useEffect(() => {
    //     if (props.containerName !== 'jupyter-notebook') {
    //         return;
    //     }

    //     const req: RequestInit = {
    //         method: 'GET',
    //         headers: {
    //             'Content-Type': 'text/plain',
    //             'Transfer-Encoding': 'chunked',
    //         },
    //     };

    //     fetch(`api/logs/pods/${props.podName}?container=${props.containerName}&follow=true`, req).then((response) => {
    //         const reader: ReadableStreamDefaultReader<Uint8Array> | undefined = response.body?.getReader();

    //         function readChunk() {
    //             return reader?.read().then(({ done, value }) => {
    //                 if (done) {
    //                     return;
    //                 }

    //                 console.log(`Chunk received: ${String.fromCharCode.apply(null, value)}`);

    //                 return readChunk();
    //             });
    //         }

    //         return readChunk();
    //     });
    // }, []);

    console.log(`Querying data from \"${url}\"`);

    return (
        <Panel isScrollable variant="bordered">
            <PanelMain maxHeight={'450px'}>
                <PanelMainBody>
                    <Title headingLevel="h1">{`Logs for Container ${props.containerName} of Pod ${props.podName}`}</Title>
                    <Divider />
                    <ScrollFollow
                        startFollowing
                        render={({ onScroll, follow, startFollowing, stopFollowing }) => (
                            <LazyLog
                                stream={true}
                                extraLines={1}
                                url={url}
                                enableSearch
                                enableSearchNavigation
                                follow={follow}
                                onScroll={onScroll}
                                enableHotKeys
                                selectableLines
                                height={400}
                            />
                        )}
                    />
                </PanelMainBody>
            </PanelMain>
        </Panel>
    );
};
