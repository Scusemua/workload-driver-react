package proto

import "go.uber.org/zap/zapcore"

func (x *RequestTrace) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddString("message_id", x.MessageId)
	encoder.AddString("message_type", x.MessageType)
	encoder.AddString("kernel_id", x.KernelId)
	//err := encoder.AddArray("traces", TracesArr(x.Traces))
	//if err != nil {
	//	return err
	//}
	encoder.AddInt64("gateway_received_request", x.GetRequestReceivedByGateway())
	encoder.AddInt64("gateway_forwarded_request", x.GetRequestSentByGateway())
	encoder.AddInt64("local_daemon_received_request", x.GetRequestReceivedByLocalDaemon())
	encoder.AddInt64("local_daemon_sent_request", x.GetRequestSentByLocalDaemon())
	encoder.AddInt64("kernel_replica_received_request", x.GetRequestReceivedByKernelReplica())
	encoder.AddInt64("kernel_replica_sent_reply", x.GetReplySentByKernelReplica())
	encoder.AddInt64("local_daemon_received_reply", x.GetReplyReceivedByLocalDaemon())
	encoder.AddInt64("local_daemon_sent_reply", x.GetReplySentByLocalDaemon())
	encoder.AddInt64("gateway_received_reply", x.GetReplyReceivedByGateway())
	encoder.AddInt64("gateway_forwarded_reply", x.GetReplySentByGateway())

	return nil
}

type RequestTraceArr []*RequestTrace

func (r RequestTraceArr) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	for _, v := range r {
		err := encoder.AppendObject(v)
		if err != nil {
			return err
		}
	}

	return nil
}

func (x *Trace) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddString("id", x.Id)
	encoder.AddString("name", x.Name)
	encoder.AddInt64("start_timestamp_unix_milliseconds", x.StartTimeUnixMicro)
	encoder.AddInt64("end_timestamp_unix_milliseconds", x.EndTimeUnixMicro)
	encoder.AddTime("start_timestamp", x.StartTimestamp.AsTime())
	encoder.AddTime("end_timestamp", x.EndTimestamp.AsTime())
	encoder.AddInt64("duration_microseconds", x.DurationMicroseconds)

	return nil
}

type TracesArr []*Trace

func (r TracesArr) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	for _, v := range r {
		err := encoder.AppendObject(v)
		if err != nil {
			return err
		}
	}

	return nil
}
