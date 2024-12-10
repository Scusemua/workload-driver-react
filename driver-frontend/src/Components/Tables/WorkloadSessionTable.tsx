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
    WarningTriangleIcon,
} from '@patternfly/react-icons';
import { ExpandableRowContent, Table, Tbody, Td, Th, Thead, ThProps, Tr } from '@patternfly/react-table';
import { GpuIcon, GpuIconAlt2 } from '@src/Assets/Icons';
import { SessionTrainingEventTable } from '@src/Components';
import { RoundToThreeDecimalPlaces, RoundToTwoDecimalPlaces } from '@src/Utils';
import React, { ReactElement } from 'react';

const tableColumns = {
    id: 'ID',
    status: 'Status',
    completedExecutions: 'Completed Executions',
    remainingExecutions: 'RemainingExecutions',
    millicpus: 'Millicpus',
    memory: 'Memory (MB)',
    gpus: 'GPUs',
    // currentGpus: 'Current vGPUs',
    // maxGpus: 'Max vGPUs',
    vram: 'VRAM (GB)',
};

const sessions_table_columns: string[] = [
    'ID',
    'Status',
    'Completed Exec.',
    'Remaining Exec.',
    'Millicpus',
    'DRAM (MB)',
    'GPUs',
    // 'Current vGPUs',
    // 'Max vGPUs',
    'VRAM (GB)',
];

/**
 * Return the number of trainings that the given session has left to complete, if that information is available.
 *
 * If that information is not available, then return the string "N/A".
 */
function getRemainingTrainings(session: Session): string | number {
    if (session.trainings) {
        return session.trainings.length - session.trainings_completed;
    }

    return 'N/A';
}

function getSessionStatusLabel(session: Session): ReactElement {
    if (session.discarded) {
        return (
            <Tooltip position="right" content="This session was discarded and will not be sampled in this workload.">
                <Label icon={<WarningTriangleIcon />} color="orange">
                    discarded
                </Label>
            </Tooltip>
        );
    }

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
                <Tooltip position="right" content="This session is actively-running, but it is not currently training.">
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
}

// Since OnSort specifies sorted columns by index, we need sortable values for our object by column index.
// This example is trivial since our data objects just contain strings, but if the data was more complex
// this would be a place to return simplified string or number versions of each column to sort by.
function getSortableRowValues(session: Session): (string | number | Date)[] {
    const { id, state, trainings, trainings_completed, current_resource_request, max_resource_request } = session;

    let status: string = state;
    if (session.discarded) {
        status = 'discarded';
    }

    return [
        id,
        status,
        trainings_completed,
        trainings.length - trainings_completed,
        current_resource_request.cpus,
        current_resource_request.memory,
        current_resource_request.gpus,
        max_resource_request.gpus,
        current_resource_request.vram,
    ];
}

export interface WorkloadSessionTableProps {
    children?: React.ReactNode;
    workload: Workload | null;
    showDiscardedSessions?: boolean;
}

// Displays the Sessions from a workload in a table.
export const WorkloadSessionTable: React.FunctionComponent<WorkloadSessionTableProps> = (props) => {
    const [page, setPage] = React.useState(1);
    const [perPage, setPerPage] = React.useState(5);

    // Index of the currently sorted column
    const [activeSortIndex, setActiveSortIndex] = React.useState<number | null>(null);

    const [expandedSessions, setExpandedSessions] = React.useState<string[]>([]);

    // Sort direction of the currently sorted column
    const [activeSortDirection, setActiveSortDirection] = React.useState<'asc' | 'desc' | null>(null);

    const [showCopySuccessContent, setShowCopySuccessContent] = React.useState(false);

    const [sortedSessions, setSortedSessions] = React.useState<Session[]>([]);

    React.useEffect(() => {
        let sorted =
            props.workload?.sessions.filter((session: Session) => {
                return props.showDiscardedSessions || !session.discarded;
            }) || [];
        if (activeSortIndex !== null) {
            sorted =
                sorted.sort((a, b) => {
                    const aValue = getSortableRowValues(a)[activeSortIndex];
                    const bValue = getSortableRowValues(b)[activeSortIndex];
                    // console.log(
                    //     `Sorting ${aValue} and ${bValue} (activeSortIndex = ${activeSortIndex}, activeSortDirection =
                    //     '${activeSortDirection}', activeSortColumn='${sessions_table_columns[activeSortIndex]}')`,
                    // );
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

        setSortedSessions(sorted);
    }, [activeSortDirection, activeSortIndex, props.workload, props.workload?.sessions, props.showDiscardedSessions]);

    const copyText: string = 'Copy session ID to clipboard';
    const doneCopyText: string = 'Successfully copied session ID to clipboard!';

    const setSessionExpanded = (session: Session, isExpanding = true) =>
        setExpandedSessions((prevExpanded) => {
            const otherExpandedSessionNames = prevExpanded.filter((r) => r !== session.id);
            return isExpanding ? [...otherExpandedSessionNames, session.id] : otherExpandedSessionNames;
        });
    const isSessionExpanded = (session: Session) => expandedSessions.includes(session.id);

    const onPerPageSelect = (
        _event: React.MouseEvent | React.KeyboardEvent | MouseEvent,
        newPerPage: number,
        newPage: number,
    ) => {
        setPerPage(newPerPage);
        setPage(newPage);
    };

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

    const tableHead = (
        <Thead noWrap>
            <Tr>
                <Th
                    key={`workload_${props.workload?.id}_column_expand_action`}
                    aria-label={`workload_${props.workload?.id}_column_expand_action`}
                />
                {sessions_table_columns.map((column, columnIndex) => (
                    <Th
                        key={`workload_${props.workload?.id}_column_${columnIndex}`}
                        sort={getSortParams(columnIndex)}
                        aria-label={`${column}-column`}
                    >
                        {column}
                    </Th>
                ))}
            </Tr>
        </Thead>
    );

    const pagination = (
        <Pagination
            itemCount={sortedSessions.length}
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
            onSetPage={(_event, newPage: number) => setPage(newPage)}
            onPerPageSelect={onPerPageSelect}
            ouiaId="WorkloadSessionsPagination"
        />
    );

    const getTableRow = (rowIndex: number): ReactElement | undefined => {
        const session: Session = sortedSessions[rowIndex];

        if (!props.showDiscardedSessions && session.discarded) {
            return undefined;
        }

        // key={`workload_event_${props.workload?.events_processed[0]?.id}_row_${rowIndex}`}
        return (
            <Tbody key={`session-${session.id}-row-${rowIndex}`}>
                <Tr>
                    <Td
                        expand={
                            session.trainings.length > 0
                                ? {
                                      rowIndex,
                                      isExpanded: isSessionExpanded(session),
                                      onToggle: () => setSessionExpanded(session, !isSessionExpanded(session)),
                                      expandId: 'composable-nested-table-expandable-example',
                                  }
                                : undefined
                        }
                    />
                    <Td dataLabel={tableColumns.id}>
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
                    <Td dataLabel={tableColumns.status}>{getSessionStatusLabel(session)}</Td>
                    <Td dataLabel={tableColumns.completedExecutions}>{session.trainings_completed || '0'}</Td>
                    <Td dataLabel={tableColumns.remainingExecutions}>{getRemainingTrainings(session)}</Td>
                    <Td dataLabel={tableColumns.millicpus}>
                        <CpuIcon />{' '}
                        {session?.current_resource_request
                            ? RoundToThreeDecimalPlaces(session?.current_resource_request.cpus)
                            : 0}
                        {'/'}
                        {RoundToThreeDecimalPlaces(session?.max_resource_request.cpus)}
                    </Td>
                    <Td dataLabel={tableColumns.memory}>
                        <MemoryIcon />
                        {session?.current_resource_request.memory
                            ? RoundToThreeDecimalPlaces(session?.current_resource_request.memory)
                            : 0}
                        {'/'}
                        {RoundToThreeDecimalPlaces(session?.max_resource_request.memory)}
                    </Td>
                    <Td dataLabel={tableColumns.gpus}>
                        <GpuIcon />
                        {session?.current_resource_request.memory
                            ? RoundToTwoDecimalPlaces(session?.current_resource_request.gpus)
                            : 0}
                        {'/'}
                        {RoundToThreeDecimalPlaces(session?.max_resource_request.gpus)}
                    </Td>
                    {/*<Td dataLabel={tableColumns.currentGpus}>*/}
                    {/*    <GpuIcon />*/}
                    {/*    {session?.current_resource_request.gpus ? session?.current_resource_request.gpus : 0}*/}
                    {/*</Td>*/}

                    {/*<Td dataLabel={tableColumns.maxGpus}>*/}
                    {/*    <GpuIcon />*/}
                    {/*    {RoundToThreeDecimalPlaces(session?.max_resource_request.gpus)}*/}
                    {/*</Td>*/}
                    <Td dataLabel={tableColumns.vram}>
                        <GpuIconAlt2 />
                        {session?.current_resource_request.vram
                            ? RoundToThreeDecimalPlaces(session?.current_resource_request.vram)
                            : 0}
                        {'/'}
                        {RoundToThreeDecimalPlaces(session?.max_resource_request.vram)}
                    </Td>
                </Tr>
                <Tr isExpanded={isSessionExpanded(session)}>
                    <Td dataLabel={`${session.id} expended`} colSpan={sessions_table_columns.length + 1}>
                        <ExpandableRowContent>
                            <SessionTrainingEventTable session={session} isNested={true} isStriped={true} />
                        </ExpandableRowContent>
                    </Td>
                </Tr>
            </Tbody>
        );
    };

    // Indices from current pagination state.
    const startIndex: number = perPage * (page - 1);
    const endIndex: number = perPage * (page - 1) + perPage;
    // const arrayIndices = Array.from({ length: endIndex - startIndex }, (_, i) => startIndex + i).map((index) => {
    //     return index;
    // });
    // const filteredSessions: Session[] | undefined = sortedSessions.slice(startIndex, endIndex);

    const getTableRows = () => {
        const tableRows: ReactElement[] = [];
        for (let i: number = startIndex; i < endIndex && i < sortedSessions.length; i++) {
            const tableRow: ReactElement | undefined = getTableRow(i);
            if (tableRow !== undefined) {
                tableRows.push(tableRow);
            }
        }

        return tableRows;
    };

    return (
        <Card isCompact isRounded isFlat>
            <CardBody>
                <Table variant="compact" borders={true} isStriped isExpandable>
                    {tableHead}
                    {/*{filteredSessions?.map((session: Session, rowIndex: number) => {*/}
                    {/*{sortedSessions.length > 0 &&*/}
                    {/*    arrayIndices.map((rowIndex: number) => {*/}
                    {/*        return getTableRow(rowIndex);*/}
                    {/*    })}*/}
                    {sortedSessions.length > 0 && getTableRows()}
                </Table>
                {pagination}
            </CardBody>
        </Card>
    );
};
