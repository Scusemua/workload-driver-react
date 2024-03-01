import React, { useEffect, useRef } from 'react';
import {
  Button,
  Card,
  CardBody,
  CardExpandableContent,
  CardHeader,
  DescriptionList,
  DescriptionListDescription,
  DescriptionListGroup,
  DescriptionListTerm,
  Tab,
  TabContent,
  TabTitleText,
  Tabs,
  Title,
} from '@patternfly/react-core';

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
      // We're specifically targeting the API endpoint I setup called "kernelspec".
      const response = await fetch('/api/kernelspec');

      const respKernels: KernelSpec[] = await response.json();

      if (!ignoreResponse.current) {
        console.log('Received kernel specs: ' + JSON.stringify(respKernels));
        setKernelSpecs(respKernels);
      }
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

  const cardHeaderActions = <Button variant="link" icon={<SyncIcon />} onClick={fetchKernelSpecs} />;

  return (
    <>
      <Card isCompact isRounded isExpanded={isCardExpanded}>
        <CardHeader onExpand={onCardExpand} actions={{ actions: cardHeaderActions, hasNoOffset: true }}>
          <Title headingLevel="h2" size="xl">
            Available Kernel Specs
          </Title>
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
                    <DescriptionListDescription>{kernelSpecs[tabIndex].name}</DescriptionListDescription>
                  </DescriptionListGroup>
                  <DescriptionListGroup>
                    <DescriptionListTerm>Display Name</DescriptionListTerm>
                    <DescriptionListDescription>{kernelSpecs[tabIndex].displayName}</DescriptionListDescription>
                  </DescriptionListGroup>
                  <DescriptionListGroup>
                    <DescriptionListTerm>Language</DescriptionListTerm>
                    <DescriptionListDescription>{kernelSpecs[tabIndex].language}</DescriptionListDescription>
                  </DescriptionListGroup>
                  <DescriptionListGroup>
                    <DescriptionListTerm>Interrupt Mode</DescriptionListTerm>
                    <DescriptionListDescription>{kernelSpecs[tabIndex].interruptMode}</DescriptionListDescription>
                  </DescriptionListGroup>
                </DescriptionList>
              </TabContent>
            ))}
          </CardBody>
        </CardExpandableContent>
      </Card>
    </>
  );
};
