package router

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

const (
	unkonwnEventType = "UNKNOWN"

	messageHandlerProcessingTimeMetricName = "message_handler_processing_time_seconds"
	messageHandlerSuccessCountMetricName   = "message_handler_success_count"
	messageHandlerErrorCountMetricName     = "message_handler_error_count"
)

func HandlerMetrics(metricMeter metric.Meter, prefix string, log *slog.Logger) (func(message.HandlerFunc) message.HandlerFunc, error) {
	messageProcessingTime, err := metricMeter.Float64Histogram(
		fmt.Sprintf("%s.%s", prefix, messageHandlerProcessingTimeMetricName),
		metric.WithDescription("Time spent by the handler processing a message"),
	)
	if err != nil {
		return nil, err
	}

	messageProcessed, err := metricMeter.Int64Counter(
		fmt.Sprintf("%s.%s", prefix, messageHandlerSuccessCountMetricName),
		metric.WithDescription("Number of messages processed by the handler"),
	)
	if err != nil {
		return nil, err
	}

	messageProcessingError, err := metricMeter.Int64Counter(
		fmt.Sprintf("%s.%s", prefix, messageHandlerErrorCountMetricName),
		metric.WithDescription("Number of messages that failed to process by the handler"),
	)
	if err != nil {
		return nil, err
	}

	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			start := time.Now()

			attrSet := metricAttributesFromMessage(msg)

			resMsg, err := h(msg)
			if err != nil {
				// This should be warning, as it might happen that the kafka message is produced later than the
				// database commit happens.
				log.Warn("Message handler failed, will retry later", "error", err, "message_metadata", msg.Metadata, "message_payload", string(msg.Payload))
				messageProcessingError.Add(msg.Context(), 1, metric.WithAttributeSet(
					attrSet,
				))
				return resMsg, err
			}

			messageProcessingTime.Record(msg.Context(), time.Since(start).Seconds(), metric.WithAttributeSet(
				attrSet,
			))
			messageProcessed.Add(msg.Context(), 1, metric.WithAttributeSet(
				attrSet,
			))
			return resMsg, nil
		}
	}, nil
}

const (
	messageProcessingErrorCountMetricName   = "message_processing_error_count"
	messageProcessingSuccessCountMetricName = "message_processing_success_count"
)

func DLQMetrics(metricMeter metric.Meter, prefix string, log *slog.Logger) (func(message.HandlerFunc) message.HandlerFunc, error) {
	messageProcessingErrorCount, err := metricMeter.Int64Counter(
		fmt.Sprintf("%s.%s", prefix, messageProcessingErrorCountMetricName),
		metric.WithDescription("Number of messages that failed to process"),
	)
	if err != nil {
		return nil, err
	}

	messageProcessingSuccessCount, err := metricMeter.Int64Counter(
		fmt.Sprintf("%s.%s", prefix, messageProcessingSuccessCountMetricName),
		metric.WithDescription("Number of messages that were successfully processed"),
	)
	if err != nil {
		return nil, err
	}

	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			attrSet := metricAttributesFromMessage(msg)

			resMsg, err := h(msg)
			if err != nil {
				log.Error("Failed to process message, message is going to DLQ", "error", err, "message_metadata", msg.Metadata, "message_payload", string(msg.Payload))
				messageProcessingErrorCount.Add(msg.Context(), 1, metric.WithAttributeSet(
					attrSet,
				))
				return resMsg, err
			}

			messageProcessingSuccessCount.Add(msg.Context(), 1, metric.WithAttributeSet(
				attrSet,
			))

			return resMsg, nil
		}
	}, nil
}

func metricAttributesFromMessage(msg *message.Message) attribute.Set {
	ce_type := msg.Metadata.Get(marshaler.CloudEventsHeaderType)
	if ce_type == "" {
		ce_type = unkonwnEventType
	}
	attrSet := attribute.NewSet(attribute.String("ce_type", ce_type))

	return attrSet
}
