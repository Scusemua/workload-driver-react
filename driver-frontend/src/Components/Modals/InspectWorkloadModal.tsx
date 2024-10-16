import {
    Workload,
    WORKLOAD_STATE_ERRED,
    WORKLOAD_STATE_FINISHED,
    WORKLOAD_STATE_READY,
    WORKLOAD_STATE_RUNNING,
    WORKLOAD_STATE_TERMINATED,
} from '@src/Data/Workload';
import { GetToastContentWithHeaderAndBody } from '@src/Utils/toast_utils';
import { RoundToThreeDecimalPlaces } from '@Components/Modals/NewWorkloadFromTemplateModal';
import { WorkloadEventTable, WorkloadSessionTable } from '@Components/Tables';
import {
    Button,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    FlexItem,
    Label,
    Modal,
    ModalVariant,
    Title,
    TitleSizes,
} from '@patternfly/react-core';
import {
    BlueprintIcon,
    CheckCircleIcon,
    ClipboardCheckIcon,
    ClockIcon,
    CloseIcon,
    CodeIcon,
    DiceIcon,
    ExclamationTriangleIcon,
    HourglassStartIcon,
    MonitoringIcon,
    PlayIcon,
    SpinnerIcon,
    StopIcon,
    Stopwatch20Icon,
    StopwatchIcon,
    TimesCircleIcon,
    UserClockIcon,
} from '@patternfly/react-icons';
import text from '@patternfly/react-styles/css/utilities/Text/text';
import React from 'react';
import toast from 'react-hot-toast';

export interface InspectWorkloadModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    onStartClicked: () => void;
    onStopClicked: () => void;
    workload: Workload | null;
}

export const InspectWorkloadModal: React.FunctionComponent<InspectWorkloadModalProps> = (props) => {
    const [currentTick, setCurrentTick] = React.useState<number>(0);

    const tickStartTime = React.useRef<number>(0);
    const tickDurations = React.useRef<number[]>([]);

    React.useEffect(() => {
        if (props.workload && props.workload?.current_tick > currentTick) {
            const tickDuration: number = performance.now() - tickStartTime.current;
            tickDurations.current.push(tickDuration);
            tickStartTime.current = performance.now();
            setCurrentTick(props.workload?.current_tick);
            toast.custom((t) =>
                GetToastContentWithHeaderAndBody(
                    'Tick Incremented',
                    `Workload ${props.workload?.name} has progressed to Tick #${props.workload?.current_tick}.`,
                  'info',
                  () => {toast.dismiss(t.id)}
                ),
                { icon: '⏱️', style: { maxWidth: 700 } },
            );
        }
    }, [props.workload?.current_tick]);

    const workloadStatus = (
        <React.Fragment>
            {props.workload?.workload_state == WORKLOAD_STATE_READY && (
                <Label icon={<HourglassStartIcon className={text.infoColor_100} />} color="blue">
                    Ready
                </Label>
            )}
            {props.workload?.workload_state == WORKLOAD_STATE_RUNNING && (
                <Label icon={<SpinnerIcon className={'loading-icon-spin ' + text.successColor_100} />} color="green">
                    Running
                </Label>
            )}
            {props.workload?.workload_state == WORKLOAD_STATE_FINISHED && (
                <Label icon={<CheckCircleIcon className={text.successColor_100} />} color="green">
                    Complete
                </Label>
            )}
            {props.workload?.workload_state == WORKLOAD_STATE_ERRED && (
                <Label icon={<TimesCircleIcon className={text.dangerColor_100} />} color="red">
                    Erred
                </Label>
            )}
            {props.workload?.workload_state == WORKLOAD_STATE_TERMINATED && (
                <Label icon={<ExclamationTriangleIcon className={text.warningColor_100} />} color="orange">
                    Terminated
                </Label>
            )}
        </React.Fragment>
    );

    const header = (
        <React.Fragment>
            <Title headingLevel="h1" size={TitleSizes['2xl']}>
                {`Workload ${props.workload?.name} `}

                {workloadStatus}
            </Title>
        </React.Fragment>
    );

    const getTimeElapsedString = () => {
        if (props.workload?.workload_state === undefined || props.workload?.workload_state === '') {
            return 'N/A';
        }

        return props.workload?.time_elapsed_str;
    };

    const getLastTickDuration = () => {
        if (tickDurations && tickDurations.current && tickDurations.current.length > 0) {
            return RoundToThreeDecimalPlaces(tickDurations.current[tickDurations.current.length - 1]);
        } else {
            return 'N/A';
        }
    };

    const getAverageTickDuration = () => {
        if (tickDurations && tickDurations.current && tickDurations.current.length > 0) {
            let sum: number = 0.0;
            tickDurations.current.forEach((val: number) => {
                sum = sum + val;
            });
            return RoundToThreeDecimalPlaces(sum / tickDurations.current.length);
        } else {
            return 'N/A';
        }
    };

    return (
        <Modal
            variant={ModalVariant.large}
            titleIconVariant={'info'}
            header={header}
            aria-label="inspect-workload-modal"
            isOpen={props.isOpen}
            width={1500}
            maxWidth={1920}
            onClose={props.onClose}
            actions={[
                <Button
                    key="start-workload"
                    variant="primary"
                    icon={<PlayIcon />}
                    onClick={props.onStartClicked}
                    isDisabled={props.workload?.workload_state != WORKLOAD_STATE_READY}
                >
                    Start Workload
                </Button>,
                <Button
                    key="stop-workload"
                    variant="danger"
                    icon={<StopIcon />}
                    onClick={props.onStopClicked}
                    isDisabled={props.workload?.workload_state != WORKLOAD_STATE_RUNNING}
                >
                    Stop Workload
                </Button>,
                <Button key="dismiss-workload" variant="secondary" icon={<CloseIcon />} onClick={props.onClose}>
                    Close Window
                </Button>,
            ]}
        >
            <Flex direction={{ default: 'column' }}>
                <FlexItem>
                    <DescriptionList columnModifier={{ lg: '3Col' }}>
                        {props.workload?.workload_preset && (
                            <DescriptionListGroup>
                                <DescriptionListTerm>
                                    Workload Preset <BlueprintIcon />{' '}
                                </DescriptionListTerm>
                                <DescriptionListDescription>
                                    &quot;{props.workload?.workload_preset_name}&quot;
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                        )}
                        {/* {props.workload?.workload_template && <DescriptionListGroup>
                            <DescriptionListTerm>Workload Template <BlueprintIcon /></DescriptionListTerm>
                            <DescriptionListDescription>&quot;{props.workload?.workload_template.name}&quot;</DescriptionListDescription>
                        </DescriptionListGroup>} */}
                        <DescriptionListGroup>
                            <DescriptionListTerm>
                                Seed <DiceIcon />{' '}
                            </DescriptionListTerm>
                            <DescriptionListDescription>{props.workload?.seed}</DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>
                                Time Adjustment Factor <ClockIcon />{' '}
                            </DescriptionListTerm>
                            <DescriptionListDescription>
                                {props.workload?.timescale_adjustment_factor}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>
                                Events Processed <MonitoringIcon />
                            </DescriptionListTerm>
                            <DescriptionListDescription>
                                {props.workload?.num_events_processed}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>
                                Training Events Completed <CodeIcon />
                            </DescriptionListTerm>
                            <DescriptionListDescription>
                                {props.workload?.num_tasks_executed}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>
                                Time Elapsed <StopwatchIcon />
                            </DescriptionListTerm>
                            <DescriptionListDescription>{getTimeElapsedString()}</DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>
                                Workload Clock Time <UserClockIcon />
                            </DescriptionListTerm>
                            <DescriptionListDescription>
                                {props.workload?.simulation_clock_time == ''
                                    ? 'N/A'
                                    : props.workload?.simulation_clock_time}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>
                                Current Tick <Stopwatch20Icon />
                            </DescriptionListTerm>
                            <DescriptionListDescription>{props.workload?.current_tick}</DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>
                                Last Tick Duration (ms) <ClockIcon />
                            </DescriptionListTerm>
                            <DescriptionListDescription>{getLastTickDuration()}</DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>
                                Average Tick Duration (ms) <ClockIcon />
                            </DescriptionListTerm>
                            <DescriptionListDescription>{getAverageTickDuration()}</DescriptionListDescription>
                        </DescriptionListGroup>
                    </DescriptionList>
                </FlexItem>
                <FlexItem>
                    <ClipboardCheckIcon /> {<strong>Events Processed:</strong>} {props.workload?.num_events_processed}
                </FlexItem>
                <FlexItem>
                    <WorkloadEventTable workload={props.workload} />
                </FlexItem>
                <FlexItem>
                    <ClipboardCheckIcon /> {<strong>Sessions:</strong>} {props.workload?.num_sessions_created} /{' '}
                    {props.workload?.sessions.length} created, {props.workload?.num_active_trainings} actively training
                </FlexItem>
                <FlexItem>
                    <WorkloadSessionTable workload={props.workload} />
                </FlexItem>
            </Flex>
            <React.Fragment />
        </Modal>
    );
};
