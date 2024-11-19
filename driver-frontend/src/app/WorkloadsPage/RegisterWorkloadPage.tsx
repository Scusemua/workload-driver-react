import { RegisterWorkloadFromPresetForm } from '@Components/Workloads';
import { RegisterWorkloadFromTemplateForm } from '@Components/Workloads/RegisterWorkloadFromTemplateForm';
import { Button, Card, CardBody, CardHeader, Flex, FlexItem, PageSection, Tooltip } from '@patternfly/react-core';
import { EditIcon, PencilAltIcon } from '@patternfly/react-icons';
import * as React from 'react';
import { useNavigate } from 'react-router-dom';

const RegisterWorkloadPage: React.FunctionComponent = () => {
    const navigate = useNavigate();

    const [registeringFromPreset, setRegisteringFromPreset] = React.useState<boolean>(false);

    const cardHeaderActions = (
        <React.Fragment>
            <Tooltip exitDelay={75} content={<div>Create new workload from template.</div>}>
                <Button
                    variant="plain"
                    aria-label="Create New Workload From Template"
                    onClick={() => setRegisteringFromPreset(true)}
                >
                    <EditIcon />
                </Button>
            </Tooltip>
        </React.Fragment>
    );

    return (
        <PageSection>
            <Card>
                <CardHeader actions={{ actions: cardHeaderActions, hasNoOffset: false }}>
                    Register a New Workload
                </CardHeader>
                <CardBody>
                    <Flex>
                        {registeringFromPreset && (
                            <FlexItem>
                                <RegisterWorkloadFromPresetForm
                                    onRegisterWorkloadFromTemplateClicked={() => {}}
                                    onConfirm={() => {}}
                                    onCancel={() => {
                                        navigate(-1);
                                    }}
                                    hideActions={false}
                                />
                            </FlexItem>
                        )}
                        {!registeringFromPreset && (
                            <FlexItem>
                                <RegisterWorkloadFromTemplateForm
                                    onConfirm={() => {}}
                                    onCancel={() => {
                                        navigate(-1);
                                    }}
                                />
                            </FlexItem>
                        )}
                        <FlexItem>
                            <Button icon={<PencilAltIcon />}>Create from Template</Button>
                        </FlexItem>
                    </Flex>
                </CardBody>
            </Card>
        </PageSection>
    );
};

export { RegisterWorkloadPage };
