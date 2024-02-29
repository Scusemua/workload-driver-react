import * as React from 'react';
import {
  Card,
  CardBody,
  CardHeader,
  DescriptionList,
  DescriptionListDescription,
  DescriptionListGroup,
  DescriptionListTerm,
  Tab,
  TabContent,
  TabTitleText,
  Tabs,
  Title
} from '@patternfly/react-core';

import { KernelSpec } from 'src/app/Data/Kernel';

const kernelSpecs: KernelSpec[] = [
  {
    name: "distributed",
    displayName: "Distributed Python3",
    language: "python3",
    interruptMode: "signal",
    kernelProvisioner: {
      name: "gateway-provisioner",
      gateway: "gateway:8080",
      valid: true
    },
    argV: [""],
  },
  {
    name: "python3",
    displayName: "Python 3 (ipykernel)",
    language: "python3",
    interruptMode: "signal",
    kernelProvisioner: {
      name: "",
      gateway: "",
      valid: false
    },
    argV: [""],
  }
];

export const KernelSpecList: React.FunctionComponent = () => {
  const [activeTabKey, setActiveTabKey] = React.useState(0);
  const handleTabClick = (_e: React.MouseEvent<HTMLElement, MouseEvent>, tabIndex: string | number) => {
    setActiveTabKey(Number(tabIndex));
  };

  const tabContent = (
    <DescriptionList columnModifier={{ lg: '2Col' }}>
      <DescriptionListGroup>
        <DescriptionListTerm>Name</DescriptionListTerm>
        <DescriptionListDescription>{kernelSpecs[activeTabKey].name}</DescriptionListDescription>
      </DescriptionListGroup>
      <DescriptionListGroup>
        <DescriptionListTerm>Display Name</DescriptionListTerm>
        <DescriptionListDescription>
          {kernelSpecs[activeTabKey].displayName}
        </DescriptionListDescription>
      </DescriptionListGroup>
      <DescriptionListGroup>
        <DescriptionListTerm>Language</DescriptionListTerm>
        <DescriptionListDescription>{kernelSpecs[activeTabKey].language}</DescriptionListDescription>
      </DescriptionListGroup>
      <DescriptionListGroup>
        <DescriptionListTerm>Interrupt Mode</DescriptionListTerm>
        <DescriptionListDescription>
          {kernelSpecs[activeTabKey].interruptMode}
        </DescriptionListDescription>
      </DescriptionListGroup>
    </DescriptionList>
  );

  return (
    <>
      <Card isCompact isRounded>
        <CardHeader>
          <Title headingLevel='h2' size='xl'>
            Available Kernel Specs
          </Title>
        </CardHeader>
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
              {tabContent}
            </TabContent>
          ))}
        </CardBody>
      </Card>
    </>
  );
}