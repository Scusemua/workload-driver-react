import React from 'react';
import { Button, DescriptionList, DescriptionListDescription, DescriptionListGroup, DescriptionListTerm, Flex, FlexItem, Label, Modal, ModalVariant, Title, TitleSizes, Tooltip } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { BlueprintIcon, CheckCircleIcon, ClipboardCheckIcon, ClockIcon, CloseIcon, CodeIcon, CpuIcon, DiceIcon, ErrorCircleOIcon, ExclamationTriangleIcon, HourglassStartIcon, MemoryIcon, MigrationIcon, MonitoringIcon, OffIcon, OutlinedCalendarAltIcon, PendingIcon, PlayIcon, QuestionCircleIcon, ResourcesEmptyIcon, RunningIcon, SpinnerIcon, StopIcon, Stopwatch20Icon, StopwatchIcon, TimesCircleIcon, UnknownIcon, UserClockIcon } from '@patternfly/react-icons';

import {
    Session,
    WORKLOAD_STATE_ERRED,
    WORKLOAD_STATE_FINISHED,
    WORKLOAD_STATE_READY,
    WORKLOAD_STATE_RUNNING,
    WORKLOAD_STATE_TERMINATED,
    Workload,
    WorkloadEvent,
} from '@app/Data/Workload';
import text from '@patternfly/react-styles/css/utilities/Text/text';
import { GpuIcon, GpuIconAlt, RamIcon } from '@app/Icons';
import { WorkloadEventTable, WorkloadSessionTable } from '../Cards';

export interface InspectWorkloadModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    onStartClicked: () => void;
    onStopClicked: () => void;
    workload: Workload | null;
}

export const InspectWorkloadModal: React.FunctionComponent<InspectWorkloadModalProps> = (props) => {
    const workloadStatus = (
        <React.Fragment>
            {props.workload?.workload_state ==
                WORKLOAD_STATE_READY && (
                    <Label icon={<HourglassStartIcon
                        className={
                            text.infoColor_100
                        }
                    />} color="blue">Ready</Label>
                )}
            {props.workload?.workload_state ==
                WORKLOAD_STATE_RUNNING && (
                    <Label icon={<SpinnerIcon
                        className={
                            'loading-icon-spin ' +
                            text.successColor_100
                        }
                    />} color="green">Running</Label>
                )}
            {props.workload?.workload_state ==
                WORKLOAD_STATE_FINISHED && (
                    <Label icon={<CheckCircleIcon
                        className={
                            text.successColor_100
                        }
                    />} color="green">Complete</Label>
                )}
            {props.workload?.workload_state ==
                WORKLOAD_STATE_ERRED && (
                    <Label icon={<TimesCircleIcon
                        className={
                            text.dangerColor_100
                        }
                    />} color="red">Erred</Label>
                )}
            {props.workload?.workload_state ==
                WORKLOAD_STATE_TERMINATED && (
                    <Label icon={<ExclamationTriangleIcon
                        className={
                            text.warningColor_100
                        }
                    />} color="orange">Terminated</Label>
                )}
        </React.Fragment>
    );

    const header = (
        <React.Fragment>
            <Title headingLevel='h1' size={TitleSizes['2xl']}>
                {`Workload ${props.workload?.name} `}

                {workloadStatus}
            </Title>
        </React.Fragment>
    )

    // const getSessionStatusLabel = (session: Session) => {
    //     const status: string = session.state;
    //     switch (status) {
    //         case "awaiting start":
    //             return (<Tooltip content="This session has not yet been created or started yet."><Label icon={<PendingIcon />} color='grey'>{status}</Label></Tooltip>);
    //         case "idle":
    //             return (<Tooltip content="This session is actively-running, but it is not currently training."><Label icon={<ResourcesEmptyIcon />} color='grey'>{status}</Label></Tooltip>);
    //         case "training":
    //             return (<Tooltip content="This session is actively training."><Label icon={<RunningIcon />} color='green'>{status}</Label></Tooltip>);
    //         case "terminated":
    //             return (<Tooltip content="This session has been stopped permanently (without error)."><Label icon={<OffIcon />} color='gold'>{status}</Label></Tooltip>);
    //         case "erred":
    //             return (<Tooltip content={`This session has been terminated due to an unexpected error: ${session.error_message}`}><Label icon={<ErrorCircleOIcon />} color='red'> {status}</Label></Tooltip>);
    //         default:
    //             return (<Tooltip content="This session is in an unknown or unexpected state."><Label icon={<UnknownIcon />} color='orange'> unknown: {status}</Label></Tooltip>);
    //     }
    // }

    // TODO: Add pagination to this table.
    // TODO: Define this table in its own file (probably).
    // const events_table_columns: string[] = ["Index", "Event Name", "Target Session ID", "Event Timestamp", "IRL Timestamp"];
    // const eventsTable = (
    //     <Table variant="compact">
    //         <Thead>
    //             <Tr>
    //                 {events_table_columns.map((column, columnIndex) => (
    //                     <Th key={columnIndex}>{column}</Th>
    //                 ))}
    //             </Tr>
    //         </Thead>
    //         <Tbody>
    //             {props.workload?.events_processed?.map((evt: WorkloadEvent, idx: number) => {
    //                 return (
    //                     <Tr key={props.workload?.events_processed[0]?.id}>
    //                         <Td dataLabel={events_table_columns[0]}>{idx}</Td>
    //                         <Td dataLabel={events_table_columns[1]}>{getEventLabel(evt?.name)}</Td>
    //                         <Td dataLabel={events_table_columns[2]}>{evt?.session}</Td>
    //                         <Td dataLabel={events_table_columns[3]}>{evt?.timestamp}</Td>
    //                         <Td dataLabel={events_table_columns[4]}>{evt?.processed_at}</Td>
    //                     </Tr>
    //                 )
    //             })}
    //         </Tbody>
    //     </Table>
    // );

    // TODO: Add pagination to this table.
    // TODO: Define this table in its own file (probably).
    // const sessions_table_columns: string[] = ["Index", "ID", "Status", "Trainings Completed", "Max vCPUs", "Max Memory (GB)", "Max vGPUs"]
    // const sessionTable = (
    //     <Table variant="compact">
    //         <Thead>
    //             <Tr>
    //                 {sessions_table_columns.map((column, columnIndex) => (
    //                     <Th key={columnIndex}>{column}</Th>
    //                 ))}
    //             </Tr>
    //         </Thead>
    //         <Tbody>
    //             {props.workload?.sessions?.map((session: Session, idx: number) => {
    //                 return (
    //                     <Tr key={props.workload?.events_processed[0]?.id}>
    //                         <Td dataLabel={sessions_table_columns[0]}>{idx}</Td>
    //                         <Td dataLabel={sessions_table_columns[1]}>{session.id}</Td>
    //                         <Td dataLabel={sessions_table_columns[2]}>{getSessionStatusLabel(session)}</Td>
    //                         <Td dataLabel={sessions_table_columns[3]}>{session.trainings_completed || '0'}</Td>
    //                         <Td dataLabel={sessions_table_columns[4]}><CpuIcon /> {session?.max_cpus}</Td>
    //                         <Td dataLabel={sessions_table_columns[5]}><GpuIcon /> {session?.max_num_gpus}</Td>
    //                         <Td dataLabel={sessions_table_columns[6]}><MemoryIcon /> {session?.max_memory_gb}</Td>
    //                     </Tr>
    //                 )
    //             })}
    //         </Tbody>
    //     </Table>
    // )

    const getTimeElapsedString = () => {
        if (props.workload?.workload_state === undefined || props.workload?.workload_state === "") {
            return "N/A"
        }

        return props.workload?.time_elapsed_str;
    }

    // console.log(`props.workload?.events_processed?: ${props.workload?.events_processed}`)

    return (
        <Modal
            variant={ModalVariant.large}
            titleIconVariant={"info"}
            header={header}
            aria-label="inspect-workload-modal"
            isOpen={props.isOpen}
            width={1500}
            maxWidth={1920}
            onClose={props.onClose}
            actions={[
                <Button key="start-workload" variant="primary" icon={<PlayIcon />} onClick={props.onStartClicked} isDisabled={props.workload?.workload_state != WORKLOAD_STATE_READY}>
                    Start Workload
                </Button>,
                <Button key="stop-workload" variant="danger" icon={<StopIcon />} onClick={props.onStopClicked} isDisabled={props.workload?.workload_state != WORKLOAD_STATE_RUNNING}>
                    Stop Workload
                </Button>,
                <Button key="dismiss-workload" variant="secondary" icon={<CloseIcon />} onClick={props.onClose}>
                    Close Window
                </Button>,
            ]}
        >
            <Flex direction={{ 'default': 'column' }}>
                <FlexItem>
                    <DescriptionList columnModifier={{ lg: '3Col' }}>
                        {props.workload?.workload_preset && <DescriptionListGroup>
                            <DescriptionListTerm>Workload Preset <BlueprintIcon /> </DescriptionListTerm>
                            <DescriptionListDescription>&quot;{props.workload?.workload_preset_name}&quot;</DescriptionListDescription>
                        </DescriptionListGroup>}
                        {props.workload?.workload_template && <DescriptionListGroup>
                            <DescriptionListTerm>Workload Template <BlueprintIcon /></DescriptionListTerm>
                            <DescriptionListDescription>&quot;{props.workload?.workload_template.name}&quot;</DescriptionListDescription>
                        </DescriptionListGroup>}
                        <DescriptionListGroup>
                            <DescriptionListTerm>Seed <DiceIcon /> </DescriptionListTerm>
                            <DescriptionListDescription>{props.workload?.seed}</DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Time Adjustment Factor <ClockIcon /> </DescriptionListTerm>
                            <DescriptionListDescription>{props.workload?.timescale_adjustment_factor}</DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Events Processed <MonitoringIcon /></DescriptionListTerm>
                            <DescriptionListDescription>{props.workload?.num_events_processed}</DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Training Events Completed <CodeIcon /></DescriptionListTerm>
                            <DescriptionListDescription>{props.workload?.num_tasks_executed}</DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Time Elapsed <StopwatchIcon /></DescriptionListTerm>
                            <DescriptionListDescription>{getTimeElapsedString()}</DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Workload Clocktime <UserClockIcon /></DescriptionListTerm>
                            <DescriptionListDescription>{props.workload?.simulation_clock_time == '' ? 'N/A' : props.workload?.simulation_clock_time}</DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Current Tick <Stopwatch20Icon /></DescriptionListTerm>
                            <DescriptionListDescription>{props.workload?.current_tick}</DescriptionListDescription>
                        </DescriptionListGroup>
                    </DescriptionList>
                </FlexItem>
                <FlexItem>
                    <ClipboardCheckIcon /> {<strong>Events Processed:</strong>} {props.workload?.num_events_processed}
                </FlexItem>
                <FlexItem>
                    <WorkloadEventTable workload={props.workload}/>
                </FlexItem>
                <FlexItem>
                    <ClipboardCheckIcon /> {<strong>Sessions:</strong>} {props.workload?.num_sessions_created} / {props.workload?.sessions.length} created, {props.workload?.num_active_trainings} actively training
                </FlexItem>
                <FlexItem>
                    <WorkloadSessionTable workload={props.workload}/>
                </FlexItem>
            </Flex>
            <React.Fragment />
        </Modal>);
}