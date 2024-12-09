import { RoundToThreeDecimalPlaces } from '@Utils/utils';
import { Workload } from '@src/Data';
import { DarkModeContext } from '@src/Providers';
import React from 'react';
import { VictoryLabel } from 'victory';
import { VictoryAxis } from 'victory-axis';
import { VictoryChart } from 'victory-chart';
import { VictoryLine } from 'victory-line';
import { VictoryTooltip } from 'victory-tooltip';
import { VictoryZoomContainer } from 'victory-zoom-container';

interface IWorkloadTickDurationChartProps {
    workload: Workload;
}

interface Datapoint {
    x: number;
    y: number;
    label: string;
}

/**
 * Chart that plots the duration of each tick of a workload.
 */
export const WorkloadTickDurationChart: React.FunctionComponent<IWorkloadTickDurationChartProps> = (
    props: IWorkloadTickDurationChartProps,
) => {
    const { darkMode } = React.useContext(DarkModeContext);

    const [, setThreshold] = React.useState<number>(0);

    const [data, setData] = React.useState<Datapoint[]>([]);

    React.useEffect(() => {
        if (props.workload.tick_durations_milliseconds.length == 0) {
            return;
        }

        const averageDuration: number =
            props.workload.sum_tick_durations_millis / props.workload.tick_durations_milliseconds.length;
        console.log(`averageDuration * 1.5: ${averageDuration * 1.5}`);
        setThreshold(averageDuration * 1.5);
    }, [
        props.workload.sum_tick_durations_millis,
        props.workload.tick_durations_milliseconds,
        props.workload.tick_durations_milliseconds.length,
    ]);

    React.useEffect(() => {
        const datapoints: Datapoint[] = [];

        for (let i: number = 0; i < props.workload.tick_durations_milliseconds.length; i++) {
            const tickDurationRounded: number = RoundToThreeDecimalPlaces(
                props.workload.tick_durations_milliseconds[i],
            );

            const datapoint: Datapoint = {
                x: i,
                y: tickDurationRounded,
                label: `${tickDurationRounded} ms`,
            };

            datapoints.push(datapoint);
        }

        setData(datapoints);
    }, [props.workload]);

    const getStyle = () => {
        if (darkMode) {
            return {
                axis: {
                    stroke: 'white',
                },
                tickLabels: {
                    fill: 'white',
                },
                grid: { stroke: '#F4F5F7', strokeWidth: 0.5 },
            };
        } else {
            return {
                grid: { stroke: '#646464', strokeWidth: 0.5 },
            };
        }
    };

    const getMaxDomainY = (): number => {
        if (props.workload.tick_durations_milliseconds.length == 0) {
            return 5000;
        }

        let max: number = 0;

        props.workload.tick_durations_milliseconds.forEach((dur: number) => {
            if (dur > max) {
                max = dur;
            }
        });

        return max * 1.1;
    };

    return (
        <VictoryChart
            containerComponent={<VictoryZoomContainer />}
            minDomain={{
                x: 0,
                y: 0,
            }}
            maxDomain={{
                x: props.workload.tick_durations_milliseconds.length,
                y: getMaxDomainY(),
            }}
            padding={{
                bottom: 100,
                left: 100,
                right: 25, // Adjusted to accommodate legend
                top: 75,
            }}
            height={300}
            width={950}
        >
            <VictoryLabel
                text="Tick Durations (milliseconds)"
                x={475}
                y={50}
                textAnchor="middle"
                style={{
                    fill: darkMode ? '#ffffff' : '#383838',
                    fontSize: 20,
                }}
            />
            <VictoryAxis style={getStyle()} />
            <VictoryAxis dependentAxis style={getStyle()} />
            <VictoryLine
                style={{
                    data: {
                        stroke: '#007dff',
                        strokeWidth: 3,
                    },
                }}
                interpolation={'natural'}
                data={data}
                labelComponent={<VictoryTooltip />}
            />
        </VictoryChart>
    );
};

export default WorkloadTickDurationChart;
