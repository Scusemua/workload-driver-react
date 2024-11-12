import { PageSection } from '@patternfly/react-core';
import { NodeListCard } from '@src/Components';
import * as React from 'react';

// eslint-disable-next-line prefer-const
const NodesPage: React.FunctionComponent = () => {
    return (
        <PageSection>
            <NodeListCard
                isDashboardList={false}
                hideAdjustVirtualGPUsButton={false}
                displayNodeToggleSwitch={true}
                nodesPerPage={10}
                selectableViaCheckboxes={false}
                hideControlPlaneNode={true}
            />
        </PageSection>
    );
};

export { NodesPage };