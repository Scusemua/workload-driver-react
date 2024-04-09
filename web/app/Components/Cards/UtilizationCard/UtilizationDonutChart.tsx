import { KubernetesNode } from '@app/Data';
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

        nodes.forEach((node: KubernetesNode) => {
            console.log(`node.AllocatedResources: ${JSON.stringify(node.AllocatedResources)}`);
            console.log(`node.CapacityResources: ${JSON.stringify(node.CapacityResources)}\n`);
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

        let percentUtilization: number = roundTo2Decimals(sumAllocated / sumCapacity);
        if (Number.isNaN(percentUtilization)) {
            percentUtilization = 0.0;
        }

        setResource({
            Capacity: sumCapacity,
            Allocated: sumAllocated,
            PercentUtilization: percentUtilization,
        });
    }, [nodes]);

    // TODO: Convert this to a HoC where we pass the name of the target resource as a string in the props.
    // TODO: Update the KubernetesNode interface to contain a map of resource name to amount for both capacity and allocated.

    const getTitleComponent = () => {
        if (darkMode) {
            return (
                <ChartLabel
                    style={{
                        fill: 'white',
                        fontSize: 24,
                    }}
                />
            );
        } else {
            return undefined;
        }
    };

    const getSubtitleComponent = () => {
        if (darkMode) {
            return (
                <ChartLabel
                    style={{
                        fill: '#d4d4d4',
                        fontSize: 14,
                    }}
                />
            );
        } else {
            return undefined;
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
                bottom: 0,
                left: 10,
                right: 170,
                top: 0,
            }}
            theme={getTheme()}
            width={props.chartWidth}
        >
            <ChartDonutUtilization
                data={{
                    x: `${props.resourceDisplayName} Usage`,
                    y: resource?.PercentUtilization,
                }}
                labels={({ datum }) => (datum.x ? `${datum.x}: ${datum.y}%` : null)}
                legendData={[
                    { name: `${props.resourceDisplayName} Utilization: 80%` },
                    { name: `Warning at 60%` },
                    { name: `Danger at 90%` },
                ]}
                legendOrientation="vertical"
                title={`${resource?.PercentUtilization}%`}
                titleComponent={getTitleComponent()}
                subTitle={`${roundTo2Decimals(resource?.Allocated || 0)} ${props.resourceUnit} of ${roundTo2Decimals(
                    resource?.Capacity || 0,
                )} ${props.resourceUnit}`}
                subTitleComponent={getSubtitleComponent()}
                theme={getTheme()}
                thresholds={[{ value: 75 }, { value: 90 }]}
            />
        </ChartDonutThreshold>
    );
};
