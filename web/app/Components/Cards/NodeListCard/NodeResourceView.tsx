import { WorkloadEvent } from '@app/Data';
import { GpuIcon } from '@app/Icons';
import { ClusterNode, GetNodeIdleResource } from '@data/Cluster';
import { Card, CardBody, CardExpandableContent, CardHeader, CardTitle, Grid, GridItem, Pagination } from '@patternfly/react-core';
import { CpuIcon, MemoryIcon } from '@patternfly/react-icons';
import { Table, Tbody, Td, Th, Thead, ThProps, Tr } from '@patternfly/react-table';
import { useNodes } from '@providers/NodeProvider';
import React from 'react';

export interface NodeResourceUsageTableProps {
    resource: 'CPU' | 'GPU' | 'vGPU' | 'Memory';
}

export const NodeResourceUsageTable: React.FunctionComponent<NodeResourceUsageTableProps> = (
    props: NodeResourceUsageTableProps,
) => {
    const table_column_names: string[] = ['Node ID', 'Idle', 'Pending', 'Allocated', 'Capacity'];

    const { nodes } = useNodes();
    const [page, setPage] = React.useState(1);
    const [perPage, setPerPage] = React.useState(10);

    // Index of the currently sorted column
    const [activeSortIndex, setActiveSortIndex] = React.useState<number | null>(null);

    // Sort direction of the currently sorted column
    const [activeSortDirection, setActiveSortDirection] = React.useState<'asc' | 'desc' | null>(null);

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

    // Since OnSort specifies sorted columns by index, we need sortable values for our object by column index.
    // This example is trivial since our data objects just contain strings, but if the data was more complex
    // this would be a place to return simplified string or number versions of each column to sort by.
    const getSortableRowValues = (node: ClusterNode): (string | number | Date)[] => {
        const { NodeId, AllocatedResources, PendingResources, CapacityResources } = node;

        const idleRes: number =
            CapacityResources[props.resource] - (AllocatedResources[props.resource] + PendingResources[props.resource]);

        // Note: We're omitting the event's "id" and "error_message" fields here.
        return [
            NodeId,
            idleRes,
            PendingResources[props.resource],
            AllocatedResources[props.resource],
            CapacityResources[props.resource],
        ];
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

    let sortedNodes: ClusterNode[] = nodes;
    if (activeSortIndex !== null) {
        sortedNodes = sortedNodes.sort((a: ClusterNode, b: ClusterNode) => {
            const aValue = getSortableRowValues(a)[activeSortIndex];
            const bValue = getSortableRowValues(b)[activeSortIndex];
            console.log(
                `Sorting ${aValue} and ${bValue} (activeSortIndex = ${activeSortIndex}, activeSortDirection = '${activeSortDirection}')`,
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
        });
    }
  const paginatedNodes: ClusterNode[] | undefined = sortedNodes?.slice(perPage * (page - 1), perPage * (page - 1) + perPage);

    return (
        <Card isCompact isRounded isFlat>
            <CardBody>
                <Table variant="compact" isStriped>
                    <Thead noWrap>
                        <Tr>
                            {table_column_names.map((column, columnIndex) => (
                                <Th
                                    key={`node-${props.resource}-column-${columnIndex}`}
                                    sort={getSortParams(columnIndex)}
                                    aria-label={`${column}-column`}
                                >
                                    {column}
                                </Th>
                            ))}
                        </Tr>
                    </Thead>
                    <Tbody>
                        {paginatedNodes.map((node) => {
                            return (
                                <Tr key={`node-${node.NodeId}-${props.resource}-usage-table-row`}>
                                    <Td dataLabel={table_column_names[0]}>{node.NodeId}</Td>
                                    <Td dataLabel={table_column_names[1]}>
                                        {GetNodeIdleResource(node, props.resource)}
                                    </Td>
                                    <Td dataLabel={table_column_names[2]}>{node.PendingResources[props.resource]}</Td>
                                    <Td dataLabel={table_column_names[3]}>{node.AllocatedResources[props.resource]}</Td>
                                    <Td dataLabel={table_column_names[4]}>{node.CapacityResources[props.resource]}</Td>
                                </Tr>
                            );
                        })}
                    </Tbody>
                </Table>
                <Pagination
                    itemCount={sortedNodes?.length}
                    isDisabled={sortedNodes?.length == 0}
                    perPage={perPage}
                    page={page}
                    perPageOptions={[
                        { title: '1 nodes', value: 1 },
                        { title: '2 nodes', value: 2 },
                        {
                            title: '3 nodes',
                            value: 3,
                        },
                        { title: '4 nodes', value: 4 },
                        { title: '5 nodes', value: 5 },
                        {
                            title: '10 nodes',
                            value: 10,
                        },
                        { title: '25 nodes', value: 25 },
                        { title: '50 nodes', value: 50 },
                        { title: '100 nodes', value: 100 },
                    ]}
                    onSetPage={onSetPage}
                    onPerPageSelect={onPerPageSelect}
                    ouiaId="WorkloadEventsPagination"
                />
            </CardBody>
        </Card>
    );
};

/**
 * NodeResourceView provides a visualization of the current and historical resource utilization of the Cluster Nodes.
 */
export const NodeResourceView: React.FunctionComponent = () => {
    const [isCpuSectionExpanded, setIsCpuSectionExpanded] = React.useState<boolean>(false);
    const [isGpuSectionExpanded, setIsGpuSectionExpanded] = React.useState<boolean>(false);
    const [isVirtualGpuSectionExpanded, setIsVirtualGpuSectionExpanded] = React.useState<boolean>(false);
    const [isMemorySectionExpanded, setIsMemorySectionExpanded] = React.useState<boolean>(false);

    return (
        <React.Fragment>
            <Card isCompact isPlain isExpanded={isCpuSectionExpanded}>
                <CardHeader
                    onExpand={() => setIsCpuSectionExpanded(!isCpuSectionExpanded)}
                    isToggleRightAligned={false}
                    toggleButtonProps={{
                        id: 'node-resource-view-cpu-section-toggle-button',
                        'aria-label': 'Toggle CPU resource usage section',
                        'aria-labelledby':
                            'node-resource-view-cpu-section-toggle-button node-resource-view-cpu-section-title',
                        'aria-expanded': isCpuSectionExpanded,
                    }}
                >
                    <CardTitle id={'node-resource-view-cpu-section-title'}>
                        <span className="pf-v5-u-font-weight-light">
                            <CpuIcon /> CPU Usage
                        </span>
                    </CardTitle>
                </CardHeader>
                <CardExpandableContent>
                    <CardBody>
                        <Grid hasGutter>
                            <NodeResourceUsageTable resource={'CPU'} />
                        </Grid>
                    </CardBody>
                </CardExpandableContent>
            </Card>
            <Card isCompact isPlain isExpanded={isMemorySectionExpanded}>
                <CardHeader
                    onExpand={() => setIsMemorySectionExpanded(!isMemorySectionExpanded)}
                    isToggleRightAligned={false}
                    toggleButtonProps={{
                        id: 'node-resource-view-memory-section-toggle-button',
                        'aria-label': 'Toggle Memory resource usage section',
                        'aria-labelledby':
                            'node-resource-view-memory-section-toggle-button node-resource-view-memory-section-title',
                        'aria-expanded': isMemorySectionExpanded,
                    }}
                >
                    <CardTitle id={'node-resource-view-memory-section-title'}>
                        <span className="pf-v5-u-font-weight-light">
                            <MemoryIcon /> Memory Usage
                        </span>
                    </CardTitle>
                </CardHeader>
                <CardExpandableContent>
                    <CardBody>
                        <Grid hasGutter>
                            <NodeResourceUsageTable resource={'Memory'} />
                        </Grid>
                    </CardBody>
                </CardExpandableContent>
            </Card>
            <Card isCompact isPlain isExpanded={isGpuSectionExpanded}>
                <CardHeader
                    onExpand={() => setIsGpuSectionExpanded(!isGpuSectionExpanded)}
                    isToggleRightAligned={false}
                    toggleButtonProps={{
                        id: 'node-resource-view-gpu-section-toggle-button',
                        'aria-label': 'Toggle GPU resource usage section',
                        'aria-labelledby':
                            'node-resource-view-gpu-section-toggle-button node-resource-view-gpu-section-title',
                        'aria-expanded': isGpuSectionExpanded,
                    }}
                >
                    <CardTitle id={'node-resource-view-gpu-section-title'}>
                        <span className="pf-v5-u-font-weight-light">
                            <GpuIcon /> GPU Usage
                        </span>
                    </CardTitle>
                </CardHeader>
                <CardExpandableContent>
                    <CardBody>
                        <Grid hasGutter>
                            <GridItem span={8}>
                            <NodeResourceUsageTable resource={'GPU'} />
                            </GridItem>
                            <GridItem span={4}>
                            <iframe src="http://localhost:3000/d-solo/ddx4dshpy2c5cd/new-dashboard?orgId=1&from=1725639256441&to=1725660856441&panelId=1" width="100%" height="800" frameBorder={0}></iframe>
                            </GridItem>
                        </Grid>
                    </CardBody>
                </CardExpandableContent>
            </Card>
            <Card isCompact isPlain isExpanded={isVirtualGpuSectionExpanded}>
                <CardHeader
                    onExpand={() => setIsVirtualGpuSectionExpanded(!isVirtualGpuSectionExpanded)}
                    isToggleRightAligned={false}
                    toggleButtonProps={{
                        id: 'node-resource-view-vgpu-section-toggle-button',
                        'aria-label': 'Toggle vGPU resource usage section',
                        'aria-labelledby':
                            'node-resource-view-vgpu-section-toggle-button node-resource-view-gpu-section-title',
                        'aria-expanded': isGpuSectionExpanded,
                    }}
                >
                    <CardTitle id={'node-resource-view-vgpu-section-title'}>
                        <span className="pf-v5-u-font-weight-light">
                            <GpuIcon /> vGPU Usage
                        </span>
                    </CardTitle>
                </CardHeader>
                <CardExpandableContent>
                    <CardBody>
                        <Grid hasGutter>
                            <NodeResourceUsageTable resource={'vGPU'} />
                        </Grid>
                    </CardBody>
                </CardExpandableContent>
            </Card>
        </React.Fragment>
    );
};
