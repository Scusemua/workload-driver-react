import { KernelManager } from '@jupyterlab/services';
import { IKernelConnection } from '@jupyterlab/services/lib/kernel/kernel';
import { Alert, AlertActionCloseButton, Flex, FlexItem, Title } from '@patternfly/react-core';
import { SpinnerIcon } from '@patternfly/react-icons';
import { RequestTraceSplitTable } from '@src/Components';
import { DistributedJupyterKernel, JupyterKernelReplica, PongResponse } from '@src/Data';
import { DefaultDismiss, GetPathForFetch, GetToastContentWithHeaderAndBody, RoundToNDecimalPlaces } from '@src/Utils';
import React from 'react';
import toast from 'react-hot-toast';

export function PingKernel(kernelId: string, socketType: 'control' | 'shell') {
    console.log(`User is pinging kernel ${kernelId} using the ${socketType} socket.`);

    const req: RequestInit = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            Authorization: 'Bearer ' + localStorage.getItem('token'),
        },
        body: JSON.stringify({
            socketType: socketType,
            kernelId: kernelId,
            createdAtTimestamp: new Date(Date.now()).toISOString(),
        }),
    };

    const toastId: string = toast.custom(
        (t) => {
            return (
                <Alert
                    title={<b>Pinging kernel {kernelId} now...</b>}
                    variant={'custom'}
                    customIcon={<SpinnerIcon className={'loading-icon-spin-pulse'} />}
                    timeout={false}
                    actionClose={<AlertActionCloseButton onClose={() => toast.dismiss(t.id)} />}
                />
            );
        },
        {
            style: {
                maxWidth: 750,
            },
            icon: <SpinnerIcon className={'loading-icon-spin-pulse'} />,
        },
    );

    const startTime: number = performance.now();
    const initialRequestTimestamp: number = Date.now();
    fetch(GetPathForFetch('api/ping-kernel'), req)
        .catch((err: Error) => {
            toast.custom(
                () =>
                    GetToastContentWithHeaderAndBody(
                        `Failed to ping one or more replicas of kernel ${kernelId}.`,
                        err.message,
                        'danger',
                        () => {
                            toast.dismiss(toastId);
                        },
                    ),
                { id: toastId, style: { maxWidth: 750 } },
            );
        })
        .then(async (resp: Response | void) => {
            if (!resp) {
                console.error('No response from ping-kernel.');
                return;
            }

            if (resp.status != 200 || !resp.ok) {
                const response = await resp.json();
                toast.custom(
                    () =>
                        GetToastContentWithHeaderAndBody(
                            `Failed to ping one or more replicas of kernel ${kernelId}.`,
                            `${JSON.stringify(response)}`,
                            'danger',
                            () => {
                                toast.dismiss(toastId);
                            },
                        ),
                    { id: toastId, style: { maxWidth: 750 } },
                );
            } else {
                const response: PongResponse = await resp.json();
                const receivedReplyAt: number = Date.now();
                const latencyMilliseconds: number = RoundToNDecimalPlaces(performance.now() - startTime, 6);

                console.log('All Request Traces:');
                console.log(JSON.stringify(response.requestTraces, null, 2));

                toast.custom(
                    <Alert
                        isExpandable
                        variant={'success'}
                        title={`Pinged kernel ${response.id} via its ${socketType} channel (${latencyMilliseconds} ms)`}
                        timeoutAnimation={30000}
                        timeout={15000}
                        onTimeout={() => toast.dismiss(toastId)}
                        actionClose={<AlertActionCloseButton onClose={() => toast.dismiss(toastId)} />}
                    >
                        {response.requestTraces && response.requestTraces.length > 0 && (
                            <Flex direction={{ default: 'column' }}>
                                <FlexItem>
                                    <Title headingLevel={'h3'}>Request Trace(s)</Title>
                                </FlexItem>
                                <FlexItem>
                                    <RequestTraceSplitTable
                                        receivedReplyAt={receivedReplyAt}
                                        initialRequestSentAt={initialRequestTimestamp}
                                        messageId={response.msg}
                                        traces={response.requestTraces}
                                    />
                                </FlexItem>
                            </Flex>
                        )}
                    </Alert>,
                    { id: toastId },
                );
            }
        });
}

export function InstructKernelToStopTraining(kernelId?: string) {
    if (!kernelId) {
        console.error('Undefined kernel specified for interrupt target...');
        return;
    }

    console.log('User is interrupting kernel ' + kernelId);

    const req: RequestInit = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            Authorization: 'Bearer ' + localStorage.getItem('token'),
            // 'Cache-Control': 'no-cache, no-transform, no-store',
        },
        body: JSON.stringify({
            session_id: '',
            kernel_id: kernelId,
        }),
    };

    toast
        .promise(fetch(GetPathForFetch('api/stop-training'), req), {
            loading: <b>Interrupting kernel {kernelId} now...</b>,
            success: (resp: Response) => {
                if (!resp.ok || resp.status != 200) {
                    console.error(`Failed to interrupt kernel ${kernelId}.`);
                    throw new Error(`HTTP ${resp.status}: ${resp.statusText}`);
                }
                console.log(`Successfully interrupted kernel ${kernelId}.`);
                return (
                    <b>
                        Successfully interrupted kernel {kernelId} (HTTP {resp.status}: {resp.statusText}).
                    </b>
                );
            },
            error: (reason: Error) =>
                GetToastContentWithHeaderAndBody(
                    `Failed to interrupt kernel ${kernelId}.`,
                    `<b>Reason:</b> ${reason.message}`,
                    'danger',
                    () => {},
                ),
        })
        .then(() => {});
}

export async function InterruptKernel(kernelId: string, kernelManager: KernelManager) {
    console.log(`Connecting to kernel ${kernelId} (so we can interrupt it) now...`);

    const kernelConnection: IKernelConnection = kernelManager.connectTo({
        model: { id: kernelId, name: kernelId },
    });

    console.log(`Connected to kernel ${kernelId}. Attempting to interrupt kernel now...`);

    await kernelConnection.interrupt();

    console.log(`Interrupted kernel ${kernelId}.`);
}

/**
 * Delete the specified kernel.
 *
 * @param kernelId The ID of the kernel to be deleted.
 * @param toastId ID of associated Toast notification
 */
export async function DeleteKernel(kernelId: string, toastId?: string) {
    console.log('Deleting Kernel ' + kernelId + ' now.');
    const startTime: number = performance.now();

    const req: RequestInit = {
        method: 'DELETE',
    };

    let resp: Response;
    try {
        resp = await fetch(`jupyter/api/kernels/${kernelId}`, req);
    } catch (err) {
        toast.custom(
            GetToastContentWithHeaderAndBody(
                `Failed to Delete Kernel ${kernelId}`,
                [`Error: ${err}`],
                'danger',
                DefaultDismiss,
            ),
            { id: toastId },
        );
        return;
    }

    if (resp.ok && resp.status == 204) {
        console.log(`Successfully deleted kernel ${kernelId}`);
        toast.custom(
            GetToastContentWithHeaderAndBody(
                `Successfully Deleted Kernel ${kernelId}`,
                null,
                'success',
                DefaultDismiss,
            ),
            { id: toastId },
        );
    } else {
        console.error(`Received HTTP ${resp.status} ${resp.statusText} when trying to delete kernel ${kernelId}.`);

        const respText: string = await resp.text();

        toast.custom(
            GetToastContentWithHeaderAndBody(
                'Failed to Delete Kernel',
                [`Failed to delete kernel ${kernelId}`, `HTTP ${resp.status} ${resp.statusText}: ${respText}`],
                'danger',
                DefaultDismiss,
            ),
            { id: toastId },
        );

        return;
    }

    await fetch(GetPathForFetch('api/metrics'), {
        method: 'PATCH',
        headers: {
            'Content-Type': 'application/json',
            Authorization: 'Bearer ' + localStorage.getItem('token'),
            // 'Cache-Control': 'no-cache, no-transform, no-store',
        },
        body: JSON.stringify({
            name: 'distributed_cluster_jupyter_session_termination_latency_seconds',
            value: performance.now() - startTime,
            metadata: {
                kernel_id: kernelId,
            },
        }),
    });
}

export async function MigrateKernelReplica(
    targetReplica: JupyterKernelReplica,
    targetKernel: DistributedJupyterKernel,
    targetNodeId: string,
) {
    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            Authorization: 'Bearer ' + localStorage.getItem('token'),
            // 'Cache-Control': 'no-cache, no-transform, no-store',
        },
        body: JSON.stringify({
            targetReplica: {
                replicaId: targetReplica.replicaId,
                kernelId: targetKernel.kernelId,
            },
            targetNodeId: targetNodeId,
        }),
    };

    targetReplica.isMigrating = true;

    console.log(
        `Migrating replica ${targetReplica.replicaId} of kernel ${targetKernel.kernelId} to node ${targetNodeId}`,
    );
    toast(`Migrating replica ${targetReplica.replicaId} of kernel ${targetKernel.kernelId} to node ${targetNodeId}`, {
        duration: 7500,
        style: { maxWidth: 850 },
    });

    const response: Response = await fetch(GetPathForFetch('/api/migrate'), requestOptions);
    console.log(
        'Received response for migration operation of replica %d of kernel %s: %s',
        targetReplica.replicaId,
        targetKernel.kernelId,
        JSON.stringify(response),
    );
}
