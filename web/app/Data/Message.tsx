export interface QueryMessageResponse {
  messageId: string;
  messageType: string;
  kernelId: string;
  gatewayReceivedRequest: number;
  gatewayForwardedRequest: number;
  gatewayReceivedReply: number;
  gatewayForwardedReply: number;
}
