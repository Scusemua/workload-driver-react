import React, { useCallback, useRef } from 'react';
import { AnsiUp } from 'ansi_up';
import {
    Button,
    Chip,
    Grid,
    GridItem,
    Panel,
    PanelMain,
    PanelMainBody,
    Switch,
    useInterval,
} from '@patternfly/react-core';

export interface KubernetesLogViewProps {
    children?: React.ReactNode;
    podName: string;
    containerName: string;
    logPollIntervalSeconds: number;
    convertToHtml: boolean;
}

const ansi_up = new AnsiUp();

export const KubernetesLogViewComponent: React.FunctionComponent<KubernetesLogViewProps> = (props) => {
    const logs = useRef<string>('');

    const fetchLogs = useCallback(async (podName: string, containerName: string) => {
        let url: string = `kubernetes/api/v1/namespaces/default/pods/${podName}/log?container=${containerName}`;
        if (logs.current.length > 0) {
            url = `kubernetes/api/v1/namespaces/default/pods/${podName}/log?container=${containerName}&sinceSeconds=${props.logPollIntervalSeconds}`;
        }

        const resp: Response = await fetch(url);
        const latestLogs: string = await resp.text();

        // Only update if we fetched some new logs.
        if (latestLogs.length > 0) {
            if (props.convertToHtml) {
                logs.current += ansi_up.ansi_to_html(latestLogs);
            } else {
                logs.current += latestLogs;
            }
            var cdiv = document.getElementById(`${props.podName}-${props.containerName}-console`);
            if (cdiv) {
                cdiv.innerHTML = logs.current;
            }
        }
    }, []);

    useInterval(() => fetchLogs(props.podName, props.containerName), props.logPollIntervalSeconds * 1000);

    return (
        <Panel isScrollable variant="bordered">
            <PanelMain maxHeight={'450px'}>
                <PanelMainBody>
                    <pre id={`${props.podName}-${props.containerName}-console`}></pre>
                </PanelMainBody>
            </PanelMain>
        </Panel>
    );
};
