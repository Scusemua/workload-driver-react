import { WorkloadInspectionView } from '@Components/Workloads/WorkloadInspectionView';
import { Button, Card, CardBody, CardFooter, Divider, Flex, FlexItem, PageSection, Text } from '@patternfly/react-core';
import { BackwardIcon, ExportIcon } from '@patternfly/react-icons';
import useNavigation from '@Providers/NavigationProvider';
import { WorkloadDataListCell } from '@src/Components/Workloads/WorkloadDataListCell';
import { Workload } from '@src/Data';
import { WorkloadContext } from '@src/Providers';
import React from 'react';
import { useParams } from 'react-router';

interface IndividualWorkloadPageProps {
    onVisualizeWorkloadClicked: (workload: Workload) => void;
}

export const IndividualWorkloadPage: React.FunctionComponent<IndividualWorkloadPageProps> = (
    props: IndividualWorkloadPageProps,
) => {
    const params = useParams();

    const { workloadsMap, exportWorkload } = React.useContext(WorkloadContext);

    const { navigate } = useNavigation();

    const [targetWorkload, setTargetWorkload] = React.useState<Workload | undefined>(undefined);

    React.useEffect(() => {
        const workloadId: string | undefined = params.workload_id;

        if (workloadId && workloadId !== '' && workloadId !== ':workload_id') {
            const workload: Workload | undefined = workloadsMap.get(workloadId);

            // console.log(`workload ${workloadId} tick durations: ${workload?.tick_durations_milliseconds}`)

            setTargetWorkload(workload);
        } else {
            // If there is no query parameter for the workload ID, then just redirect back to the workloads page.
            navigate('/workloads');
        }
    }, [navigate, params, workloadsMap]);

    /**
     * Return the content to be rendered on the page.
     */
    const getPageContent = (): React.ReactNode => {
        if (targetWorkload) {
            return (
                <PageSection>
                    <Card isFullHeight isRounded>
                        <CardBody>
                            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsLg' }}>
                                <FlexItem>
                                    <WorkloadDataListCell
                                        workload={targetWorkload}
                                        onVisualizeWorkloadClicked={props.onVisualizeWorkloadClicked}
                                    />
                                </FlexItem>
                                <FlexItem>
                                    <Divider />
                                </FlexItem>
                                <FlexItem>
                                    <WorkloadInspectionView workload={targetWorkload} showTickDurationChart={true} />
                                </FlexItem>
                            </Flex>
                        </CardBody>
                        <CardFooter>
                            <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsSm' }}>
                                <FlexItem>
                                    <Button icon={<BackwardIcon />} onClick={() => navigate(-1)}>
                                        Go Back
                                    </Button>
                                </FlexItem>
                                <FlexItem>
                                    <Button
                                        key="export_workload_state_button"
                                        aria-label={'Export workload state'}
                                        variant="secondary"
                                        icon={<ExportIcon />}
                                        onClick={() => {
                                            if (targetWorkload) {
                                                exportWorkload(targetWorkload);
                                            }
                                        }}
                                    >
                                        Export
                                    </Button>
                                </FlexItem>
                            </Flex>
                        </CardFooter>
                    </Card>
                </PageSection>
            );
        } else {
            return (
                <PageSection>
                    <Text>Unknown workload: &quot;{params.workload_id}&quot;</Text>
                </PageSection>
            );
        }
    };

    return getPageContent();
};

export default IndividualWorkloadPage;
