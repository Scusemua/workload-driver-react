import { Label, Pagination, Title } from '@patternfly/react-core';
import { CheckCircleIcon, PendingIcon, RunningIcon, UnknownIcon } from '@patternfly/react-icons';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Session, TrainingEvent } from '@src/Data';
import { RoundToThreeDecimalPlaces } from '@src/Utils';
import React from 'react';

interface ISessionTrainingEventTableProps {
    session: Session;

    // Flag indicating this table is nested within another table
    isNested: boolean;

    // Flag indicating that this table should apply the striping pattern.
    isStriped?: boolean;
}

export const SessionTrainingEventTable: React.FunctionComponent<ISessionTrainingEventTableProps> = (
    props: ISessionTrainingEventTableProps,
) => {
    const [page, setPage] = React.useState(1);
    const [perPage, setPerPage] = React.useState(5);

    const table_names: string[] = [
        'Training Index',
        'Status',
        'Start Tick',
        'Stop Tick',
        'Duration (Ticks)',
        'Millicpus',
        'Memory (MB)',
        'vGPUs',
        'VRAM (GB)',
    ];

    const columnNames = {
        trainingIndex: 'Index',
        status: 'Status',
        startTick: 'Starting Tick',
        stopTick: 'Stop Tick',
        duration: 'Duration (Ticks)',
        cpus: 'Millicpus',
        memory: 'Memory (MB)',
        gpus: 'vGPUs',
        vram: 'VRAM (GB)',
    };

    const getStatusIcon = (statusText: 'unknown' | 'completed' | 'awaiting start' | 'in progress') => {
        if (statusText === 'unknown') {
            return <UnknownIcon />;
        }

        if (statusText === 'completed') {
            return <CheckCircleIcon />;
        }

        if (statusText === 'awaiting start') {
            return <PendingIcon />;
        }

        if (statusText === 'in progress') {
            return <RunningIcon />;
        }

        return <UnknownIcon />;
    };

    const getStatusColor = (statusText: 'unknown' | 'completed' | 'awaiting start' | 'in progress') => {
        if (statusText === 'unknown') {
            return 'gold';
        }

        if (statusText === 'completed') {
            return 'green';
        }

        if (statusText === 'awaiting start') {
            return 'grey';
        }

        if (statusText === 'in progress') {
            return 'blue';
        }

        return 'gold';
    };

    const getStatusText = (training: TrainingEvent, index: number) => {
        if (props.session.state === 'training' && index == props.session.trainings_completed) {
            return 'in progress';
        }

        if (index >= props.session.trainings_completed) {
            return 'awaiting start';
        }

        if (index < props.session.trainings_completed) {
            return 'completed';
        }

        return 'unknown';
    };

    const getStatus = (training: TrainingEvent, index: number) => {
        const statusText: 'unknown' | 'completed' | 'awaiting start' | 'in progress' = getStatusText(training, index);
        return (
            <Label icon={getStatusIcon(statusText)} color={getStatusColor(statusText)}>
                {statusText}
            </Label>
        );
    };

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

    const paginatedTrainingEvents: TrainingEvent[] | undefined = props.session.trainings?.slice(
        perPage * (page - 1),
        perPage * (page - 1) + perPage,
    );

    return (
        <React.Fragment>
            <Title headingLevel={'h3'}>Training Events ({props.session.trainings?.length})</Title>
            <Table isNested={props.isNested} isStriped={props.isStriped} variant={'compact'}>
                <Thead>
                    <Tr>
                        {table_names.map((column, columnIndex) => (
                            <Th
                                key={`session_${props.session?.id}_column_${columnIndex}`}
                                aria-label={`${column}-column`}
                            >
                                {column}
                            </Th>
                        ))}
                    </Tr>
                </Thead>
                <Tbody>
                    {paginatedTrainingEvents.map((training: TrainingEvent, index: number) => {
                        return (
                            <Tr key={`session_${props.session?.id}_training_${index}`}>
                                <Td dataLabel={columnNames.trainingIndex}>{index}</Td>
                                <Td dataLabel={columnNames.status}>{getStatus(training, index)}</Td>
                                <Td dataLabel={columnNames.startTick}>{training.start_tick}</Td>
                                <Td dataLabel={columnNames.stopTick}>
                                    {training.start_tick + training.duration_in_ticks}
                                </Td>
                                <Td dataLabel={columnNames.duration}>{training.duration_in_ticks}</Td>
                                <Td dataLabel={columnNames.cpus}>{RoundToThreeDecimalPlaces(training.cpus)}</Td>
                                <Td dataLabel={columnNames.memory}>{RoundToThreeDecimalPlaces(training.memory)}</Td>
                                <Td dataLabel={columnNames.gpus}>{training.gpu_utilizations.length}</Td>
                                <Td dataLabel={columnNames.vram}>{RoundToThreeDecimalPlaces(training.vram)}</Td>
                            </Tr>
                        );
                    })}
                </Tbody>
            </Table>
            <Pagination
                itemCount={props.session.trainings?.length}
                isDisabled={props.session.trainings?.length == 0}
                perPage={perPage}
                page={page}
                perPageOptions={[
                    { title: '3 events', value: 3 },
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
        </React.Fragment>
    );
};

export default SessionTrainingEventTable;
