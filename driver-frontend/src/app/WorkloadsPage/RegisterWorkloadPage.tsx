import { RegisterWorkloadFromPresetForm } from '@Components/Workloads';
import { RegisterWorkloadFromTemplateForm } from '@Components/Workloads/RegistrationForms/RegisterWorkloadFromTemplateForm';
import { WorkloadPreset } from '@Data/Workload';
import { Button, Card, CardBody, CardHeader, PageSection, Tooltip } from '@patternfly/react-core';
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

    const onConfirmRegisterWorkloadFromPreset = (
        workloadName: string,
        selectedPreset: WorkloadPreset,
        workloadSeedString: string,
        debugLoggingEnabled: boolean,
        timescaleAdjustmentFactor: number,
        workloadSessionSamplePercent: number,
    ) => {
        registerWorkloadFromPreset(
            workloadName,
            selectedPreset,
            workloadSeedString,
            debugLoggingEnabled,
            timescaleAdjustmentFactor,
            workloadSessionSamplePercent,
        );
        navigate('workloads');
    };

    const onConfirmRegisterWorkloadFromTemplate = (workloadName: string, workloadRegistrationRequest: string) => {
        registerWorkloadFromTemplate(workloadName, workloadRegistrationRequest);
        navigate('workloads');
    };

    return (
        <PageSection>
            <Card>
                <CardHeader actions={{ actions: cardHeaderActions, hasNoOffset: false }}>
                    Register a New Workload
                </CardHeader>
                <CardBody>
                    {registeringFromPreset && (
                        <RegisterWorkloadFromPresetForm
                            onConfirm={onConfirmRegisterWorkloadFromPreset}
                            onCancel={() => {
                                navigate('workloads');
                            }}
                            hideActions={false}
                        />
                    )}
                    {!registeringFromPreset && (
                        <RegisterWorkloadFromTemplateForm
                            onConfirm={onConfirmRegisterWorkloadFromTemplate}
                            onCancel={() => {
                                navigate('workloads');
                            }}
                        />
                    )}
                </CardBody>
            </Card>
        </PageSection>
    );
};

export { RegisterWorkloadPage };
