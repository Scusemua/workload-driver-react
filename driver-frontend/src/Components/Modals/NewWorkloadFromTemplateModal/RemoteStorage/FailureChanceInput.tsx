import { FormGroup, NumberInput, Popover } from '@patternfly/react-core';
import HelpIcon from '@patternfly/react-icons/dist/esm/icons/help-icon';
import styles from '@patternfly/react-styles/css/components/Form/form';
import React from 'react';
import { Controller, useFormContext } from 'react-hook-form';

interface IFailureChanceInputProps {
    operationName: 'Read' | 'Write';
}

const FailureChanceInput: React.FunctionComponent<IFailureChanceInputProps> = (props: IFailureChanceInputProps) => {
    const { control, setValue, getValues } = useFormContext(); // retrieve all hook methods

    return (
        <FormGroup
            label={`${props.operationName} Failure Chance (%)`}
            labelIcon={
                <Popover
                    aria-label={`workload-${props.operationName}-rate-input`}
                    headerContent={`${props.operationName} Failure Chance (%)`}
                    bodyContent={`The likelihood as a percentage (value between 0 and 100) that an error occurs during any single ${props.operationName.toLowerCase()} operation.`}
                >
                    <button
                        type="button"
                        aria-label={`The likelihood as a percentage (value between 0 and 100) that an error occurs during any single ${props.operationName.toLowerCase()} operation.`}
                        onClick={(e) => e.preventDefault()}
                        className={styles.formGroupLabelHelp}
                    >
                        <HelpIcon />
                    </button>
                </Popover>
            }
        >
            <Controller
                name={`${props.operationName.toLowerCase()}_failure_chance_percentage`}
                control={control}
                defaultValue={0}
                rules={{ min: 0, max: 100 }}
                render={({ field }) => (
                    <NumberInput
                        inputName={`${props.operationName.toLowerCase()}-failure-chance-percentage`}
                        id={`${props.operationName.toLowerCase()}-failure-chance-percentage`}
                        type="number"
                        min={0}
                        max={100}
                        onBlur={field.onBlur}
                        onChange={field.onChange}
                        name={field.name}
                        value={field.value}
                        widthChars={10}
                        aria-label={`Text input for the remote storage ${props.operationName} rate`}
                        onPlus={() => {
                            const curr: number = getValues(`${props.operationName.toLowerCase()}Rate`) || 0;
                            let next: number = curr + 1;

                            if (next < 0) {
                                next = 0;
                            }

                            if (next > 100) {
                                next = 100;
                            }

                            setValue(`${props.operationName.toLowerCase()}_failure_chance_percentage`, next);
                        }}
                        onMinus={() => {
                            const curr: number = getValues(`${props.operationName.toLowerCase()}Rate`) || 0;
                            let next: number = curr - 1;

                            if (next < 0) {
                                next = 0;
                            }

                            if (next > 100) {
                                next = 100;
                            }

                            setValue(`${props.operationName.toLowerCase()}_failure_chance_percentage`, next);
                        }}
                    />
                )}
            />
        </FormGroup>
    );
};

export default FailureChanceInput;
