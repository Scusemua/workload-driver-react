import React from 'react';
import { Button, DescriptionList, DescriptionListDescription, DescriptionListGroup, DescriptionListTerm, Flex, FlexItem, Label, Modal, ModalVariant, Title, TitleSizes, Tooltip } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { BlueprintIcon, CheckCircleIcon, ClipboardCheckIcon, ClockIcon, CpuIcon, DiceIcon, ExclamationTriangleIcon, HourglassStartIcon, OutlinedCalendarAltIcon, SpinnerIcon, TimesCircleIcon } from '@patternfly/react-icons';

import {
    Session,
    WORKLOAD_STATE_ERRED,
    WORKLOAD_STATE_FINISHED,
    WORKLOAD_STATE_READY,
    WORKLOAD_STATE_RUNNING,
    WORKLOAD_STATE_TERMINATED,
    Workload,
    WorkloadPreset,
    WorkloadTemplate,
    GetWorkloadStatusTooltip,
    WorkloadEvent,
} from '@app/Data/Workload';
import text from '@patternfly/react-styles/css/utilities/Text/text';
import { GpuIcon, GpuIconAlt, RamIcon } from '@app/Icons';

export interface InspectWorkloadModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
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

    // TODO: Add pagination to this table.
    // TODO: Define this table in its own file (probably).
    const events_table_columns: string[] = ["Index", "Event Name", "Target Session ID", "Event Timestamp", "IRL Timestamp"];
    const eventsTable = (
        <Table variant="compact">
            <Thead>
                <Tr>
                    {events_table_columns.map((column, columnIndex) => (
                        <Th key={columnIndex}>{column}</Th>
                    ))}
                </Tr>
            </Thead>
            <Tbody>
                {props.workload?.events_processed?.map((evt: WorkloadEvent, idx: number) => {
                    return (
                        <Tr key={props.workload?.events_processed[0]?.id}>
                            <Td dataLabel={events_table_columns[0]}>{idx}</Td>
                            <Td dataLabel={events_table_columns[1]}>{evt?.name}</Td>
                            <Td dataLabel={events_table_columns[2]}>{evt?.session}</Td>
                            <Td dataLabel={events_table_columns[3]}>{evt?.timestamp}</Td>
                            <Td dataLabel={events_table_columns[4]}>{evt?.processed_at}</Td>
                        </Tr>
                    )
                })}
            </Tbody>
        </Table>
    );

    // TODO: Add pagination to this table.
    // TODO: Define this table in its own file (probably).
    const sessions_table_columns: string[] = ["Index", "ID", "Status", "#Events Processed", "Max vCPUs", "Max Memory (GB)", "Max vGPUs"]
    const sessionTable = (
        <Table variant="compact">
            <Thead>
                <Tr>
                    {sessions_table_columns.map((column, columnIndex) => (
                        <Th key={columnIndex}>{column}</Th>
                    ))}
                </Tr>
            </Thead>
            <Tbody>
                {props.workload?.sessions?.map((session: Session, idx: number) => {
                    return (
                        <Tr key={props.workload?.events_processed[0]?.id}>
                            <Td dataLabel={sessions_table_columns[0]}>{idx}</Td>
                            <Td dataLabel={sessions_table_columns[1]}>{session?.id}</Td>
                            <Td dataLabel={sessions_table_columns[2]}>{session?.state}</Td>
                            <Td dataLabel={sessions_table_columns[3]}>{session?.num_events_processed}</Td>
                            <Td dataLabel={sessions_table_columns[4]}><CpuIcon/> {session?.max_cpus}</Td>
                            <Td dataLabel={sessions_table_columns[5]}><GpuIcon/> {session?.max_num_gpus}</Td>
                            <Td dataLabel={sessions_table_columns[6]}><RamIcon/> {session?.max_memory_gb}</Td>
                        </Tr>
                    )
                })}
            </Tbody>
        </Table>
    )

    console.log(`props.workload?.events_processed?: ${props.workload?.events_processed}`)

    return (
        <Modal
            variant={ModalVariant.large}
            titleIconVariant={"info"}
            header={header}
            isOpen={props.isOpen}
            width={1280}
            maxWidth={1920}
            onClose={props.onClose}
            actions={[
                <Button key="dismiss" variant="primary" onClick={props.onClose}>
                    Dismiss
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
                    </DescriptionList>
                </FlexItem>
                <FlexItem>
                    <ClipboardCheckIcon /> {<strong>Events Processed:</strong>} {props.workload?.num_events_processed}
                </FlexItem>
                <FlexItem>
                    {eventsTable}
                </FlexItem>
                <FlexItem>
                    <ClipboardCheckIcon /> {<strong>Sessions:</strong>} {props.workload?.num_sessions_created}
                </FlexItem>
                <FlexItem>
                    {sessionTable}
                </FlexItem>
            </Flex>
            <React.Fragment />
        </Modal>);
}