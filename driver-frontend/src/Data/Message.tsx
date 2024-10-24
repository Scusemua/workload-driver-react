export interface QueryMessageResponse {
    requestTraces: RequestTrace[];
}

export interface RequestTrace {
    messageId: string;
    messageType: string;
    kernelId: string;
    replicaId: number;
    requestReceivedByGateway: number;
    requestSentByGateway: number;
    requestReceivedByLocalDaemon: number;
    requestSentByLocalDaemon: number;
    requestReceivedByKernelReplica: number;
    replySentByKernelReplica: number;
    replyReceivedByLocalDaemon: number;
    replySentByLocalDaemon: number;
    replyReceivedByGateway: number;
    replySentByGateway: number;
    e2eLatencyMilliseconds: number;
}

export type SplitName =
    | 'ClusterRequestToGateway'
    | 'GatewayProcessRequest'
    | 'GatewayRequestToLocalDaemon'
    | 'LocalDaemonProcessRequest'
    | 'LocalDaemonRequestToKernel'
    | 'KernelProcessRequest'
    | 'KernelReplyToLocalDaemon'
    | 'LocalDaemonProcessReply'
    | 'LocalDaemonReplyToGateway'
    | 'GatewayProcessReply'
    | 'GatewayReplyToClient';

export const SplitNames: SplitName[] = [
    'ClusterRequestToGateway',
    'GatewayProcessRequest',
    'GatewayRequestToLocalDaemon',
    'LocalDaemonProcessRequest',
    'LocalDaemonRequestToKernel',
    'KernelProcessRequest',
    'KernelReplyToLocalDaemon',
    'LocalDaemonProcessReply',
    'LocalDaemonReplyToGateway',
    'GatewayProcessReply',
    'GatewayReplyToClient',
];

export const AdjustedSplitNames: string[] = [
    'Client → Gateway',
    'Gateway Processing Request',
    'Gateway → Scheduler Daemon',
    'Scheduler Daemon Processing Request',
    'Scheduler Daemon → Kernel',
    'Kernel Processing Request',
    'Kernel → SchedulerDaemon',
    'Scheduler Daemon Processing Reply',
    'Scheduler Daemon → Gateway',
    'Gateway Processing Reply',
    'Gateway → Client',
];

export interface RequestTraceSplit {
    messageId: string;
    messageType: string;
    kernelId: string;
    splitName: SplitName;
    start: number;
    end: number;
    latencyMilliseconds: number;
}

export function GetAverageRequestTrace(traces: RequestTrace[]): RequestTrace | void {
    if (traces.length == 0) {
        return;
    }

    const sumTrace: RequestTrace = traces.reduce((acc: RequestTrace, val: RequestTrace) => {
        acc.requestReceivedByGateway += val.requestReceivedByGateway;
        acc.requestSentByGateway += val.requestSentByGateway;
        acc.requestReceivedByLocalDaemon += val.requestReceivedByLocalDaemon;
        acc.requestSentByLocalDaemon += val.requestSentByLocalDaemon;
        acc.requestReceivedByKernelReplica += val.requestReceivedByKernelReplica;
        acc.replySentByKernelReplica += val.replySentByKernelReplica;
        acc.replyReceivedByLocalDaemon += val.replyReceivedByLocalDaemon;
        acc.replySentByLocalDaemon += val.replySentByLocalDaemon;
        acc.replyReceivedByGateway += val.replyReceivedByGateway;
        acc.replySentByGateway += val.replySentByGateway;

        return acc;
    });

    sumTrace.messageId = traces[0].messageId;
    sumTrace.messageType = traces[0].messageType;
    sumTrace.kernelId = traces[0].kernelId;
    sumTrace.requestReceivedByGateway = sumTrace.requestReceivedByGateway / traces.length;
    sumTrace.requestSentByGateway = sumTrace.requestSentByGateway / traces.length;
    sumTrace.requestReceivedByLocalDaemon = sumTrace.requestReceivedByLocalDaemon / traces.length;
    sumTrace.requestSentByLocalDaemon = sumTrace.requestSentByLocalDaemon / traces.length;
    sumTrace.requestReceivedByKernelReplica = sumTrace.requestReceivedByKernelReplica / traces.length;
    sumTrace.replySentByKernelReplica = sumTrace.replySentByKernelReplica / traces.length;
    sumTrace.replyReceivedByLocalDaemon = sumTrace.replyReceivedByLocalDaemon / traces.length;
    sumTrace.replySentByLocalDaemon = sumTrace.replySentByLocalDaemon / traces.length;
    sumTrace.replyReceivedByGateway = sumTrace.replyReceivedByGateway / traces.length;
    sumTrace.replySentByGateway = sumTrace.replySentByGateway / traces.length;

    return sumTrace;
}

/**
 * Generate and return a slice of RequestTraceSplit objects from the given RequestTrace.
 * @param replyReceived the unix milliseconds at which the reply was received by the frontend client.
 * @param trace the RequestTrace from which a slice of RequestTraceSplit objects will be created.
 * @param initialRequestSentAt the time at which the frontend client initially sent the request.
 */
export function GetSplitsFromRequestTrace(
    replyReceived: number,
    trace: RequestTrace,
    initialRequestSentAt: number | undefined,
): RequestTraceSplit[] {
    let splitClientToGateway: RequestTraceSplit;

    if (initialRequestSentAt !== undefined) {
        splitClientToGateway = {
            messageId: trace.messageId,
            messageType: trace.messageType,
            kernelId: trace.kernelId,
            splitName: 'GatewayProcessRequest',
            start: initialRequestSentAt,
            end: trace.requestReceivedByGateway,
            latencyMilliseconds: trace.requestReceivedByGateway - initialRequestSentAt,
        };
    } else {
        splitClientToGateway = {
            messageId: trace.messageId,
            messageType: trace.messageType,
            kernelId: trace.kernelId,
            splitName: 'GatewayProcessRequest',
            start: trace.requestReceivedByGateway,
            end: trace.requestReceivedByGateway,
            latencyMilliseconds: trace.requestReceivedByGateway - trace.requestReceivedByGateway,
        };
    }

    const splitGatewayProcessRequest: RequestTraceSplit = {
        messageId: trace.messageId,
        messageType: trace.messageType,
        kernelId: trace.kernelId,
        splitName: 'GatewayProcessRequest',
        start: trace.requestReceivedByGateway,
        end: trace.requestSentByGateway,
        latencyMilliseconds: trace.requestSentByGateway - trace.requestReceivedByGateway,
    };

    const splitGatewayRequestToLocalDaemon: RequestTraceSplit = {
        messageId: trace.messageId,
        messageType: trace.messageType,
        kernelId: trace.kernelId,
        splitName: 'GatewayRequestToLocalDaemon',
        start: trace.requestSentByGateway,
        end: trace.requestReceivedByLocalDaemon,
        latencyMilliseconds: trace.requestReceivedByLocalDaemon - trace.requestSentByGateway,
    };

    const splitLocalDaemonProcessRequest: RequestTraceSplit = {
        messageId: trace.messageId,
        messageType: trace.messageType,
        kernelId: trace.kernelId,
        splitName: 'LocalDaemonProcessRequest',
        start: trace.requestReceivedByLocalDaemon,
        end: trace.requestSentByLocalDaemon,
        latencyMilliseconds: trace.requestSentByLocalDaemon - trace.requestReceivedByLocalDaemon,
    };

    const splitLocalDaemonRequestToKernel: RequestTraceSplit = {
        messageId: trace.messageId,
        messageType: trace.messageType,
        kernelId: trace.kernelId,
        splitName: 'LocalDaemonRequestToKernel',
        start: trace.requestSentByLocalDaemon,
        end: trace.requestReceivedByKernelReplica,
        latencyMilliseconds: trace.requestReceivedByKernelReplica - trace.requestSentByLocalDaemon,
    };

    const splitKernelProcessRequest: RequestTraceSplit = {
        messageId: trace.messageId,
        messageType: trace.messageType,
        kernelId: trace.kernelId,
        splitName: 'KernelProcessRequest',
        start: trace.requestReceivedByKernelReplica,
        end: trace.replySentByKernelReplica,
        latencyMilliseconds: trace.replySentByKernelReplica - trace.requestReceivedByKernelReplica,
    };

    const splitKernelReplyToLocalDaemon: RequestTraceSplit = {
        messageId: trace.messageId,
        messageType: trace.messageType,
        kernelId: trace.kernelId,
        splitName: 'KernelReplyToLocalDaemon',
        start: trace.replySentByKernelReplica,
        end: trace.replyReceivedByLocalDaemon,
        latencyMilliseconds: trace.replyReceivedByLocalDaemon - trace.replySentByKernelReplica,
    };

    const splitLocalDaemonProcessReply: RequestTraceSplit = {
        messageId: trace.messageId,
        messageType: trace.messageType,
        kernelId: trace.kernelId,
        splitName: 'LocalDaemonProcessReply',
        start: trace.replyReceivedByLocalDaemon,
        end: trace.replySentByLocalDaemon,
        latencyMilliseconds: trace.replySentByLocalDaemon - trace.replyReceivedByLocalDaemon,
    };

    const splitLocalDaemonReplyToGateway: RequestTraceSplit = {
        messageId: trace.messageId,
        messageType: trace.messageType,
        kernelId: trace.kernelId,
        splitName: 'LocalDaemonReplyToGateway',
        start: trace.replySentByLocalDaemon,
        end: trace.replyReceivedByGateway,
        latencyMilliseconds: trace.replyReceivedByGateway - trace.replySentByLocalDaemon,
    };

    const splitGatewayProcessReply: RequestTraceSplit = {
        messageId: trace.messageId,
        messageType: trace.messageType,
        kernelId: trace.kernelId,
        splitName: 'GatewayProcessReply',
        start: trace.replyReceivedByGateway,
        end: trace.replySentByGateway,
        latencyMilliseconds: trace.replySentByGateway - trace.replyReceivedByGateway,
    };

    const splitGatewayReplyToClient: RequestTraceSplit = {
        messageId: trace.messageId,
        messageType: trace.messageType,
        kernelId: trace.kernelId,
        splitName: 'GatewayReplyToClient',
        start: trace.replySentByGateway,
        end: replyReceived,
        latencyMilliseconds: replyReceived - trace.replySentByGateway,
    };

    return [
        splitClientToGateway,
        splitGatewayProcessRequest,
        splitGatewayRequestToLocalDaemon,
        splitLocalDaemonProcessRequest,
        splitLocalDaemonRequestToKernel,
        splitKernelProcessRequest,
        splitKernelReplyToLocalDaemon,
        splitLocalDaemonProcessReply,
        splitLocalDaemonReplyToGateway,
        splitGatewayProcessReply,
        splitGatewayReplyToClient,
    ];
}

/**
 * Sent as a response to pinging a kernel.
 */
export interface PongResponse {
    id: string;
    success: boolean;
    msg: string;
    requestTraces: RequestTrace[];
}

/**
 * The format of a JSON-serialized buffers frame from a Jupyter kernel, containing a RequestTrace.
 */
export interface FirstJupyterKernelBuffersFrame {
    request_trace: RequestTrace;
}
