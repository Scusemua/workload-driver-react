import { GpuIcon } from '@src/Assets/Icons';
import { useNodes } from '@src/Providers';
import {
    Button,
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Checkbox,
    Flex,
    FlexItem,
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
import React from 'react';
import { UtilizationDonutChart } from './UtilizationDonutChart';

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
            <ToolbarGroup variant="button-group">
                <ToolbarItem variant="search-filter">
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
            <ToolbarGroup variant="icon-button-group">
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
        <Card isFullHeight isRounded id="utilization-card" component="div">
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
                        <Flex justifyContent={{ default: 'justifyContentCenter' }}>
                            <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsSm' }}>
                                <FlexItem>
                                    <CpuIcon />
                                </FlexItem>
                                <FlexItem>
                                    <b>CPU (vCPU)</b>
                                </FlexItem>
                            </Flex>
                            <FlexItem>
                                <UtilizationDonutChart
                                    chartHeight={props.chartHeight}
                                    chartWidth={props.chartWidth}
                                    resourceDisplayName="CPU"
                                    resourceUnit="vCPU"
                                    resourceStatus={selectedResourceType}
                                    randomizeUtilizations={randomizeUtilizations}
                                    showLegend={false}
                                />
                            </FlexItem>
                        </Flex>
                    </GridItem>
                    <GridItem span={3} rowSpan={1}>
                        <Flex justifyContent={{ default: 'justifyContentCenter' }}>
                            <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsSm' }}>
                                <FlexItem>
                                    <MemoryIcon />
                                </FlexItem>
                                <FlexItem>
                                    <b>Memory (GB)</b>
                                </FlexItem>
                            </Flex>
                            <FlexItem>
                                <UtilizationDonutChart
                                    chartHeight={props.chartHeight}
                                    chartWidth={props.chartWidth}
                                    resourceDisplayName="Memory"
                                    resourceUnit="GB"
                                    resourceStatus={selectedResourceType}
                                    randomizeUtilizations={randomizeUtilizations}
                                    converter={(val: number) => {
                                        return val * 0.001048576; // Convert from MiB to GB.
                                    }}
                                    showLegend={false}
                                />
                            </FlexItem>
                        </Flex>
                    </GridItem>
                    <GridItem span={3} rowSpan={1}>
                        <Flex justifyContent={{ default: 'justifyContentCenter' }}>
                            <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsSm' }}>
                                <FlexItem>
                                    <GpuIcon scale={1.25} />
                                </FlexItem>
                                <FlexItem>
                                    <b>VRAM (GB)</b>
                                </FlexItem>
                            </Flex>
                            <FlexItem>
                                <UtilizationDonutChart
                                    chartHeight={props.chartHeight}
                                    chartWidth={props.chartWidth}
                                    resourceDisplayName="VRAM"
                                    resourceUnit="GB"
                                    resourceStatus={selectedResourceType}
                                    randomizeUtilizations={randomizeUtilizations}
                                    showLegend={false}
                                />
                            </FlexItem>
                        </Flex>
                    </GridItem>
                    <GridItem span={3} rowSpan={1}>
                        <Flex justifyContent={{ default: 'justifyContentCenter' }}>
                            <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsSm' }}>
                                <FlexItem>
                                    <GpuIcon scale={1.25} />
                                </FlexItem>
                                <FlexItem>
                                    <b>GPU</b>
                                </FlexItem>
                            </Flex>
                            <FlexItem>
                                <UtilizationDonutChart
                                    chartHeight={props.chartHeight}
                                    chartWidth={props.chartWidth}
                                    resourceDisplayName="GPU"
                                    resourceUnit="GPUs"
                                    resourceStatus={selectedResourceType}
                                    randomizeUtilizations={randomizeUtilizations}
                                    showLegend={false}
                                />
                            </FlexItem>
                        </Flex>
                    </GridItem>
                </Grid>
            </CardBody>
        </Card>
    );
};
