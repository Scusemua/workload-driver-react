/* eslint-disable camelcase */
import React, { useCallback, useEffect, useState } from 'react';
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
    Button,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
    CodeBlock,
    CodeBlockCode,
} from '@patternfly/react-core';

import { Console, Hook, Unhook } from 'console-feed';
import { Message } from 'console-feed/lib/definitions/Console';
import { Message as MessageComponent } from 'console-feed/lib/definitions/Component';
import { CheckCircleIcon, ExclamationCircleIcon, SyncIcon } from '@patternfly/react-icons';
import { toast } from 'react-hot-toast';

export const ConsoleLogCard: React.FunctionComponent = () => {
    const [activeTabKey, setActiveTabKey] = React.useState(0);
    const [activeLocalDaemonTabKey, setActiveLocalDaemonTabKey] = React.useState(0);
    const [browserConsoleLogs, setLogsBrowserConsoleLogs] = useState<MessageComponent[]>([]);
    const [podsAreRefreshing, setPodsAreRefreshing] = useState(false);

    const [gatewayPodLogs, setGatewayPodLogs] = useState('');
    const [jupyterPodLogs, setJupyterPodLogs] = useState('');

    const [gatewayPod, setGatewayPod] = React.useState('');
    const [jupyterPod, setJupyterPod] = React.useState('');

    const [localDaemonLogs, setLocalDaemonLogs] = React.useState<Map<number, string>>(new Map());

    useEffect(() => {
        const hookedConsole = Hook(
            window.console,
            (log: Message) =>
                setLogsBrowserConsoleLogs((currLogs: MessageComponent[]) => [...currLogs, log as MessageComponent]),
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

    const fetchLogs = useCallback(
        async (podName: string, containerName: string) => {
            console.log(`Fetching logs for container ${containerName} of pod ${podName} now...`);
            const resp: Response = await fetch(
                `kubernetes/api/v1/namespaces/default/pods/${podName}/log?container=${containerName}`,
            );
            return await resp.text();
        },
        [setGatewayPodLogs, setJupyterPodLogs],
    );

    const refreshPods = useCallback(async () => {
        const requestOptions = {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
                'Sec-Fetch-Dest': 'document',
            },
        };

        console.log('Retrieving Pods now.');
        const response: Response = await fetch('kubernetes/api/v1/namespaces/default/pods', requestOptions);
        console.log(`Response for Pods refresh: ${response.status} ${response.statusText}`);
        const responseJson: Record<string, any> = await response.json();

        const podsJson: Record<string, any>[] = responseJson['items'];
        podsJson.map((pod: Record<string, any>) => {
            const podName: string = pod['metadata']['name'];
            const containerName: string = pod['spec']['containers'][0]['name'];
            console.log(`Discovered Pod ${podName} with Container ${containerName}`);

            if (podName.includes('gateway')) {
                fetchLogs(podName, containerName).then((logs: string) => {
                    setGatewayPod(podName);
                    setGatewayPodLogs(logs);
                });
            } else if (podName.includes('jupyter')) {
                fetchLogs(podName, containerName).then((logs: string) => {
                    setJupyterPod(podName);
                    setJupyterPodLogs(logs);
                });
            } else if (podName.includes('local-daemon')) {
                fetchLogs(podName, containerName).then((logs: string) => {
                    const idx: number = Number.parseInt(podName.slice(podName.lastIndexOf('-') + 1, podName.length));
                    setLocalDaemonLogs((m) => new Map(m).set(idx, logs));
                });
            }
        });
    }, [setGatewayPod, setJupyterPod, fetchLogs]);

    useEffect(() => {
        refreshPods();
    }, [refreshPods]);

    const cardHeaderActions = (
        <ToolbarGroup variant="icon-button-group">
            <ToolbarItem>
                <Tooltip exitDelay={75} content={<div>Refresh kernel specs.</div>}>
                    <Button
                        label="refresh-kernel-specs-button"
                        aria-label="refresh-kernel-specs-button"
                        variant="plain"
                        isDisabled={podsAreRefreshing}
                        className={
                            (podsAreRefreshing && 'loading-icon-spin-toggleable') ||
                            'loading-icon-spin-toggleable paused'
                        }
                        onClick={() => {
                            setPodsAreRefreshing(true);
                            toast
                                .promise(
                                    refreshPods(),
                                    {
                                        loading: <b>Refreshing Kubernetes Pods...</b>,
                                        success: <b>Refreshed Kubernetes Pods!</b>,
                                        error: (reason: Error) => {
                                            let reasonUI = <FlexItem>{reason.message}</FlexItem>;

                                            if (reason.message.includes("Unexpected token 'E'")) {
                                                reasonUI = <FlexItem>HTTP 504: Gateway Timeout</FlexItem>;
                                            }

                                            return (
                                                <Flex
                                                    direction={{ default: 'column' }}
                                                    spaceItems={{ default: 'spaceItemsNone' }}
                                                >
                                                    <FlexItem>
                                                        <b>Could not refresh Kuberentes Pods.</b>
                                                    </FlexItem>
                                                    {reasonUI}
                                                </Flex>
                                            );
                                        },
                                    },
                                    {
                                        style: {
                                            padding: '8px',
                                        },
                                    },
                                )
                                .then(() => {
                                    setPodsAreRefreshing(false);
                                });
                        }}
                        icon={<SyncIcon />}
                    />
                </Tooltip>
            </ToolbarItem>
        </ToolbarGroup>
    );

    const localDaemonIDs: number[] = [0, 1, 2, 3];

    const getLocalDaemonTabContent = (idx: number) => {
        return (
            <Panel isScrollable variant="bordered">
                <PanelMain maxHeight={'450px'}>
                    <PanelMainBody>
                        <CodeBlock>
                            <CodeBlockCode id="code-content">
                                {localDaemonLogs.get(idx) || `No logs available for Local Daemon ${idx}`}
                            </CodeBlockCode>
                        </CodeBlock>
                    </PanelMainBody>
                </PanelMain>
            </Panel>
        );
    };

    return (
        <Card isRounded id="console-log-view-card">
            <CardHeader actions={{ actions: cardHeaderActions, hasNoOffset: false }}>
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
                        tabContentId={`tab-content-local-daemon-browserConsoleLogs`}
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
                                <Console logs={browserConsoleLogs} variant="dark" />
                            </PanelMainBody>
                        </PanelMain>
                    </Panel>
                </TabContent>
                <TabContent
                    key={1}
                    eventKey={1}
                    id={`tabContent${1}`}
                    activeKey={activeTabKey}
                    hidden={1 !== activeTabKey}
                >
                    <Panel isScrollable variant="bordered">
                        <PanelMain maxHeight={'450px'}>
                            <PanelMainBody>
                                <CodeBlock>
                                    <CodeBlockCode id="code-content">{gatewayPodLogs}</CodeBlockCode>
                                </CodeBlock>
                            </PanelMainBody>
                        </PanelMain>
                    </Panel>
                </TabContent>
                <TabContent
                    key={2}
                    eventKey={2}
                    id={`tabContent${2}`}
                    activeKey={activeTabKey}
                    hidden={2 !== activeTabKey}
                >
                    <Panel isScrollable variant="bordered">
                        <PanelMain maxHeight={'450px'}>
                            <PanelMainBody>
                                <CodeBlock>
                                    <CodeBlockCode id="code-content">{jupyterPodLogs}</CodeBlockCode>
                                </CodeBlock>
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
                        {getLocalDaemonTabContent(id)}
                    </TabContent>
                ))}
            </CardBody>
        </Card>
    );
};
