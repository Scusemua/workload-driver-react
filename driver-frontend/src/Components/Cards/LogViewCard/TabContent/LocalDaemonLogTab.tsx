/* eslint-disable camelcase */
import React from 'react';

import { Tab, TabTitleIcon, TabTitleText, Tabs } from '@patternfly/react-core';

import { ServerIcon } from '@patternfly/react-icons';
import { KubernetesPodLogView } from '@Cards/LogViewCard/Views/';
import { LogHeightContext } from '../LogViewCard';
import { useNodes } from '@src/Providers';

export interface LocalDaemonLogTabContentProps {
    children?: React.ReactNode;
    abortController: AbortController;
}

export const LocalDaemonLogTabContent: React.FunctionComponent<LocalDaemonLogTabContentProps> = (
    props: LocalDaemonLogTabContentProps,
) => {
    const { nodes } = useNodes();
    const [numNodes, setNumNodes] = React.useState(0);

    const logHeight = React.useContext(LogHeightContext);
    const [activeLocalDaemonTabKey, setActiveLocalDaemonTabKey] = React.useState(0);

    const handleLocalDaemonTabClick = (_e: React.MouseEvent<HTMLElement, MouseEvent>, tab: string | number) => {
        console.log(`Setting local daemon tab to ${tab}`);
        setActiveLocalDaemonTabKey(Number(tab));
    };

    React.useEffect(() => {
        setNumNodes(nodes.length);
    }, [nodes]);

    return (
        <Tabs
            isSecondary
            isFilled
            id="local-daemon-tabs"
            activeKey={activeLocalDaemonTabKey}
            onSelect={handleLocalDaemonTabClick}
        >
            {Array.from(Array(numNodes).keys()).map((_val: number, id: number) => {
                return (
                    <Tab
                        id={`local-daemon-${id}-tab`}
                        key={id}
                        eventKey={id}
                        title={
                            <>
                                <TabTitleIcon>
                                    <ServerIcon />
                                </TabTitleIcon>
                                <TabTitleText>{`Local Daemon ${id}`}</TabTitleText>
                            </>
                        }
                    >
                        <KubernetesPodLogView
                            podName={`local-daemon-${id}`}
                            containerName={'local-daemon'}
                            logPollIntervalSeconds={1}
                            convertToHtml={false}
                            signal={props.abortController.signal}
                            height={logHeight}
                        />
                    </Tab>
                );
            })}
        </Tabs>
    );
};
