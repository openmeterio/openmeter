package metrics

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"sync"

	"github.com/rcrowley/go-metrics"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

type TransformedMetric struct {
	Name       string
	Attributes attribute.Set
	Drop       bool
}

type (
	TransformMetricsNameToOtel func(string) TransformedMetric
	ErrorHandler               func(error)
)

func LoggingErrorHandler(dest *slog.Logger) ErrorHandler {
	return func(err error) {
		dest.Error("error registering meter", "err", err)
	}
}

type NewRegistryOptions struct {
	MetricMeter     otelmetric.Meter
	NameTransformFn TransformMetricsNameToOtel
	ErrorHandler    ErrorHandler
}

func NewRegistry(opts NewRegistryOptions) (metrics.Registry, error) {
	if opts.MetricMeter == nil {
		return nil, errors.New("metric meter is required")
	}

	if opts.NameTransformFn == nil {
		opts.NameTransformFn = func(name string) TransformedMetric {
			return TransformedMetric{
				Name:       name,
				Attributes: attribute.NewSet(),
			}
		}
	}

	if opts.ErrorHandler == nil {
		opts.ErrorHandler = func(err error) {
			// no-op
		}
	}

	return &registry{
		Registry:        metrics.NewRegistry(),
		meticMeter:      opts.MetricMeter,
		nameTransformFn: opts.NameTransformFn,
		errorHandler:    opts.ErrorHandler,
	}, nil
}

type registry struct {
	metrics.Registry

	mu              sync.Mutex
	meticMeter      otelmetric.Meter
	nameTransformFn TransformMetricsNameToOtel
	errorHandler    ErrorHandler
}

func (r *registry) GetOrRegister(name string, def interface{}) interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()

	existingMeter := r.Registry.Get(name)
	if existingMeter != nil {
		return existingMeter
	}

	wrappedMeter, err := r.getWrappedMeter(name, def)
	if err != nil {
		r.errorHandler(err)
		return def
	}

	if err := r.Registry.Register(name, wrappedMeter); err != nil {
		r.errorHandler(err)
	}

	return wrappedMeter
}

func (r *registry) Register(name string, def interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	wrappedMeter, err := r.getWrappedMeter(name, def)
	if err != nil {
		return err
	}

	return r.Registry.Register(name, wrappedMeter)
}

func (r *registry) getWrappedMeter(name string, def interface{}) (interface{}, error) {
	// def might be a function that returns the actual metric, not an interface{}, so we need to have reflect here, to instantiate
	// the actual metric in such cases
	if v := reflect.ValueOf(def); v.Kind() == reflect.Func {
		def = v.Call(nil)[0].Interface()
	}

	transfomedMetric := r.nameTransformFn(name)

	if transfomedMetric.Drop {
		// If we are not interested in the metric, let's just return the original metric
		return def, nil
	}

	switch meterDef := def.(type) {
	case metrics.Meter:
		otelMeter, err := r.meticMeter.Int64Counter(transfomedMetric.Name)
		if err != nil {
			return def, err
		}

		return &wrappedMeter{Meter: meterDef, otelMeter: otelMeter, attributes: transfomedMetric.Attributes}, nil
	case metrics.Counter:
		otelMeter, err := r.meticMeter.Int64UpDownCounter(transfomedMetric.Name)
		if err != nil {
			return def, err
		}

		return &wrappedCounter{Counter: meterDef, otelMeter: otelMeter, attributes: transfomedMetric.Attributes}, nil
	case metrics.GaugeFloat64:
		otelMeter, err := r.meticMeter.Float64Gauge(transfomedMetric.Name)
		if err != nil {
			return def, err
		}

		return &wrappedGaugeFloat64{GaugeFloat64: meterDef, otelMeter: otelMeter, attributes: transfomedMetric.Attributes}, nil
	case metrics.Gauge:
		otelMeter, err := r.meticMeter.Int64Gauge(transfomedMetric.Name)
		if err != nil {
			return def, err
		}

		return &wrappedGauge{Gauge: meterDef, otelMeter: otelMeter, attributes: transfomedMetric.Attributes}, nil
	case metrics.Histogram:
		otelMeter, err := r.meticMeter.Int64Histogram(transfomedMetric.Name)
		if err != nil {
			r.errorHandler(err)
			break
		}

		return &wrappedHistogram{Histogram: meterDef, otelMeter: otelMeter, attributes: transfomedMetric.Attributes}, nil
	default:
		// this is just a safety net, as we should have handled all the cases above (based on the lib)
		r.errorHandler(fmt.Errorf("unsupported metric type (name=%s): %v", name, def))
	}

	return def, nil
}

type wrappedMeter struct {
	metrics.Meter
	otelMeter  otelmetric.Int64Counter
	attributes attribute.Set
}

func (m *wrappedMeter) Mark(n int64) {
	m.otelMeter.Add(context.Background(), n, otelmetric.WithAttributeSet(m.attributes))
	m.Meter.Mark(n)
}

type wrappedCounter struct {
	metrics.Counter
	otelMeter  otelmetric.Int64UpDownCounter
	attributes attribute.Set
}

func (m *wrappedCounter) Inc(n int64) {
	m.otelMeter.Add(context.Background(), n, otelmetric.WithAttributeSet(m.attributes))
	m.Counter.Inc(n)
}

func (m *wrappedCounter) Dec(n int64) {
	m.otelMeter.Add(context.Background(), -n, otelmetric.WithAttributeSet(m.attributes))
	m.Counter.Dec(n)
}

type wrappedGaugeFloat64 struct {
	metrics.GaugeFloat64
	otelMeter  otelmetric.Float64Gauge
	attributes attribute.Set
}

func (m *wrappedGaugeFloat64) Update(newVal float64) {
	m.otelMeter.Record(context.Background(), newVal, otelmetric.WithAttributeSet(m.attributes))
	m.GaugeFloat64.Update(newVal)
}

type wrappedGauge struct {
	metrics.Gauge
	otelMeter  otelmetric.Int64Gauge
	attributes attribute.Set
}

func (m *wrappedGauge) Update(newVal int64) {
	m.otelMeter.Record(context.Background(), newVal, otelmetric.WithAttributeSet(m.attributes))
	m.Gauge.Update(newVal)
}

type wrappedHistogram struct {
	metrics.Histogram
	otelMeter  otelmetric.Int64Histogram
	attributes attribute.Set
}

func (m *wrappedHistogram) Update(newVal int64) {
	m.otelMeter.Record(context.Background(), newVal, otelmetric.WithAttributeSet(m.attributes))
	m.Histogram.Update(newVal)
}
