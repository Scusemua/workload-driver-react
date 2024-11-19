import { RegisterWorkloadForm } from '@Components/Workloads';
import { Button, Flex, FlexItem, PageSection } from '@patternfly/react-core';
import { PencilAltIcon } from '@patternfly/react-icons';
import * as React from 'react';
import { useNavigate } from 'react-router-dom';

const RegisterWorkloadPage: React.FunctionComponent = () => {
    const navigate = useNavigate();

    return (
        <PageSection>
            <Flex>
                <FlexItem>
                    <RegisterWorkloadForm
                        onRegisterWorkloadFromTemplateClicked={() => {}}
                        onConfirm={() => {}}
                        onCancel={() => {
                            navigate(-1);
                        }}
                    />
                </FlexItem>
                <FlexItem>
                    <Button icon={<PencilAltIcon />}>Create from Template</Button>
                </FlexItem>
            </Flex>
        </PageSection>
    );
};

export { RegisterWorkloadPage };
