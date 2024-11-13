import { RegisterWorkloadForm } from '@Components/Workloads';
import { PageSection } from '@patternfly/react-core';
import * as React from 'react';
import { useNavigate } from 'react-router-dom';

const RegisterWorkloadPage: React.FunctionComponent = () => {
    const navigate = useNavigate();

    return (
        <PageSection>
            <RegisterWorkloadForm
                onRegisterWorkloadFromTemplateClicked={() => {}}
                onConfirm={() => {}}
                onCancel={() => {
                    navigate(-1);
                }}
            />
        </PageSection>
    );
};

export { RegisterWorkloadPage };
