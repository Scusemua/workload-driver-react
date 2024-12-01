import { ClampValue } from '@Components/Modals';
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

    const formName: string = `remoteStorageDefinition.${props.operationName.toLowerCase()}FailureChancePercentage`;

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
                name={formName}
                control={control}
                defaultValue={0}
                rules={{ min: 0, max: 100 }}
                render={({ field }) => (
                    <NumberInput
                        inputName={formName}
                        id={formName}
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
                        aria-label={`Text input for the remote storage ${props.operationName} rate`}
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

export default FailureChanceInput;
