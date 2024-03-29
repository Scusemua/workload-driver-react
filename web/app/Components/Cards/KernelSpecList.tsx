import React, { useEffect, useRef } from 'react';
import {
    Button,
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Tab,
    TabContent,
    TabTitleText,
    Tabs,
    Title,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
    Flex,
    FlexItem,
} from '@patternfly/react-core';
import { SyncIcon } from '@patternfly/react-icons';
import { useKernelSpecs } from '@app/Providers';
import { toast } from 'react-hot-toast';

export const KernelSpecList: React.FunctionComponent = () => {
    const [activeTabKey, setActiveTabKey] = React.useState(0);
    const { kernelSpecs, kernelSpecsAreLoading, refreshKernelSpecs } = useKernelSpecs();

    const handleTabClick = (_e: React.MouseEvent<HTMLElement, MouseEvent>, tabIndex: string | number) => {
        setActiveTabKey(Number(tabIndex));
    };

    const cardHeaderActions = (
        <ToolbarGroup variant="icon-button-group">
            <ToolbarItem>
                <Tooltip exitDelay={75} content={<div>Refresh kernel specs.</div>}>
                    <Button
                        label="refresh-kernel-specs-button"
                        aria-label="refresh-kernel-specs-button"
                        variant="plain"
                        isDisabled={kernelSpecsAreLoading}
                        className={
                            (kernelSpecsAreLoading && 'loading-icon-spin-toggleable') ||
                            'loading-icon-spin-toggleable paused'
                        }
                        onClick={() => {
                            toast.promise(
                                refreshKernelSpecs(),
                                {
                                    loading: 'Refreshing Jupyter KernelSpecs...',
                                    success: <b>Refreshed Jupyter KernelSpecs!</b>,
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
                                                    <b>Could not refresh Jupyter KernelSpecs.</b>
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
                            );
                        }}
                        icon={<SyncIcon />}
                    />
                </Tooltip>
            </ToolbarItem>
        </ToolbarGroup>
    );

    return (
        <Card isRounded>
            <CardHeader actions={{ actions: cardHeaderActions, hasNoOffset: false }}>
                <CardTitle>
                    <Title headingLevel="h1" size="xl">
                        Kernel Specs
                    </Title>
                </CardTitle>
            </CardHeader>
            <CardBody>
                <Tabs isFilled id="status-tabs" activeKey={activeTabKey} onSelect={handleTabClick}>
                    {Object.keys(kernelSpecs).map((key, tabIndex) => (
                        <Tab
                            key={tabIndex}
                            eventKey={tabIndex}
                            title={<TabTitleText>{kernelSpecs[key]?.spec.display_name}</TabTitleText>}
                            tabContentId={`tabContent${tabIndex}`}
                        />
                    ))}
                </Tabs>
            </CardBody>
            <CardBody>
                {Object.keys(kernelSpecs).map((key, tabIndex) => (
                    <TabContent
                        key={tabIndex}
                        eventKey={tabIndex}
                        id={`tabContent${tabIndex}`}
                        activeKey={activeTabKey}
                        hidden={tabIndex !== activeTabKey}
                    >
                        <DescriptionList columnModifier={{ lg: '3Col' }}>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Name</DescriptionListTerm>
                                <DescriptionListDescription>{kernelSpecs[key].name}</DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Display Name</DescriptionListTerm>
                                <DescriptionListDescription>
                                    {kernelSpecs[key]?.spec.display_name}
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Language</DescriptionListTerm>
                                <DescriptionListDescription>
                                    {kernelSpecs[key]?.spec.language}
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                        </DescriptionList>
                    </TabContent>
                ))}
            </CardBody>
        </Card>
    );
};
