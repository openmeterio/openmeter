package consumer

import (
	"log/slog"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// safeCloseChannel safely closes a channel, recovering from panic if channel is already closed.
func safeCloseChannel[T any](ch chan T, logger *slog.Logger) {
	defer func() {
		if r := recover(); r != nil {
			// Channel was already closed, which is fine
			logger.Debug("channel was already closed", "panic", r)
		}
	}()

	// Close the channel (will panic if already closed, but we recover from it)
	close(ch)
}

// prettyPartitions formats partitions for logging.
func prettyPartitions(partitions []kafka.TopicPartition) []string {
	result := make([]string, len(partitions))
	for i, p := range partitions {
		result[i] = p.String()
	}
	return result
}
