import React from 'react';
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
    Flex,
    FlexItem,
    Tab,
    TabContent,
    TabTitleText,
    Tabs,
    Title,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
    Skeleton,
    CodeBlock,
    CodeBlockCode,
} from '@patternfly/react-core';
import { SyncIcon } from '@patternfly/react-icons';
import { useKernelSpecs } from '@app/Providers';
import { toast } from 'react-hot-toast';
import { JupyterKernelSpecWrapper } from '@app/Data';

export const KernelSpecList: React.FunctionComponent = () => {
    const [activeTabKey, setActiveTabKey] = React.useState(0);
    const { kernelSpecs, kernelSpecsAreLoading, refreshKernelSpecs } = useKernelSpecs();

    const handleTabClick = (_e: React.MouseEvent<HTMLElement, MouseEvent>, idx: string | number) => {
        setActiveTabKey(Number(idx));
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
                                    loading: <b>Refreshing Jupyter KernelSpecs...</b>,
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
        <Card isFullHeight isRounded>
            <CardHeader actions={{ actions: cardHeaderActions, hasNoOffset: false }}>
                <CardTitle>
                    <Title headingLevel="h1" size="xl">
                        Kernel Specs
                    </Title>
                </CardTitle>
            </CardHeader>
            <CardBody>
                <Tabs
                    isFilled
                    id="status-tabs"
                    activeKey={activeTabKey}
                    onSelect={handleTabClick}
                    hidden={kernelSpecs.length == 0}
                >
                    {kernelSpecs.map((kernelSpec: JupyterKernelSpecWrapper, idx: number) => {
                        return (
                            <Tab
                                key={idx}
                                eventKey={idx}
                                title={<TabTitleText>{kernelSpec.spec.display_name}</TabTitleText>}
                                tabContentId={`kernel-spec-${idx}-tab-content`}
                            />
                        );
                    })}
                </Tabs>
                <div style={{ height: '50px' }} hidden={kernelSpecs.length > 0}>
                    <Skeleton height="70%" width="40%" style={{ float: 'left', margin: '8px' }} />
                    <Skeleton height="70%" width="15%" style={{ float: 'left', margin: '8px' }} />
                    <Skeleton height="70%" width="35%" style={{ float: 'left', margin: '8px' }} />

                    <Skeleton height="70%" width="35%" style={{ float: 'left', margin: '8px' }} />
                    <Skeleton height="70%" width="25%" style={{ float: 'left', margin: '8px' }} />
                    <Skeleton height="70%" width="30%" style={{ float: 'left', margin: '8px' }} />

                    <Skeleton height="100%" width="93.5%" style={{ float: 'left', margin: '8px' }} />
                </div>
            </CardBody>
            <CardBody hidden={kernelSpecs.length == 0}>
                {kernelSpecs.map((kernelSpec: JupyterKernelSpecWrapper, idx: number) => (
                    <TabContent
                        key={idx}
                        eventKey={idx}
                        id={`kernel-spec-${idx}-tab-content`}
                        activeKey={activeTabKey}
                        hidden={idx !== activeTabKey}
                    >
                        <DescriptionList isCompact isFillColumns columnModifier={{ default: '3Col' }}>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Name</DescriptionListTerm>
                                <DescriptionListDescription>{kernelSpec.name}</DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Display Name</DescriptionListTerm>
                                <DescriptionListDescription>{kernelSpec.spec.display_name}</DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Language</DescriptionListTerm>
                                <DescriptionListDescription>{kernelSpec.spec.language}</DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Interrupt Mode</DescriptionListTerm>
                                <DescriptionListDescription>
                                    {kernelSpec.spec.interrupt_mode}
                                </DescriptionListDescription>
                            </DescriptionListGroup>

                            <DescriptionListGroup>
                                <DescriptionListTerm>Command Line Arguments</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <CodeBlock>
                                        <CodeBlockCode>{kernelSpec.spec.argv.join(' ')}</CodeBlockCode>
                                    </CodeBlock>
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                        </DescriptionList>
                    </TabContent>
                ))}
            </CardBody>
        </Card>
    );
};
