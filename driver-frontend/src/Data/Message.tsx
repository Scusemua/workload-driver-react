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
    requestTraceUuid: string;
    cudaInitMicroseconds: number;
    downloadDependencyMicroseconds: number;
    downloadModelAndTrainingDataMicroseconds: number;
    uploadModelAndTrainingDataMicroseconds: number;
    // executionTimeMicroseconds is the amount of time spent executing user-submitted code, excluding any other
    // overheads, if applicable. The units are microseconds.
    executionTimeMicroseconds: number;
    executionStartUnixMillis: number;
    executionEndUnixMillis: number;
    replayTimeMicroseconds: number;
    copyFromCpuToGpuMicroseconds: number;
    copyFromGpuToCpuMicroseconds: number;
    // leaderElectionTimeMicroseconds is the amount of time, in microseconds, that the kernel spent handling the
    // leader election prior to executing the user-submitted code, if applicable.
    leaderElectionTimeMicroseconds: number;
    // electionCreationTime is the time at which the kernel created its Election object, if applicable.
    electionCreationTime: number;
    // electionProposalPhaseStartTime is the time at which the kernel started its Election, if applicable.
    electionProposalPhaseStartTime: number;
    // electionExecutionPhaseStartTime is when the leader was selected.
    electionExecutionPhaseStartTime: number;
    // electionEndTime is when the execution fully completed and/or the follower was notified by the leader that it
    // finished executing.
    electionEndTime: number;
    e2eLatencyMilliseconds: number;
}

export type SplitName =
    | 'Client → Global Scheduler'
    | 'Global Scheduler Processing Request'
    | 'Global Scheduler → Local Scheduler'
    | 'Local Scheduler Processing Request'
    | 'Local Scheduler → Kernel'
    | 'Kernel Processing Request'
    | 'Kernel Preprocessing Request'
    | 'Kernel Creating Election'
    | 'Kernel Election Proposal/Vote Phase'
    | 'Kernel Executing Code'
    | 'Kernel Postprocessing Request'
    | 'Kernel → Local Scheduler'
    | 'Local Scheduler Processing Reply'
    | 'Local Scheduler → Global Scheduler'
    | 'Global Scheduler Processing Reply'
    | 'Global Scheduler → Client';
// | 'ClusterRequestToGateway'
// | 'GatewayProcessRequest'
// | 'GatewayRequestToLocalDaemon'
// | 'LocalDaemonProcessRequest'
// | 'LocalDaemonRequestToKernel'
// | 'KernelProcessRequest'
// | 'KernelPreprocessRequest'
// | 'KernelElectionCreation'
// | 'KernelProposalVotePhase'
// | 'KernelExecuteCodePhase'
// | 'KernelReplyToLocalDaemon'
// | 'LocalDaemonProcessReply'
// | 'LocalDaemonReplyToGateway'
// | 'GatewayProcessReply'
// | 'GatewayReplyToClient';

// export const AdjustedSplitNames: string[] = [
//     'Client → Gateway',
//     'Gateway Processing Request',
//     'Gateway → Scheduler Daemon',
//     'Scheduler Daemon Processing Request',
//     'Scheduler Daemon → Kernel',
//     'Kernel Processing Request',
//     'Kernel Pre-Processing Request',
//     'Kernel Creating Election',
//     'Kernel Election Proposal/Vote Phase',
//     'Kernel Executing Code',
//     'Kernel → SchedulerDaemon',
//     'Scheduler Daemon Processing Reply',
//     'Scheduler Daemon → Gateway',
//     'Gateway Processing Reply',
//     'Gateway → Client',
// ];

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
    const requestTraceSplits: RequestTraceSplit[] = [];

    let splitClientToGateway: RequestTraceSplit;

    if (initialRequestSentAt !== undefined) {
        splitClientToGateway = {
            messageId: trace.messageId,
            messageType: trace.messageType,
            kernelId: trace.kernelId,
            splitName: 'Client → Global Scheduler',
            start: initialRequestSentAt,
            end: trace.requestReceivedByGateway,
            latencyMilliseconds: trace.requestReceivedByGateway - initialRequestSentAt,
        };
    } else {
        splitClientToGateway = {
            messageId: trace.messageId,
            messageType: trace.messageType,
            kernelId: trace.kernelId,
            splitName: 'Client → Global Scheduler',
            start: trace.requestReceivedByGateway,
            end: trace.requestReceivedByGateway,
            latencyMilliseconds: trace.requestReceivedByGateway - trace.requestReceivedByGateway,
        };
    }
    requestTraceSplits.push(splitClientToGateway);

    const splitGatewayProcessRequest: RequestTraceSplit = {
        messageId: trace.messageId,
        messageType: trace.messageType,
        kernelId: trace.kernelId,
        splitName: 'Global Scheduler Processing Request',
        start: trace.requestReceivedByGateway,
        end: trace.requestSentByGateway,
        latencyMilliseconds: trace.requestSentByGateway - trace.requestReceivedByGateway,
    };
    requestTraceSplits.push(splitGatewayProcessRequest);

    const splitGatewayRequestToLocalDaemon: RequestTraceSplit = {
        messageId: trace.messageId,
        messageType: trace.messageType,
        kernelId: trace.kernelId,
        splitName: 'Global Scheduler → Local Scheduler',
        start: trace.requestSentByGateway,
        end: trace.requestReceivedByLocalDaemon,
        latencyMilliseconds: trace.requestReceivedByLocalDaemon - trace.requestSentByGateway,
    };
    requestTraceSplits.push(splitGatewayRequestToLocalDaemon);

    const splitLocalDaemonProcessRequest: RequestTraceSplit = {
        messageId: trace.messageId,
        messageType: trace.messageType,
        kernelId: trace.kernelId,
        splitName: 'Local Scheduler Processing Request',
        start: trace.requestReceivedByLocalDaemon,
        end: trace.requestSentByLocalDaemon,
        latencyMilliseconds: trace.requestSentByLocalDaemon - trace.requestReceivedByLocalDaemon,
    };
    requestTraceSplits.push(splitLocalDaemonProcessRequest);

    const splitLocalDaemonRequestToKernel: RequestTraceSplit = {
        messageId: trace.messageId,
        messageType: trace.messageType,
        kernelId: trace.kernelId,
        splitName: 'Local Scheduler → Kernel',
        start: trace.requestSentByLocalDaemon,
        end: trace.requestReceivedByKernelReplica,
        latencyMilliseconds: trace.requestReceivedByKernelReplica - trace.requestSentByLocalDaemon,
    };
    requestTraceSplits.push(splitLocalDaemonRequestToKernel);

    if (trace.messageType === 'execute_request' || trace.messageType === 'yield_request') {
        const splitKernelPreprocessRequest: RequestTraceSplit = {
            messageId: trace.messageId,
            messageType: trace.messageType,
            kernelId: trace.kernelId,
            splitName: 'Kernel Preprocessing Request',
            start: trace.requestReceivedByKernelReplica,
            end: trace.electionCreationTime,
            latencyMilliseconds: trace.electionCreationTime - trace.requestReceivedByKernelReplica,
        };
        requestTraceSplits.push(splitKernelPreprocessRequest);

        const splitKernelCreateElection: RequestTraceSplit = {
            messageId: trace.messageId,
            messageType: trace.messageType,
            kernelId: trace.kernelId,
            splitName: 'Kernel Creating Election',
            start: trace.electionCreationTime,
            end: trace.electionProposalPhaseStartTime,
            latencyMilliseconds: trace.electionProposalPhaseStartTime - trace.electionCreationTime,
        };
        requestTraceSplits.push(splitKernelCreateElection);

        const splitKernelProposalVotingPhase: RequestTraceSplit = {
            messageId: trace.messageId,
            messageType: trace.messageType,
            kernelId: trace.kernelId,
            splitName: 'Kernel Election Proposal/Vote Phase',
            start: trace.electionProposalPhaseStartTime,
            end: trace.electionExecutionPhaseStartTime,
            latencyMilliseconds: trace.electionExecutionPhaseStartTime - trace.electionProposalPhaseStartTime,
        };
        requestTraceSplits.push(splitKernelProposalVotingPhase);

        const splitKernelExecuteCodePhase: RequestTraceSplit = {
            messageId: trace.messageId,
            messageType: trace.messageType,
            kernelId: trace.kernelId,
            splitName: 'Kernel Executing Code',
            start: trace.executionStartUnixMillis,
            end: trace.executionEndUnixMillis,
            latencyMilliseconds: trace.executionEndUnixMillis - trace.executionStartUnixMillis,
        };
        requestTraceSplits.push(splitKernelExecuteCodePhase);

        const splitKernelPostprocessRequest: RequestTraceSplit = {
            messageId: trace.messageId,
            messageType: trace.messageType,
            kernelId: trace.kernelId,
            splitName: 'Kernel Postprocessing Request',
            start: trace.executionEndUnixMillis,
            end: trace.replySentByKernelReplica,
            latencyMilliseconds: trace.replySentByKernelReplica - trace.executionEndUnixMillis,
        };
        requestTraceSplits.push(splitKernelPostprocessRequest);
    } else {
        const splitKernelProcessRequest: RequestTraceSplit = {
            messageId: trace.messageId,
            messageType: trace.messageType,
            kernelId: trace.kernelId,
            splitName: 'Kernel Processing Request',
            start: trace.requestReceivedByKernelReplica,
            end: trace.replySentByKernelReplica,
            latencyMilliseconds: trace.replySentByKernelReplica - trace.requestReceivedByKernelReplica,
        };
        requestTraceSplits.push(splitKernelProcessRequest);
    }

    const splitKernelReplyToLocalDaemon: RequestTraceSplit = {
        messageId: trace.messageId,
        messageType: trace.messageType,
        kernelId: trace.kernelId,
        splitName: 'Kernel → Local Scheduler',
        start: trace.replySentByKernelReplica,
        end: trace.replyReceivedByLocalDaemon,
        latencyMilliseconds: trace.replyReceivedByLocalDaemon - trace.replySentByKernelReplica,
    };
    requestTraceSplits.push(splitKernelReplyToLocalDaemon);

    const splitLocalDaemonProcessReply: RequestTraceSplit = {
        messageId: trace.messageId,
        messageType: trace.messageType,
        kernelId: trace.kernelId,
        splitName: 'Local Scheduler Processing Reply',
        start: trace.replyReceivedByLocalDaemon,
        end: trace.replySentByLocalDaemon,
        latencyMilliseconds: trace.replySentByLocalDaemon - trace.replyReceivedByLocalDaemon,
    };
    requestTraceSplits.push(splitLocalDaemonProcessReply);

    const splitLocalDaemonReplyToGateway: RequestTraceSplit = {
        messageId: trace.messageId,
        messageType: trace.messageType,
        kernelId: trace.kernelId,
        splitName: 'Local Scheduler → Global Scheduler',
        start: trace.replySentByLocalDaemon,
        end: trace.replyReceivedByGateway,
        latencyMilliseconds: trace.replyReceivedByGateway - trace.replySentByLocalDaemon,
    };
    requestTraceSplits.push(splitLocalDaemonReplyToGateway);

    const splitGatewayProcessReply: RequestTraceSplit = {
        messageId: trace.messageId,
        messageType: trace.messageType,
        kernelId: trace.kernelId,
        splitName: 'Global Scheduler Processing Reply',
        start: trace.replyReceivedByGateway,
        end: trace.replySentByGateway,
        latencyMilliseconds: trace.replySentByGateway - trace.replyReceivedByGateway,
    };
    requestTraceSplits.push(splitGatewayProcessReply);

    const splitGatewayReplyToClient: RequestTraceSplit = {
        messageId: trace.messageId,
        messageType: trace.messageType,
        kernelId: trace.kernelId,
        splitName: 'Global Scheduler → Client',
        start: trace.replySentByGateway,
        end: replyReceived,
        latencyMilliseconds: replyReceived - trace.replySentByGateway,
    };
    requestTraceSplits.push(splitGatewayReplyToClient);

    return requestTraceSplits;
    // return [
    //     splitClientToGateway,
    //     splitGatewayProcessRequest,
    //     splitGatewayRequestToLocalDaemon,
    //     splitLocalDaemonProcessRequest,
    //     splitLocalDaemonRequestToKernel,
    //     splitKernelProcessRequest,
    //     splitKernelReplyToLocalDaemon,
    //     splitLocalDaemonProcessReply,
    //     splitLocalDaemonReplyToGateway,
    //     splitGatewayProcessReply,
    //     splitGatewayReplyToClient,
    // ];
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
