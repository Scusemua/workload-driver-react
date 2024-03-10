import React from 'react';
import {
    Button,
    Dropdown,
    DropdownItem,
    DropdownList,
    Form,
    FormGroup,
    FormHelperText,
    HelperText,
    HelperTextItem,
    MenuToggle,
    MenuToggleElement,
    Modal,
    ModalVariant,
    Popover,
    TextInput,
} from '@patternfly/react-core';

import HelpIcon from '@patternfly/react-icons/dist/esm/icons/help-icon';
import styles from '@patternfly/react-styles/css/components/Form/form';

import { WorkloadPreset } from '@app/Data';

export interface StartWorkloadModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    onConfirm: (string, WorkloadPreset) => void;
    workloadPresets: WorkloadPreset[];
}

export const StartWorkloadModal: React.FunctionComponent<StartWorkloadModalProps> = (props) => {
    const defaultWorkloadKey = 'Select workload data';

    const [workloadTitle, setWorkloadTitle] = React.useState('');
    const [isWorkloadDataDropdownOpen, setIsWorkloadDataDropdownOpen] = React.useState(false);
    const [selectedWorkloadPreset, setSelectedWorkloadPreset] = React.useState<WorkloadPreset | null>(null);

    const handleWorkloadTitleChanged = (_event, title: string) => {
        setWorkloadTitle(title);
    };

    const onWorkloadDataDropdownToggleClick = () => {
        setIsWorkloadDataDropdownOpen(!isWorkloadDataDropdownOpen);
    };

    const onWorkloadDataDropdownSelect = (
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined,
    ) => {
        // eslint-disable-next-line no-console
        console.log('selected', value);

        if (value != undefined) {
            setSelectedWorkloadPreset(props.workloadPresets[value]);
        } else {
            setSelectedWorkloadPreset(null);
        }
        setIsWorkloadDataDropdownOpen(false);
    };

    const isSubmitButtonEnabled = () => {
        if (props.workloadPresets.length == 0) {
            return true;
        }

        if (workloadTitle == '') {
            return true;
        }

        if (selectedWorkloadPreset == null) {
            return true;
        }

        return false;
    };

    return (
        <Modal
            variant={ModalVariant.small}
            titleIconVariant={'info'}
            title={'Launch Workload'}
            isOpen={props.isOpen}
            onClose={props.onClose}
            actions={[
                <Button
                    key="submit"
                    variant="primary"
                    onClick={() => {
                        props.onConfirm(workloadTitle, selectedWorkloadPreset);
                    }}
                    isDisabled={isSubmitButtonEnabled()}
                >
                    Submit
                </Button>,
                <Button key="cancel" variant="link" onClick={props.onClose}>
                    Cancel
                </Button>,
            ]}
        >
            <Form>
                <FormGroup
                    label="Workload name:"
                    labelIcon={
                        <Popover
                            headerContent={<div>Workload Title</div>}
                            bodyContent={
                                <div>
                                    This is an identifier (that is not necessarily unique, but probably should be) to
                                    help you identify the specific workload.
                                </div>
                            }
                        >
                            <button
                                type="button"
                                aria-label="This is an identifier (that is not necessarily unique, but probably should be) to help you identify the specific workload."
                                onClick={(e) => e.preventDefault()}
                                aria-describedby="simple-form-workload-name-01"
                                className={styles.formGroupLabelHelp}
                            >
                                <HelpIcon />
                            </button>
                        </Popover>
                    }
                >
                    <TextInput
                        isRequired
                        type="text"
                        id="simple-form-name-01"
                        name="simple-form-name-01"
                        aria-describedby="simple-form-name-01-helper"
                        value={workloadTitle}
                        onChange={handleWorkloadTitleChanged}
                    />
                    <FormHelperText>
                        <HelperText>
                            <HelperTextItem>Provide a title to help you identify the workload.</HelperTextItem>
                        </HelperText>
                    </FormHelperText>
                </FormGroup>
                <FormGroup
                    label="Workload data:"
                    labelIcon={
                        <Popover
                            headerContent={<div>Workload Data</div>}
                            bodyContent={<div>Select the preprocessed data to use for driving the workload.</div>}
                        >
                            <button
                                type="button"
                                aria-label="Select the preprocessed data to use for driving the workload."
                                onClick={(e) => e.preventDefault()}
                                aria-describedby="simple-form-workload-data-01"
                                className={styles.formGroupLabelHelp}
                            >
                                <HelpIcon />
                            </button>
                        </Popover>
                    }
                >
                    {props.workloadPresets.length == 0 && (
                        <TextInput
                            id="disabled-workload-data-select-text"
                            isDisabled
                            type="text"
                            value="No workload presets available."
                        />
                    )}
                    {props.workloadPresets.length > 0 && (
                        <Dropdown
                            isOpen={isWorkloadDataDropdownOpen}
                            onSelect={onWorkloadDataDropdownSelect}
                            onOpenChange={(isOpen: boolean) => setIsWorkloadDataDropdownOpen(isOpen)}
                            toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                                <MenuToggle
                                    ref={toggleRef}
                                    isFullWidth
                                    onClick={onWorkloadDataDropdownToggleClick}
                                    isExpanded={isWorkloadDataDropdownOpen}
                                >
                                    {selectedWorkloadPreset?.name}
                                </MenuToggle>
                            )}
                            shouldFocusToggleOnSelect
                        >
                            <DropdownList>
                                {props.workloadPresets.map((value: WorkloadPreset, index: number) => {
                                    return (
                                        <DropdownItem value={index} key={value.key} description={value.description}>
                                            {value.name}
                                        </DropdownItem>
                                    );
                                })}
                            </DropdownList>
                        </Dropdown>
                    )}
                </FormGroup>
            </Form>
        </Modal>
    );
};
