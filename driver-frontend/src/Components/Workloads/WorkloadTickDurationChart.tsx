import { RoundToThreeDecimalPlaces } from '@Components/Modals';
import { Chart, ChartAxis, ChartGroup, ChartLine, ChartScatter, ChartTooltip } from '@patternfly/react-charts';
import { Workload } from '@src/Data';
import { DarkModeContext } from '@src/Providers';
import React from 'react';
import { VictoryZoomContainer } from 'victory-zoom-container';

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
        <Chart
            ariaDesc={'Line chart of tick durations'}
            ariaTitle={'Line chart of tick durations'}
            containerComponent={<VictoryZoomContainer allowPan allowZoom minimumZoom={{ x: 0.5, y: 0.5 }} />}
            legendOrientation="vertical"
            legendPosition="right"
            height={300}
            minDomain={{
                x: 0,
                y: 0,
            }}
            maxDomain={{
                x: props.workload.tick_durations_milliseconds.length,
                y: getMaxDomainY(),
            }}
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
                    style={{
                        data: {
                            strokeWidth: 5,
                        },
                    }}
                    interpolation={'natural'}
                    data={props.workload.tick_durations_milliseconds?.map((tickDurationMs: number, index: number) => {
                        const tickDurationRounded: number = RoundToThreeDecimalPlaces(tickDurationMs);

                        return {
                            name: 'Tick Duration',
                            x: index,
                            y: tickDurationRounded,
                            label: `${tickDurationRounded} ms`,
                        };
                    })}
                    labelComponent={<ChartTooltip />}
                />
                <ChartScatter
                    data={props.workload.tick_durations_milliseconds?.map((tickDurationMs: number, index: number) => {
                        const tickDurationRounded: number = RoundToThreeDecimalPlaces(tickDurationMs);

                        return {
                            name: 'Tick Duration',
                            x: index,
                            y: tickDurationRounded,
                            label: `${tickDurationRounded} ms`,
                        };
                    })}
                    size={6}
                    labelComponent={<ChartTooltip />}
                />
            </ChartGroup>
        </Chart>
    );
};

export default WorkloadTickDurationChart;
