import {
    Button,
    Dropdown,
    DropdownItem,
    DropdownList,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    FormHelperText,
    Grid,
    GridItem,
    HelperText,
    HelperTextItem,
    MenuToggle,
    MenuToggleElement,
    NumberInput,
    Popover,
    Switch,
    TextInput,
    ValidatedOptions,
} from '@patternfly/react-core';
import HelpIcon from '@patternfly/react-icons/dist/esm/icons/help-icon';
import styles from '@patternfly/react-styles/css/components/Form/form';
import { AuthorizationContext } from '@Providers/AuthProvider';
import useNavigation from '@Providers/NavigationProvider';
import { useWorkloadPresets } from '@Providers/WorkloadPresetProvider';

import { WorkloadPreset } from '@src/Data';
import SampleSessionsPopover from '@Workloads/RegistrationForms/SampleSessionsPopover';
import React from 'react';

import { v4 as uuidv4 } from 'uuid';

export interface IRegisterWorkloadFormProps {
    onConfirm: (
        workloadTitle: string,
        preset: WorkloadPreset,
        workloadSeed: string,
        debugLoggingEnabled: boolean,
        timescaleAdjustmentFactor: number,
        workloadSessionSamplePercent: number,
    ) => void;
    onCancel: () => void;
    hideActions: boolean;
}

function assertIsNumber(value: number | ''): asserts value is number {
    if (value === '') {
        throw new Error('value is not number');
    }
}

export const RegisterWorkloadFromPresetForm: React.FunctionComponent<IRegisterWorkloadFormProps> = (
    props: IRegisterWorkloadFormProps,
) => {
    const [workloadTitle, setWorkloadTitle] = React.useState<string>('');
    const [workloadTitleIsValid, setWorkloadTitleIsValid] = React.useState<boolean>(true);
    const [workloadSeed, setWorkloadSeed] = React.useState<string>('');
    const [workloadSessionSamplePercent, setWorkloadSessionSamplePercent] = React.useState<number | ''>(1.0);
    const [workloadSeedIsValid, setWorkloadSeedIsValid] = React.useState<boolean>(true);
    const [isWorkloadDataDropdownOpen, setIsWorkloadDataDropdownOpen] = React.useState<boolean>(false);
    const [selectedWorkloadPreset, setSelectedWorkloadPreset] = React.useState<WorkloadPreset | null>(null);
    const [debugLoggingEnabled, setDebugLoggingEnabled] = React.useState<boolean>(true);
    const [timescaleAdjustmentFactor, setTimescaleAdjustmentFactor] = React.useState<number | ''>(0.05);

    const defaultWorkloadTitle = React.useRef(uuidv4());

    const { workloadPresets } = useWorkloadPresets();

    const { authenticated } = React.useContext(AuthorizationContext);

    const { navigate } = useNavigation();

    React.useEffect(() => {
        // Automatically close the modal of we are logged out.
        if (!authenticated) {
            navigate('login');
        }
    }, [authenticated, navigate]);

    const handleWorkloadTitleChanged = (_event, title: string) => {
        setWorkloadTitle(title);
        setWorkloadTitleIsValid(title.length >= 0 && title.length <= 36);
    };

    const handleWorkloadSeedChanged = (_event, seed: string) => {
        const validSeed: boolean = /[0-9]/.test(seed) || seed == '';

        // If it's either the empty string, or we can't even convert the value to a number,
        // then update the state accordingly.
        if (!validSeed || seed == '') {
            setWorkloadSeedIsValid(validSeed);
            setWorkloadSeed('');
            return;
        }

        // Convert to a number.
        const parsed: number = parseInt(seed, 10);

        // If it's a float or something, then just default to no seed.
        if (Number.isNaN(parsed)) {
            setWorkloadSeed('');
            return;
        }

        // If it's greater than the max value, then it is invalid.
        if (parsed > 2147483647 || parsed < 0) {
            setWorkloadSeedIsValid(false);
            setWorkloadSeed(seed);
            return;
        }

        setWorkloadSeed(parsed.toString());
        setWorkloadSeedIsValid(true);
    };

    const onWorkloadDataDropdownToggleClick = () => {
        setIsWorkloadDataDropdownOpen(!isWorkloadDataDropdownOpen);
    };

    const onWorkloadDataDropdownSelect = (
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined,
    ) => {
        if (value != undefined) {
            setSelectedWorkloadPreset(workloadPresets[value]);
        } else {
            setSelectedWorkloadPreset(null);
        }
        setIsWorkloadDataDropdownOpen(false);
    };

    const getWorkloadSeedValidatedState = () => {
        if (!workloadSeedIsValid) {
            return ValidatedOptions.error;
        }

        if (workloadSeed == '') {
            return ValidatedOptions.default;
        }

        return ValidatedOptions.success;
    };

    const isSubmitButtonDisabled = (): boolean => {
        if (!authenticated) {
            return true;
        }

        if (workloadPresets.length == 0) {
            return true;
        }

        if (!workloadTitleIsValid) {
            return true;
        }

        if (selectedWorkloadPreset == null) {
            return true;
        }

        if (!workloadSeedIsValid) {
            return true;
        }

        return validateTimescaleAdjustmentFactor() == 'error';
    };

    // Called when the 'submit' button is clicked.
    const onSubmitWorkload = () => {
        if (!authenticated) {
            return;
        }

        // If the user left the workload title blank, then use the default workload title, which is a randomly-generated UUID.
        let workloadTitleToSubmit: string = workloadTitle;
        if (workloadTitleToSubmit.length == 0) {
            workloadTitleToSubmit = defaultWorkloadTitle.current;
        }

        assertIsNumber(timescaleAdjustmentFactor);
        assertIsNumber(workloadSessionSamplePercent);

        props.onConfirm(
            workloadTitleToSubmit,
            selectedWorkloadPreset!,
            workloadSeed,
            debugLoggingEnabled,
            timescaleAdjustmentFactor,
            workloadSessionSamplePercent,
        );

        // Reset all the fields.
        setSelectedWorkloadPreset(null);
        setWorkloadSeed('');
        setWorkloadTitle('');

        defaultWorkloadTitle.current = uuidv4();
    };

    const validateSessionSamplePercentage = () => {
        if (workloadSessionSamplePercent === '' || Number.isNaN(workloadSessionSamplePercent)) {
            return 'error';
        }

        return workloadSessionSamplePercent <= 0 || workloadSessionSamplePercent > 1 ? 'error' : 'success';
    };

    const validateTimescaleAdjustmentFactor = () => {
        if (timescaleAdjustmentFactor === '' || Number.isNaN(timescaleAdjustmentFactor)) {
            return 'error';
        }

        return timescaleAdjustmentFactor <= 0 || timescaleAdjustmentFactor > 10 ? 'error' : 'success';
    };

    const sampleSessionsPercentFormGroup = (
        <FormGroup label={'Sample Sessions %'} labelIcon={<SampleSessionsPopover />}>
            <NumberInput
                value={workloadSessionSamplePercent}
                onMinus={() => setWorkloadSessionSamplePercent((workloadSessionSamplePercent || 0) - 0.01)}
                onChange={(event: React.FormEvent<HTMLInputElement>) => {
                    const value = (event.target as HTMLInputElement).value;
                    setWorkloadSessionSamplePercent(value === '' ? value : +value);
                }}
                onPlus={() => setWorkloadSessionSamplePercent((timescaleAdjustmentFactor || 0) + 0.01)}
                inputName="sessions-sample-percentage-input"
                inputAriaLabel="sessions-sample-percentage-input"
                minusBtnAriaLabel="minus"
                plusBtnAriaLabel="plus"
                validated={validateSessionSamplePercentage()}
                widthChars={4}
                min={0}
                max={1}
            />
        </FormGroup>
    );

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsLg' }}>
            <FlexItem>
                <Form>
                    <Grid hasGutter md={6}>
                        <GridItem span={12}>
                            <FormGroup
                                label="Workload name"
                                labelIcon={
                                    <Popover
                                        aria-label="workload-title-popover"
                                        headerContent={<div>Workload Title</div>}
                                        bodyContent={
                                            <div>
                                                This is an identifier (that is not necessarily unique, but probably
                                                should be) to help you identify the specific workload. Please note that
                                                the title must be between 1 and 36 characters in length.
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
                                    label="workload-title-text-input"
                                    aria-label="workload-title-text-input"
                                    type="text"
                                    id="workload-title-text-input"
                                    name="workload-title-text-input"
                                    aria-describedby="workload-title-text-input-helper"
                                    value={workloadTitle}
                                    placeholder={defaultWorkloadTitle.current}
                                    validated={
                                        (workloadTitleIsValid && ValidatedOptions.success) || ValidatedOptions.error
                                    }
                                    onChange={handleWorkloadTitleChanged}
                                />
                                <FormHelperText
                                    label="workload-title-text-input-helper"
                                    aria-label="workload-title-text-input-helper"
                                >
                                    <HelperText
                                        label="workload-title-text-input-helper"
                                        aria-label="workload-title-text-input-helper"
                                    >
                                        <HelperTextItem
                                            aria-label="workload-title-text-input-helper"
                                            label="workload-title-text-input-helper"
                                        >
                                            Provide a title to help you identify the workload. The title must be between
                                            1 and 36 characters in length.
                                        </HelperTextItem>
                                    </HelperText>
                                </FormHelperText>
                            </FormGroup>
                        </GridItem>
                        <GridItem span={6}>
                            <FormGroup
                                label="Workload Seed"
                                labelIcon={
                                    <Popover
                                        aria-label="workload-seed-popover"
                                        headerContent={<div>Workload Title</div>}
                                        bodyContent={
                                            <div>
                                                This is an integer seed for the random number generator used by the
                                                workload generator. You may leave this blank to refrain from seeding the
                                                random number generator. Please note that if you do specify a seed, then
                                                the value must be between 0 and 2,147,483,647.
                                            </div>
                                        }
                                    >
                                        <button
                                            type="button"
                                            aria-label="This is an integer seed (between 0 and 2,147,483,647) for the random number generator used by the workload generator. You may leave this blank to refrain from seeding the random number generator."
                                            onClick={(e) => e.preventDefault()}
                                            aria-describedby="simple-form-workload-seed-01"
                                            className={styles.formGroupLabelHelp}
                                        >
                                            <HelpIcon />
                                        </button>
                                    </Popover>
                                }
                            >
                                <TextInput
                                    isRequired
                                    label="workload-seed-text-input"
                                    aria-label="workload-seed-text-input"
                                    type="number"
                                    id="workload-seed-text-input"
                                    name="workload-seed-text-input"
                                    placeholder="No seed"
                                    value={workloadSeed}
                                    aria-describedby="workload-seed-text-input-helper"
                                    validated={getWorkloadSeedValidatedState()}
                                    onChange={handleWorkloadSeedChanged}
                                />
                                <FormHelperText
                                    label="workload-seed-text-input-helper"
                                    aria-label="workload-seed-text-input-helper"
                                >
                                    <HelperText
                                        label="workload-seed-text-input-helper"
                                        aria-label="workload-seed-text-input-helper"
                                    >
                                        <HelperTextItem
                                            aria-label="workload-seed-text-input-helper"
                                            label="workload-seed-text-input-helper"
                                        >
                                            Provide an optional integer seed for the workload&apos;s random number
                                            generator.
                                        </HelperTextItem>
                                    </HelperText>
                                </FormHelperText>
                            </FormGroup>
                        </GridItem>
                        <GridItem span={6}>
                            <FormGroup
                                label="Workload preset:"
                                labelIcon={
                                    <Popover
                                        aria-label="workload-preset-text-header"
                                        headerContent={<div>Workload Preset</div>}
                                        bodyContent={
                                            <div>
                                                Select the preprocessed data to use for driving the workload. This
                                                largely determines which subset of trace data will be used to generate
                                                the workload.
                                            </div>
                                        }
                                    >
                                        <button
                                            type="button"
                                            aria-label="Select the preprocessed data to use for driving the workload. This largely determines which subset of trace data will be used to generate the workload."
                                            onClick={(e) => e.preventDefault()}
                                            className={styles.formGroupLabelHelp}
                                        >
                                            <HelpIcon />
                                        </button>
                                    </Popover>
                                }
                            >
                                {workloadPresets.length == 0 && (
                                    <TextInput
                                        label="workload-presetset-disabled-text"
                                        aria-label="workload-presetset-disabled-text"
                                        id="disabled-workload-preset-select-text"
                                        isDisabled
                                        type="text"
                                        validated={ValidatedOptions.warning}
                                        value="No workload presets available..."
                                    />
                                )}
                                {workloadPresets.length > 0 && (
                                    <Dropdown
                                        aria-label="workload-presetset-dropdown-menu"
                                        isScrollable
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
                                        <DropdownList aria-label="workload-presetset-dropdown-list">
                                            {workloadPresets.map((value: WorkloadPreset, index: number) => {
                                                return (
                                                    <DropdownItem
                                                        aria-label={'workload-presetset-dropdown-item' + index}
                                                        value={index}
                                                        key={value.key}
                                                        description={value.description}
                                                    >
                                                        {value.name}
                                                    </DropdownItem>
                                                );
                                            })}
                                        </DropdownList>
                                    </Dropdown>
                                )}
                                <FormHelperText
                                    label="workload-preset-dropdown-input-helper"
                                    aria-label="workload-preset-dropdown-input-helper"
                                >
                                    <HelperText
                                        label="workload-preset-dropdown-input-helper"
                                        aria-label="workload-preset-dropdown-input-helper"
                                    >
                                        <HelperTextItem
                                            aria-label="workload-preset-dropdown-input-helper"
                                            label="workload-preset-dropdown-input-helper"
                                        >
                                            Select a configuration/data preset for the workload.
                                        </HelperTextItem>
                                    </HelperText>
                                </FormHelperText>
                            </FormGroup>
                        </GridItem>
                        <GridItem span={4}>
                            <FormGroup
                                label={'Timescale Adjustment Factor'}
                                labelIcon={
                                    <Popover
                                        aria-label="timescale-adjustment-factor-header"
                                        headerContent={<div>Timescale Adjustment Factor</div>}
                                        bodyContent={
                                            <div>
                                                This quantity adjusts the timescale at which the trace data is replayed.
                                                For example, if each tick is 60 seconds, then setting this value to 1.0
                                                will instruct the Workload Driver to simulate each tick for the full 60
                                                seconds. Alternatively, setting this quantity to 2.0 will instruct the
                                                Workload Driver to spend 120 seconds on each tick. Setting the quantity
                                                to 0.5 will instruct the Workload Driver to spend 30 seconds on each
                                                tick.
                                            </div>
                                        }
                                    >
                                        <button
                                            type="button"
                                            aria-label="Set the Timescale Adjustment Factor."
                                            onClick={(e) => e.preventDefault()}
                                            className={styles.formGroupLabelHelp}
                                        >
                                            <HelpIcon />
                                        </button>
                                    </Popover>
                                }
                            >
                                <NumberInput
                                    value={timescaleAdjustmentFactor}
                                    onMinus={() =>
                                        setTimescaleAdjustmentFactor((timescaleAdjustmentFactor || 0) - 0.25)
                                    }
                                    onChange={(event: React.FormEvent<HTMLInputElement>) => {
                                        const value = (event.target as HTMLInputElement).value;
                                        setTimescaleAdjustmentFactor(value === '' ? value : +value);
                                    }}
                                    onPlus={() => setTimescaleAdjustmentFactor((timescaleAdjustmentFactor || 0) + 0.25)}
                                    inputName="timescale-adjustment-factor-input"
                                    inputAriaLabel="timescale-adjustment-factor-input"
                                    minusBtnAriaLabel="minus"
                                    plusBtnAriaLabel="plus"
                                    validated={validateTimescaleAdjustmentFactor()}
                                    widthChars={4}
                                    min={0}
                                    max={10}
                                />
                            </FormGroup>
                        </GridItem>
                        <GridItem span={4}>{sampleSessionsPercentFormGroup}</GridItem>
                        <GridItem span={4}>
                            <FormGroup
                                label={'Verbose Server-Side Log Output'}
                                labelIcon={
                                    <Popover
                                        aria-label="workload-debug-logging-header"
                                        headerContent={<div>Verbose Server-Side Log Output</div>}
                                        bodyContent={
                                            <div>
                                                Enable or disable server-side debug (i.e., verbose) log output from the
                                                workload generator and workload driver.
                                            </div>
                                        }
                                    >
                                        <button
                                            type="button"
                                            aria-label="Select the preprocessed data to use for driving the workload. This largely determines which subset of trace data will be used to generate the workload."
                                            onClick={(e) => e.preventDefault()}
                                            className={styles.formGroupLabelHelp}
                                        >
                                            <HelpIcon />
                                        </button>
                                    </Popover>
                                }
                            >
                                <Switch
                                    id="debug-logging-switch-preset"
                                    label="Debug logging enabled"
                                    labelOff="Debug logging disabled"
                                    aria-label="debug-logging-switch-preset"
                                    isChecked={debugLoggingEnabled}
                                    ouiaId="DebugLoggingSwitchPreset"
                                    onChange={(_event: React.FormEvent<HTMLInputElement>, checked: boolean) => {
                                        setDebugLoggingEnabled(checked);
                                    }}
                                />
                            </FormGroup>
                        </GridItem>
                    </Grid>
                </Form>
            </FlexItem>
            {!props.hideActions && (
                <Flex spaceItems={{ default: 'spaceItemsLg' }}>
                    <FlexItem>
                        <Button
                            key="submit"
                            variant="primary"
                            onClick={onSubmitWorkload}
                            isDisabled={isSubmitButtonDisabled()}
                        >
                            Submit
                        </Button>
                    </FlexItem>
                    <FlexItem>
                        <Button key="cancel" variant="link" onClick={props.onCancel}>
                            Cancel
                        </Button>
                    </FlexItem>
                </Flex>
            )}
        </Flex>
    );
};
