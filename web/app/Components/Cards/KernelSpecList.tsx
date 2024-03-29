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
} from '@patternfly/react-core';
import { KernelSpecManager, ServerConnection } from '@jupyterlab/services';
import { SyncIcon } from '@patternfly/react-icons';
import { useKernelSpecs } from '@app/Providers';

export const KernelSpecList: React.FunctionComponent = () => {
    const [activeTabKey, setActiveTabKey] = React.useState(0);
    const kernelSpecManager = useRef<KernelSpecManager | null>(null);
    const { kernelSpecs, kernelSpecsAreLoading, refreshKernelSpecs } = useKernelSpecs();

    useEffect(() => {
        async function initializeKernelManagers() {
            if (kernelSpecManager.current === null) {
                const kernelSpecManagerOptions: KernelSpecManager.IOptions = {
                    serverSettings: ServerConnection.makeSettings({
                        token: '',
                        appendToken: false,
                        baseUrl: 'jupyter',
                        fetch: fetch,
                    }),
                };
                kernelSpecManager.current = new KernelSpecManager(kernelSpecManagerOptions);

                console.log('Waiting for kernel spec manager to be ready.');

                kernelSpecManager.current.connectionFailure.connect((_sender: KernelSpecManager, err: Error) => {
                    console.log(
                        '[ERROR] An error has occurred while preparing the Kernel Spec Manager. ' +
                            err.name +
                            ': ' +
                            err.message,
                    );
                });

                await kernelSpecManager.current.ready.then(() => {
                    console.log('Kernel spec manager is ready!');
                });
            }
        }

        initializeKernelManagers();
    }, []);

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
                            // ignoreResponse.current = false;
                            // fetchKernelSpecs();
                            refreshKernelSpecs();
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
                            title={<TabTitleText>{kernelSpecs[key]?.display_name}</TabTitleText>}
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
                                <DescriptionListDescription>{kernelSpecs[key]?.name}</DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Display Name</DescriptionListTerm>
                                <DescriptionListDescription>
                                    {kernelSpecs[key]?.display_name}
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Language</DescriptionListTerm>
                                <DescriptionListDescription>{kernelSpecs[key]?.language}</DescriptionListDescription>
                            </DescriptionListGroup>
                            {/* <DescriptionListGroup>
                  <DescriptionListTerm>Interrupt Mode</DescriptionListTerm>
                  <DescriptionListDescription>{kernelSpecs[key]?.interrupt_mode}</DescriptionListDescription>
                </DescriptionListGroup> */}
                            {/* {kernelSpec.kernelProvisioner.valid && (
                  <React.Fragment>
                    <DescriptionListGroup>
                      <DescriptionListTerm>Provisioner</DescriptionListTerm>
                      <DescriptionListDescription>{kernelSpec.kernelProvisioner.name}</DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                      <DescriptionListTerm>Provisioner Gateway</DescriptionListTerm>
                      <DescriptionListDescription>{kernelSpec.kernelProvisioner.gateway}</DescriptionListDescription>
                    </DescriptionListGroup>
                  </React.Fragment>
                )} */}
                        </DescriptionList>
                    </TabContent>
                ))}
            </CardBody>
        </Card>
    );
};
