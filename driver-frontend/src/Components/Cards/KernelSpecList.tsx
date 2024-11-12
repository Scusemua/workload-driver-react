import {
    Button,
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    CodeBlock,
    CodeBlockCode,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Skeleton,
    Tab,
    TabContent,
    Tabs,
    TabTitleText,
    Title,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
} from '@patternfly/react-core';
import { SyncIcon } from '@patternfly/react-icons';
import { JupyterKernelSpecWrapper } from '@src/Data';
import { useKernelSpecs } from '@src/Providers';
import { ToastRefresh } from '@src/Utils/toast_utils';
import React from 'react';

export const KernelSpecList: React.FunctionComponent = () => {
    const [activeTabKey, setActiveTabKey] = React.useState(0);

    const { kernelSpecs, kernelSpecsAreLoading, refreshKernelSpecs } = useKernelSpecs();

    const handleTabClick = (_e: React.MouseEvent<HTMLElement, MouseEvent>, idx: string | number) => {
        setActiveTabKey(Number(idx));
    };

    const cardHeaderActions = (
        <ToolbarGroup variant="action-group-plain">
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
                            ToastRefresh(
                                refreshKernelSpecs,
                                'Refreshing Jupyter KernelSpecs...',
                                'Failed to refresh Jupyter KernelSpecs',
                                'Refreshed Jupyter KernelSpecs',
                            );
                        }}
                        icon={<SyncIcon />}
                    />
                </Tooltip>
            </ToolbarItem>
        </ToolbarGroup>
    );

    return (
        <Card isFullHeight  id="kernel-spec-list-card">
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
