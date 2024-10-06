package proto

import "go.uber.org/zap/zapcore"

func (x *QueryMessageResponse) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddString("message_id", x.MessageId)
	encoder.AddString("message_type", x.MessageType)
	encoder.AddString("kernel_id", x.KernelId)
	encoder.AddInt64("gateway_received_request", x.GatewayReceivedRequest)
	encoder.AddInt64("gateway_forwarded_request", x.GatewayForwardedRequest)
	encoder.AddInt64("gateway_received_reply", x.GatewayReceivedReply)
	encoder.AddInt64("gateway_forwarded_reply", x.GatewayForwardedReply)

	return nil
}

func (x *QueryMessageRequest) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddString("message_id", x.MessageId)
	encoder.AddString("message_type", x.MessageType)
	encoder.AddString("kernel_id", x.KernelId)

	return nil
}
