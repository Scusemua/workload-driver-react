import { GpuIcon } from '@app/Icons';
import { useNodes } from '@app/Providers';
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
    ToggleGroup,
    ToggleGroupItem,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
} from '@patternfly/react-core';
import { CpuIcon, MemoryIcon, SyncIcon } from '@patternfly/react-icons';
import React from 'react';
import { toast } from 'react-hot-toast';
import { UtilizationDonutChart } from './UtilizationDonutChart';

export interface UtilizationCardProps {
    chartWidth?: number;
    chartHeight?: number;
}

function roundToTwo(num: number) {
    return +(Math.round(Number.parseFloat(num.toString() + 'e+2')).toString() + 'e-2');
}

export const UtilizationCard: React.FunctionComponent<UtilizationCardProps> = (props: UtilizationCardProps) => {
    const { refreshNodes, nodesAreLoading } = useNodes();
    const [randomizeUtilizations, setRandomizeUtilizations] = React.useState(false);
    const [selectedResourceType, setSelectedResourceType] = React.useState<'idle' | 'pending' | 'committed'>(
        'committed',
    );

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
            <ToolbarGroup>
                <ToolbarItem>
                    <ToggleGroup
                        aria-label={
                            "Select the type of resource to visualize (i.e., 'idle', 'pending', or 'committed')."
                        }
                    >
                        <ToggleGroupItem
                            text={'Idle'}
                            buttonId={'toggle-idle-resource-utilization'}
                            isSelected={selectedResourceType == 'idle'}
                            onChange={() => setSelectedResourceType('idle')}
                        />
                        <ToggleGroupItem
                            text={'Pending'}
                            buttonId={'toggle-pending-resource-utilization'}
                            isSelected={selectedResourceType == 'pending'}
                            onChange={() => setSelectedResourceType('pending')}
                        />
                        <ToggleGroupItem
                            text={'Committed'}
                            buttonId={'toggle-committed-resource-utilization'}
                            isSelected={selectedResourceType == 'committed'}
                            onChange={() => setSelectedResourceType('committed')}
                        />
                    </ToggleGroup>
                </ToolbarItem>
            </ToolbarGroup>
            <ToolbarGroup variant="icon-button-group">
                <ToolbarItem>
                    <Tooltip exitDelay={75} content={<div>Refresh cluster resource utilization data.</div>}>
                        <Button
                            variant="plain"
                            onClick={() => {
                                // const st: number = performance.now();
                                refreshNodes().then(() => {});
                                // toast
                                //     .promise(
                                //         refreshNodes(),
                                //         {
                                //             loading: <b>Refreshing cluster resource utilization data...</b>,
                                //             success: () => (
                                //                 <div>
                                //                     <Flex
                                //                         direction={{ default: 'column' }}
                                //                         spaceItems={{ default: 'spaceItemsNone' }}
                                //                     >
                                //                         <FlexItem>
                                //                             <b>Refreshed cluster resource utilization data.</b>
                                //                         </FlexItem>
                                //                         <FlexItem>
                                //                             <Text component={TextVariants.small}>
                                //                                 Time elapsed: {roundToTwo(performance.now() - st)} ms.
                                //                             </Text>
                                //                         </FlexItem>
                                //                     </Flex>
                                //                 </div>
                                //             ),
                                //             error: (reason: Error) => {
                                //                 let explanation: string = reason.message;
                                //                 if (reason.name === 'SyntaxError') {
                                //                     explanation = 'HTTP 504 Gateway Timeout';
                                //                 }
                                //
                                //                 return (
                                //                     <div>
                                //                         <Flex
                                //                             direction={{ default: 'column' }}
                                //                             spaceItems={{ default: 'spaceItemsNone' }}
                                //                         >
                                //                             <FlexItem>
                                //                                 <b>
                                //                                     Could not refresh cluster resource utilization data.
                                //                                 </b>
                                //                             </FlexItem>
                                //                             <FlexItem>{explanation}</FlexItem>
                                //                         </Flex>
                                //                     </div>
                                //                 );
                                //             },
                                //         },
                                //         {
                                //             style: { maxWidth: 450 },
                                //         },
                                //     )
                                //     .then(() => {});
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
                                    resourceStatus={selectedResourceType}
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
                                    resourceStatus={selectedResourceType}
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
                                    resourceStatus={selectedResourceType}
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
                                    resourceStatus={selectedResourceType}
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
