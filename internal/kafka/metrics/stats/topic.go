package stats

type TopicStats struct {
	// Topic name
	Topic string `json:"topic"`
	// Age of client's topic object (milliseconds)
	Age int64 `json:"age"`
	// Age of metadata from broker for this topic (milliseconds)
	MetadataAge int64 `json:"metadata_age"`
	// Batch sizes in bytes
	BatchSize WindowStats `json:"batch_size"`
	// Batch message counts
	BatchCount WindowStats `json:"batch_count"`
	// Partitions
	Partitions map[string]Partition `json:"partitions"`
}

type Partition struct {
	// Partition Id (-1 for internal UA/UnAssigned partition)
	Partition int64 `json:"partition"`
	// The id of the broker that messages are currently being fetched from
	Broker int64 `json:"broker"`
	// Current leader broker id
	Leader int64 `json:"leader"`
	// Partition is explicitly desired by application
	Desired bool `json:"desired"`
	// Partition not seen in topic metadata from broker
	Unknown bool `json:"unknown"`
	// Number of messages waiting to be produced in first-level queue
	MessagesInQueue int64 `json:"msgq_cnt"`
	// Number of bytes in msgq_cnt
	MessageBytesInQueue int64 `json:"msgq_bytes"`
	// Number of messages ready to be produced in transmit queue
	MessagesReadyToTransmit int64 `json:"xmit_msgq_cnt"`
	// Number of bytes ready to be produced in transmit queue
	MessageBytesReadyToTransmit int64 `json:"xmit_msgq_bytes"`
	// Number of pre-fetched messages in fetch queue
	MessagesInFetchQueue int64 `json:"fetchq_cnt"`
	// Number of message bytes pre-fetched in fetch queue
	MessageBytesInFetchQueue int64 `json:"fetchq_size"`
	// Current/Last logical offset query
	QueryOffset int64 `json:"query_offset"`
	// Next offset to fetch
	NextOffset int64 `json:"next_offset"`
	// Offset of last message passed to application + 1
	AppOffset int64 `json:"app_offset"`
	// Offset to be committed
	StoredOffset int64 `json:"stored_offset"`
	// Last committed offset
	CommittedOffset int64 `json:"committed_offset"`
	// Last PARTITION_EOF signaled offset
	EOFOffset int64 `json:"eof_offset"`
	// Partition's low watermark offset on broker
	LowWatermarkOffset int64 `json:"lo_offset"`
	// Partition's high watermark offset on broker
	HighWatermarkOffset int64 `json:"hi_offset"`
	// Partition's last stable offset on broker, or same as HighWatermarkOffset is broker version is less than 0.11.0.0.
	LastStableOffsetOnBroker int64 `json:"ls_offset"`
	// Difference between (HighWatermarkOffset or LowWatermarkOffset) and CommittedOffset). HighWatermarkOffset is used when isolation.level=read_uncommitted, otherwise LastStableOffsetOnBroker.
	ConsumerLag int64 `json:"consumer_lag"`
	// Difference between (HighWatermarkOffset or LastStableOffsetOnBroker) and StoredOffset. See consumer_lag and StoredOffset.
	ConsumerLagStored int64 `json:"consumer_lag_stored"`
	// Total number of messages transmitted (produced)
	MessagesSent int64 `json:"txmsgs"`
	// Total number of bytes transmitted for MessagesSent
	MessageBytesSent int64 `json:"txbytes"`
	// Total number of messages consumed, not including ignored messages (due to offset, etc).
	MessagesReceived int64 `json:"rxmsgs"`
	// Total number of bytes received for MessageBytesReceived
	MessageBytesReceived int64 `json:"rxbytes"`
	// Total number of messages received (consumer, same as MessageBytesReceived), or total number of messages produced (possibly not yet transmitted) (producer).
	TotalNumOfMessages int64 `json:"msgs"`
	// Current number of messages in-flight to/from broker
	MessagesInflight int64 `json:"msginflight"`
}
