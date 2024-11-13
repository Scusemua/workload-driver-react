import { PageSection } from '@patternfly/react-core';
import { KernelListCard } from '@src/Components';
import * as React from 'react';

export const KernelsPage: React.FunctionComponent = () => (
    <PageSection>
        <KernelListCard
            kernelsPerPage={10}
            openMigrationModal={() => {}}
            perPageOption={[
                {
                    title: '1 kernels',
                    value: 1,
                },
                {
                    title: '2 kernels',
                    value: 2,
                },
                {
                    title: '3 kernels',
                    value: 3,
                },
                {
                    title: '5 kernels',
                    value: 5,
                },
                {
                    title: '10 kernels',
                    value: 10,
                },
                {
                    title: '25 kernels',
                    value: 25,
                },
                {
                    title: '50 kernels',
                    value: 50,
                },
                {
                    title: '100 kernels',
                    value: 100,
                },
            ]}
        />
    </PageSection>
);

export default KernelsPage;
