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
    <WorkloadCard
      workloadsPerPage={3}
      inspectInModal={false}
      perPageOption={[
        {
          title: '1 workloads',
          value: 1,
        },
        {
          title: '2 workloads',
          value: 2,
        },
        {
          title: '3 workloads',
          value: 3,
        },
        {
          title: '5 workloads',
          value: 5,
        },
        {
          title: '10 workloads',
          value: 10,
        },
      ]}
    />
  </PageSection>
);

export { WorkloadsPage };
