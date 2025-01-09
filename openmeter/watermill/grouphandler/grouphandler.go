package grouphandler

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/samber/lo"
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

type NoPublishingHandler struct {
	meters         *meters
	marshaler      cqrs.CommandEventMarshaler
	typeHandlerMap map[string][]cqrs.GroupEventHandler
	handlerLock    sync.RWMutex
}

func (h *NoPublishingHandler) Handle(msg *message.Message) error {
	eventName := h.marshaler.NameFromMessage(msg)

	meterAttributeCEType := attribute.String("ce_type", eventName)

	h.handlerLock.Lock()
	defer h.handlerLock.Unlock()

	groupHandler, ok := h.typeHandlerMap[eventName]
	if !ok || len(groupHandler) == 0 {
		h.meters.handlerMessageCount.Add(msg.Context(), 1, metric.WithAttributes(
			meterAttributeCEType,
			meterAttributeStatusIgnored,
		))
		return nil
	}

	event := groupHandler[0].NewEvent()

	if err := h.marshaler.Unmarshal(msg, event); err != nil {
		return err
	}

	startedAt := time.Now()
	err := errors.Join(lo.Map(groupHandler, func(handler GroupEventHandler, _ int) error {
		return handler.Handle(msg.Context(), event)
	})...)
	if err != nil {
		h.meters.handlerMessageCount.Add(msg.Context(), 1, metric.WithAttributes(
			meterAttributeCEType,
			meterAttributeStatusFailed,
		))
		h.meters.handlerProcessingTime.Record(msg.Context(), time.Since(startedAt).Milliseconds(), metric.WithAttributes(
			meterAttributeCEType,
			meterAttributeStatusFailed,
		))

		return err
	}

	h.meters.handlerProcessingTime.Record(msg.Context(), time.Since(startedAt).Milliseconds(), metric.WithAttributes(
		meterAttributeCEType,
		meterAttributeStatusSuccess,
	))
	h.meters.handlerMessageCount.Add(msg.Context(), 1, metric.WithAttributes(
		meterAttributeCEType,
		meterAttributeStatusSuccess,
	))

	return nil
}

func (h *NoPublishingHandler) AddHandler(handler GroupEventHandler) {
	h.handlerLock.Lock()
	defer h.handlerLock.Unlock()

	event := handler.NewEvent()
	h.typeHandlerMap[h.marshaler.Name(event)] = append(h.typeHandlerMap[h.marshaler.Name(event)], handler)
}

// NewNoPublishingHandler creates a NoPublishHandlerFunc that will handle events with the provided GroupEventHandlers.
func NewNoPublishingHandler(marshaler cqrs.CommandEventMarshaler, metricMeter metric.Meter, groupHandlers ...GroupEventHandler) (*NoPublishingHandler, error) {
	meters, err := getMeters(metricMeter)
	if err != nil {
		return nil, err
	}

	typeHandlerMap := make(map[string][]cqrs.GroupEventHandler)
	for _, groupHandler := range groupHandlers {
		event := groupHandler.NewEvent()
		typeHandlerMap[marshaler.Name(event)] = append(typeHandlerMap[marshaler.Name(event)], groupHandler)
	}

	return &NoPublishingHandler{
		marshaler:      marshaler,
		meters:         meters,
		typeHandlerMap: typeHandlerMap,
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
