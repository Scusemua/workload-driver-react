import { RoundToThreeDecimalPlaces } from '@Components/Modals';
import { Chart, ChartAxis, ChartGroup, ChartLine, ChartVoronoiContainer } from '@patternfly/react-charts';
import { Workload } from '@src/Data';
import { DarkModeContext } from '@src/Providers';
import React from 'react';

interface IWorkloadTickDurationChartProps {
    workload: Workload;
}

/**
 * Chart that plots the duration of each tick of a workload.
 */
export const WorkloadTickDurationChart: React.FunctionComponent<IWorkloadTickDurationChartProps> = (
    props: IWorkloadTickDurationChartProps,
) => {
    const { darkMode } = React.useContext(DarkModeContext);

    const getStyle = () => {
        if (darkMode) {
            return {
                axis: {
                    stroke: 'white',
                },
                tickLabels: {
                    fill: 'white',
                },
            };
        }

        return {};
    };

    return (
        <Chart
            ariaDesc={'Line chart of tick durations'}
            ariaTitle={'Line chart of tick durations'}
            containerComponent={
                <ChartVoronoiContainer labels={({ datum }) => `${datum.name}: ${datum.y}`} constrainToVisibleArea />
            }
            legendOrientation="vertical"
            legendPosition="right"
            height={300}
            name="tickDurations"
            title={'Tick Durations (Milliseconds)'}
            padding={{
                bottom: 100,
                left: 100,
                right: 25, // Adjusted to accommodate legend
                top: 75,
            }}
            width={950}
        >
            <ChartAxis name={'Tick'} label={'Tick'} showGrid style={getStyle()} />
            <ChartAxis dependentAxis showGrid style={getStyle()} />
            <ChartGroup>
                <ChartLine
                    data={props.workload.tick_durations_milliseconds?.map((tickDurationMs: number, index: number) => {
                        return {
                            name: 'Tick Duration',
                            x: index,
                            y: RoundToThreeDecimalPlaces(tickDurationMs),
                        };
                    })}
                />
            </ChartGroup>
        </Chart>
    );
};

export default WorkloadTickDurationChart;
