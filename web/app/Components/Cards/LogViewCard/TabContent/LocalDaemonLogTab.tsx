/* eslint-disable camelcase */
import React from 'react';

import { Tab, TabTitleIcon, TabTitleText, Tabs } from '@patternfly/react-core';

import { ServerIcon } from '@patternfly/react-icons';
import { KubernetesPodLogView } from '@cards/LogViewCard/Views/';
import { LogHeightContext } from '../LogViewCard';

export interface LocalDaemonLogTabContentProps {
    children?: React.ReactNode;
    abortController: AbortController;
}

export const LocalDaemonLogTabContent: React.FunctionComponent<LocalDaemonLogTabContentProps> = (
    props: LocalDaemonLogTabContentProps,
) => {
    const logHeight = React.useContext(LogHeightContext);
    const [activeLocalDaemonTabKey, setActiveLocalDaemonTabKey] = React.useState(0);

    const handleLocalDaemonTabClick = (_e: React.MouseEvent<HTMLElement, MouseEvent>, tab: string | number) => {
        console.log(`Setting local daemon tab to ${tab}`);
        setActiveLocalDaemonTabKey(Number(tab));
    };

    const localDaemonIDs: number[] = [0, 1, 2, 3];

    return (
        <Tabs
            isSecondary
            isFilled
            id="local-daemon-tabs"
            activeKey={activeLocalDaemonTabKey}
            onSelect={handleLocalDaemonTabClick}
        >
            {localDaemonIDs.map((id: number) => {
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
