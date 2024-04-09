import React from 'react';
import {
    Button,
    Card,
    CardTitle,
    CardBody,
    Flex,
    FlexItem,
    Title,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
    CardHeader,
    Grid,
    GridItem,
} from '@patternfly/react-core';
import { SyncIcon } from '@patternfly/react-icons';
import { UtilizationDonutChart } from './UtilizationDonutChart';
import { useNodes } from '@app/Providers';
import { toast } from 'react-hot-toast';

export interface UtilizationCardProps {
    chartWidth?: number;
    chartHeight?: number;
}

export const UtilizationCard: React.FunctionComponent<UtilizationCardProps> = (props: UtilizationCardProps) => {
    const { refreshNodes, nodesAreLoading } = useNodes();

    const toolbar = (
        <React.Fragment>
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
                        Cluster Resource Utilization
                    </Title>
                </CardTitle>
            </CardHeader>
            <CardBody>
                <Flex spaceItems={{ default: 'spaceItemsXl' }} direction={{ default: 'column' }}>
                    <Flex spaceItems={{ default: 'spaceItemsNone' }} direction={{ default: 'row' }}>
                        <FlexItem>
                            <div style={{ height: props.chartHeight, width: props.chartWidth }}>
                                <UtilizationDonutChart
                                    chartHeight={props.chartHeight}
                                    chartWidth={props.chartWidth}
                                    resourceDisplayName="CPU"
                                    resourceUnit="vCPU"
                                />
                            </div>
                        </FlexItem>
                        <FlexItem>
                            <div style={{ height: props.chartHeight, width: props.chartWidth }}>
                                <UtilizationDonutChart
                                    chartHeight={props.chartHeight}
                                    chartWidth={props.chartWidth}
                                    resourceDisplayName="Memory"
                                    resourceUnit="GB"
                                    converter={(val: number) => {
                                        return val * 0.001048576; // Convert from MiB to GB.
                                    }}
                                />
                            </div>
                        </FlexItem>
                    </Flex>
                    <Flex spaceItems={{ default: 'spaceItemsNone' }} direction={{ default: 'row' }}>
                        <FlexItem>
                            <div style={{ height: props.chartHeight, width: props.chartWidth }}>
                                <UtilizationDonutChart
                                    chartHeight={props.chartHeight}
                                    chartWidth={props.chartWidth}
                                    resourceDisplayName="vGPU"
                                    resourceUnit="vGPUs"
                                />
                            </div>
                        </FlexItem>
                        <FlexItem>
                            <div style={{ height: props.chartHeight, width: props.chartWidth }}>
                                <UtilizationDonutChart
                                    chartHeight={props.chartHeight}
                                    chartWidth={props.chartWidth}
                                    resourceDisplayName="GPU"
                                    resourceUnit="GPUs"
                                />
                            </div>
                        </FlexItem>
                    </Flex>
                </Flex>
            </CardBody>
        </Card>
    );
};
