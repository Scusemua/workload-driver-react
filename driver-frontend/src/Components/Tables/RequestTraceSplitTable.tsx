import {
    Card,
    CardBody,
    Flex,
    FlexItem,
    ToggleGroup,
    ToggleGroupItem,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { ClockIcon } from '@patternfly/react-icons';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { GetSplitsFromRequestTrace, RequestTrace, RequestTraceSplit } from '@src/Data';
import { RoundToTwoDecimalPlaces } from '@Utils/utils';
import React from 'react';

export interface RequestTraceSplitTableProps {
    children?: React.ReactNode;
    traces: RequestTrace[];
    messageId: string;
    receivedReplyAt: number; // Time that we, the frontend client, received the reply.
    initialRequestSentAt?: number; // Time that we, the frontend client, initially sent the request.
}

// Displays the "splits" from a RequestTrace in a table, with the latency of each part of
// the request trace shown in its own row.
export const RequestTraceSplitTable: React.FunctionComponent<RequestTraceSplitTableProps> = (props) => {
    const table_columns: string[] = [
        'Index',
        'Split Name',
        'Start',
        'Stop',
        'Latency (ms)',
        'Relative Percent',
        'Cumulative Latency (ms)',
    ];
    const table_icons: (React.ReactNode | null)[] = [null, null, null, null, <ClockIcon key={'clock_icon'} />];

    const [selectedTrace, setSelectedTrace] = React.useState<number>(0);

    const [splits, setSplits] = React.useState<RequestTraceSplit[][]>([]);

    React.useEffect(() => {
        const _splits: RequestTraceSplit[][] = [];

        console.log(`props.traces: ${JSON.stringify(props.traces, null, 2)}`);

        props.traces.forEach((trace: RequestTrace) => {
            const requestTraceSplits: RequestTraceSplit[] = GetSplitsFromRequestTrace(
                props.receivedReplyAt,
                trace,
                props.initialRequestSentAt,
            );
            trace.e2eLatencyMilliseconds =
                props.receivedReplyAt - (props.initialRequestSentAt || requestTraceSplits[0].start);
            _splits.push(requestTraceSplits);
        });

        setSplits(_splits);

        console.log(
            `Generated the following splits from the assigned RequestTrace:\n${JSON.stringify(_splits, null, 2)}`,
        );
    }, [props.traces, props.initialRequestSentAt, props.receivedReplyAt]);

    const getColumnDefinitionContent = (column: string, index: number) => {
        if (table_icons[index] === null) {
            return column;
        } else {
            return (
                <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsXs' }}>
                    <FlexItem>{table_icons[index]}</FlexItem>
                    <FlexItem>{column}</FlexItem>
                </Flex>
            );
        }
    };

    const getCumulativeLatency = (splits: RequestTraceSplit[], start: number, end: number): number => {
        return splits[end].end - splits[start].start;
    };

    const getRelativePercent = (traceIndex: number, targetSplitIndex: number): number => {
        return (
            splits[traceIndex][targetSplitIndex].latencyMilliseconds / props.traces[traceIndex].e2eLatencyMilliseconds
        );
    };

    return (
        <Card isCompact isRounded isFlat>
            <CardBody>
                <Toolbar id={`request-${props.messageId}-trace-split-table-toolbar`}>
                    <ToolbarContent>
                        {/* We only display the ToggleGroup if there are 2 or more individual traces to display. */}
                        <ToolbarItem variant={'chip-group'} hidden={props.traces.length <= 1}>
                            <ToggleGroup aria-label={'Specify which request trace to view'}>
                                {props.traces.map((trace: RequestTrace, idx: number) => {
                                    return (
                                        <ToggleGroupItem
                                            text={`Kernel #${trace.replicaId}`}
                                            key={`request-${props.messageId}-trace-split-table-kernel-${trace.replicaId}-toggle-key`}
                                            buttonId={`request-${props.messageId}-trace-split-table-kernel-${trace.replicaId}-toggle`}
                                            isSelected={selectedTrace == idx}
                                            onChange={() => setSelectedTrace(idx)}
                                        />
                                    );
                                })}
                            </ToggleGroup>
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>
                <Table id={`request-${props.messageId}-trace-split-table`} variant="compact" borders={true} isStriped>
                    <Thead noWrap>
                        <Tr>
                            {table_columns.map((column, columnIndex) => (
                                <Th key={columnIndex} aria-label={`${column}-column`}>
                                    {getColumnDefinitionContent(column, columnIndex)}
                                </Th>
                            ))}
                        </Tr>
                    </Thead>
                    <Tbody>
                        {splits.length > 0 &&
                            splits[selectedTrace].map((split: RequestTraceSplit, idx: number) => {
                                let startDateString: string = 'N/A';
                                let endDateString: string = 'N/A';

                                let startDate: Date | null;
                                try {
                                    startDate = new Date(split.start);
                                    startDateString = startDate.toISOString();
                                    // eslint-disable-next-line @typescript-eslint/no-unused-vars
                                } catch (err) {
                                    startDate = null;
                                }

                                let endDate: Date | null;
                                try {
                                    endDate = new Date(split.end);
                                    endDateString = endDate.toISOString();
                                    // eslint-disable-next-line @typescript-eslint/no-unused-vars
                                } catch (err) {
                                    endDate = null;
                                }

                                return (
                                    <Tr key={`request-${props.messageId}-split-table-${idx}}`}>
                                        <Td dataLabel={table_columns[0]}>{idx}</Td>
                                        <Td dataLabel={table_columns[1]}>{split.splitName}</Td>
                                        <Td dataLabel={table_columns[2]}>{startDateString}</Td>
                                        <Td dataLabel={table_columns[3]}>{endDateString}</Td>
                                        <Td dataLabel={table_columns[4]}>{split.latencyMilliseconds}</Td>
                                        <Td dataLabel={table_columns[5]}>
                                            {RoundToTwoDecimalPlaces(getRelativePercent(selectedTrace, idx) * 100)}
                                            {'%'}
                                        </Td>
                                        <Td dataLabel={table_columns[6]}>
                                            {getCumulativeLatency(splits[selectedTrace], 0, idx)}
                                        </Td>
                                    </Tr>
                                );
                            })}
                    </Tbody>
                </Table>
            </CardBody>
        </Card>
    );
};
