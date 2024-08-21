package router

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/metric"
)

const (
	messageProcessingTimeMetricName = "message_processing_time"
	messageProcessedCount           = "message_processed_count"
	messageProcessingErrorCount     = "message_processing_error_count"
)

func Metrics(metricMeter metric.Meter, prefix string, log *slog.Logger) (func(message.HandlerFunc) message.HandlerFunc, error) {
	messageProcessingTime, err := metricMeter.Float64Histogram(
		fmt.Sprintf("%s.%s", prefix, messageProcessingTimeMetricName),
		metric.WithDescription("Time spent processing a message"),
	)
	if err != nil {
		return nil, err
	}

	messageProcessed, err := metricMeter.Int64Counter(
		fmt.Sprintf("%s.%s", prefix, messageProcessedCount),
		metric.WithDescription("Number of messages processed"),
	)
	if err != nil {
		return nil, err
	}

	messageProcessingError, err := metricMeter.Int64Counter(
		fmt.Sprintf("%s.%s", prefix, messageProcessingErrorCount),
		metric.WithDescription("Number of messages that failed to process"),
	)
	if err != nil {
		return nil, err
	}

	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			start := time.Now()

			resMsg, err := h(msg)
			if err != nil {
				log.Error("Failed to process message", "error", err, "message_metadata", msg.Metadata, "message_payload", string(msg.Payload))
				messageProcessingError.Add(msg.Context(), 1)
				return resMsg, err
			}

			messageProcessingTime.Record(msg.Context(), time.Since(start).Seconds())
			messageProcessed.Add(msg.Context(), 1)
			return resMsg, nil
		}
	}, nil
}
