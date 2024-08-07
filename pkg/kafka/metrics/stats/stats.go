// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stats

// https://github.com/confluentinc/librdkafka/blob/v2.4.0/STATISTICS.md
type Stats struct {
	// Handle instance name
	Name string `json:"name"`
	// The configured (or default) client.id
	ClientID string `json:"client_id"`
	// Instance type (producer or consumer)
	Type string `json:"type"`
	// Time since this client instance was created (microseconds)
	Age int64 `json:"age"`
	// Number of ops (callbacks, events, etc) waiting in queue for application to serve with rd_kafka_poll()
	ReplyQueue int64 `json:"replyq"`
	// Current number of messages in producer queues
	MessageCount int64 `json:"msg_cnt"`
	// Current total size of messages in producer queues
	MessageSize int64 `json:"msg_size"`
	// Threshold: maximum number of messages allowed on the producer queues
	MessageMax int64 `json:"msg_max"`
	// Total number of requests sent to Kafka brokers
	RequestsSent int64 `json:"tx"`
	// Total number of bytes transmitted to Kafka brokers
	RequestsBytesSent int64 `json:"tx_bytes"`
	// Total number of responses received from Kafka brokers
	RequestsReceived int64 `json:"rx"`
	// Total number of bytes received from Kafka brokers
	RequestsBytesReceived int64 `json:"rx_bytes"`
	// Total number of messages transmitted (produced) to Kafka brokers
	MessagesProduced int64 `json:"txmsgs"`
	// Total number of message bytes (including framing, such as per-Message framing and MessageSet/batch framing) transmitted to Kafka brokers
	MessagesBytesProduced int64 `json:"txmsg_bytes"`
	// Total number of messages consumed, not including ignored messages (due to offset, etc), from Kafka brokers.
	MessagesConsumed int64 `json:"rxmsgs"`
	// Total number of message bytes (including framing) received from Kafka brokers
	MessagesBytesConsumed int64 `json:"rxmsg_bytes"`
	// Number of topics in the metadata cache
	TopicsInMetadataCache int64                  `json:"metadata_cache_cnt"`
	Brokers               map[string]BrokerStats `json:"brokers"`
	Topics                map[string]TopicStats  `json:"topics"`
	ConsumerGroup         ConsumerGroupStats     `json:"cgrp"`
}

// WindowStats stores rolling window statistics. The values are in microseconds unless otherwise stated.
type WindowStats struct {
	// Smallest value
	Min int64 `json:"min"`
	// Largest value
	Max int64 `json:"max"`
	// Average value
	Avg int64 `json:"avg"`
	// Sum of values
	Sum int64 `json:"sum"`
	// Number of values sampled
	Count int64 `json:"count"`
	// Standard deviation (based on histogram)
	StdDev int64 `json:"stddev"`
	// Memory size of Hdr Histogram
	HdrSize int64 `json:"hdrsize"`
	// 50th percentile
	P50 int64 `json:"p50"`
	// 75th percentile
	P75 int64 `json:"p75"`
	// 90th percentile
	P90 int64 `json:"p90"`
	// 95th percentile
	P95 int64 `json:"p95"`
	// 99th percentile
	P99 int64 `json:"p99"`
	// 99.99th percentile
	P9999 int64 `json:"p99_99"`
}
