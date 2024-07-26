package flushhandler

import (
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

type metrics struct {
	eventsReceived      metric.Int64Counter
	eventsProcessed     metric.Int64Counter
	eventsFailed        metric.Int64Counter
	eventProcessingTime metric.Float64Histogram
	eventChannelFull    metric.Int64Counter
}

func newMetrics(handlerName string, meter metric.Meter) (*metrics, error) {
	var err error

	r := &metrics{}

	if r.eventsReceived, err = meter.Int64Counter(fmt.Sprintf("sink.flush_handler.%s.events_received", handlerName)); err != nil {
		return nil, err
	}

	if r.eventsProcessed, err = meter.Int64Counter(fmt.Sprintf("sink.flush_handler.%s.events_processed", handlerName)); err != nil {
		return nil, err
	}

	if r.eventsFailed, err = meter.Int64Counter(fmt.Sprintf("sink.flush_handler.%s.events_failed", handlerName)); err != nil {
		return nil, err
	}

	if r.eventChannelFull, err = meter.Int64Counter(fmt.Sprintf("sink.flush_handler.%s.event_channel_full", handlerName)); err != nil {
		return nil, err
	}

	if r.eventProcessingTime, err = meter.Float64Histogram(fmt.Sprintf("sink.flush_handler.%s.event_processing_time", handlerName)); err != nil {
		return nil, err
	}

	return r, nil
}
