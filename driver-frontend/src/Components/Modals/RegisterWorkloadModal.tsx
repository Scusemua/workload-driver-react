import { Button, Modal, ModalVariant, Text, Tooltip } from '@patternfly/react-core';
import { EditIcon } from '@patternfly/react-icons';
import { AuthorizationContext } from '@Providers/AuthProvider';
import { RegisterWorkloadFromPresetForm } from '@src/Components';

import { WorkloadPreset } from '@src/Data';
import React from 'react';

export interface IRegisterWorkloadModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    onRegisterWorkloadFromTemplateClicked: () => void;
    onConfirm: (
        workloadTitle: string,
        preset: WorkloadPreset,
        workloadSeed: string,
        debugLoggingEnabled: boolean,
        timescaleAdjustmentFactor: number,
        workloadSessionSamplePercent: number,
    ) => void;
}

export const RegisterWorkloadModal: React.FunctionComponent<IRegisterWorkloadModalProps> = (props) => {
    const { authenticated } = React.useContext(AuthorizationContext);

    React.useEffect(() => {
        // Automatically close the modal of we are logged out.
        if (!authenticated) {
            props.onClose();
        }
    }, [props, authenticated]);

    return (
        <Modal
            variant={ModalVariant.medium}
            titleIconVariant={'info'}
            title={'Register Workload from Preset'}
            aria-label="register-workload-from-preset-modal"
            isOpen={props.isOpen}
            onClose={props.onClose}
            help={
                <Tooltip exitDelay={75} content={<div>Create new workload from template.</div>}>
                    <Button
                        variant="plain"
                        aria-label="Create New Workload From Template"
                        onClick={props.onRegisterWorkloadFromTemplateClicked}
                    >
                        <EditIcon />
                    </Button>
                </Tooltip>
            }
        >
            <Text>
                You can also create new workloads using templates by clicking the + button in the top-right of this
                modal.
            </Text>
            <RegisterWorkloadFromPresetForm onConfirm={props.onConfirm} onCancel={props.onClose} hideActions={false} />
        </Modal>
    );
};
