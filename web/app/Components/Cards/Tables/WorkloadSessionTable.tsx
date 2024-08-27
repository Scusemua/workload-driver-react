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
    const [perPage, setPerPage] = React.useState(5);

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
                return (<Tooltip position='right' content="This session has not yet been created or started yet."><Label icon={<PendingIcon />} color='grey'>{status}</Label></Tooltip>);
            case "idle":
                return (<Tooltip position='right' content="This session is actively-running, but it is not currently training."><Label icon={<ResourcesEmptyIcon />} color='blue'>{status}</Label></Tooltip>);
            case "training":
                return (<Tooltip position='right' content="This session is actively training."><Label icon={<RunningIcon />} color='green'>{status}</Label></Tooltip>);
            case "terminated":
                return (<Tooltip position='right' content="This session has been stopped permanently (without error)."><Label icon={<OffIcon />} color='orange'>{status}</Label></Tooltip>);
            case "erred":
                return (<Tooltip position='right' content={`This session has been terminated due to an unexpected error: ${session.error_message}`}><Label icon={<ErrorCircleOIcon />} color='red'> {status}</Label></Tooltip>);
            default:
                return (<Tooltip position='right' content="This session is in an unknown or unexpected state."><Label icon={<UnknownIcon />} color='red'> unknown: {status}</Label></Tooltip>);
        }
    }

    const filteredSessions: Session[] | undefined = props.workload?.sessions?.slice(perPage * (page - 1), perPage * (page - 1) + perPage);

    return (
        <Card isCompact>
            <CardBody>
                <Table variant="compact" borders={true} isStriped>
                    <Thead noWrap>
                        <Tr>
                            {sessions_table_columns.map((column, columnIndex) => (
                                <Th key={columnIndex} aria-label={`${column}-column`}>{column}</Th>
                            ))}
                        </Tr>
                    </Thead>
                    <Tbody>
                        {filteredSessions?.map((session: Session, idx: number) => {
                            return (
                                <Tr key={props.workload?.events_processed[0]?.id} >
                                    <Td dataLabel={sessions_table_columns[0]}>{idx}</Td>
                                    <Td dataLabel={sessions_table_columns[1]}>{session.id}</Td>
                                    <Td dataLabel={sessions_table_columns[2]}>{getSessionStatusLabel(session)}</Td>
                                    <Td dataLabel={sessions_table_columns[3]}>{session.trainings_completed || '0'}</Td>
                                    <Td dataLabel={sessions_table_columns[4]}><CpuIcon /> {session?.resource_request.cpus}</Td>
                                    <Td dataLabel={sessions_table_columns[5]}><MemoryIcon /> {session?.resource_request.mem_gb} </Td>
                                    <Td dataLabel={sessions_table_columns[6]}><GpuIcon /> {session?.resource_request.gpus}</Td>
                                </Tr>
                            )
                        })}
                    </Tbody>
                </Table>
                <Pagination
                    itemCount={props.workload?.sessions.length}
                    perPage={perPage}
                    page={page}
                    perPageOptions={[{ title: "1 session", value: 1 }, { title: "2 sessions", value: 2 }, { title: "3 sessions", value: 3 }, { title: "4 sessions", value: 4 }, { title: "5 sessions", value: 5 }, { title: "10 sessions", value: 10 }, { title: "25 sessions", value: 25 }, { title: "50 sessions", value: 50 }]}
                    onSetPage={onSetPage}
                    onPerPageSelect={onPerPageSelect}
                    ouiaId="WorkloadSessionsPagination"
                />
            </CardBody>
        </Card>
    );
}