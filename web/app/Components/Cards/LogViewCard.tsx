/* eslint-disable camelcase */
import React, { useState } from 'react';
import {
    Button,
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Flex,
    FlexItem,
    Skeleton,
    Tab,
    TabContent,
    TabTitleIcon,
    TabTitleText,
    Tabs,
    Title,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
} from '@patternfly/react-core';

import {
    BugIcon,
    LaptopCodeIcon,
    ServerAltIcon,
    ServerGroupIcon,
    ServerIcon,
    StopIcon,
    SyncIcon,
} from '@patternfly/react-icons';
import { toast } from 'react-hot-toast';
import { ConsoleLogViewComponent } from '../ConsoleLogView';
import { KubernetesLogViewComponent } from '../KubernetesLogView';
import { useKernels, usePodNames } from '@app/Providers';
import { DistributedJupyterKernel, JupyterKernelReplica } from '@app/Data';
import { CloudServerIcon } from '@app/Icons';

export const LogViewCard: React.FunctionComponent = () => {
    const [activeTabKey, setActiveTabKey] = React.useState(0);
    const [activeLocalDaemonTabKey, setActiveLocalDaemonTabKey] = React.useState(0);
    const [activeKernelTabKey, setActiveKernelTabKey] = React.useState(0);
    const [activeKernelReplicaTabKey, setActiveKernelReplicaTabKey] = React.useState(0);
    const [podsAreRefreshing, setPodsAreRefreshing] = useState(false);

    const { gatewayPod, jupyterPod, refreshPodNames } = usePodNames();

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

    const abortController = React.useRef<AbortController | null>(null);
    if (abortController.current == null) {
        abortController.current = new AbortController();
    }

    const cardHeaderActions = (
        <ToolbarGroup variant="icon-button-group">
            <ToolbarItem>
                <Button
                    variant="plain"
                    icon={<StopIcon />}
                    onClick={() => {
                        try {
                            console.warn('Aborting all Kuberntes logs now.');
                            abortController.current?.abort();
                        } catch (error) {
                            console.error(`Error occurred whilst aborting: ${error}`);
                        }
                    }}
                />
            </ToolbarItem>
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
                                    refreshPodNames(),
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

    if (!gatewayPod || !jupyterPod || gatewayPod.length == 0 || jupyterPod.length == 0) {
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
                    <Skeleton height={'400'} />
                </CardBody>
            </Card>
        );
    }

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
                        tabContentId={`browser-console-logs-tab-content`}
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
                        tabContentId={`cluster-gateway-logs-tab-content`}
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
                        tabContentId={`jupyter-notebook-server-tab-content`}
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
                        tabContentId={`local-daemon-tab-content`}
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
                                        tabContentId={`local-daemon-${id}-tab-content`}
                                    />
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
                                                            signal={abortController.current?.signal}
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
                    id={`browser-console-logs-tab-content`}
                    activeKey={activeTabKey}
                    hidden={0 !== activeTabKey}
                >
                    <ConsoleLogViewComponent />
                </TabContent>
                <TabContent
                    key={1}
                    eventKey={1}
                    id={`cluster-gateway-logs-tab-content`}
                    activeKey={activeTabKey}
                    hidden={1 !== activeTabKey}
                >
                    <KubernetesLogViewComponent
                        podName={gatewayPod}
                        containerName={'gateway'}
                        logPollIntervalSeconds={1}
                        convertToHtml={false}
                        signal={abortController.current?.signal}
                    />
                </TabContent>
                <TabContent
                    key={2}
                    eventKey={2}
                    id={`jupyter-notebook-server-tab-content`}
                    activeKey={activeTabKey}
                    hidden={2 !== activeTabKey}
                >
                    <KubernetesLogViewComponent
                        podName={jupyterPod}
                        containerName={'jupyter-notebook'}
                        logPollIntervalSeconds={1}
                        convertToHtml={false}
                        signal={abortController.current?.signal}
                    />
                </TabContent>
                {localDaemonIDs.map((id: number) => (
                    <TabContent
                        key={id}
                        eventKey={id}
                        id={`local-daemon-${id}-tab-content`}
                        activeKey={activeLocalDaemonTabKey}
                        hidden={id !== activeLocalDaemonTabKey || 3 !== activeTabKey}
                    >
                        <KubernetesLogViewComponent
                            podName={`local-daemon-${id}`}
                            containerName={'local-daemon'}
                            logPollIntervalSeconds={1}
                            convertToHtml={false}
                            signal={abortController.current?.signal}
                        />
                    </TabContent>
                ))}
            </CardBody>
        </Card>
    );
};
