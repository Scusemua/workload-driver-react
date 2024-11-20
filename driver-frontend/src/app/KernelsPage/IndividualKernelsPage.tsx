import { KernelInfoIcons } from '@Cards/KernelListCard/KernelInfoBar';
import { Card, CardBody, Flex, FlexItem, PageSection, Text } from '@patternfly/react-core';
import useNavigation from '@Providers/NavigationProvider';
import { KernelOverflowMenu, KernelReplicaTable } from '@src/Components';
import { DistributedJupyterKernel } from '@src/Data';
import { useKernels } from '@src/Providers';
import React from 'react';
import { useParams } from 'react-router';

export const IndividualKernelsPage: React.FunctionComponent = () => {
    const params = useParams();

    const { kernelsMap } = useKernels(false);

    const { navigate } = useNavigation();

    const [targetKernel, setTargetKernel] = React.useState<DistributedJupyterKernel | undefined>(undefined);

    React.useEffect(() => {
        const kernelId: string | undefined = params.kernel_id;

        if (kernelId && kernelId !== '' && kernelId !== ':kernel_id') {
            const kernel: DistributedJupyterKernel | undefined = kernelsMap.get(kernelId);

            // console.log(`workload ${workloadId} tick durations: ${workload?.tick_durations_milliseconds}`)

            setTargetKernel(kernel);
        } else {
            // If there is no query parameter for the workload ID, then just redirect back to the workloads page.
            navigate('/kernels');
        }
    }, [navigate, params, kernelsMap]);

    /**
     * Return the content to be rendered on the page.
     */
    const getPageContent = (): React.ReactNode => {
        if (targetKernel) {
            return (
                <PageSection>
                    <Card isFullHeight isRounded>
                        <CardBody>
                            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsLg' }}>
                                <Flex direction={{ default: 'row' }}>
                                    <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsXs' }}>
                                        <FlexItem align={{ default: 'alignLeft' }}>
                                            <Text component={'h1'}>Kernel {targetKernel?.kernelId}</Text>
                                        </FlexItem>
                                        <FlexItem align={{ default: 'alignLeft' }}>
                                            <KernelInfoIcons
                                                kernel={targetKernel}
                                                iconSizes={'lg'}
                                                iconSpacingOverride={'spaceItemsXl'}
                                            />
                                        </FlexItem>
                                    </Flex>
                                    <FlexItem align={{ default: 'alignRight' }}>
                                        <KernelOverflowMenu
                                            kernel={targetKernel}
                                            onExecuteCodeClicked={() => {}}
                                            onPingKernelClicked={() => {}}
                                            onInterruptKernelClicked={() => {}}
                                            onTerminateKernelClicked={() => {}}
                                            onStopTrainingClicked={() => {}}
                                            onToggleOrSelectKernelDropdown={() => {}}
                                            openKernelDropdownMenu={targetKernel?.kernelId}
                                        />
                                    </FlexItem>
                                </Flex>
                                <FlexItem>
                                    <KernelReplicaTable
                                        kernel={targetKernel}
                                        openMigrationModal={() => {}}
                                        onExecuteCodeClicked={() => {}}
                                        setOpenReplicaDropdownMenu={() => {}}
                                        setOpenKernelDropdownMenu={() => {}}
                                        openReplicaDropdownMenu={targetKernel?.kernelId}
                                    />
                                </FlexItem>
                            </Flex>
                        </CardBody>
                    </Card>
                </PageSection>
            );
        } else {
            return (
                <PageSection>
                    <Text>Unknown kernel: &quot;{params.kernel_id}&quot;</Text>
                </PageSection>
            );
        }
    };

    return getPageContent();
};

export default IndividualKernelsPage;
