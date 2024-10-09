export interface QueryMessageResponse {
    messageId: string;
    messageType: string;
    kernelId: string;
    gatewayReceivedRequest: number;
    gatewayForwardedRequest: number;
    gatewayReceivedReply: number;
    gatewayForwardedReply: number;
}

export interface RequestTrace {
    messageId: string;
    messageType: string;
    kernelId: string;
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
}

export type SplitName =
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

export function GetSplitsFromRequestTrace(trace: RequestTrace): RequestTraceSplit[] {
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
        end: Date.now(),
        latencyMilliseconds: Date.now() - trace.replySentByGateway,
    };

    return [
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
