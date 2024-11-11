import { WorkloadInspectionView } from '@Components/Workloads/WorkloadInspectionView';
import { Button, Label, Modal, ModalVariant, Title, TitleSizes } from '@patternfly/react-core';
import {
    CheckCircleIcon,
    CloseIcon,
    ExclamationTriangleIcon,
    ExportIcon,
    HourglassStartIcon,
    PlayIcon,
    SpinnerIcon,
    StopIcon,
    TimesCircleIcon,
} from '@patternfly/react-icons';
import text from '@patternfly/react-styles/css/utilities/Text/text';
import { AuthorizationContext } from '@Providers/AuthProvider';
import {
    Workload,
    WORKLOAD_STATE_ERRED,
    WORKLOAD_STATE_FINISHED,
    WORKLOAD_STATE_READY,
    WORKLOAD_STATE_RUNNING,
    WORKLOAD_STATE_TERMINATED,
} from '@src/Data/Workload';
import { GetToastContentWithHeaderAndBody } from '@src/Utils/toast_utils';
import React from 'react';
import toast from 'react-hot-toast';

export interface InspectWorkloadModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    onExportClicked: (currentLocalWorkload: Workload) => void;
    onStartClicked: () => void;
    onStopClicked: () => void;
    workload: Workload;
}

export const InspectWorkloadModal: React.FunctionComponent<InspectWorkloadModalProps> = (props) => {
    const { authenticated } = React.useContext(AuthorizationContext);

    React.useEffect(() => {
        // Automatically close the modal of we are logged out.
        if (!authenticated) {
            props.onClose();
        }
    }, [props, authenticated]);

    const workloadStatus = (
        <React.Fragment>
            {props.workload?.workload_state == WORKLOAD_STATE_READY && (
                <Label icon={<HourglassStartIcon className={text.infoColor_100} />} color="blue">
                    Ready
                </Label>
            )}
            {props.workload?.workload_state == WORKLOAD_STATE_RUNNING && (
                <Label icon={<SpinnerIcon className={'loading-icon-spin ' + text.successColor_100} />} color="green">
                    Running
                </Label>
            )}
            {props.workload?.workload_state == WORKLOAD_STATE_FINISHED && (
                <Label icon={<CheckCircleIcon className={text.successColor_100} />} color="green">
                    Complete
                </Label>
            )}
            {props.workload?.workload_state == WORKLOAD_STATE_ERRED && (
                <Label icon={<TimesCircleIcon className={text.dangerColor_100} />} color="red">
                    Erred
                </Label>
            )}
            {props.workload?.workload_state == WORKLOAD_STATE_TERMINATED && (
                <Label icon={<ExclamationTriangleIcon className={text.warningColor_100} />} color="orange">
                    Terminated
                </Label>
            )}
        </React.Fragment>
    );

    const header = (
        <React.Fragment>
            <Title headingLevel="h1" size={TitleSizes['2xl']}>
                {`Workload ${props.workload?.name} `}

                {workloadStatus}
            </Title>
        </React.Fragment>
    );

    return (
        <Modal
            variant={ModalVariant.large}
            titleIconVariant={'info'}
            header={header}
            aria-label="inspect-workload-modal"
            isOpen={props.isOpen}
            width={1500}
            maxWidth={1920}
            onClose={props.onClose}
            actions={[
                <Button
                    key="start-workload-button"
                    variant="primary"
                    aria-label={'Start workload'}
                    icon={<PlayIcon />}
                    onClick={props.onStartClicked}
                    isDisabled={props.workload?.workload_state != WORKLOAD_STATE_READY || !authenticated}
                >
                    Start Workload
                </Button>,
                <Button
                    key="stop-workload-button"
                    variant="danger"
                    aria-label={'Stop workload'}
                    icon={<StopIcon />}
                    onClick={props.onStopClicked}
                    isDisabled={props.workload?.workload_state != WORKLOAD_STATE_RUNNING || !authenticated}
                >
                    Stop Workload
                </Button>,
                <Button
                    key="export_workload_state_button"
                    aria-label={'Export workload state'}
                    variant="secondary"
                    icon={<ExportIcon />}
                    onClick={() => {
                        if (props.workload) {
                            props.onExportClicked(props.workload);
                        }
                    }}
                >
                    Export
                </Button>,
                <Button
                    key="close-inspect-workload-modal-button"
                    variant="secondary"
                    aria-label={'Inspect workload'}
                    icon={<CloseIcon />}
                    onClick={props.onClose}
                >
                    Close Window
                </Button>,
            ]}
        >
            <WorkloadInspectionView workload={props.workload} />
        </Modal>
    );
};
