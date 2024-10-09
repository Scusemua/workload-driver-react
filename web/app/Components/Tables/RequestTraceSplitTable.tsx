import { AdjustedSplitNames, GetSplitsFromRequestTrace, RequestTrace, RequestTraceSplit } from '@app/Data';
import {
    Card,
    CardBody,
    Checkbox,
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
import React from 'react';

export interface RequestTraceSplitTableProps {
    children?: React.ReactNode;
    traces: RequestTrace[];
    messageId: string;
}

// Displays the Sessions from a workload in a table.
export const RequestTraceSplitTable: React.FunctionComponent<RequestTraceSplitTableProps> = (props) => {
    const [useAlternativeSplitNames, setUseAlternativeSplitNames] = React.useState<boolean>(true);

    const table_columns: string[] = ['Index', 'Split Name', 'Start', 'Stop', 'Latency (ms)', 'Cumulative Latency (ms)'];
    const table_icons: (React.ReactNode | null)[] = [null, null, null, null, <ClockIcon key={'clock_icon'} />];

    const [selectedTrace, setSelectedTrace] = React.useState<number>(0);

    const [splits, setSplits] = React.useState<RequestTraceSplit[][]>([]);

    React.useEffect(() => {
        const _splits: RequestTraceSplit[][] = [];
        props.traces.forEach((trace: RequestTrace) => {
            const requestTraceSplits: RequestTraceSplit[] = GetSplitsFromRequestTrace(trace);
            _splits.push(requestTraceSplits);
        });

        setSplits(_splits);
    }, [props.traces]);

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

    return (
        <Card isCompact isRounded isFlat>
            <CardBody>
                <Toolbar id={`request-${props.messageId}-trace-split-table-toolbar`}>
                    <ToolbarContent>
                        <ToolbarItem variant={'label'}>
                            <Checkbox
                                id={`request-${props.messageId}-trace-split-table-alt-name-checkbox`}
                                label={'Use Alternative Split Names'}
                                isChecked={useAlternativeSplitNames}
                                onChange={() => {
                                    setUseAlternativeSplitNames((curr) => !curr);
                                }}
                                name={`request-${props.messageId}-trace-split-table-alt-name-checkbox`}
                            />
                        </ToolbarItem>
                        <ToolbarItem variant={'chip-group'}>
                            <ToggleGroup aria-label={'Specify which request trace to view'}>
                                {props.traces.map((trace: RequestTrace, idx: number) => {
                                    return (
                                        <ToggleGroupItem
                                            text={`Kernel #${idx}`}
                                            key={`request-${props.messageId}-trace-split-table-kernel-${idx}-toggle-key`}
                                            buttonId={`request-${props.messageId}-trace-split-table-kernel-${idx}-toggle`}
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
                                return (
                                    <Tr key={`request-${props.messageId}-split-table-${idx}}`}>
                                        <Td dataLabel={table_columns[0]}>{idx}</Td>
                                        <Td dataLabel={table_columns[1]}>
                                            {useAlternativeSplitNames ? AdjustedSplitNames[idx] : split.splitName}
                                        </Td>
                                        <Td dataLabel={table_columns[2]}>{new Date(split.start).toISOString()}</Td>
                                        <Td dataLabel={table_columns[3]}>{new Date(split.end).toISOString()}</Td>
                                        <Td dataLabel={table_columns[4]}>{split.latencyMilliseconds}</Td>
                                        <Td dataLabel={table_columns[5]}>
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
