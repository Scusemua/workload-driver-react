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
    Title,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
} from '@patternfly/react-core';
import { SyncIcon } from '@patternfly/react-icons';
import { UtilizationDonutChart } from './UtilizationDonutChart';
import { useNodes } from '@app/Providers';
import { toast } from 'react-hot-toast';
import { ChartDonutThreshold, ChartDonutUtilization } from '@patternfly/react-charts';

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
                        Current Cluster Resource Utilization
                    </Title>
                </CardTitle>
            </CardHeader>
            <CardBody>
                <Grid>
                    <GridItem span={6} rowSpan={1} style={{ marginRight: -16, marginLeft: -16 }}>
                        <UtilizationDonutChart
                            chartHeight={props.chartHeight}
                            chartWidth={props.chartWidth}
                            resourceDisplayName="CPU"
                            resourceUnit="vCPU"
                            randomizeUtilizations={randomizeUtilizations}
                        />
                    </GridItem>
                    <GridItem span={6} rowSpan={1} style={{ marginLeft: -16, marginRight: 16 }}>
                        <UtilizationDonutChart
                            chartHeight={props.chartHeight}
                            chartWidth={props.chartWidth}
                            resourceDisplayName="Memory"
                            resourceUnit="GB"
                            randomizeUtilizations={randomizeUtilizations}
                            converter={(val: number) => {
                                return val * 0.001048576; // Convert from MiB to GB.
                            }}
                        />
                    </GridItem>
                    <GridItem span={6} rowSpan={1} style={{ marginRight: -16, marginLeft: -16 }}>
                        <UtilizationDonutChart
                            chartHeight={props.chartHeight}
                            chartWidth={props.chartWidth}
                            resourceDisplayName="vGPU"
                            resourceUnit="vGPUs"
                            randomizeUtilizations={randomizeUtilizations}
                        />
                    </GridItem>
                    <GridItem span={6} rowSpan={1} style={{ marginLeft: -16, marginRight: 16 }}>
                        <UtilizationDonutChart
                            chartHeight={props.chartHeight}
                            chartWidth={props.chartWidth}
                            resourceDisplayName="GPU"
                            resourceUnit="GPUs"
                            randomizeUtilizations={randomizeUtilizations}
                        />
                    </GridItem>
                </Grid>
            </CardBody>
        </Card>
    );
};
