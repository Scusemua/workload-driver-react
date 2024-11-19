import { RegisterWorkloadFromPresetForm } from '@Components/Workloads';
import { RegisterWorkloadFromTemplateForm } from '@Components/Workloads/RegistrationForms/RegisterWorkloadFromTemplateForm';
import { Button, Card, CardBody, CardHeader, Flex, FlexItem, PageSection, Tooltip } from '@patternfly/react-core';
import { EditIcon, ListIcon } from '@patternfly/react-icons';
import useNavigation from '@Providers/NavigationProvider';
import { WorkloadContext } from '@src/Providers';
import * as React from 'react';

const RegisterWorkloadPage: React.FunctionComponent = () => {
    const { navigate } = useNavigation();

    const [registeringFromPreset, setRegisteringFromPreset] = React.useState<boolean>(true);

    const { registerWorkloadFromPreset, registerWorkloadFromTemplate } = React.useContext(WorkloadContext);

    const cardHeaderActions = (
        <React.Fragment>
            <Tooltip exitDelay={75} content={<div>Create new workload from template.</div>}>
                <Button
                    variant="plain"
                    aria-label="Create New Workload From Template"
                    onClick={() => setRegisteringFromPreset((curr: boolean) => !curr)}
                >
                    {registeringFromPreset ? <EditIcon /> : <ListIcon />}
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
                                    onConfirm={registerWorkloadFromPreset}
                                    onCancel={() => {
                                        navigate('workloads');
                                    }}
                                    hideActions={false}
                                />
                            </FlexItem>
                        )}
                        {!registeringFromPreset && (
                            <FlexItem>
                                <RegisterWorkloadFromTemplateForm
                                    onConfirm={registerWorkloadFromTemplate}
                                    onCancel={() => {
                                        navigate('workloads');
                                    }}
                                />
                            </FlexItem>
                        )}
                    </Flex>
                </CardBody>
            </Card>
        </PageSection>
    );
};

export { RegisterWorkloadPage };
