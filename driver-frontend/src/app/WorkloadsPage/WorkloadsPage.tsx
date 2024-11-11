import { WorkloadCard } from '@src/Components';
import * as React from 'react';
import {
  PageSection,
} from '@patternfly/react-core';

export interface IWorkloadsPageProps {
  sampleProp?: string;
}

// eslint-disable-next-line prefer-const
let WorkloadsPage: React.FunctionComponent<IWorkloadsPageProps> = () => (
  <PageSection>
    <WorkloadCard workloadsPerPage={3}/>
  </PageSection>
);

export { WorkloadsPage };
