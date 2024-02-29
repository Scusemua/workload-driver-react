import '@patternfly/react-core/dist/styles/base.css';

import React from 'react';
import { Grid, GridItem, PageSection, Title } from '@patternfly/react-core';

import { KernelList } from '@app/Components/KernelList';
import { KubernetesNodeList } from '@app/Components/NodeList';
import { KernelSpecList } from '@app/Components/KernelSpecList'

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
