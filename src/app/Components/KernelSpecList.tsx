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
import { KernelAPI, KernelManager, KernelMessage, KernelSpecManager, ServerConnection } from '@jupyterlab/services';

import { SyncIcon } from '@patternfly/react-icons';
import { KernelSpec } from '@data/Kernel';

// const kernelSpecs: KernelSpec[] = [
//   {
//     name: 'distributed',
//     displayName: 'Distributed Python3',
//     language: 'python3',
//     interruptMode: 'signal',
//     kernelProvisioner: {
//       name: 'gateway-provisioner',
//       gateway: 'gateway:8080',
//       valid: true,
//     },
//     argV: [''],
//   },
//   {
//     name: 'python3',
//     displayName: 'Python 3 (ipykernel)',
//     language: 'python3',
//     interruptMode: 'signal',
//     kernelProvisioner: {
//       name: '',
//       gateway: '',
//       valid: false,
//     },
//     argV: [''],
//   },
// ];

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
            baseUrl: '/jupyter',
            fetch: fetch,
          }),
        };
        kernelSpecManager.current = new KernelSpecManager(kernelSpecManagerOptions);

        console.log('Waiting for kernel spec manager to be ready.');

        kernelSpecManager.current.disposed.connect(() => {
          console.log('Spec manager was disposed.');
        });

        kernelSpecManager.current.connectionFailure.connect((_sender: KernelSpecManager, err: Error) => {
          console.log('An error has occurred. ' + err.name + ': ' + err.message);
        });

        await kernelSpecManager.current.ready.then(() => {
          console.log('Kernel spec manager is ready!');
          const kernelSpecs = kernelSpecManager.current?.specs;
          console.log('KernelSpecs have been refreshed. Specs: ' + JSON.stringify(kernelSpecs));
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
        if (!ignoreResponse.current) {
          const respKernels = kernelSpecManager.current?.specs;
          console.log('Received kernel specs: ' + JSON.stringify(respKernels));
          // setKernelSpecs(respKernels);
        }
      });
    } catch (e) {
      console.error(e);
    }
  }

  const [kernelSpecs, setKernelSpecs] = React.useState<KernelSpec[]>([]);
  useEffect(() => {
    ignoreResponse.current = false;
    fetchKernelSpecs();

    // Periodically refresh the automatically kernel specs every 5 minutes.
    setInterval(fetchKernelSpecs, 300000);

    return () => {
      ignoreResponse.current = true;
    };
  }, []);

  const cardHeaderActions = (
    <ToolbarGroup variant="icon-button-group">
      <ToolbarItem>
        <Tooltip exitDelay={75} content={<div>Refresh kernel specs.</div>}>
          <Button variant="plain" onClick={fetchKernelSpecs}>
            <SyncIcon />
          </Button>
        </Tooltip>
      </ToolbarItem>
    </ToolbarGroup>
  );

  return (
    <Card isCompact isRounded isExpanded={isCardExpanded}>
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
            {kernelSpecs.map((kernelSpec, tabIndex) => (
              <Tab
                key={tabIndex}
                eventKey={tabIndex}
                title={<TabTitleText>{kernelSpec.displayName}</TabTitleText>}
                tabContentId={`tabContent${tabIndex}`}
              />
            ))}
          </Tabs>
        </CardBody>
        <CardBody>
          {kernelSpecs.map((kernelSpec, tabIndex) => (
            <TabContent
              key={tabIndex}
              eventKey={tabIndex}
              id={`tabContent${tabIndex}`}
              activeKey={activeTabKey}
              hidden={tabIndex !== activeTabKey}
            >
              <DescriptionList columnModifier={{ lg: '2Col' }}>
                <DescriptionListGroup>
                  <DescriptionListTerm>Name</DescriptionListTerm>
                  <DescriptionListDescription>{kernelSpec.name}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>Display Name</DescriptionListTerm>
                  <DescriptionListDescription>{kernelSpec.displayName}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>Language</DescriptionListTerm>
                  <DescriptionListDescription>{kernelSpec.language}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>Interrupt Mode</DescriptionListTerm>
                  <DescriptionListDescription>{kernelSpec.interruptMode}</DescriptionListDescription>
                </DescriptionListGroup>
                {kernelSpec.kernelProvisioner.valid && (
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
                )}
              </DescriptionList>
            </TabContent>
          ))}
        </CardBody>
      </CardExpandableContent>
    </Card>
  );
};
