import { ClusterNode } from '@app/Data';
import { DarkModeContext, useNodes } from '@app/Providers';
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
    resourceDisplayName: string;
    resourceUnit: string;
    chartWidth?: number;
    chartHeight?: number;
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
            if (node.NodeId.includes('control-plane')) {
                return;
            }

            const allocated: number | undefined = node.AllocatedResources[props.resourceDisplayName];
            if (allocated) {
                sumAllocated += allocated;
            }

            const capacity: number | undefined = node.CapacityResources[props.resourceDisplayName];
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
            data={[
                { x: `Warning at 75%`, y: 75 },
                { x: `Danger at 90%`, y: 90 },
            ]}
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
                        { name: `Warning at 60%`, symbol: { fill: '#D2D2D2' } },
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
                thresholds={[
                    { value: 0, color: '#3E8635' },
                    { value: 75, color: '#F0AB00' },
                    { value: 90, color: '#C9190B' },
                ]}
            />
        </ChartDonutThreshold>
    );
};
