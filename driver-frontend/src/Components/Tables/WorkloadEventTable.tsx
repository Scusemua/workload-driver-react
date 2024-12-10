import { Workload, WorkloadEvent } from '@Data/Workload';
import { Card, CardBody, Label, Pagination, Tooltip } from '@patternfly/react-core';
import {
    CheckCircleIcon,
    CheckIcon,
    ErrorCircleOIcon,
    ExclamationCircleIcon,
    MigrationIcon,
    MonitoringIcon,
    OffIcon,
    PendingIcon,
    QuestionCircleIcon,
    StarIcon,
} from '@patternfly/react-icons';
import { Table, Tbody, Td, Th, Thead, ThProps, Tr } from '@patternfly/react-table';
import React, { ReactElement } from 'react';

export interface WorkloadEventTableProps {
    children?: React.ReactNode;
    workload: Workload | null;
    showDiscardedEvents?: boolean;
}

export function GetEventLabel(event_name: string): ReactElement {
    switch (event_name) {
        case 'workload-started':
            return (
                <Label color="gold" icon={<StarIcon />}>
                    {event_name}
                </Label>
            );
        case 'workload-complete':
            return (
                <Label color="purple" icon={<CheckCircleIcon />}>
                    {event_name}
                </Label>
            );
        case 'session-started':
            return (
                <Label color="cyan" icon={<MigrationIcon />}>
                    {event_name}
                </Label>
            );
        case 'session-ready':
            return (
                <Label color="grey" icon={<PendingIcon />}>
                    {event_name}
                </Label>
            );
        case 'training':
            return (
                <Label color="green" icon={<MonitoringIcon />}>
                    {event_name}
                </Label>
            );
        case 'training-started':
            return (
                <Label color="green" icon={<MonitoringIcon />}>
                    {event_name}
                </Label>
            );
        case 'training-ended':
            return (
                <Label color="blue" icon={<CheckIcon />}>
                    {event_name}
                </Label>
            );
        case 'session-stopped':
            return (
                <Label color="orange" icon={<OffIcon />}>
                    {event_name}
                </Label>
            );
        case 'update-gpu-util':
            return <Label color="grey">{event_name}</Label>;
        case 'workload-terminated':
            return (
                <Label color="orange" icon={<ExclamationCircleIcon />}>
                    {event_name}
                </Label>
            );
        case null:
            return (
                <Label color="gold" icon={<QuestionCircleIcon />}>
                    Unknown
                </Label>
            );
        default:
            // console.error(`Unexpected event name: "${event_name}"`);
            return (
                <Label color="gold" icon={<QuestionCircleIcon />}>
                    {event_name}
                </Label>
            );
    }
}

export const WorkloadEventTable: React.FunctionComponent<WorkloadEventTableProps> = (props) => {
    const [page, setPage] = React.useState(1);
    const [perPage, setPerPage] = React.useState(10);

    // Index of the currently sorted column
    const [activeSortIndex, setActiveSortIndex] = React.useState<number | null>(null);

    // Sort direction of the currently sorted column
    const [activeSortDirection, setActiveSortDirection] = React.useState<'asc' | 'desc' | null>(null);

    // Since OnSort specifies sorted columns by index, we need sortable values for our object by column index.
    // This example is trivial since our data objects just contain strings, but if the data was more complex
    // this would be a place to return simplified string or number versions of each column to sort by.
    const getSortableRowValues = (event: WorkloadEvent): (string | number | Date)[] => {
        const { idx, name, session, timestamp, processed_at, processed_successfully } = event;
        const timestamp_adjusted: string = timestamp.substring(0, timestamp.length - 10);
        const processed_at_adjusted: string = processed_at.substring(0, 27);

        // console.log(`Timestamp Adjusted: ${timestamp_adjusted}, Processed-At Adjusted: ${processed_at_adjusted}`)

        // Note: We're omitting the event's "id" and "error_message" fields here.
        return [
            idx,
            name,
            session,
            Date.parse(timestamp_adjusted),
            Date.parse(processed_at_adjusted),
            processed_successfully ? 1 : 0,
        ];
    };

    // Note that we perform the sort as part of the component's render logic and not in onSort.
    // We shouldn't store the list of data in state because we don't want to have to sync that with props.
    let sortedEvents = props.workload?.statistics.events_processed;
    if (activeSortIndex !== null) {
        sortedEvents = props.workload?.statistics.events_processed.sort((a, b) => {
            const aValue = getSortableRowValues(a)[activeSortIndex];
            const bValue = getSortableRowValues(b)[activeSortIndex];
            // console.log(
            //     `Sorting ${aValue} and ${bValue} (activeSortIndex = ${activeSortIndex}, activeSortDirection = '${activeSortDirection}')`,
            // );
            if (typeof aValue === 'number') {
                // Numeric sort
                if (activeSortDirection === 'asc') {
                    return (aValue as number) - (bValue as number);
                }
                return (bValue as number) - (aValue as number);
            } else if (aValue instanceof Date && bValue instanceof Date) {
                const aDate: Date = aValue as Date;
                const bDate: Date = bValue as Date;
                if (activeSortDirection === 'asc') {
                    return aDate.getTime() - bDate.getTime();
                }
                return bDate.getTime() - aDate.getTime();
            } else {
                // String sort
                if (activeSortDirection === 'asc') {
                    return (aValue as string).localeCompare(bValue as string);
                }
                return (bValue as string).localeCompare(aValue as string);
            }
        });
    }

    const getSortParams = (columnIndex: number): ThProps['sort'] => ({
        sortBy: {
            index: activeSortIndex!,
            direction: activeSortDirection!,
            defaultDirection: 'asc', // starting sort direction when first sorting a column. Defaults to 'asc'
        },
        onSort: (_event, index, direction) => {
            setActiveSortIndex(index);
            setActiveSortDirection(direction);
        },
        columnIndex,
    });

    const onSetPage = (_event: React.MouseEvent | React.KeyboardEvent | MouseEvent, newPage: number) => {
        setPage(newPage);
    };

    const onPerPageSelect = (
        _event: React.MouseEvent | React.KeyboardEvent | MouseEvent,
        newPerPage: number,
        newPage: number,
    ) => {
        setPerPage(newPerPage);
        setPage(newPage);
    };

    const events_table_columns: string[] = [
        'Index',
        'Name',
        'Target Session',
        'Event Timestamp',
        'IRL Timestamp',
        'Status',
    ];

    const getStatusLabel = (evt: WorkloadEvent) => {
        if (evt.processed_successfully) {
            return (
                <Tooltip position="left-start" content={'The event was processed successfully.'}>
                    <Label color="green" icon={<CheckCircleIcon />}>
                        Processed
                    </Label>
                </Tooltip>
            );
        } else {
            return (
                <Tooltip
                    position="left-start"
                    content={`The event was NOT processed successfully. Reason: ${evt.error_message !== undefined ? evt.error_message : 'N/A'}`}
                >
                    <Label color="red" icon={<ErrorCircleOIcon />}>
                        {evt.status || 'Error'}
                    </Label>
                </Tooltip>
            );
        }
    };

    let filteredEvents: WorkloadEvent[] | undefined = sortedEvents?.slice(
        perPage * (page - 1),
        perPage * (page - 1) + perPage,
    );

    if (!props.showDiscardedEvents) {
        filteredEvents = filteredEvents?.filter((event: WorkloadEvent) => event.status != 'Discarded');
    }

    return (
        <Card isCompact isRounded isFlat>
            <CardBody>
                <Table variant="compact" isStriped>
                    <Thead noWrap>
                        <Tr>
                            {events_table_columns.map((column, columnIndex) => (
                                <Th
                                    key={`workload-${props.workload?.id}-column-${columnIndex}`}
                                    sort={getSortParams(columnIndex)}
                                    aria-label={`${column}-column`}
                                >
                                    {column}
                                </Th>
                            ))}
                        </Tr>
                    </Thead>
                    <Tbody>
                        {filteredEvents?.map((evt: WorkloadEvent) => {
                            return (
                                <Tr key={`workload-${props.workload?.id}-event-${evt?.idx}`}>
                                    <Td dataLabel={events_table_columns[0]}>{evt?.idx}</Td>
                                    <Td dataLabel={events_table_columns[1]}>{GetEventLabel(evt?.name)}</Td>
                                    <Td dataLabel={events_table_columns[2]}>{evt?.session}</Td>
                                    <Td dataLabel={events_table_columns[3]}>
                                        {evt?.timestamp.substring(0, evt?.timestamp.length - 10)}
                                    </Td>
                                    <Td dataLabel={events_table_columns[4]}>{evt?.processed_at.substring(0, 27)}</Td>
                                    <Td dataLabel={events_table_columns[5]}>{getStatusLabel(evt)}</Td>
                                </Tr>
                            );
                        })}
                    </Tbody>
                </Table>
                <Pagination
                    itemCount={sortedEvents?.length}
                    isDisabled={sortedEvents?.length == 0}
                    perPage={perPage}
                    page={page}
                    perPageOptions={[
                        { title: '1 event', value: 1 },
                        { title: '2 events', value: 2 },
                        { title: '3 events', value: 3 },
                        { title: '4 events', value: 4 },
                        { title: '5 events', value: 5 },
                        { title: '10 events', value: 10 },
                        { title: '25 events', value: 25 },
                        { title: '50 events', value: 50 },
                        { title: '100 events', value: 100 },
                    ]}
                    onSetPage={onSetPage}
                    onPerPageSelect={onPerPageSelect}
                    ouiaId="WorkloadEventsPagination"
                />
            </CardBody>
        </Card>
    );
};
