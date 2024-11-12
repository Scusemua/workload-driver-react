import { WorkloadInspectionView } from '@Components/Workloads/WorkloadInspectionView';
import { Card, CardBody, Divider, Flex, FlexItem, PageSection, Text } from '@patternfly/react-core';
import { WorkloadDataListCell } from '@src/Components/Workloads/WorkloadDataListCell';
import { Workload } from '@src/Data';
import { WorkloadContext } from '@src/Providers';
import { JoinPaths } from '@src/Utils/path_utils';
import React from 'react';
import { useParams } from 'react-router';
import { useNavigate } from 'react-router-dom';

interface IndividualWorkloadPageProps {
  onVisualizeWorkloadClicked: (workload: Workload) => void;
}

export const IndividualWorkloadPage: React.FunctionComponent<IndividualWorkloadPageProps> = (
    props: IndividualWorkloadPageProps,
) => {
    const params = useParams();

    const { workloadsMap } = React.useContext(WorkloadContext);

    const navigate = useNavigate();

    const [targetWorkload, setTargetWorkload] = React.useState<Workload | undefined>(undefined);

    React.useEffect(() => {
        const workloadId: string | undefined = params.workload_id;

        if (workloadId && workloadId !== '' && workloadId !== ':workload_id') {
            const workload: Workload | undefined = workloadsMap.get(workloadId);

            console.log(`workload ${workloadId} tick durations: ${workload?.tick_durations_milliseconds}`)

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
