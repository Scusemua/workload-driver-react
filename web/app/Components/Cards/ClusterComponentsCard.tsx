import React, { useCallback, useEffect, useRef } from 'react';
import {
    Badge,
    Button,
    Card,
    CardBody,
    CardExpandableContent,
    CardHeader,
    CardTitle,
    DataList,
    DataListAction,
    DataListCell,
    DataListCheck,
    DataListContent,
    DataListItem,
    DataListItemCells,
    DataListItemRow,
    DataListToggle,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    FlexItem,
    InputGroup,
    InputGroupItem,
    Menu,
    MenuContent,
    MenuItem,
    MenuList,
    MenuToggle,
    Pagination,
    PaginationVariant,
    Popper,
    SearchInput,
    Tab,
    TabContent,
    Tabs,
    TabTitleIcon,
    TabTitleText,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarFilter,
    ToolbarGroup,
    ToolbarItem,
    ToolbarToggleGroup,
    Tooltip,
} from '@patternfly/react-core';

import global_success_color_100 from '@patternfly/react-tokens/dist/esm/global_success_color_100';
import global_danger_color_100 from '@patternfly/react-tokens/dist/esm/global_danger_color_100';
import global_Color_200 from '@patternfly/react-tokens/dist/esm/global_Color_200';

import { KernelManager, ServerConnection } from '@jupyterlab/services';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import {
    BundleIcon,
    CheckCircleIcon,
    ClusterIcon,
    CodeIcon,
    CubesIcon,
    ExclamationTriangleIcon,
    FilterIcon,
    HourglassHalfIcon,
    MigrationIcon,
    MulticlusterIcon,
    PauseIcon,
    PlusIcon,
    RebootingIcon,
    ServerAltIcon,
    ServerIcon,
    SkullIcon,
    SpinnerIcon,
    StopCircleIcon,
    SyncIcon,
    TrashIcon,
    VirtualMachineIcon,
} from '@patternfly/react-icons';

import { IKernelConnection } from '@jupyterlab/services/lib/kernel/kernel';
// import { IInfoReplyMsg } from '@jupyterlab/services/lib/kernel/messages';

import {
    ConfirmationModal,
    ConfirmationWithTextInputModal,
    ExecuteCodeOnKernelModal,
    InformationModal,
} from '@app/Components/Modals';
import { DistributedJupyterKernel, JupyterKernelReplica } from '@data/Kernel';

export interface ClusterComponentsCardProps {}

export const ClusterComponentsCard: React.FunctionComponent<ClusterComponentsCardProps> = (
    props: ClusterComponentsCardProps,
) => {
    const [activeTabKey, setActiveTabKey] = React.useState('cluster-gateway-tab');
    const [isCardExpanded, setIsCardExpanded] = React.useState(true);
    const handleTabClick = (_e: React.MouseEvent<HTMLElement, MouseEvent>, tab: string | number) => {
        setActiveTabKey(tab.toString());
    };

    const gatewayTabContent = (
        <TabContent
            key={'cluster-gateway-tab'}
            eventKey={'cluster-gateway-tab'}
            id={'tabContentClusterGateway'}
            activeKey={activeTabKey}
            hidden={'cluster-gateway-tab' !== activeTabKey}
        >
            <DescriptionListGroup key={0}>
                <DescriptionListTerm>
                    <Flex>
                        <FlexItem>
                            <ServerAltIcon />
                        </FlexItem>
                        <FlexItem>
                            <Title headingLevel="h4" size="md">
                                "Active"
                            </Title>
                        </FlexItem>
                    </Flex>
                </DescriptionListTerm>
                <DescriptionListDescription>
                    <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                        <FlexItem>
                            <CheckCircleIcon color={global_success_color_100.var} />
                        </FlexItem>
                        <FlexItem>
                            <span>Cluster</span>
                        </FlexItem>
                    </Flex>
                </DescriptionListDescription>
            </DescriptionListGroup>
        </TabContent>
    );

    const getLocalDaemonTabContent = (daemonIndex: number) => {
        if (daemonIndex <= 0 || daemonIndex > 3) {
            console.error('Invalid Local Daemon index specified: %d.', daemonIndex);
            return;
        }
        return (
            <TabContent
                key={`"local-daemon${daemonIndex}-tab`}
                eventKey={`"local-daemon${daemonIndex}-tab`}
                id={`tabContentLocalDaemon${daemonIndex}`}
                activeKey={activeTabKey}
                hidden={`"local-daemon${daemonIndex}-tab` !== activeTabKey}
            >
                <DescriptionListGroup key={daemonIndex}>
                    <DescriptionListTerm>
                        <Flex>
                            <FlexItem>
                                <ServerIcon />
                            </FlexItem>
                            <FlexItem>
                                <Title headingLevel="h4" size="md">
                                    "Alive"
                                </Title>
                            </FlexItem>
                        </Flex>
                    </DescriptionListTerm>
                    <DescriptionListDescription>
                        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                            <FlexItem>
                                <CheckCircleIcon color={global_success_color_100.var} />
                            </FlexItem>
                            <FlexItem>
                                <span>Cluster</span>
                            </FlexItem>
                        </Flex>
                    </DescriptionListDescription>
                </DescriptionListGroup>
            </TabContent>
        );
    };

    const onCardExpand = () => {
        setIsCardExpanded(!isCardExpanded);
    };

    return (
        <Card isCompact isRounded isExpanded={isCardExpanded}>
            <CardHeader
                onExpand={onCardExpand}
                toggleButtonProps={{
                    id: 'expand-kube-nodes-button',
                    'aria-label': 'expand-kube-nodes-button',
                    'aria-expanded': isCardExpanded,
                }}
            >
                <CardTitle>
                    <Title headingLevel="h1" size="xl">
                        Distributed Notebook Cluster Components
                    </Title>
                </CardTitle>
            </CardHeader>
            <CardExpandableContent>
                <CardBody>
                    <Tabs isFilled id="status-tabs" activeKey={activeTabKey} onSelect={handleTabClick}>
                        <Tab
                            key={'cluster-gateway-tab'}
                            eventKey={'cluster-gateway-tab'}
                            title={
                                <React.Fragment>
                                    <TabTitleIcon>
                                        <ClusterIcon />
                                    </TabTitleIcon>
                                    <TabTitleText>{`Cluster Gateway`}</TabTitleText>
                                </React.Fragment>
                            }
                            tabContentId={'tabContentClusterGateway'}
                        />
                        <Tab
                            key={'local-daemon1-tab'}
                            eventKey={'local-daemon1-tab'}
                            title={<TabTitleText>{`Local Daemon 1`}</TabTitleText>}
                            tabContentId={'tabContentLocalDaemon1'}
                        />
                        <Tab
                            key={'local-daemon2-tab'}
                            eventKey={'local-daemon2-tab'}
                            title={<TabTitleText>{`Local Daemon 2`}</TabTitleText>}
                            tabContentId={'tabContentLocalDaemon2'}
                        />
                        <Tab
                            key={'local-daemon3-tab'}
                            eventKey={'local-daemon3-tab'}
                            title={<TabTitleText>{`Local Daemon 3`}</TabTitleText>}
                            tabContentId={'tabContentLocalDaemon3'}
                        />
                    </Tabs>
                </CardBody>
                <CardBody>
                    {gatewayTabContent}
                    {getLocalDaemonTabContent(1)}
                    {getLocalDaemonTabContent(2)}
                    {getLocalDaemonTabContent(3)}
                </CardBody>
            </CardExpandableContent>
        </Card>
    );
};
