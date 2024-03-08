import React, { useEffect, useRef } from 'react';
import {
    Button,
    Card,
    CardBody,
    CardExpandableContent,
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
    const [isCardExpanded, setIsCardExpanded] = React.useState(true);
    const kernelSpecManager = useRef<KernelSpecManager | null>(null);

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

    const onCardExpand = () => {
        setIsCardExpanded(!isCardExpanded);
    };

    const handleTabClick = (_e: React.MouseEvent<HTMLElement, MouseEvent>, tabIndex: string | number) => {
        setActiveTabKey(Number(tabIndex));
    };

    const ignoreResponse = useRef(false);
    async function fetchKernelSpecs() {
        try {
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
                }
            });
        } catch (e) {
            console.error(e);
        }
    }

    const [kernelSpecs, setKernelSpecs] = React.useState<{ [key: string]: ISpecModel | undefined }>({});
    useEffect(() => {
        ignoreResponse.current = false;
        fetchKernelSpecs();

        // Periodically refresh the automatically kernel specs every 5 minutes.
        setInterval(() => {
            ignoreResponse.current = false;
            fetchKernelSpecs();
            ignoreResponse.current = true;
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
                        variant="plain"
                        onClick={() => {
                            ignoreResponse.current = false;
                            fetchKernelSpecs();
                            ignoreResponse.current = true;
                        }}
                    >
                        <SyncIcon />
                    </Button>
                </Tooltip>
            </ToolbarItem>
        </ToolbarGroup>
    );

    return (
        <Card isRounded isExpanded={isCardExpanded}>
            <CardHeader
                onExpand={onCardExpand}
                actions={{ actions: cardHeaderActions, hasNoOffset: true }}
                toggleButtonProps={{
                    id: 'toggle-button',
                    'aria-label': 'Actions',
                    'aria-labelledby': 'titleId toggle-button',
                    'aria-expanded': isCardExpanded,
                }}
            >
                <CardTitle>
                    <Title headingLevel="h4" size="xl">
                        Kernel Specs
                    </Title>
                </CardTitle>
            </CardHeader>
            <CardExpandableContent>
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
                                    <DescriptionListDescription>
                                        {kernelSpecs[key]?.language}
                                    </DescriptionListDescription>
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
            </CardExpandableContent>
        </Card>
    );
};
