import { RegisterWorkloadFromTemplateForm } from '@Components/Workloads/RegisterWorkloadFromTemplateForm';
import { Button, Flex, FlexItem, Modal, ModalVariant, Popover, Tooltip } from '@patternfly/react-core';
import { DownloadIcon, PencilAltIcon } from '@patternfly/react-icons';
import HelpIcon from '@patternfly/react-icons/dist/esm/icons/help-icon';
import React from 'react';

export interface NewWorkloadFromTemplateModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    onConfirm: (workloadName: string, workloadRegistrationRequestJson: string) => void;
}

// Clamp a value between two extremes.
function clamp(value: number, min: number, max: number) {
    return Math.max(Math.min(value, max), min);
}

interface readFile {
    fileName: string;
    data?: string;
    loadResult?: 'danger' | 'success';
    loadError?: DOMException;
}

// Important: this component must be wrapped in a <SessionTabsDataProvider></SessionTabsDataProvider>!
export const NewWorkloadFromTemplateModal: React.FunctionComponent<NewWorkloadFromTemplateModalProps> = (props) => {
    return (
        <React.Fragment>
            <Modal
                variant={ModalVariant.large}
                titleIconVariant={PencilAltIcon}
                aria-label="Modal to create a new workload from a template"
                title={'Create New Workload from Template'}
                isOpen={props.isOpen}
                onClose={props.onClose}
                help={
                    <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsXs' }}>
                        <FlexItem>
                            <Popover
                                headerContent={<div>Creating New Workloads from Templates</div>}
                                bodyContent={
                                    <div>
                                        You can create and register a new workload using a &quot;template&quot;. This
                                        allows for a greater degree of dynamicity in the workload&apos;s execution.
                                        <br />
                                        <br />
                                        Specifically, templates enable you to customize various properties of the
                                        workload, such as the number of sessions, the resource utilization of these
                                        sessions, when the sessions start and stop, and the training events processed by
                                        the workload&apos;s sessions.
                                    </div>
                                }
                            >
                                <Button variant="plain" aria-label="Create New Workload From Template Helper">
                                    <HelpIcon />
                                </Button>
                            </Popover>
                        </FlexItem>
                    </Flex>
                }
            >
                <RegisterWorkloadFromTemplateForm onCancel={props.onClose} onConfirm={() => {}} />
            </Modal>
        </React.Fragment>
    );
};
