import React from 'react';
import { Button, DescriptionList, DescriptionListDescription, DescriptionListGroup, DescriptionListTerm, Flex, FlexItem, Label, Modal, ModalVariant, Title, TitleSizes, Tooltip } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { BlueprintIcon, CheckCircleIcon, ClipboardCheckIcon, ClockIcon, DiceIcon, ExclamationTriangleIcon, HourglassStartIcon, OutlinedCalendarAltIcon, SpinnerIcon, TimesCircleIcon } from '@patternfly/react-icons';

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

export interface InspectWorkloadModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    workload: Workload | null;
}

export const InspectWorkloadModal: React.FunctionComponent<InspectWorkloadModalProps> = (props) => {
    const events_processed_columns: string[] = ["Event Name", "Target Session ID", "Event Timestamp", "IRL Timestamp"];

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
                    // <React.Fragment>

                    //     Running
                    // </React.Fragment>
                )}
            {props.workload?.workload_state ==
                WORKLOAD_STATE_FINISHED && (
                    <Label icon={<CheckCircleIcon
                        className={
                            text.successColor_100
                        }
                    />} color="green">Complete</Label>
                    // <React.Fragment>
                    //     <CheckCircleIcon
                    //         className={
                    //             text.successColor_100
                    //         }
                    //     />
                    //     {' Complete'}
                    // </React.Fragment>
                )}
            {props.workload?.workload_state ==
                WORKLOAD_STATE_ERRED && (
                    <Label icon={<TimesCircleIcon
                        className={
                            text.dangerColor_100
                        }
                    />} color="red">Erred</Label>
                    // <React.Fragment>
                    //     <TimesCircleIcon
                    //         className={
                    //             text.dangerColor_100
                    //         }
                    //     />
                    //     {' Erred'}
                    // </React.Fragment>
                )}
            {props.workload?.workload_state ==
                WORKLOAD_STATE_TERMINATED && (
                    <Label icon={<ExclamationTriangleIcon
                        className={
                            text.warningColor_100
                        }
                    />} color="orange">Terminated</Label>
                    // <React.Fragment>
                    //     <ExclamationTriangleIcon
                    //         className={
                    //             text.warningColor_100
                    //         }
                    //     />
                    //     {' Terminated'}
                    // </React.Fragment>
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

    console.log(`props.workload?.events_processed?: ${props.workload?.events_processed}`)

    return (
        <Modal
            variant={ModalVariant.large}
            titleIconVariant={"info"}
            header={header}
            isOpen={props.isOpen}
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
                    <Table variant="compact">
                        <Thead>
                            <Tr>
                                {events_processed_columns.map((column, columnIndex) => (
                                    <Th key={columnIndex}>{column}</Th>
                                ))}
                            </Tr>
                        </Thead>
                        <Tbody>
                            {props.workload?.events_processed?.map((evt: WorkloadEvent) => {
                                return (
                                    <Tr key={props.workload?.events_processed[0]?.id}>
                                        <Td dataLabel={events_processed_columns[0]}>{evt?.name}</Td>
                                        <Td dataLabel={events_processed_columns[1]}>{evt?.session}</Td>
                                        <Td dataLabel={events_processed_columns[2]}>{evt?.timestamp}</Td>
                                        <Td dataLabel={events_processed_columns[3]}>{evt?.processed_at}</Td>
                                    </Tr>
                                )
                            })}
                        </Tbody>
                    </Table>
                </FlexItem>
            </Flex>
            <React.Fragment />
        </Modal>);
}