package grouphandler

import (
	"context"
	"time"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	meterNameHandlerMessageCount   = "watermill.grouphandler.message_count"
	meterNameHandlerProcessingTime = "watermill.grouphandler.processing_time_ms"
)

var (
	meterAttributeStatusIgnored = attribute.String("status", "ignored")
	meterAttributeStatusFailed  = attribute.String("status", "failed")
	meterAttributeStatusSuccess = attribute.String("status", "success")
)

type GroupEventHandler = cqrs.GroupEventHandler

func NewGroupEventHandler[T any](handleFunc func(ctx context.Context, event *T) error) GroupEventHandler {
	return cqrs.NewGroupEventHandler(handleFunc)
}

// NewNoPublishingHandler creates a NoPublishHandlerFunc that will handle events with the provided GroupEventHandlers.
func NewNoPublishingHandler(marshaler cqrs.CommandEventMarshaler, metricMeter metric.Meter, groupHandlers ...GroupEventHandler) (message.NoPublishHandlerFunc, error) {
	meters, err := getMeters(metricMeter)
	if err != nil {
		return nil, err
	}

	typeHandlerMap := make(map[string]cqrs.GroupEventHandler)
	for _, groupHandler := range groupHandlers {
		event := groupHandler.NewEvent()
		typeHandlerMap[marshaler.Name(event)] = groupHandler
	}

	return func(msg *message.Message) error {
		eventName := marshaler.NameFromMessage(msg)

		meterAttributeCEType := attribute.String("ce_type", eventName)

		groupHandler, ok := typeHandlerMap[eventName]
		if !ok {
			meters.handlerMessageCount.Add(msg.Context(), 1, metric.WithAttributes(
				meterAttributeCEType,
				meterAttributeStatusIgnored,
			))
			return nil
		}

		event := groupHandler.NewEvent()

		if err := marshaler.Unmarshal(msg, event); err != nil {
			return err
		}

		startedAt := time.Now()
		err := groupHandler.Handle(msg.Context(), event)
		if err != nil {
			meters.handlerMessageCount.Add(msg.Context(), 1, metric.WithAttributes(
				meterAttributeCEType,
				meterAttributeStatusFailed,
			))
			meters.handlerProcessingTime.Record(msg.Context(), time.Since(startedAt).Milliseconds(), metric.WithAttributes(
				meterAttributeCEType,
				meterAttributeStatusFailed,
			))

			return err
		}

		meters.handlerProcessingTime.Record(msg.Context(), time.Since(startedAt).Milliseconds(), metric.WithAttributes(
			meterAttributeCEType,
			meterAttributeStatusSuccess,
		))
		meters.handlerMessageCount.Add(msg.Context(), 1, metric.WithAttributes(
			meterAttributeCEType,
			meterAttributeStatusSuccess,
		))

		return nil
	}, nil
}

type meters struct {
	handlerMessageCount   metric.Int64Counter
	handlerProcessingTime metric.Int64Histogram
}

func getMeters(meter metric.Meter) (*meters, error) {
	handlerMessageCount, err := meter.Int64Counter(meterNameHandlerMessageCount)
	if err != nil {
		return nil, err
	}

	handlerProcessingTime, err := meter.Int64Histogram(meterNameHandlerProcessingTime)
	if err != nil {
		return nil, err
	}

	return &meters{
		handlerMessageCount:   handlerMessageCount,
		handlerProcessingTime: handlerProcessingTime,
	}, nil
}
