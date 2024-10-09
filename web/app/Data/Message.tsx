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

/**
 * Sent as a response to pinging a kernel.
 */
export interface PongResponse {
  id: string;
  success: boolean;
  msg: string;
  requestTraces: RequestTrace[];
}
