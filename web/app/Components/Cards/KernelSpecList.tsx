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
import { ISpecModel } from '@jupyterlab/services/lib/kernelspec/restapi';
import { SyncIcon } from '@patternfly/react-icons';

export const KernelSpecList: React.FunctionComponent = () => {
    const [activeTabKey, setActiveTabKey] = React.useState(0);
    const kernelSpecManager = useRef<KernelSpecManager | null>(null);
    const [refreshingKernelSpecs, setRefreshingKernelSpecs] = React.useState(false);

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

    const ignoreResponse = useRef(false);
    async function fetchKernelSpecs() {
        const startTime = performance.now();
        try {
            setRefreshingKernelSpecs(true);
            console.log('Refreshing kernel specs.');

            // Make a network request to the backend. The server infrastructure handles proxying/routing the request to the correct host.
            // We're specifically targeting the API endpoint I setup called "kernelspecs".
            // const response = await fetch('/api/jupyter/kernelspecs');
            // const respKernels: KernelSpec[] = await response.json();

            kernelSpecManager.current?.refreshSpecs().then(() => {
                if (!ignoreResponse.current && kernelSpecManager.current?.specs?.kernelspecs != undefined) {
                    const respKernels: { [key: string]: ISpecModel | undefined } =
                        kernelSpecManager.current?.specs.kernelspecs;

                    if (respKernels !== undefined) {
                        setKernelSpecs(respKernels!);
                    }

                    ignoreResponse.current = true;
                }

                setRefreshingKernelSpecs(false);
            });
        } catch (e) {
            console.error(e);
        }

        console.log(`Refresh kernel specs: ${(performance.now() - startTime).toFixed(4)} ms`);
    }

    const [kernelSpecs, setKernelSpecs] = React.useState<{ [key: string]: ISpecModel | undefined }>({});
    useEffect(() => {
        ignoreResponse.current = false;
        fetchKernelSpecs();

        // Periodically refresh the automatically kernel specs every 5 minutes.
        setInterval(() => {
            ignoreResponse.current = false;
            fetchKernelSpecs().then(() => {
                ignoreResponse.current = true;
            });
        }, 300000);

        return () => {
            ignoreResponse.current = true;
        };
    }, []);

    const cardHeaderActions = (
        <ToolbarGroup variant="icon-button-group">
            <ToolbarItem>
                <Tooltip exitDelay={75} content={<div>Refresh kernel specs.</div>}>
                    <Button
                        label="refresh-kernel-specs-button"
                        aria-label="refresh-kernel-specs-button"
                        variant="plain"
                        isDisabled={refreshingKernelSpecs}
                        className={
                            (refreshingKernelSpecs && 'loading-icon-spin-toggleable') ||
                            'loading-icon-spin-toggleable paused'
                        }
                        onClick={() => {
                            ignoreResponse.current = false;
                            fetchKernelSpecs();
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
