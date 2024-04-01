/* eslint-disable camelcase */
import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Title,
    Tab,
    TabContent,
    TabTitleText,
    Tabs,
    Flex,
    FlexItem,
    Button,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
    TabTitleIcon,
    CardExpandableContent,
} from '@patternfly/react-core';

import { BugIcon, LaptopCodeIcon, ServerAltIcon, ServerGroupIcon, ServerIcon, SyncIcon } from '@patternfly/react-icons';
import { toast } from 'react-hot-toast';
import { ConsoleLogViewComponent } from '../ConsoleLogView';
import { KubernetesLogViewComponent } from '../KubernetesLogView';
import { useKernels } from '@app/Providers';
import { DistributedJupyterKernel, JupyterKernelReplica } from '@app/Data';
import { CloudServerIcon, ClusterIcon } from '@app/Icons';
import { LazyLog } from '@melloware/react-logviewer';

export const LogViewCard: React.FunctionComponent = () => {
    const [activeTabKey, setActiveTabKey] = React.useState(0);
    const [activeLocalDaemonTabKey, setActiveLocalDaemonTabKey] = React.useState(0);
    const [activeKernelTabKey, setActiveKernelTabKey] = React.useState(0);
    const [activeKernelReplicaTabKey, setActiveKernelReplicaTabKey] = React.useState(0);
    const [podsAreRefreshing, setPodsAreRefreshing] = useState(false);

    const [isCardExpanded, setIsCardExpanded] = useState(true);

    const [gatewayPod, setGatewayPod] = React.useState('');
    const [jupyterPod, setJupyterPod] = React.useState('');

    const { kernels } = useKernels();

    const handleTabClick = (_e: React.MouseEvent<HTMLElement, MouseEvent>, tabIndex: string | number) => {
        setActiveTabKey(Number(tabIndex));
    };

    const handleLocalDaemonTabClick = (_e: React.MouseEvent<HTMLElement, MouseEvent>, tabIndex: string | number) => {
        setActiveLocalDaemonTabKey(Number(tabIndex));
    };

    const handleKernelTabClick = (_e: React.MouseEvent<HTMLElement, MouseEvent>, tabIndex: string | number) => {
        setActiveKernelTabKey(Number(tabIndex));
    };

    const handleKernelReplicaTabClick = (_e: React.MouseEvent<HTMLElement, MouseEvent>, tabIndex: string | number) => {
        setActiveKernelReplicaTabKey(Number(tabIndex));
    };

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
                console.log(`Identified Gateway Pod: ${podName}`);
                setGatewayPod(podName);
            } else if (podName.includes('jupyter')) {
                console.log(`Identified Jupyter Pod: ${podName}`);
                setJupyterPod(podName);
            }
        });
    }, [setGatewayPod, setJupyterPod]);

    useEffect(() => {
        refreshPods();
    }, [refreshPods]);

    const cardHeaderActions = (
        <ToolbarGroup variant="icon-button-group">
            <ToolbarItem>
                <Tooltip exitDelay={75} content={<div>Refresh pod names.</div>}>
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
                                        loading: <b>Refreshing Kubernetes pod names...</b>,
                                        success: <b>Refreshed Kubernetes pod names!</b>,
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
                                                        <b>Could not refresh Kuberentes pod names.</b>
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

    const onCardExpand = (event: React.MouseEvent, id: string) => {
        setIsCardExpanded(!isCardExpanded);
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
                        title={
                            <>
                                <TabTitleIcon>
                                    <BugIcon />
                                </TabTitleIcon>
                                <TabTitleText>{'Browser Debug Console'}</TabTitleText>
                            </>
                        }
                        tabContentId={`tab-content-browser-debug-console`}
                    />
                    <Tab
                        key={1}
                        eventKey={1}
                        title={
                            <>
                                <TabTitleIcon>
                                    <ServerAltIcon />
                                </TabTitleIcon>
                                <TabTitleText>{'Cluster Gateway'}</TabTitleText>
                            </>
                        }
                        tabContentId={`tab-content-gateway`}
                    />
                    <Tab
                        key={2}
                        eventKey={2}
                        title={
                            <>
                                <TabTitleIcon>
                                    <LaptopCodeIcon />
                                </TabTitleIcon>
                                <TabTitleText>{'Jupyter Server'}</TabTitleText>
                            </>
                        }
                        tabContentId={`tab-content-jupyter-server`}
                    />
                    <Tab
                        key={3}
                        eventKey={3}
                        title={
                            <>
                                <TabTitleIcon>
                                    <ServerGroupIcon />
                                </TabTitleIcon>
                                <TabTitleText>{'Local Daemons'}</TabTitleText>
                            </>
                        }
                        tabContentId={`tab-content-local-daemon-browserConsoleLogs`}
                    >
                        <Tabs
                            isFilled
                            id="local-daemon-tabs"
                            activeKey={activeLocalDaemonTabKey}
                            onSelect={handleLocalDaemonTabClick}
                            isBox={true}
                        >
                            {localDaemonIDs.map((id: number) => {
                                return (
                                    <Tab
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
                                        tabContentId={`tab-content-local-daemon${id}`}
                                    ></Tab>
                                );
                            })}
                        </Tabs>
                    </Tab>
                    <Tab
                        key={4}
                        eventKey={4}
                        isDisabled={kernels.length == 0}
                        title={
                            <>
                                <TabTitleIcon>
                                    <CloudServerIcon scale={1.25} />
                                </TabTitleIcon>
                                <TabTitleText>{`Kernels`}</TabTitleText>
                            </>
                        }
                        tabContentId={'tab-content-kernels'}
                    >
                        <Tabs isFilled id="kernel-tabs" activeKey={activeKernelTabKey} onSelect={handleKernelTabClick}>
                            {kernels.map((kernel: DistributedJupyterKernel, idx: number) => {
                                return (
                                    <Tab
                                        key={idx}
                                        eventKey={idx}
                                        title={
                                            <>
                                                <TabTitleIcon>
                                                    <ServerIcon />
                                                </TabTitleIcon>
                                                <TabTitleText>{`Kernel ${kernel.kernelId.slice(
                                                    0,
                                                    8,
                                                )}...`}</TabTitleText>
                                            </>
                                        }
                                        tabContentId={`tab-content-kernel-${kernel.kernelId}`}
                                    >
                                        <Tabs
                                            isFilled
                                            id="kernel-tabs"
                                            activeKey={activeKernelReplicaTabKey}
                                            onSelect={handleKernelReplicaTabClick}
                                            isBox={true}
                                        >
                                            {kernel?.replicas?.map((replica: JupyterKernelReplica) => {
                                                return (
                                                    <Tab
                                                        key={replica.replicaId}
                                                        eventKey={replica.replicaId}
                                                        title={
                                                            <>
                                                                <TabTitleIcon>
                                                                    <ServerIcon />
                                                                </TabTitleIcon>
                                                                <TabTitleText>{`Replica ${replica.replicaId}`}</TabTitleText>
                                                            </>
                                                        }
                                                        tabContentId={`tab-content-kernel-${kernel.kernelId}-${replica.replicaId}`}
                                                    >
                                                        <KubernetesLogViewComponent
                                                            podName={replica.podId}
                                                            containerName="kernel"
                                                            convertToHtml={false}
                                                            logPollIntervalSeconds={1}
                                                        />
                                                    </Tab>
                                                );
                                            })}
                                        </Tabs>
                                    </Tab>
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
                    <ConsoleLogViewComponent />
                </TabContent>
                <TabContent
                    key={1}
                    eventKey={1}
                    id={`tabContent${1}`}
                    activeKey={activeTabKey}
                    hidden={1 !== activeTabKey}
                >
                    {gatewayPod.length > 0 && (
                        <KubernetesLogViewComponent
                            podName={gatewayPod}
                            containerName={'gateway'}
                            logPollIntervalSeconds={1}
                            convertToHtml={false}
                        />
                    )}
                </TabContent>
                <TabContent
                    key={2}
                    eventKey={2}
                    id={`tabContent${2}`}
                    activeKey={activeTabKey}
                    hidden={2 !== activeTabKey}
                >
                    {jupyterPod.length > 0 && (
                        <KubernetesLogViewComponent
                            podName={jupyterPod}
                            containerName={'jupyter-notebook'}
                            logPollIntervalSeconds={1}
                            convertToHtml={false}
                        />
                    )}
                </TabContent>
                {localDaemonIDs.map((id: number) => (
                    <TabContent
                        key={id}
                        eventKey={id}
                        id={`local-daemin-${id}-tabcontent`}
                        activeKey={activeLocalDaemonTabKey}
                        hidden={id !== activeLocalDaemonTabKey || 3 !== activeTabKey}
                    >
                        <KubernetesLogViewComponent
                            podName={`local-daemon-${id}`}
                            containerName={'local-daemon'}
                            logPollIntervalSeconds={1}
                            convertToHtml={false}
                        />
                    </TabContent>
                ))}
            </CardBody>
        </Card>
    );
};
