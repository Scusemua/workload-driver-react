import { WorkloadSeedDelta } from '@Components/Modals';
import { Button, FormGroup, NumberInput, Popover } from '@patternfly/react-core';
import HelpIcon from '@patternfly/react-icons/dist/esm/icons/help-icon';
import styles from '@patternfly/react-styles/css/components/Form/form';
import React from 'react';
import { Controller, useFormContext } from 'react-hook-form';

interface ITransferRateInputProps {
    rateName: 'Upload' | 'Download';
}

const TransferRateInput: React.FunctionComponent<ITransferRateInputProps> = (props: ITransferRateInputProps) => {
    const { control, setValue, getValues } = useFormContext(); // retrieve all hook methods

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
                name={`${props.rateName.toLowerCase()}_rate`}
                control={control}
                defaultValue={1e6}
                rules={{ min: 0 }}
                render={({ field }) => (
                    <NumberInput
                        inputName={`${props.rateName.toLowerCase()}-rate-input`}
                        id={`${props.rateName.toLowerCase()}-rate-input`}
                        type="number"
                        min={0}
                        onBlur={field.onBlur}
                        onChange={field.onChange}
                        name={field.name}
                        value={field.value}
                        widthChars={10}
                        aria-label={`Text input for the remote storage ${props.rateName} rate`}
                        onPlus={() => {
                            const curr: number = getValues(`${props.rateName.toLowerCase()}Rate`) || 0;
                            let next: number = curr + 1e6;

                            if (next < 0) {
                                next = 0;
                            }

                            setValue(`${props.rateName.toLowerCase()}_rate`, next);
                        }}
                        onMinus={() => {
                            const curr: number = getValues(`${props.rateName.toLowerCase()}Rate`) || 0;
                            let next: number = curr - 1e6;

                            if (next < 0) {
                                next = 0;
                            }

                            setValue(`${props.rateName.toLowerCase()}_rate`, next);
                        }}
                    />
                )}
            />
        </FormGroup>
    );
};

export default TransferRateInput;
