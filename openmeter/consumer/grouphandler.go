package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

const (
	meterNameHandlerMessageCount   = "consumer.grouphandler.message_count"
	meterNameHandlerProcessingTime = "consumer.grouphandler.processing_time_ms"
)

var (
	meterAttributeStatusIgnored = attribute.String("status", "ignored")
	meterAttributeStatusFailed  = attribute.String("status", "failed")
	meterAttributeStatusSuccess = attribute.String("status", "success")
)

// GroupEventHandler is an alias for cqrs.GroupEventHandler to maintain compatibility
// with existing watermill-based handlers.
type GroupEventHandler = cqrs.GroupEventHandler

// NewGroupEventHandler creates a new GroupEventHandler from a function.
// This is a wrapper around cqrs.NewGroupEventHandler for convenience.
func NewGroupEventHandler[T any](handleFunc func(ctx context.Context, event *T) error) GroupEventHandler {
	return cqrs.NewGroupEventHandler(handleFunc)
}

// KafkaMessageHandler handles Kafka messages and routes them to appropriate event handlers.
type KafkaMessageHandler struct {
	meters         *handlerMeters
	marshaler      marshaler.Marshaler
	typeHandlerMap map[string][]GroupEventHandler
	mux            sync.RWMutex
}

// Handle processes a Kafka message and routes it to the appropriate event handlers.
func (h *KafkaMessageHandler) Handle(ctx context.Context, kafkaMsg *kafka.Message) error {
	// Extract event name from Kafka message
	eventName := h.ExtractEventName(kafkaMsg)

	meterAttributeCEType := attribute.String("ce_type", eventName)
	if eventName == "" {
		eventName = marshaler.UnknownEventName
		meterAttributeCEType = attribute.String("ce_type", marshaler.UnknownEventName)
	}

	h.mux.RLock()
	groupHandler, ok := h.typeHandlerMap[eventName]
	h.mux.RUnlock()

	if !ok || len(groupHandler) == 0 {
		h.meters.handlerMessageCount.Add(ctx, 1, metric.WithAttributes(
			meterAttributeCEType,
			meterAttributeStatusIgnored,
		))
		return nil
	}

	// Create event instance
	event := groupHandler[0].NewEvent()

	// Unmarshal Kafka message to event
	if err := h.unmarshalKafkaMessage(kafkaMsg, event); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Process through all handlers for this event type
	startedAt := time.Now()
	err := errors.Join(lo.Map(groupHandler, func(handler GroupEventHandler, _ int) error {
		return handler.Handle(ctx, event)
	})...)

	if err != nil {
		h.meters.handlerMessageCount.Add(ctx, 1, metric.WithAttributes(
			meterAttributeCEType,
			meterAttributeStatusFailed,
		))
		h.meters.handlerProcessingTime.Record(ctx, time.Since(startedAt).Milliseconds(), metric.WithAttributes(
			meterAttributeCEType,
			meterAttributeStatusFailed,
		))

		return err
	}

	h.meters.handlerProcessingTime.Record(ctx, time.Since(startedAt).Milliseconds(), metric.WithAttributes(
		meterAttributeCEType,
		meterAttributeStatusSuccess,
	))
	h.meters.handlerMessageCount.Add(ctx, 1, metric.WithAttributes(
		meterAttributeCEType,
		meterAttributeStatusSuccess,
	))

	return nil
}

// AddHandler adds an event handler to the handler map.
func (h *KafkaMessageHandler) AddHandler(handler GroupEventHandler) {
	h.mux.Lock()
	defer h.mux.Unlock()

	event := handler.NewEvent()
	eventName := h.marshaler.Name(event)
	h.typeHandlerMap[eventName] = append(h.typeHandlerMap[eventName], handler)
}

// ExtractEventName extracts the CloudEvents type from a Kafka message.
// It first checks headers, then falls back to parsing the CloudEvent payload.
func (h *KafkaMessageHandler) ExtractEventName(kafkaMsg *kafka.Message) string {
	// First, check headers for CloudEvents type
	for _, header := range kafkaMsg.Headers {
		if header.Key == marshaler.CloudEventsHeaderType {
			return string(header.Value)
		}
	}

	// Fall back to parsing CloudEvent from payload
	var ce cloudevents.Event
	if err := json.Unmarshal(kafkaMsg.Value, &ce); err == nil {
		if ce.Type() != "" {
			return ce.Type()
		}
	}

	return ""
}

// unmarshalKafkaMessage unmarshals a Kafka message to an event using the marshaler.
// It converts the Kafka message to a watermill message format that the marshaler expects.
func (h *KafkaMessageHandler) unmarshalKafkaMessage(kafkaMsg *kafka.Message, event interface{}) error {
	// Create a dummy watermill message from the Kafka message
	// The marshaler expects a watermill message with the CloudEvent JSON as payload
	watermillMsg := message.NewMessage("", kafkaMsg.Value)
	
	// Copy headers from Kafka message to watermill message metadata
	for _, header := range kafkaMsg.Headers {
		watermillMsg.Metadata.Set(header.Key, string(header.Value))
	}

	// Use the marshaler's Unmarshal method which handles CloudEvent unmarshaling and validation
	return h.marshaler.Unmarshal(watermillMsg, event)
}

// NewKafkaMessageHandler creates a new KafkaMessageHandler with the provided marshaler and handlers.
func NewKafkaMessageHandler(marshaler marshaler.Marshaler, metricMeter metric.Meter, handlers ...GroupEventHandler) (*KafkaMessageHandler, error) {
	meters, err := getHandlerMeters(metricMeter)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics: %w", err)
	}

	typeHandlerMap := make(map[string][]GroupEventHandler)
	for _, handler := range handlers {
		event := handler.NewEvent()
		eventName := marshaler.Name(event)
		typeHandlerMap[eventName] = append(typeHandlerMap[eventName], handler)
	}

	return &KafkaMessageHandler{
		marshaler:      marshaler,
		meters:         meters,
		typeHandlerMap: typeHandlerMap,
	}, nil
}

type handlerMeters struct {
	handlerMessageCount   metric.Int64Counter
	handlerProcessingTime metric.Int64Histogram
}

func getHandlerMeters(meter metric.Meter) (*handlerMeters, error) {
	handlerMessageCount, err := meter.Int64Counter(meterNameHandlerMessageCount)
	if err != nil {
		return nil, fmt.Errorf("failed to create message count metric: %w", err)
	}

	handlerProcessingTime, err := meter.Int64Histogram(meterNameHandlerProcessingTime)
	if err != nil {
		return nil, fmt.Errorf("failed to create processing time metric: %w", err)
	}

	return &handlerMeters{
		handlerMessageCount:   handlerMessageCount,
		handlerProcessingTime: handlerProcessingTime,
	}, nil
}

