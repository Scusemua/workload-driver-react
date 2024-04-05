/* eslint-disable camelcase */
import React from 'react';
import { Tab, TabTitleIcon, TabTitleText, Tabs } from '@patternfly/react-core';

import { KubernetesPodLogView } from '@cards/LogViewCard/Views/';
import { useKernels } from '@app/Providers';
import { DistributedJupyterKernel, JupyterKernelReplica } from '@app/Data';
import { LogHeightContext } from '../LogViewCard';
import { ServerIcon } from '@patternfly/react-icons';

export interface KernelLogTabContentProps {
    children?: React.ReactNode;
    abortController: AbortController;
}

export const KernelLogTabContent: React.FunctionComponent<KernelLogTabContentProps> = (
    props: KernelLogTabContentProps,
) => {
    const { kernels } = useKernels();

    const [activeKernelTabKey, setActiveKernelTabKey] = React.useState(
        kernels.length >= 1 ? `kernel-${kernels[0].kernelId}-tab` : '',
    );
    const [activeKernelReplicaTabKey, setActiveKernelReplicaTabKey] = React.useState(1);

    const logHeight = React.useContext(LogHeightContext);

    const handleKernelTabClick = (_e: React.MouseEvent<HTMLElement, MouseEvent>, tab: string | number) => {
        setActiveKernelTabKey(tab.toString());
    };

    const handleKernelReplicaTabClick = (_e: React.MouseEvent<HTMLElement, MouseEvent>, tab: string | number) => {
        setActiveKernelReplicaTabKey(Number(tab));
    };

    return (
        <Tabs isFilled isSecondary id="kernel-tabs" activeKey={activeKernelTabKey} onSelect={handleKernelTabClick}>
            {kernels.map((kernel: DistributedJupyterKernel) => {
                return (
                    <Tab
                        id={`kernel-${kernel.kernelId}-tab`}
                        key={kernel.kernelId}
                        eventKey={kernel.kernelId}
                        title={
                            <>
                                <TabTitleIcon>
                                    <ServerIcon />
                                </TabTitleIcon>
                                <TabTitleText>{`Kernel ${kernel.kernelId}`}</TabTitleText>
                            </>
                        }
                        tabContentId={`tab-content-kernel-${kernel.kernelId}`}
                    >
                        <Tabs
                            isFilled
                            isSecondary
                            id={`kernel-${kernel.kernelId}-replica-tabs`}
                            activeKey={activeKernelReplicaTabKey}
                            onSelect={handleKernelReplicaTabClick}
                            isBox={true}
                        >
                            {kernel?.replicas?.map((replica: JupyterKernelReplica) => {
                                return (
                                    <Tab
                                        id={`kernel-${kernel.kernelId}-replica-${replica.replicaId}-tab`}
                                        key={replica.replicaId}
                                        eventKey={replica.replicaId}
                                        title={
                                            <>
                                                <TabTitleIcon>
                                                    <ServerIcon />
                                                </TabTitleIcon>
                                                <TabTitleText>{`Replica ${replica.replicaId}`}</TabTitleText>
                                            </>
                                        }
                                    >
                                        <KubernetesPodLogView
                                            height={logHeight}
                                            podName={replica.podId}
                                            containerName="kernel"
                                            convertToHtml={false}
                                            logPollIntervalSeconds={1}
                                            signal={props.abortController.signal}
                                        />
                                    </Tab>
                                );
                            })}
                        </Tabs>
                    </Tab>
                );
            })}
        </Tabs>
    );
};
