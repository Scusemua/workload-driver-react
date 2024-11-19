import FailureChanceInput from '@Components/Modals/NewWorkloadFromTemplateModal/RemoteStorage/FailureChanceInput';
import TransferRateInput from '@Components/Modals/NewWorkloadFromTemplateModal/RemoteStorage/TransferRateInput';
import TransferVarianceInput from '@Components/Modals/NewWorkloadFromTemplateModal/RemoteStorage/TransferVarianceInput';
import { FormGroup, FormSection, Grid, GridItem, Popover, TextInput } from '@patternfly/react-core';
import HelpIcon from '@patternfly/react-icons/dist/esm/icons/help-icon';
import styles from '@patternfly/react-styles/css/components/Form/form';
import React from 'react';

import { Controller, useFormContext } from 'react-hook-form';

export function ClampValue(val: number, minVal: number = 0, maxVal: number = 100): number {
    if (val > maxVal) {
        val = maxVal;
    }

    if (val < minVal) {
        val = minVal;
    }

    return val;
}

const RemoteStorageDefinitionForm: React.FunctionComponent = () => {
    const { control } = useFormContext(); // retrieve all hook methods

    return (
        <FormSection title="Remote Storage Definition" titleElement={'h1'}>
            <Grid hasGutter>
                <GridItem span={12} colSpan={12} rowSpan={1}>
                    <FormGroup
                        label="Remote Storage Name:"
                        labelIcon={
                            <Popover
                                aria-label="remote-storage-name-popover"
                                headerContent={<div>Remote Storage Name</div>}
                                bodyContent={'The name of the simulated Remote Storage that you are defining.'}
                            >
                                <button
                                    type="button"
                                    aria-label="The name of the simulated Remote Storage that you are defining."
                                    onClick={(e) => e.preventDefault()}
                                    aria-describedby="simple-form-remote-storage-name"
                                    className={styles.formGroupLabelHelp}
                                >
                                    <HelpIcon />
                                </button>
                            </Popover>
                        }
                    >
                        <Controller
                            name="name"
                            control={control}
                            defaultValue={'AWS S3'}
                            render={({ field }) => (
                                <TextInput
                                    id="remote-storage-name-input"
                                    onBlur={field.onBlur}
                                    onChange={field.onChange}
                                    name={field.name}
                                    value={field.value}
                                    aria-label="Text input for the 'remote storage name'"
                                />
                            )}
                        />
                    </FormGroup>
                </GridItem>
                <GridItem span={3} colSpan={3} rowSpan={1}>
                    <TransferRateInput rateName={'Download'} />
                </GridItem>
                <GridItem span={3} colSpan={3} rowSpan={1}>
                    <TransferRateInput rateName={'Upload'} />
                </GridItem>
                <GridItem span={3} colSpan={3} rowSpan={1}>
                    <TransferVarianceInput rateName={'Download'} />
                </GridItem>
                <GridItem span={3} colSpan={3} rowSpan={1}>
                    <TransferVarianceInput rateName={'Upload'} />
                </GridItem>
                <GridItem span={3} colSpan={3} rowSpan={1}>
                    <FailureChanceInput operationName={'Read'} />
                </GridItem>
                <GridItem span={3} colSpan={3} rowSpan={1}>
                    <FailureChanceInput operationName={'Write'} />
                </GridItem>
            </Grid>
        </FormSection>
    );
};

export default RemoteStorageDefinitionForm;
