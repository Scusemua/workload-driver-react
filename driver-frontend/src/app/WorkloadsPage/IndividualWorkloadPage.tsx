import { WorkloadInspectionView } from '@Components/Workloads/WorkloadInspectionView';
import { Card, CardBody, Divider, Flex, FlexItem, PageSection, Text } from '@patternfly/react-core';
import { WorkloadDataListCell } from '@src/Components/Workloads/WorkloadDataListCell';
import { Workload } from '@src/Data';
import { useWorkloads } from '@src/Providers';
import { JoinPaths } from '@src/Utils/path_utils';
import React from 'react';
import { useParams } from 'react-router';
import { useNavigate } from 'react-router-dom';

interface IndividualWorkloadPageProps {
  onPauseWorkloadClicked: (workload: Workload) => void;
  toggleDebugLogs: (workloadId: string, enabled: boolean) => void;
  onVisualizeWorkloadClicked: (workload: Workload) => void;
  onStartWorkloadClicked: (workload: Workload) => void;
  onStopWorkloadClicked: (workload: Workload) => void;
}

export const IndividualWorkloadPage: React.FunctionComponent<IndividualWorkloadPageProps> = (
    props: IndividualWorkloadPageProps,
) => {
    const params = useParams();

    const { workloadsMap } = useWorkloads();

    const navigate = useNavigate();

    const [targetWorload, setTargetWorkload] = React.useState<Workload | undefined>(undefined);

    React.useEffect(() => {
        const workloadId: string | undefined = params.workload_id;

        if (workloadId && workloadId !== '' && workloadId !== ':workload_id') {
            const workload: Workload | undefined = workloadsMap.get(workloadId);

            setTargetWorkload(workload);
        } else {
            // If there is no query parameter for the workload ID, then just redirect back to the workloads page.
            navigate(JoinPaths(process.env.PUBLIC_PATH || '/', '/workloads'));
        }
    }, [navigate, params, workloadsMap]);

    /**
     * Return the content to be rendered on the page.
     */
    const getPageContent = (): React.ReactNode => {
        if (targetWorload) {
            return (
                <PageSection>
                    <Card isFullHeight isRounded>
                        <CardBody>
                            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsLg' }}>
                                <FlexItem>
                                    <WorkloadDataListCell
                                        workload={targetWorload}
                                        onPauseWorkloadClicked={() => {}}
                                        toggleDebugLogs={() => {}}
                                        onVisualizeWorkloadClicked={() => {}}
                                        onStartWorkloadClicked={() => {}}
                                        onStopWorkloadClicked={() => {}}
                                    />
                                </FlexItem>
                                <FlexItem>
                                    <Divider />
                                </FlexItem>
                                <FlexItem>
                                    <WorkloadInspectionView workload={targetWorload} showTickDurationChart={true} />
                                </FlexItem>
                            </Flex>
                        </CardBody>
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
