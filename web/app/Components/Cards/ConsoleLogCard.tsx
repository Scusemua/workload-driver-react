/* eslint-disable camelcase */
import React, { useEffect, useState } from 'react';
import {
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Panel,
    PanelMain,
    PanelMainBody,
    Title,
    Tab,
    TabContent,
    TabTitleText,
    Tabs,
    DescriptionList,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    FlexItem,
    DescriptionListDescription,
} from '@patternfly/react-core';

import { Console, Hook, Unhook } from 'console-feed';
import { Message } from 'console-feed/lib/definitions/Console';
import { Message as MessageComponent } from 'console-feed/lib/definitions/Component';
import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';

export const ConsoleLogCard: React.FunctionComponent = () => {
    const [activeTabKey, setActiveTabKey] = React.useState(0);
    const [activeLocalDaemonTabKey, setActiveLocalDaemonTabKey] = React.useState(0);
    const [logs, setLogs] = useState<MessageComponent[]>([]);

    useEffect(() => {
        const hookedConsole = Hook(
            window.console,
            (log: Message) => setLogs((currLogs: MessageComponent[]) => [...currLogs, log as MessageComponent]),
            false,
        );
        return () => {
            Unhook(hookedConsole);
        };
    }, []);

    const handleTabClick = (_e: React.MouseEvent<HTMLElement, MouseEvent>, tabIndex: string | number) => {
        setActiveTabKey(Number(tabIndex));
    };

    const handleLocalDaemonTabClick = (_e: React.MouseEvent<HTMLElement, MouseEvent>, tabIndex: string | number) => {
        setActiveLocalDaemonTabKey(Number(tabIndex));
    };

    const descriptionListData = [
        {
            status: 'Running',
            resourceName: 'Resource name that is long and can wrap',
            detail: '121 Systems',
            icon: <CheckCircleIcon />,
        },
        {
            status: 'Ready',
            resourceName: 'Resource name that is long and can wrap',
            detail: '123 Systems',
            icon: <ExclamationCircleIcon />,
        },
        {
            status: 'Running',
            resourceName: 'Resource name that is long and can wrap',
            detail: '122 Systems',
            icon: <CheckCircleIcon />,
        },
        {
            status: 'Ready',
            resourceName: 'Resource name that is long and can wrap',
            detail: '124 Systems',
            icon: <ExclamationCircleIcon />,
        },
    ];

    const tabContent = (
        <DescriptionList isHorizontal columnModifier={{ lg: '2Col' }}>
            {descriptionListData.map(({ status, resourceName, detail, icon }, index) => (
                <DescriptionListGroup key={index}>
                    <DescriptionListTerm>
                        <Flex>
                            <FlexItem>{icon}</FlexItem>
                            <FlexItem>
                                <Title headingLevel="h4" size="md">
                                    {status}
                                </Title>
                            </FlexItem>
                        </Flex>
                    </DescriptionListTerm>
                    <DescriptionListDescription>
                        <a href="#">{resourceName}</a>
                        <div>{detail}</div>
                    </DescriptionListDescription>
                </DescriptionListGroup>
            ))}
        </DescriptionList>
    );

    const localDaemonIDs: number[] = [0, 1, 2, 3];

    return (
        <Card isRounded id="console-log-view-card">
            <CardHeader>
                <CardTitle>
                    <Title headingLevel="h1" size="xl">
                        Logs
                    </Title>
                </CardTitle>
            </CardHeader>
            <CardBody>
                <Tabs isFilled id="cluster-component-log-tabs" activeKey={activeTabKey} onSelect={handleTabClick}>
                    <Tab
                        key={0}
                        eventKey={0}
                        title={<TabTitleText>{'Browser Debug Console'}</TabTitleText>}
                        tabContentId={`tab-content-browser-debug-console`}
                    />
                    <Tab
                        key={1}
                        eventKey={1}
                        title={<TabTitleText>{'Cluster Gateway'}</TabTitleText>}
                        tabContentId={`tab-content-gateway`}
                    />
                    <Tab
                        key={2}
                        eventKey={2}
                        title={<TabTitleText>{'Jupyter Server'}</TabTitleText>}
                        tabContentId={`tab-content-jupyter-server`}
                    />
                    <Tab
                        key={3}
                        eventKey={3}
                        title={<TabTitleText>{'Local Daemons'}</TabTitleText>}
                        tabContentId={`tab-content-local-daemon-logs`}
                    >
                        <Tabs
                            isFilled
                            id="local-daemon-tabs"
                            activeKey={activeLocalDaemonTabKey}
                            onSelect={handleLocalDaemonTabClick}
                        >
                            {localDaemonIDs.map((id: number) => {
                                return (
                                    <Tab
                                        key={id}
                                        eventKey={id}
                                        title={<TabTitleText>{`Local Daemon ${id}`}</TabTitleText>}
                                        tabContentId={`tab-content-local-daemon${id}`}
                                    ></Tab>
                                );
                            })}
                        </Tabs>
                    </Tab>
                </Tabs>
            </CardBody>
            <CardBody>
                <TabContent
                    key={0}
                    eventKey={0}
                    id={`tabContent${0}`}
                    activeKey={activeTabKey}
                    hidden={0 !== activeTabKey}
                >
                    <Panel isScrollable variant="bordered">
                        <PanelMain maxHeight={'450px'}>
                            <PanelMainBody>
                                <Console logs={logs} variant="dark" />
                            </PanelMainBody>
                        </PanelMain>
                    </Panel>
                </TabContent>
                {localDaemonIDs.map((id: number) => (
                    <TabContent
                        key={id}
                        eventKey={id}
                        id={`local-daemin-${id}-tabcontent`}
                        activeKey={activeLocalDaemonTabKey}
                        hidden={id !== activeLocalDaemonTabKey || 3 !== activeTabKey}
                    >
                        <DescriptionList isHorizontal columnModifier={{ lg: '2Col' }}>
                            {descriptionListData.map(({ status, resourceName, detail, icon }, index) => (
                                <DescriptionListGroup key={index}>
                                    <DescriptionListTerm>
                                        <Flex>
                                            <FlexItem>{icon}</FlexItem>
                                            <FlexItem>
                                                <Title headingLevel="h4" size="md">
                                                    {status}
                                                </Title>
                                            </FlexItem>
                                        </Flex>
                                    </DescriptionListTerm>
                                    <DescriptionListDescription>
                                        <a href="#">{resourceName}</a>
                                        <div>{detail}</div>
                                    </DescriptionListDescription>
                                </DescriptionListGroup>
                            ))}
                        </DescriptionList>
                    </TabContent>
                ))}
            </CardBody>
        </Card>
    );
};
