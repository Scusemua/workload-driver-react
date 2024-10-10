package proto

import "go.uber.org/zap/zapcore"

func (x *QueryMessageResponse) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	err := encoder.AddObject("request_trace", x.RequestTrace)
	if err != nil {
		return err
	}

	return nil
}

func (x *QueryMessageRequest) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddString("message_id", x.MessageId)
	encoder.AddString("message_type", x.MessageType)
	encoder.AddString("kernel_id", x.KernelId)

	return nil
}
