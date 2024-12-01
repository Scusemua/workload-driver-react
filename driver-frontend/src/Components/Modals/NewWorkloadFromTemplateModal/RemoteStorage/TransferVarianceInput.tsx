import { ClampValue } from '@Components/Modals';
import { FormGroup, NumberInput, Popover } from '@patternfly/react-core';
import HelpIcon from '@patternfly/react-icons/dist/esm/icons/help-icon';
import styles from '@patternfly/react-styles/css/components/Form/form';
import React from 'react';
import { Controller, useFormContext } from 'react-hook-form';

interface ITransferVarianceInputProps {
    rateName: 'Upload' | 'Download';
}

const TransferVarianceInput: React.FunctionComponent<ITransferVarianceInputProps> = (
    props: ITransferVarianceInputProps,
) => {
    const { control, setValue, getValues } = useFormContext(); // retrieve all hook methods

    const formName: string = `remoteStorageDefinition.${props.rateName.toLowerCase()}RateVariancePercentage`;

    return (
        <FormGroup
            label={`${props.rateName} Rate Variance (%)`}
            labelIcon={
                <Popover
                    aria-label={`workload-${props.rateName}-rate-variance-input`}
                    headerContent={`${props.rateName} Rate Variance (%)`}
                    bodyContent={`The maximum amount by which the ${props.rateName.toLowerCase()} rate can vary/deviate from its set value during a simulated I/O operation.`}
                >
                    <button
                        type="button"
                        aria-label={`The maximum amount by which the ${props.rateName.toLowerCase()} rate can vary/deviate from its set value during a simulated I/O operation.`}
                        onClick={(e) => e.preventDefault()}
                        className={styles.formGroupLabelHelp}
                    >
                        <HelpIcon />
                    </button>
                </Popover>
            }
        >
            <Controller
                name={formName}
                control={control}
                defaultValue={5}
                rules={{ min: 0, max: 100 }}
                render={({ field }) => (
                    <NumberInput
                        inputName={`${props.rateName.toLowerCase()}RateVariancePercentage-input`}
                        id={`${props.rateName.toLowerCase()}RateVariancePercentage-input`}
                        type="number"
                        min={0}
                        max={100}
                        onBlur={field.onBlur}
                        onChange={(event: React.FormEvent<HTMLInputElement>) => {
                            field.onChange(parseFloat((event.target as HTMLInputElement).value));
                        }}
                        name={field.name}
                        value={field.value}
                        widthChars={10}
                        aria-label={`Text input for the remote storage ${props.rateName} rate variance percentage`}
                        onPlus={() => {
                            const curr: number = (getValues(formName) as number) || 0;
                            setValue(formName, ClampValue(curr + 1, 0, 100));
                        }}
                        onMinus={() => {
                            const curr: number = (getValues(formName) as number) || 0;
                            setValue(formName, ClampValue(curr - 1, 0, 100));
                        }}
                    />
                )}
            />
        </FormGroup>
    );
};

export default TransferVarianceInput;
