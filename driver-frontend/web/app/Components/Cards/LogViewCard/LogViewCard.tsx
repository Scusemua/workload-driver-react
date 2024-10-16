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
    TabTitleIcon,
    TabTitleText,
    Tabs,
    TextInput,
    TextInputProps,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
} from '@patternfly/react-core';

import { BugIcon, LaptopCodeIcon, ServerAltIcon, ServerGroupIcon, SyncIcon } from '@patternfly/react-icons';
import { toast } from 'react-hot-toast';
import { BrowserDebugConsoleLogView, KubernetesPodLogView } from '@cards/LogViewCard/Views/';
import { GatewayLogTabContent, KernelLogTabContent, LocalDaemonLogTabContent } from '@cards/LogViewCard/TabContent';
import { usePodNames } from '@app/Providers';
import { CloudServerIcon } from '@app/Icons';

const default_card_height: number = 400;
const min_card_height: number = 100;
const max_card_height: number = 2500;

export const LogHeightContext = React.createContext(default_card_height);

export const LogViewCard: React.FunctionComponent = () => {
    const [activeTabKey, setActiveTabKey] = React.useState(0);
    const [podsAreRefreshing, setPodsAreRefreshing] = useState(false);
    const [logHeight, setLogHeight] = useState(default_card_height);
    const [logHeightString, setLogHeightString] = React.useState(default_card_height.toString());
    const [logHeightValidated, setLogHeightValidated] = React.useState<TextInputProps['validated']>('default');

    const { gatewayPod, jupyterPod, refreshPodNames } = usePodNames();

    const handleTabClick = (_e: React.MouseEvent<HTMLElement, MouseEvent>, tabIndex: string | number) => {
        setActiveTabKey(Number(tabIndex));
    };

    const abortController = React.useRef<AbortController | null>(null);
    if (abortController.current == null) {
        abortController.current = new AbortController();
    }

    const onHeightTextboxChanged = (_event: React.FormEvent<HTMLInputElement>, value: string) => {
        setLogHeightString(value);

        if (value == '') {
            setLogHeight(default_card_height);
            return;
        }

        const height: number = Number.parseInt(value);
        if (Number.isNaN(height)) {
            setLogHeightValidated('error');
            return;
        }

        if (height < min_card_height) {
            setLogHeightValidated('error');
            return;
        }

        if (height > max_card_height) {
            setLogHeightValidated('error');
            return;
        }

        setLogHeightValidated('default');
        setLogHeight(height);
    };

    const cardHeaderActions = (
        <Toolbar>
            <ToolbarContent>
                <React.Fragment>
                    <ToolbarGroup>
                        <ToolbarItem>
                            <Tooltip
                                id="log-card-height-text-input-tooltip"
                                aria-label="log-card-height-text-input-tooltip"
                                exitDelay={75}
                                content={<div>Specify the height of the &quot;Logs&quot; card.</div>}
                            >
                                <TextInput
                                    aria-label="log-card-height-text-input"
                                    id="log-card-height-text-input"
                                    placeholder={logHeight.toString()}
                                    value={logHeightString}
                                    type="number"
                                    validated={logHeightValidated}
                                    onChange={onHeightTextboxChanged}
                                />
                            </Tooltip>
                        </ToolbarItem>
                    </ToolbarGroup>
                    <ToolbarGroup variant="icon-button-group">
                        <ToolbarItem>
                            <Tooltip exitDelay={75} content={<div>Refresh container names.</div>}>
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
                                        const promise = toast
                                            .promise(
                                                refreshPodNames(),
                                                {
                                                    loading: <b>Refreshing Kubernetes container names...</b>,
                                                    success: <b>Refreshed Kubernetes container names!</b>,
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
                                                                    <b>Could not refresh Kuberentes container names.</b>
                                                                </FlexItem>
                                                                {reasonUI}
                                                            </Flex>
                                                        );
                                                    },
                                                },
                                                {
                                                    style: {
                                                        padding: '8px',
                                                        minWidth: '425px',
                                                    },
                                                    id: "manuallyRefreshPodNames",
                                                },
                                            );
                                        // Need to do this whole rigmarole so that, if the above fails for whatever reason, there's not an "unhandled error" exception.
                                        // If the above fails due to the fetcher in the PodNameProvider failing and throwing an error, we need to catch the error explicitly here.
                                        promise.then(() => {
                                            setPodsAreRefreshing(false);
                                        }).catch((reason: Error) => { // Explicitly catch the potential error.
                                        toast.error(() => {
                                            setPodsAreRefreshing(false);

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
                                                        <b>Could not refresh Kuberentes container names.</b>
                                                    </FlexItem>
                                                    {reasonUI}
                                                </Flex>
                                            );
                                        }, {
                                            style: {
                                                padding: '8px',
                                                minWidth: '425px',
                                            },
                                            id: "manuallyRefreshPodNames"
                                        })
                                    })
                                    }}
                                    icon={<SyncIcon />}
                                />
                            </Tooltip>
                        </ToolbarItem>
                    </ToolbarGroup>
                </React.Fragment>
            </ToolbarContent>
        </Toolbar>
    );

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
                    <div style={{ height: '200px' }}>
                        <Skeleton height="15%" width="40%" style={{ float: 'left', margin: '8px' }} />
                        <Skeleton height="15%" width="40%" style={{ float: 'left', margin: '8px' }} />
                        <Skeleton height="15%" width="10%" style={{ float: 'left', margin: '8px' }} />
                        <Skeleton height="15%" width="5%" style={{ float: 'left', margin: '8px' }} />

                        <Skeleton height="15%" width="10%" style={{ float: 'left', margin: '8px' }} />
                        <Skeleton height="15%" width="20%" style={{ float: 'left', margin: '8px' }} />
                        <Skeleton height="15%" width="30%" style={{ float: 'left', margin: '8px' }} />
                        <Skeleton height="15%" width="35%" style={{ float: 'left', margin: '8px' }} />

                        <Skeleton height="15%" width="25%" style={{ float: 'left', margin: '8px' }} />
                        <Skeleton height="15%" width="45%" style={{ float: 'left', margin: '8px' }} />
                        <Skeleton height="15%" width="10%" style={{ float: 'left', margin: '8px' }} />
                        <Skeleton height="15%" width="15%" style={{ float: 'left', margin: '8px' }} />

                        <Skeleton height="15%" width="15%" style={{ float: 'left', margin: '8px' }} />
                        <Skeleton height="15%" width="25%" style={{ float: 'left', margin: '8px' }} />
                        <Skeleton height="15%" width="45%" style={{ float: 'left', margin: '8px' }} />
                        <Skeleton height="15%" width="10%" style={{ float: 'left', margin: '8px' }} />

                        <Skeleton height="15%" width="20%" style={{ float: 'left', margin: '8px' }} />
                        <Skeleton height="15%" width="35%" style={{ float: 'left', margin: '8px' }} />
                        <Skeleton height="15%" width="20%" style={{ float: 'left', margin: '8px' }} />
                        <Skeleton height="15%" width="20%" style={{ float: 'left', margin: '8px' }} />

                        <Skeleton height="15%" width="25%" style={{ float: 'left', margin: '8px' }} />
                        <Skeleton height="15%" width="40%" style={{ float: 'left', margin: '8px' }} />
                        <Skeleton height="15%" width="20%" style={{ float: 'left', margin: '8px' }} />
                        <Skeleton height="15%" width="10%" style={{ float: 'left', margin: '8px' }} />
                    </div>
                </CardBody>
            </Card>
        );
    }

    return (
        <LogHeightContext.Provider value={logHeight}>
            <Card isRounded id="console-log-view-card" isFullHeight>
                <CardHeader actions={{ actions: cardHeaderActions, hasNoOffset: true }}>
                    <CardTitle>
                        <Title headingLevel="h1" size="xl">
                            Logs
                        </Title>
                    </CardTitle>
                </CardHeader>
                <CardBody>
                    <Tabs isFilled id="cluster-component-log-tabs" activeKey={activeTabKey} onSelect={handleTabClick}>
                        <Tab
                            aria-label={'browser-console-logs-tab'}
                            id={'browser-console-logs-tab'}
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
                        >
                            <BrowserDebugConsoleLogView height={logHeight} />
                        </Tab>
                        <Tab
                            aria-label={'cluster-gateway-logs-tab'}
                            id={'cluster-gateway-logs-tab'}
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
                        >
                            {/* {gatewayPod !== '' && (
                                <KubernetesPodLogView
                                    podName={gatewayPod}
                                    containerName={'gateway'}
                                    logPollIntervalSeconds={1}
                                    convertToHtml={false}
                                    signal={abortController.current?.signal}
                                    height={logHeight}
                                />
                            )} */}
                            {gatewayPod !== '' && (
                                <GatewayLogTabContent
                                    abortController={abortController.current}
                                    gatewayPodName={gatewayPod}
                                />
                            )}
                        </Tab>
                        <Tab
                            aria-label={'jupyter-server-logs-tab'}
                            id={'jupyter-server-logs-tab'}
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
                        >
                            {jupyterPod !== '' && (
                                <KubernetesPodLogView
                                    podName={jupyterPod}
                                    containerName={'jupyter-notebook'}
                                    logPollIntervalSeconds={1}
                                    convertToHtml={false}
                                    signal={abortController.current?.signal}
                                    height={logHeight}
                                />
                            )}
                        </Tab>
                        <Tab
                            aria-label={'local-daemons-logs-tab'}
                            id={'local-daemons-logs-tab'}
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
                            <LocalDaemonLogTabContent abortController={abortController.current} />
                        </Tab>
                        <Tab
                            aria-label={'kernels-logs-tab'}
                            id={'kernels-logs-tab'}
                            key={4}
                            eventKey={4}
                            title={
                                <>
                                    <TabTitleIcon>
                                        <CloudServerIcon scale={1.25} />
                                    </TabTitleIcon>
                                    <TabTitleText>{`Kernels`}</TabTitleText>
                                </>
                            }
                            tabContentId={'kernel-tab-content'}
                        >
                            <KernelLogTabContent abortController={abortController.current} />
                        </Tab>
                    </Tabs>
                </CardBody>
            </Card>
        </LogHeightContext.Provider>
    );
};
