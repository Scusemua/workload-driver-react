import { Session, Workload } from '@Data/Workload';
import { Button, Card, CardBody, Label, Pagination, Text, Tooltip } from '@patternfly/react-core';
import {
    CopyIcon,
    CpuIcon,
    ErrorCircleOIcon,
    MemoryIcon,
    OffIcon,
    PendingIcon,
    ResourcesEmptyIcon,
    RunningIcon,
    UnknownIcon,
} from '@patternfly/react-icons';
import { Table, Tbody, Td, Th, Thead, ThProps, Tr } from '@patternfly/react-table';
import { GpuIcon, GpuIconAlt2 } from '@src/Assets/Icons';
import { RoundToThreeDecimalPlaces } from '@src/Utils';
import React from 'react';

export interface WorkloadSessionTableProps {
    children?: React.ReactNode;
    workload: Workload | null;
}

// Displays the Sessions from a workload in a table.
export const WorkloadSessionTable: React.FunctionComponent<WorkloadSessionTableProps> = (props) => {
    const [page, setPage] = React.useState(1);
    const [perPage, setPerPage] = React.useState(8);

    // Index of the currently sorted column
    const [activeSortIndex, setActiveSortIndex] = React.useState<number | null>(null);

    // Sort direction of the currently sorted column
    const [activeSortDirection, setActiveSortDirection] = React.useState<'asc' | 'desc' | null>(null);

    const [showCopySuccessContent, setShowCopySuccessContent] = React.useState(false);

    const copyText: string = 'Copy session ID to clipboard';
    const doneCopyText: string = 'Successfully copied session ID to clipboard!';

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

    const sessions_table_columns: string[] = [
        'Index',
        'ID',
        'Status',
        'Exec. Completed',
        'Exec. Remaining',
        'milliCPUs',
        'Memory (MB)',
        'vGPUs',
        'VRAM (GB)',
    ];

    const getSessionStatusLabel = (session: Session) => {
        const status: string = session.state;
        switch (status) {
            case 'awaiting start':
                return (
                    <Tooltip position="right" content="This session has not yet been created or started yet.">
                        <Label icon={<PendingIcon />} color="grey">
                            {status}
                        </Label>
                    </Tooltip>
                );
            case 'idle':
                return (
                    <Tooltip
                        position="right"
                        content="This session is actively-running, but it is not currently training."
                    >
                        <Label icon={<ResourcesEmptyIcon />} color="blue">
                            {status}
                        </Label>
                    </Tooltip>
                );
            case 'training':
                return (
                    <Tooltip position="right" content="This session is actively training.">
                        <Label icon={<RunningIcon />} color="green">
                            {status}
                        </Label>
                    </Tooltip>
                );
            case 'terminated':
                return (
                    <Tooltip position="right" content="This session has been stopped permanently (without error).">
                        <Label icon={<OffIcon />} color="orange">
                            {status}
                        </Label>
                    </Tooltip>
                );
            case 'erred':
                return (
                    <Tooltip
                        position="right"
                        content={`This session has been terminated due to an unexpected error: ${session.error_message}`}
                    >
                        <Label icon={<ErrorCircleOIcon />} color="red">
                            {' '}
                            {status}
                        </Label>
                    </Tooltip>
                );
            default:
                return (
                    <Tooltip position="right" content="This session is in an unknown or unexpected state.">
                        <Label icon={<UnknownIcon />} color="red">
                            {' '}
                            unknown: {status}
                        </Label>
                    </Tooltip>
                );
        }
    };

    /**
     * Return the number of trainings that the given session has left to complete, if that information is available.
     *
     * If that information is not available, then return the string "N/A".
     */
    const getRemainingTrainings = (session: Session): string | number => {
        if (session.trainings) {
            return session.trainings.length - session.trainings_completed;
        }

        return 'N/A';
    };

    // Since OnSort specifies sorted columns by index, we need sortable values for our object by column index.
    // This example is trivial since our data objects just contain strings, but if the data was more complex
    // this would be a place to return simplified string or number versions of each column to sort by.
    const getSortableRowValues = (session: Session): (string | number | Date)[] => {
        const { id, state, trainings, trainings_completed, current_resource_request } = session;

        return [
            id,
            state,
            trainings_completed,
            trainings.length - trainings_completed,
            current_resource_request.cpus,
            current_resource_request.memory_mb,
            current_resource_request.gpus,
            current_resource_request.vram,
        ];
    };

    // Note that we perform the sort as part of the component's render logic and not in onSort.
    // We shouldn't store the list of data in state because we don't want to have to sync that with props.
    let sortedSessions = props.workload?.sessions || [];
    if (activeSortIndex !== null) {
        sortedSessions =
            props.workload?.sessions.sort((a, b) => {
                const aValue = getSortableRowValues(a)[activeSortIndex];
                const bValue = getSortableRowValues(b)[activeSortIndex];
                console.log(
                    `Sorting ${aValue} and ${bValue} (activeSortIndex = ${activeSortIndex}, activeSortDirection = '${activeSortDirection}', activeSortColumn='${sessions_table_columns[activeSortIndex]}')`,
                );
                if (typeof aValue === 'number') {
                    // Numeric sort
                    if (activeSortDirection === 'asc') {
                        return (aValue as number) - (bValue as number);
                    }
                    return (bValue as number) - (aValue as number);
                } else {
                    // String sort
                    if (activeSortDirection === 'asc') {
                        return (aValue as string).localeCompare(bValue as string);
                    }
                    return (bValue as string).localeCompare(aValue as string);
                }
            }) || [];
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

    const filteredSessions: Session[] | undefined = sortedSessions.slice(
        perPage * (page - 1),
        perPage * (page - 1) + perPage,
    );

    return (
        <Card isCompact isRounded isFlat>
            <CardBody>
                <Table variant="compact" borders={true} isStriped>
                    <Thead noWrap>
                        <Tr>
                            {sessions_table_columns.map((column, columnIndex) => (
                                <Th
                                    key={`workload_${props.workload?.id}_column_${columnIndex}`}
                                    sort={columnIndex > 0 ? getSortParams(columnIndex - 1) : undefined}
                                    aria-label={`${column}-column`}
                                >
                                    {column}
                                </Th>
                            ))}
                        </Tr>
                    </Thead>
                    <Tbody>
                        {filteredSessions?.map((session: Session, idx: number) => {
                            return (
                                <Tr key={`workload_event_${props.workload?.events_processed[0]?.id}_row_${idx}`}>
                                    <Td dataLabel={sessions_table_columns[0]}>{idx}</Td>
                                    <Td dataLabel={sessions_table_columns[1]}>
                                        <Text component={'small'}>{session.id}</Text>
                                        <Tooltip
                                            content={showCopySuccessContent ? doneCopyText : copyText}
                                            position={'right'}
                                            entryDelay={75}
                                            exitDelay={200}
                                            onTooltipHidden={() => setShowCopySuccessContent(false)}
                                        >
                                            <Button
                                                icon={<CopyIcon />}
                                                variant={'plain'}
                                                onClick={async (event) => {
                                                    event.preventDefault();
                                                    await navigator.clipboard.writeText(session.id);

                                                    setShowCopySuccessContent(!showCopySuccessContent);
                                                }}
                                            />
                                        </Tooltip>
                                    </Td>
                                    <Td dataLabel={sessions_table_columns[2]}>{getSessionStatusLabel(session)}</Td>
                                    <Td dataLabel={sessions_table_columns[3]}>{session.trainings_completed || '0'}</Td>
                                    <Td dataLabel={sessions_table_columns[4]}>{getRemainingTrainings(session)}</Td>
                                    <Td dataLabel={sessions_table_columns[5]}>
                                        <CpuIcon />{' '}
                                        {session?.current_resource_request
                                            ? RoundToThreeDecimalPlaces(session?.current_resource_request.cpus)
                                            : 0}
                                        {'/'}
                                        {RoundToThreeDecimalPlaces(session?.max_resource_request.cpus)}
                                    </Td>
                                    <Td dataLabel={sessions_table_columns[6]}>
                                        <MemoryIcon />
                                        {session?.current_resource_request.memory_mb
                                            ? RoundToThreeDecimalPlaces(session?.current_resource_request.memory_mb)
                                            : 0}
                                        {'/'}
                                        {RoundToThreeDecimalPlaces(session?.max_resource_request.memory_mb)}
                                    </Td>
                                    <Td dataLabel={sessions_table_columns[7]}>
                                        <GpuIcon />
                                        {session?.current_resource_request.gpus
                                            ? session?.current_resource_request.gpus
                                            : 0}
                                        {'/'}
                                        {RoundToThreeDecimalPlaces(session?.max_resource_request.gpus)}
                                    </Td>
                                    <Td dataLabel={sessions_table_columns[8]}>
                                        <GpuIconAlt2 />
                                        {session?.current_resource_request.vram
                                            ? session?.current_resource_request.vram
                                            : 0}
                                        {'/'}
                                        {session?.max_resource_request.vram}
                                    </Td>
                                </Tr>
                            );
                        })}
                    </Tbody>
                </Table>
                <Pagination
                    itemCount={props.workload?.sessions.length}
                    perPage={perPage}
                    page={page}
                    perPageOptions={[
                        { title: '1 session', value: 1 },
                        { title: '2 sessions', value: 2 },
                        {
                            title: '3 sessions',
                            value: 3,
                        },
                        { title: '4 sessions', value: 4 },
                        { title: '5 sessions', value: 5 },
                        {
                            title: '10 sessions',
                            value: 10,
                        },
                        { title: '25 sessions', value: 25 },
                        { title: '50 sessions', value: 50 },
                    ]}
                    onSetPage={onSetPage}
                    onPerPageSelect={onPerPageSelect}
                    ouiaId="WorkloadSessionsPagination"
                />
            </CardBody>
        </Card>
    );
};
