import { RequestTrace, RequestTraceSplit } from '@app/Data';
import { Card, CardBody, Flex, FlexItem } from '@patternfly/react-core';
import { ClockIcon } from '@patternfly/react-icons';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import React from 'react';

export interface RequestTraceSplitTableProps {
    children?: React.ReactNode;
    trace: RequestTrace;
    splits: RequestTraceSplit[];
}

// Displays the Sessions from a workload in a table.
export const RequestTraceSplitTable: React.FunctionComponent<RequestTraceSplitTableProps> = (props) => {
    const table_columns: string[] = ['Index', 'Split Name', 'Start', 'Stop', 'Latency (ms)'];
    const table_icons: (React.ReactNode | null)[] = [null, null, null, null, <ClockIcon key={'clock_icon'} />];

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

    return (
        <Card isCompact isRounded isFlat>
            <CardBody>
                <Table variant="compact" borders={true} isStriped>
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
                        {props.splits?.map((split: RequestTraceSplit, idx: number) => {
                            return (
                                <Tr key={`request-${props.trace.messageId}-split-table-${idx}}`}>
                                    <Td dataLabel={table_columns[0]}>{idx}</Td>
                                    <Td dataLabel={table_columns[1]}>{split.splitName}</Td>
                                    <Td dataLabel={table_columns[2]}>{new Date(split.start).toISOString()}</Td>
                                    <Td dataLabel={table_columns[3]}>{new Date(split.end).toISOString()}</Td>
                                    <Td dataLabel={table_columns[4]}>{split.latencyMilliseconds}</Td>
                                </Tr>
                            );
                        })}
                    </Tbody>
                </Table>
            </CardBody>
        </Card>
    );
};
