package consumer

import "time"

const (
	defaultDLQDeliveryTimeout     = 10 * time.Second
	defaultStopWorkersTimeout     = 30 * time.Second
	defaultMsgChanBufferSize      = 100
	defaultPollTimeout            = 100 * time.Millisecond
	defaultPartitionWorkerMapSize = 256
)
