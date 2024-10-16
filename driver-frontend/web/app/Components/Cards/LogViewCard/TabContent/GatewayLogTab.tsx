/* eslint-disable camelcase */
import React from 'react';

import { Tab, TabTitleIcon, TabTitleText, Tabs } from '@patternfly/react-core';

import { ServerIcon } from '@patternfly/react-icons';
import { KubernetesPodLogView } from '@cards/LogViewCard/Views/';
import { LogHeightContext } from '../LogViewCard';
import { SchedulerIcon } from '@app/Icons';

export interface GatewayLogTabContentProps {
    children?: React.ReactNode;
    abortController: AbortController;
    gatewayPodName: string;
}

export const GatewayLogTabContent: React.FunctionComponent<GatewayLogTabContentProps> = (
    props: GatewayLogTabContentProps,
) => {
    const logHeight = React.useContext(LogHeightContext);
    const [activeGatewayTabKey, setActiveGatewayTabKey] = React.useState(0);

    const handleGatewayTabClick = (_e: React.MouseEvent<HTMLElement, MouseEvent>, tab: string | number) => {
        console.log(`Setting gateway tab to ${tab}`);
        setActiveGatewayTabKey(Number(tab));
    };

    return (
        <Tabs
            isSecondary
            isFilled
            id="local-daemon-tabs"
            activeKey={activeGatewayTabKey}
            onSelect={handleGatewayTabClick}
        >
            <Tab
                id={`gateway-gateway-tab`}
                key={0}
                eventKey={0}
                title={
                    <>
                        <TabTitleIcon>
                            <ServerIcon />
                        </TabTitleIcon>
                        <TabTitleText>{`Gateway Container`}</TabTitleText>
                    </>
                }
            >
                <KubernetesPodLogView
                    podName={props.gatewayPodName}
                    containerName={'gateway'}
                    logPollIntervalSeconds={1}
                    convertToHtml={false}
                    signal={props.abortController.signal}
                    height={logHeight}
                />
            </Tab>
            <Tab
                id={`gateway-scheduler-tab`}
                key={1}
                eventKey={1}
                title={
                    <>
                        <TabTitleIcon>
                            <SchedulerIcon scale={2} />
                        </TabTitleIcon>
                        <TabTitleText>{`2nd Kube Scheduler`}</TabTitleText>
                    </>
                }
            >
                <KubernetesPodLogView
                    podName={props.gatewayPodName}
                    containerName={'scheduler-ctr'}
                    logPollIntervalSeconds={1}
                    convertToHtml={false}
                    signal={props.abortController.signal}
                    height={logHeight}
                />
            </Tab>
            <Tab
                id={`gateway-scheduler-extender-tab`}
                key={2}
                eventKey={2}
                title={
                    <>
                        <TabTitleIcon>
                            <ServerIcon />
                        </TabTitleIcon>
                        <TabTitleText>{`Scheduler Extender`}</TabTitleText>
                    </>
                }
            >
                <KubernetesPodLogView
                    podName={props.gatewayPodName}
                    containerName={'scheduler-extender-ctr'}
                    logPollIntervalSeconds={1}
                    convertToHtml={false}
                    signal={props.abortController.signal}
                    height={logHeight}
                />
            </Tab>
        </Tabs>
    );
};
