import {
    ClusterNode,
    GetNodeAllocatedResource,
    GetNodeId,
    GetNodeIdleResource,
    GetNodePendingResource,
    GetNodeSpecResource,
} from '@src/Data';
import { DarkModeContext, useNodes } from '@src/Providers';
import {
    ChartDonutThreshold,
    ChartDonutUtilization,
    ChartLabel,
    ChartThemeColor,
    ChartThemeDefinitionInterface,
    getCustomTheme,
} from '@patternfly/react-charts';
import React from 'react';

export interface UtilizationDonutChart {
    resourceDisplayName: 'CPU' | 'GPU' | 'VRAM' | 'Memory';
    resourceUnit: string;
    chartWidth?: number;
    chartHeight?: number;
    resourceStatus: 'idle' | 'pending' | 'committed';
    randomizeUtilizations?: boolean;
    showLegend?: boolean;
    converter?: (val: number) => number;
}

interface Resource {
    Capacity: number;
    Allocated: number;
    PercentUtilization: number;
}

function roundTo2Decimals(val: number) {
    return Math.round((val + Number.EPSILON) * 100) / 100;
}

export const UtilizationDonutChart: React.FunctionComponent<UtilizationDonutChart> = (props: UtilizationDonutChart) => {
    const { darkMode } = React.useContext(DarkModeContext);
    const [resource, setResource] = React.useState<Resource | undefined>(undefined);
    const { nodes } = useNodes();

    React.useEffect(() => {
        let sumAllocated: number = 0.0;
        let sumCapacity: number = 0.0;

        nodes.forEach((node: ClusterNode) => {
            if (GetNodeId(node).includes('control-plane')) {
                return;
            }

            let amountUsed: number | undefined = 0;

            if (props.resourceStatus == 'idle') {
                amountUsed = GetNodeIdleResource(node, props.resourceDisplayName);
            } else if (props.resourceStatus == 'pending') {
                amountUsed = GetNodePendingResource(node, props.resourceDisplayName);
            } else {
                amountUsed = GetNodeAllocatedResource(node, props.resourceDisplayName);
            }

            // const amountUsed: number | undefined = node.AllocatedResources[props.resourceDisplayName];
            if (amountUsed) {
                sumAllocated += amountUsed;
            }

            const capacity: number | undefined = GetNodeSpecResource(node, props.resourceDisplayName);
            if (capacity !== undefined) {
                sumCapacity += capacity;
            }
        });

        if (props.converter !== undefined) {
            sumAllocated = roundTo2Decimals(props.converter(sumAllocated)); // Math.round((props.converter(sumAllocated) + Number.EPSILON) * 100) / 100;
            sumCapacity = roundTo2Decimals(props.converter(sumCapacity)); // Math.round((props.converter(sumCapacity) + Number.EPSILON) * 100) / 100;
        }

        if (props.randomizeUtilizations) {
            if (sumCapacity == 0) {
                sumCapacity = Math.floor(Math.random() * 100.0);
                console.log(`Randomized sumCapacity for ${props.resourceDisplayName}: ${sumCapacity}`);
            }
            sumAllocated = Math.floor(Math.random() * sumCapacity);
            console.log(`Randomized sumAllocated for ${props.resourceDisplayName}: ${sumAllocated}`);
        }
        let percentUtilization: number = roundTo2Decimals((sumAllocated * 100.0) / sumCapacity);
        if (Number.isNaN(percentUtilization)) {
            percentUtilization = 0.0;
        }

        setResource({
            Capacity: sumCapacity,
            Allocated: sumAllocated,
            PercentUtilization: percentUtilization,
        });
    }, [nodes, props]);

    // TODO: Convert this to a HoC where we pass the name of the target resource as a string in the props.
    // TODO: Update the ClusterNode interface to contain a map of resource name to amount for both capacity and allocated.

    const getTitleComponent = () => {
        if (darkMode) {
            return (
                <ChartLabel
                    style={{
                        fill: 'white',
                        fontSize: 58,
                    }}
                />
            );
        } else {
            return (
                <ChartLabel
                    style={{
                        fontSize: 58,
                    }}
                />
            );
        }
    };

    const getSubtitleComponent = () => {
        if (darkMode) {
            return (
                <ChartLabel
                    style={{
                        fill: '#d4d4d4',
                        fontSize: 32,
                    }}
                    textAnchor={'middle'}
                    transform="translate(0,16)"
                />
            );
        } else {
            return (
                <ChartLabel
                    style={{
                        fontSize: 32,
                        fill: '#6A6E73',
                    }}
                    textAnchor={'middle'}
                    transform="translate(0,16)"
                />
            );
        }
    };

    const getTheme = () => {
        if (darkMode) {
            const theme: ChartThemeDefinitionInterface = getCustomTheme(ChartThemeColor.default, {
                axis: {
                    style: {
                        tickLabels: {
                            // this changed the color of my numbers to white
                            fill: 'white',
                        },
                    },
                },
                legend: {
                    style: {
                        labels: {
                            fill: 'white',
                        },
                    },
                },
            });

            return theme;
        } else {
            return undefined;
        }
    };

    const getThresholds = () => {
        if (props.resourceStatus == 'idle') {
            return [
                { value: 0, color: '#C9190B' },
                { value: 10, color: '#e67300' },
                { value: 25, color: '#ffdd00' },
                { value: 40, color: '#3E8635' },
            ];
        } else {
            return [
                { value: 0, color: '#3E8635' },
                { value: 60, color: '#ffdd00' },
                { value: 75, color: '#e67300' },
                { value: 90, color: '#C9190B' },
            ];
        }
    };

    const getData = () => {
        if (props.resourceStatus == 'idle') {
            return [
                { x: `Very High Resource Utilization`, y: 10 },
                { x: `High Resource Utilization`, y: 25 },
                { x: `Moderate Resource Utilization`, y: 40 },
                { x: `Low Resource Utilization`, y: 100 },
            ];
        } else {
            return [
                { x: `Low Resource Utilization`, y: 60 },
                { x: `Moderate Resource Utilization`, y: 75 },
                { x: `High Resource Utilization`, y: 90 },
                { x: `Very High Resource Utilization`, y: 100 },
            ];
        }
    };

    // const possiblyConvertToExponent = (val: number) => {
    //     if (val.toString().length > 6) {
    //         return val.toExponential(2);
    //     }
    //     return val;
    // };

    return (
        <ChartDonutThreshold
            ariaDesc={`Cluster ${props.resourceDisplayName} resource usage`}
            ariaTitle={`Cluster ${props.resourceDisplayName} resource usage`}
            constrainToVisibleArea={true}
            data={getData()}
            height={props.chartHeight}
            labels={({ datum }) => (datum.x ? datum.x : null)}
            padding={{
                bottom: 55,
                left: 25,
                right: 25,
                top: 0,
            }}
            colorScale={['#F0F0F0', '#D2D2D2', '#6A6E73']}
            width={props.chartWidth}
            subTitlePosition="bottom"
        >
            <ChartDonutUtilization
                data={{
                    x: `${props.resourceDisplayName} Usage`,
                    y: resource?.PercentUtilization,
                }}
                labels={({ datum }) => (datum.x ? `${datum.x}: ${datum.y}%` : null)}
                legendData={
                    (props.showLegend && [
                        { name: `${props.resourceDisplayName} Utilization` },
                        { name: `Minor Warning at 50%`, symbol: { fill: '#D2D2D2' } },
                        { name: `Major Warning at 75%`, symbol: { fill: '#D2D2D2' } },
                        { name: `Danger at 90%`, symbol: { fill: '#6A6E73 ' } },
                    ]) ||
                    []
                }
                legendOrientation="vertical"
                title={`${resource?.PercentUtilization}%`}
                titleComponent={getTitleComponent()}
                subTitle={`${roundTo2Decimals(resource?.Allocated || 0)} ${props.resourceUnit} of ${roundTo2Decimals(
                    resource?.Capacity || 0,
                )} ${props.resourceUnit}`}
                subTitleComponent={getSubtitleComponent()}
                // colorScale={['#3E8635', '#F0AB00', '#C9190B']}
                theme={getTheme()}
                thresholds={getThresholds()}
            />
        </ChartDonutThreshold>
    );
};
