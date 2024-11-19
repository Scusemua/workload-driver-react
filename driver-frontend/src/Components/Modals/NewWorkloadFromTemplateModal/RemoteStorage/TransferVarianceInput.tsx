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
                name={`${props.rateName.toLowerCase()}_rate_variance_percent`}
                control={control}
                defaultValue={5}
                rules={{ min: 0, max: 100 }}
                render={({ field }) => (
                    <NumberInput
                        inputName={`${props.rateName.toLowerCase()}-rate-variance-percent-input`}
                        id={`${props.rateName.toLowerCase()}-rate-variance-percent-input`}
                        type="number"
                        min={0}
                        max={100}
                        onBlur={field.onBlur}
                        onChange={field.onChange}
                        name={field.name}
                        value={field.value}
                        widthChars={10}
                        aria-label={`Text input for the remote storage ${props.rateName} rate variance percentage`}
                        onPlus={() => {
                            const curr: number = getValues(`${props.rateName.toLowerCase()}Rate`) || 0;
                            let next: number = curr + 1;

                            if (next < 0) {
                                next = 0;
                            }

                            if (next > 100) {
                                next = 100;
                            }

                            setValue(`${props.rateName.toLowerCase()}_rate_variance_percentage`, next);
                        }}
                        onMinus={() => {
                            const curr: number = getValues(`${props.rateName.toLowerCase()}Rate`) || 0;
                            let next: number = curr - 1;

                            if (next < 0) {
                                next = 0;
                            }

                            if (next > 100) {
                                next = 100;
                            }

                            setValue(`${props.rateName.toLowerCase()}_rate_variance_percentage`, next);
                        }}
                    />
                )}
            />
        </FormGroup>
    );
};

export default TransferVarianceInput;
