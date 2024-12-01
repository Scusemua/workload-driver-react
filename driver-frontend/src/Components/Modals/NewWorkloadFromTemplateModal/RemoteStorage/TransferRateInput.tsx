import { ClampValue } from '@Components/Modals';
import { FormGroup, NumberInput, Popover } from '@patternfly/react-core';
import HelpIcon from '@patternfly/react-icons/dist/esm/icons/help-icon';
import styles from '@patternfly/react-styles/css/components/Form/form';
import { MAX_SAFE_INTEGER } from 'lib0/number';
import React from 'react';
import { Controller, useFormContext } from 'react-hook-form';

interface ITransferRateInputProps {
    rateName: 'Upload' | 'Download';
}

const TransferRateInput: React.FunctionComponent<ITransferRateInputProps> = (props: ITransferRateInputProps) => {
    const { control, setValue, getValues } = useFormContext(); // retrieve all hook methods

    const formName: string = `remoteStorageDefinition.${props.rateName.toLowerCase()}Rate`;

    return (
        <FormGroup
            label={`${props.rateName} Rate (bytes/second)`}
            labelIcon={
                <Popover
                    aria-label={`workload-${props.rateName}-rate-input`}
                    headerContent={`${props.rateName} Rate (bytes/second)`}
                    bodyContent={`The average ${props.rateName} rate, in bytes/second, of the remote storage that you are defining.`}
                >
                    <button
                        type="button"
                        aria-label={`The average ${props.rateName} rate, in bytes/second, of the remote storage that you are defining.`}
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
                defaultValue={1e6}
                rules={{ min: 0 }}
                render={({ field }) => (
                    <NumberInput
                        inputName={`${props.rateName.toLowerCase()}Rate-input`}
                        id={`${props.rateName.toLowerCase()}Rate-input`}
                        type="number"
                        min={0}
                        onBlur={field.onBlur}
                        onChange={(event: React.FormEvent<HTMLInputElement>) => {
                            field.onChange(parseInt((event.target as HTMLInputElement).value));
                        }}
                        name={field.name}
                        value={field.value}
                        widthChars={10}
                        aria-label={`Text input for the remote storage ${props.rateName} rate`}
                        onPlus={() => {
                            const curr: number = (getValues(formName) as number) || 0;
                            setValue(formName, ClampValue(curr + 1e6, 0, MAX_SAFE_INTEGER));
                        }}
                        onMinus={() => {
                            const curr: number = (getValues(formName) as number) || 0;
                            setValue(formName, ClampValue(curr - 1e6, 0, MAX_SAFE_INTEGER));
                        }}
                    />
                )}
            />
        </FormGroup>
    );
};

export default TransferRateInput;
