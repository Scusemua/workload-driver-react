import '@patternfly/react-core/dist/styles/base.css';

import React from 'react';
import { Grid, GridItem, PageSection, Title } from '@patternfly/react-core';

import { KernelList } from '@components/KernelList';
import { KubernetesNodeList } from '@components/NodeList';
import { KernelSpecList } from '@components/KernelSpecList';

const Dashboard: React.FunctionComponent = () => (
  <PageSection>
    <Title headingLevel="h1" size="lg">
      Workload Driver: Dashboard
    </Title>
    <Grid hasGutter>
      <GridItem span={6} rowSpan={3}>
        <KernelList />
      </GridItem>
      <GridItem span={6} rowSpan={1}>
        <KubernetesNodeList />
      </GridItem>
      <GridItem span={6} rowSpan={1}>
        <KernelSpecList />
      </GridItem>
    </Grid>
  </PageSection>
);

export { Dashboard };
