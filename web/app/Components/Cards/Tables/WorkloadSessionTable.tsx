import React from 'react';
import { Card, CardBody, Label, Pagination, Tooltip } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { CpuIcon, ErrorCircleOIcon, MemoryIcon, OffIcon, PendingIcon, ResourcesEmptyIcon, RunningIcon, UnknownIcon } from '@patternfly/react-icons';

import {
    Session,
    Workload,
} from '@app/Data/Workload';
import { GpuIcon } from '@app/Icons';

export interface WorkloadSessionTableProps {
    children?: React.ReactNode;
    workload: Workload | null;
};

// Displays the Sessions from a workload in a table.
export const WorkloadSessionTable: React.FunctionComponent<WorkloadSessionTableProps> = (props) => {
    const [page, setPage] = React.useState(1);
    const [perPage, setPerPage] = React.useState(4);

    const onSetPage = (_event: React.MouseEvent | React.KeyboardEvent | MouseEvent, newPage: number) => {
        setPage(newPage);
    };

    const onPerPageSelect = (
        _event: React.MouseEvent | React.KeyboardEvent | MouseEvent,
        newPerPage: number,
        newPage: number
    ) => {
        setPerPage(newPerPage);
        setPage(newPage);
    };

    const sessions_table_columns: string[] = ["Index", "ID", "Status", "Trainings Completed", "Max vCPUs", "Max Memory (GB)", "Max vGPUs"]

    const getSessionStatusLabel = (session: Session) => {
        const status: string = session.state;
        switch (status) {
            case "awaiting start":
                return (<Tooltip content="This session has not yet been created or started yet."><Label icon={<PendingIcon />} color='grey'>{status}</Label></Tooltip>);
            case "idle":
                return (<Tooltip content="This session is actively-running, but it is not currently training."><Label icon={<ResourcesEmptyIcon />} color='grey'>{status}</Label></Tooltip>);
            case "training":
                return (<Tooltip content="This session is actively training."><Label icon={<RunningIcon />} color='green'>{status}</Label></Tooltip>);
            case "terminated":
                return (<Tooltip content="This session has been stopped permanently (without error)."><Label icon={<OffIcon />} color='gold'>{status}</Label></Tooltip>);
            case "erred":
                return (<Tooltip content={`This session has been terminated due to an unexpected error: ${session.error_message}`}><Label icon={<ErrorCircleOIcon />} color='red'> {status}</Label></Tooltip>);
            default:
                return (<Tooltip content="This session is in an unknown or unexpected state."><Label icon={<UnknownIcon />} color='orange'> unknown: {status}</Label></Tooltip>);
        }
    }

    return (
        <Card isCompact>
            <CardBody>
                <Table variant="compact" borders={true} isStriped>
                    <Thead noWrap>
                        <Tr>
                            {sessions_table_columns.map((column, columnIndex) => (
                                <Th key={columnIndex}>{column}</Th>
                            ))}
                        </Tr>
                    </Thead>
                    <Tbody>
                        {props.workload?.sessions?.map((session: Session, idx: number) => {
                            return (
                                <Tr key={props.workload?.events_processed[0]?.id} >
                                    <Td dataLabel={sessions_table_columns[0]}>{idx}</Td>
                                    <Td dataLabel={sessions_table_columns[1]}>{session.id}</Td>
                                    <Td dataLabel={sessions_table_columns[2]}>{getSessionStatusLabel(session)}</Td>
                                    <Td dataLabel={sessions_table_columns[3]}>{session.trainings_completed || '0'}</Td>
                                    <Td dataLabel={sessions_table_columns[4]}><CpuIcon /> {session?.max_cpus}</Td>
                                    <Td dataLabel={sessions_table_columns[5]}><GpuIcon /> {session?.max_num_gpus}</Td>
                                    <Td dataLabel={sessions_table_columns[6]}><MemoryIcon /> {session?.max_memory_gb}</Td>
                                </Tr>
                            )
                        })}
                    </Tbody>
                </Table>
                <Pagination
                    itemCount={props.workload?.sessions.length}
                    perPage={perPage}
                    page={page}
                    isCompact
                    perPageOptions={[{ title: "1 session", value: 1 }, { title: "2 sessions", value: 2 }, { title: "3 sessions", value: 3 }, { title: "4 sessions", value: 4 }, { title: "5 sessions", value: 5 }]}
                    onSetPage={onSetPage}
                    onPerPageSelect={onPerPageSelect}
                    ouiaId="WorkloadSessionsPagination"
                />
            </CardBody>
        </Card>
    );
}