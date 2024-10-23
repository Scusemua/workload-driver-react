import { Flex, FlexItem } from '@patternfly/react-core';
import { UtilizationDonutChart } from '@src/Components';
import React from 'react';

export interface UtilizationEntryProps {
    chartWidth?: number;
    chartHeight?: number;
    icon: React.ReactNode;
    resourceDisplayName: 'CPU' | 'GPU' | 'VRAM' | 'Memory';
    resourceUnit: string;
    selectedResourceType: 'idle' | 'pending' | 'committed';
    randomizeUtilizations: boolean;
}

export const UtilizationEntry: React.FunctionComponent<UtilizationEntryProps> = (props: UtilizationEntryProps) => {
    return (
        <Flex justifyContent={{ default: 'justifyContentCenter' }}>
            <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsSm' }}>
                <FlexItem>{props.icon}</FlexItem>
                <FlexItem>
                    <b>
                        {props.resourceDisplayName} {props.resourceUnit}
                    </b>
                </FlexItem>
            </Flex>
            <FlexItem>
                <UtilizationDonutChart
                    chartHeight={props.chartHeight}
                    chartWidth={props.chartWidth}
                    resourceDisplayName={props.resourceDisplayName}
                    resourceUnit={props.resourceUnit}
                    resourceStatus={props.selectedResourceType}
                    randomizeUtilizations={props.randomizeUtilizations}
                    showLegend={false}
                />
            </FlexItem>
        </Flex>
    );
};
