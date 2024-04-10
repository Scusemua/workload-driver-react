import React from 'react';
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
    Text,
    TextVariants,
    Title,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
} from '@patternfly/react-core';
import { CpuIcon, MemoryIcon, SyncIcon } from '@patternfly/react-icons';
import { UtilizationDonutChart } from './UtilizationDonutChart';
import { useNodes } from '@app/Providers';
import { toast } from 'react-hot-toast';
import { ChartDonutThreshold, ChartDonutUtilization } from '@patternfly/react-charts';
import { GpuIcon } from '@app/Icons';

export interface UtilizationCardProps {
    chartWidth?: number;
    chartHeight?: number;
}

export const UtilizationCard: React.FunctionComponent<UtilizationCardProps> = (props: UtilizationCardProps) => {
    const { refreshNodes, nodesAreLoading } = useNodes();
    const [randomizeUtilizations, setRandomizeUtilizations] = React.useState(false);

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
            <ToolbarGroup variant="icon-button-group">
                <ToolbarItem>
                    <Tooltip exitDelay={75} content={<div>Refresh cluster resource utilization data.</div>}>
                        <Button
                            variant="plain"
                            onClick={() => {
                                toast.promise(refreshNodes(), {
                                    loading: <b>Refreshing cluster resource utilization data...</b>,
                                    success: <b>Refreshed cluster resource utilization data!</b>,
                                    error: (reason: Error) => {
                                        let explanation: string = reason.message;
                                        if (reason.name === 'SyntaxError') {
                                            explanation = 'HTTP 504 Gateway Timeout';
                                        }

                                        return (
                                            <div>
                                                <Flex
                                                    direction={{ default: 'column' }}
                                                    spaceItems={{ default: 'spaceItemsNone' }}
                                                >
                                                    <FlexItem>
                                                        <b>Could not refresh cluster resource utilization data.</b>
                                                    </FlexItem>
                                                    <FlexItem>{explanation}</FlexItem>
                                                </Flex>
                                            </div>
                                        );
                                    },
                                });
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
                                    <b>vGPU</b>
                                </FlexItem>
                            </Flex>
                            <FlexItem>
                                <UtilizationDonutChart
                                    chartHeight={props.chartHeight}
                                    chartWidth={props.chartWidth}
                                    resourceDisplayName="vGPU"
                                    resourceUnit="vGPUs"
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
