import {
    Button,
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Checkbox,
    Grid,
    GridItem,
    Title,
    ToggleGroup,
    ToggleGroupItem,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
} from '@patternfly/react-core';
import { CpuIcon, MemoryIcon, SyncIcon } from '@patternfly/react-icons';
import { GpuIcon } from '@src/Assets/Icons';
import { UtilizationEntry } from '@src/Components';
import { useNodes } from '@src/Providers';
import React from 'react';

export interface UtilizationCardProps {
    chartWidth?: number;
    chartHeight?: number;
}

export const UtilizationCard: React.FunctionComponent<UtilizationCardProps> = (props: UtilizationCardProps) => {
    const { refreshNodes, nodesAreLoading } = useNodes();
    const [randomizeUtilizations, setRandomizeUtilizations] = React.useState(false);
    const [selectedResourceType, setSelectedResourceType] = React.useState<'idle' | 'pending' | 'committed'>(
        'committed',
    );

    const toolbar = (
        <React.Fragment>
            <ToolbarGroup variant="action-group">
                <ToolbarItem>
                    <Checkbox
                        label="Randomize utilizations"
                        id={'randomized-utilizations-checkbox'}
                        isChecked={randomizeUtilizations}
                        onChange={(_event: React.FormEvent<HTMLInputElement>, checked: boolean) =>
                            setRandomizeUtilizations(checked)
                        }
                    />
                </ToolbarItem>
            </ToolbarGroup>
            <ToolbarGroup>
                <ToolbarItem>
                    <ToggleGroup
                        aria-label={
                            "Select the type of resource to visualize (i.e., 'idle', 'pending', or 'committed')."
                        }
                    >
                        <ToggleGroupItem
                            text={'Idle'}
                            buttonId={'toggle-idle-resource-utilization'}
                            isSelected={selectedResourceType == 'idle'}
                            onChange={() => setSelectedResourceType('idle')}
                        />
                        <ToggleGroupItem
                            text={'Pending'}
                            buttonId={'toggle-pending-resource-utilization'}
                            isSelected={selectedResourceType == 'pending'}
                            onChange={() => setSelectedResourceType('pending')}
                        />
                        <ToggleGroupItem
                            text={'Committed'}
                            buttonId={'toggle-committed-resource-utilization'}
                            isSelected={selectedResourceType == 'committed'}
                            onChange={() => setSelectedResourceType('committed')}
                        />
                    </ToggleGroup>
                </ToolbarItem>
            </ToolbarGroup>
            <ToolbarGroup variant="action-group-plain">
                <ToolbarItem>
                    <Tooltip exitDelay={75} content={<div>Refresh cluster resource utilization data.</div>}>
                        <Button
                            variant="plain"
                            onClick={() => {
                                refreshNodes();
                            }}
                            label="refresh-cluster-utilization-data-button"
                            aria-label="refresh-cluster-utilization-data-button"
                            isDisabled={nodesAreLoading}
                            className={
                                (nodesAreLoading && 'loading-icon-spin-toggleable') ||
                                'loading-icon-spin-toggleable paused'
                            }
                            icon={<SyncIcon />}
                        />
                    </Tooltip>
                </ToolbarItem>
            </ToolbarGroup>
        </React.Fragment>
    );

    return (
        <Card isFullHeight id="utilization-card" component="div">
            <CardHeader actions={{ actions: toolbar, hasNoOffset: false }}>
                <CardTitle>
                    <Title headingLevel="h1" size="xl">
                        Resource Utilization
                    </Title>
                </CardTitle>
            </CardHeader>
            <CardBody>
                <Grid>
                    <GridItem span={3} rowSpan={1}>
                        <UtilizationEntry
                            resourceUnit={'vCPU'}
                            resourceDisplayName={'CPU'}
                            icon={<CpuIcon />}
                            selectedResourceType={selectedResourceType}
                            randomizeUtilizations={randomizeUtilizations}
                            chartWidth={props.chartWidth}
                            chartHeight={props.chartHeight}
                        />
                    </GridItem>
                    <GridItem span={3} rowSpan={1}>
                        <UtilizationEntry
                            resourceUnit={'GB'}
                            resourceDisplayName={'Memory'}
                            icon={<MemoryIcon />}
                            selectedResourceType={selectedResourceType}
                            randomizeUtilizations={randomizeUtilizations}
                            chartWidth={props.chartWidth}
                            chartHeight={props.chartHeight}
                        />
                    </GridItem>
                    <GridItem span={3} rowSpan={1}>
                        <UtilizationEntry
                            resourceUnit={'GB'}
                            resourceDisplayName={'VRAM'}
                            icon={<GpuIcon scale={1.25} />}
                            selectedResourceType={selectedResourceType}
                            randomizeUtilizations={randomizeUtilizations}
                            chartWidth={props.chartWidth}
                            chartHeight={props.chartHeight}
                        />
                    </GridItem>
                    <GridItem span={3} rowSpan={1}>
                        <UtilizationEntry
                            resourceUnit={'GPUs'}
                            resourceDisplayName={'GPU'}
                            icon={<GpuIcon scale={1.25} />}
                            selectedResourceType={selectedResourceType}
                            randomizeUtilizations={randomizeUtilizations}
                            chartWidth={props.chartWidth}
                            chartHeight={props.chartHeight}
                        />
                    </GridItem>
                </Grid>
            </CardBody>
        </Card>
    );
};
