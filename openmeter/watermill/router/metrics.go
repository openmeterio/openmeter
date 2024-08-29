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

	messageHandlerProcessingTimeMetricName = "message_handler_processing_time_ms"
	messageHandlerMessageCountMetricName   = "message_handler_message_count"
)

var (
	meterAttributeStatusFailed  = attribute.String("status", "failed")
	meterAttributeStatusSuccess = attribute.String("status", "success")
)

func HandlerMetrics(metricMeter metric.Meter, prefix string, log *slog.Logger) (func(message.HandlerFunc) message.HandlerFunc, error) {
	meterMessageProcessingTime, err := metricMeter.Int64Histogram(
		fmt.Sprintf("%s.%s", prefix, messageHandlerProcessingTimeMetricName),
		metric.WithDescription("Time spent by the handler processing a message"),
	)
	if err != nil {
		return nil, err
	}

	meterMessageCount, err := metricMeter.Int64Counter(
		fmt.Sprintf("%s.%s", prefix, messageHandlerMessageCountMetricName),
		metric.WithDescription("Number of messages processed by the handler"),
	)
	if err != nil {
		return nil, err
	}

	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			start := time.Now()

			meterAttributeType := metricAttributeTypeFromMessage(msg)

			resMsg, err := h(msg)
			if err != nil {
				// This should be warning, as it might happen that the kafka message is produced later than the
				// database commit happens.
				log.Warn("Message handler failed, will retry later", "error", err, "message_metadata", msg.Metadata, "message_payload", string(msg.Payload))
				meterMessageCount.Add(msg.Context(), 1, metric.WithAttributes(
					meterAttributeType,
					meterAttributeStatusFailed,
				))

				meterMessageProcessingTime.Record(msg.Context(), time.Since(start).Milliseconds(), metric.WithAttributes(
					meterAttributeType,
					meterAttributeStatusFailed,
				))
				return resMsg, err
			}

			meterMessageProcessingTime.Record(msg.Context(), time.Since(start).Milliseconds(), metric.WithAttributes(
				meterAttributeType,
				meterAttributeStatusSuccess,
			))
			meterMessageCount.Add(msg.Context(), 1, metric.WithAttributes(
				meterAttributeType,
				meterAttributeStatusSuccess,
			))
			return resMsg, nil
		}
	}, nil
}

const (
	messageProcessingCountMetricName = "message_processing_count"
	messageProcessingTimeMetricName  = "message_processing_time_ms"
)

func DLQMetrics(metricMeter metric.Meter, prefix string, log *slog.Logger) (func(message.HandlerFunc) message.HandlerFunc, error) {
	meterMessageProcessingCount, err := metricMeter.Int64Counter(
		fmt.Sprintf("%s.%s", prefix, messageProcessingCountMetricName),
		metric.WithDescription("Number of messages processed"),
	)
	if err != nil {
		return nil, err
	}

	meterMessageProcessingTime, err := metricMeter.Int64Histogram(
		fmt.Sprintf("%s.%s", prefix, messageProcessingTimeMetricName),
		metric.WithDescription("Time spent processing a message (including retries)"),
	)
	if err != nil {
		return nil, err
	}

	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			start := time.Now()

			meterAttributeCEType := metricAttributeTypeFromMessage(msg)

			resMsg, err := h(msg)
			if err != nil {
				log.Error("Failed to process message, message is going to DLQ", "error", err, "message_metadata", msg.Metadata, "message_payload", string(msg.Payload))

				meterMessageProcessingCount.Add(msg.Context(), 1, metric.WithAttributes(
					meterAttributeCEType,
					meterAttributeStatusFailed,
				))
				meterMessageProcessingTime.Record(msg.Context(), time.Since(start).Milliseconds(), metric.WithAttributes(
					meterAttributeCEType,
					meterAttributeStatusFailed,
				))

				return resMsg, err
			}

			meterMessageProcessingCount.Add(msg.Context(), 1, metric.WithAttributes(
				meterAttributeCEType,
				meterAttributeStatusSuccess,
			))
			meterMessageProcessingTime.Record(msg.Context(), time.Since(start).Milliseconds(), metric.WithAttributes(
				meterAttributeCEType,
				meterAttributeStatusSuccess,
			))

			return resMsg, nil
		}
	}, nil
}

func metricAttributeTypeFromMessage(msg *message.Message) attribute.KeyValue {
	ce_type := msg.Metadata.Get(marshaler.CloudEventsHeaderType)
	if ce_type == "" {
		ce_type = unkonwnEventType
	}

	return attribute.String("ce_type", ce_type)
}
