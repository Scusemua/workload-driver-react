import { KernelInfoIcons } from '@Cards/KernelListCard/KernelInfoBar';
import { ConfirmationModal, ExecuteCodeOnKernelModal, MigrationModal, PingKernelModal } from '@Components/Modals';
import { Card, CardBody, Flex, FlexItem, PageSection, Text, Title } from '@patternfly/react-core';
import { SpinnerIcon } from '@patternfly/react-icons';
import { ExecutionOutputTabsDataProvider } from '@Providers/ExecutionOutputTabsDataProvider';
import { useKernelAndSessionManagers } from '@Providers/KernelAndSessionManagerProvider';
import useNavigation from '@Providers/NavigationProvider';
import {
    DeleteKernel,
    ExecuteCodeOnKernelPanel,
    InstructKernelToStopTraining,
    InterruptKernel,
    KernelOverflowMenu,
    KernelReplicaTable,
    MigrateKernelReplica,
    PingKernel,
} from '@src/Components';
import { DistributedJupyterKernel, JupyterKernelReplica } from '@src/Data';
import { useKernels } from '@src/Providers';
import { DefaultDismiss, GetToastContentWithHeaderAndBody } from '@src/Utils';
import React from 'react';
import toast from 'react-hot-toast';
import { useParams } from 'react-router';

export const IndividualKernelsPage: React.FunctionComponent = () => {
    const params = useParams();

    const { kernelsMap } = useKernels(false);

    const { navigate } = useNavigation();

    const [migrateReplica, setMigrateReplica] = React.useState<JupyterKernelReplica | null>(null);
    const [isMigrateModalOpen, setIsMigrateModalOpen] = React.useState(false);
    const [executeCodeKernelReplica, setExecuteCodeKernelReplica] = React.useState<JupyterKernelReplica | null>(null);
    const [isExecuteCodeModalOpen, setIsExecuteCodeModalOpen] = React.useState(false);
    const [isConfirmDeleteKernelModalOpen, setIsConfirmDeleteKernelModalOpen] = React.useState(false);
    const [isPingKernelModalOpen, setIsPingKernelModalOpen] = React.useState(false);
    const [targetKernel, setTargetKernel] = React.useState<DistributedJupyterKernel | undefined>(undefined);

    const { kernelManager, kernelManagerIsInitializing } = useKernelAndSessionManagers();

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

    function onPingKernelClicked() {
        setIsPingKernelModalOpen(true);
    }

    const onConfirmPingKernelClicked = (kernelId: string, socketType: 'control' | 'shell') => {
        setIsPingKernelModalOpen(false);

        PingKernel(kernelId, socketType);
    };

    const onCancelDeleteKernelClicked = () => {
        setIsConfirmDeleteKernelModalOpen(false);
    };

    const onStopTrainingClicked = (kernel: DistributedJupyterKernel) => {
        InstructKernelToStopTraining(kernel.kernelId);
    };

    const onInterruptKernelClicked = async (kernel: DistributedJupyterKernel) => {
        if (!kernelManager || kernelManagerIsInitializing) {
            toast.custom(() =>
                GetToastContentWithHeaderAndBody(
                    `Cannot Interrupt Kernel ${kernel.kernelId}`,
                    'Kernel Manager is initializing. Please try again in a few seconds.',
                    'warning',
                    DefaultDismiss,
                ),
            );
            return;
        }

        await InterruptKernel(kernel.kernelId, kernelManager);
    };

    const onConfirmDeleteKernelsClicked = async () => {
        setIsConfirmDeleteKernelModalOpen(false);

        if (!targetKernel) {
            return;
        }

        // Create a new kernel.
        if (!kernelManager || kernelManagerIsInitializing) {
            console.error('Kernel Manager is not available. Will try to connect...');
            toast.custom(() =>
                GetToastContentWithHeaderAndBody(
                    `Cannot Stop Kernel ${targetKernel.kernelId}`,
                    'Kernel Manager is initializing. Please try again in a few seconds.',
                    'warning',
                    DefaultDismiss,
                ),
            );
            return;
        }

        const toastId: string = toast.custom(
            GetToastContentWithHeaderAndBody(
                'Deleting Kernel',
                `Deleting kernel ${targetKernel.kernelId}`,
                'info',
                DefaultDismiss,
                undefined,
                <SpinnerIcon className={'loading-icon-spin-pulse'} />,
            ),
        );
        await DeleteKernel(targetKernel.kernelId, toastId);
    };

    /**
     * Handles 'Execute' clicks within the replica table.
     */
    const onExecuteCodeClicked = (kernel?: DistributedJupyterKernel, replicaIdx?: number | undefined) => {
        if (!kernel || kernel !== targetKernel) {
            return;
        }

        // If we clicked the 'Execute' button associated with a specific replica, then set the state for that replica.
        if (replicaIdx !== undefined) {
            // Need to use "!== undefined" because a `replicaIdx` of 0 will be coerced to false if by itself.
            console.log(
                'Will be executing code on replica %d of kernel %s.',
                targetKernel.replicas[replicaIdx].replicaId,
                targetKernel.kernelId,
            );
            setExecuteCodeKernelReplica(targetKernel.replicas[replicaIdx]);
        } else {
            setExecuteCodeKernelReplica(null);
        }

        setIsExecuteCodeModalOpen(true);
    };

    const onConfirmMigrateReplica = async (
        targetReplica: JupyterKernelReplica,
        targetKernel: DistributedJupyterKernel,
        targetNodeId: string,
    ) => {
        // Close the migration modal and reset its state.
        setIsMigrateModalOpen(false);
        setMigrateReplica(null);

        await MigrateKernelReplica(targetReplica, targetKernel, targetNodeId);
    };

    const closeMigrateReplicaModal = () => {
        // Close the migration modal and reset its state.
        setIsMigrateModalOpen(false);
        setMigrateReplica(null);
    };

    const openMigrationModal = (kernel: DistributedJupyterKernel, replica: JupyterKernelReplica) => {
        if (kernel !== targetKernel) {
            return;
        }

        setMigrateReplica(replica);
        setIsMigrateModalOpen(true);
    };

    /**
     * Return the content to be rendered on the page.
     */
    const getPageContent = (): React.ReactNode => {
        if (targetKernel) {
            return (
                <PageSection>
                    <Card isFullHeight isRounded>
                        <CardBody>
                            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItems2xl' }}>
                                <Flex direction={{ default: 'row' }}>
                                    <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
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
                                            onExecuteCodeClicked={() => setIsExecuteCodeModalOpen(true)}
                                            onPingKernelClicked={onPingKernelClicked}
                                            onInterruptKernelClicked={onInterruptKernelClicked}
                                            onTerminateKernelClicked={() => setIsConfirmDeleteKernelModalOpen(true)}
                                            onStopTrainingClicked={onStopTrainingClicked}
                                            onToggleOrSelectKernelDropdown={() => {}}
                                            openKernelDropdownMenu={targetKernel?.kernelId}
                                        />
                                    </FlexItem>
                                </Flex>
                                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
                                    <FlexItem>
                                        <Title headingLevel={'h2'}>Replicas</Title>
                                    </FlexItem>
                                    <FlexItem>
                                        <KernelReplicaTable
                                            kernel={targetKernel}
                                            openMigrationModal={openMigrationModal}
                                            onExecuteCodeClicked={onExecuteCodeClicked}
                                            setOpenReplicaDropdownMenu={() => {}}
                                            setOpenKernelDropdownMenu={() => {}}
                                            openReplicaDropdownMenu={targetKernel?.kernelId}
                                        />
                                    </FlexItem>
                                </Flex>
                                <FlexItem>
                                    <ExecuteCodeOnKernelPanel kernel={targetKernel} replicaId={-1} />
                                </FlexItem>
                            </Flex>
                        </CardBody>
                    </Card>
                    <PingKernelModal
                        isOpen={isPingKernelModalOpen}
                        onClose={() => setIsPingKernelModalOpen(false)}
                        onConfirm={onConfirmPingKernelClicked}
                        kernelId={targetKernel.kernelId}
                    />
                    <ConfirmationModal
                        isOpen={isConfirmDeleteKernelModalOpen}
                        onConfirm={() => onConfirmDeleteKernelsClicked()}
                        onClose={onCancelDeleteKernelClicked}
                        title={'Terminate Kernel'}
                        message={"Are you sure you'd like to delete the specified kernel?"}
                    />
                    <ExecutionOutputTabsDataProvider>
                        <ExecuteCodeOnKernelModal
                            kernel={targetKernel}
                            replicaId={executeCodeKernelReplica?.replicaId}
                            isOpen={isExecuteCodeModalOpen}
                            onClose={() => setIsExecuteCodeModalOpen(false)}
                        />
                    </ExecutionOutputTabsDataProvider>
                    {migrateReplica && (
                        <MigrationModal
                            isOpen={isMigrateModalOpen}
                            onClose={closeMigrateReplicaModal}
                            onConfirm={onConfirmMigrateReplica}
                            targetKernel={targetKernel}
                            targetReplica={migrateReplica}
                        />
                    )}
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
