package metrics

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"sync"

	"github.com/rcrowley/go-metrics"
	otelmetric "go.opentelemetry.io/otel/metric"
)

type (
	TransformMetricsNameToOtel func(string) string
	ErrorHandler               func(error)
)

func LoggingErrorHandler(dest *slog.Logger) ErrorHandler {
	return func(err error) {
		dest.Error("error registering meter", "err", err)
	}
}

func MetricAddNamePrefix(prefix string) TransformMetricsNameToOtel {
	return func(name string) string {
		return prefix + name
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
		opts.NameTransformFn = func(name string) string {
			return name
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

	switch meterDef := def.(type) {
	case metrics.Meter:
		otelMeter, err := r.meticMeter.Int64Counter(r.nameTransformFn(name))
		if err != nil {
			return def, err
		}

		return &wrappedMeter{Meter: meterDef, otelMeter: otelMeter}, nil
	case metrics.Counter:
		otelMeter, err := r.meticMeter.Int64UpDownCounter(r.nameTransformFn(name))
		if err != nil {
			return def, err
		}

		return &wrappedCounter{Counter: meterDef, otelMeter: otelMeter}, nil
	case metrics.GaugeFloat64:
		otelMeter, err := r.meticMeter.Float64Gauge(r.nameTransformFn(name))
		if err != nil {
			return def, err
		}

		return &wrappedGaugeFloat64{GaugeFloat64: meterDef, otelMeter: otelMeter}, nil
	case metrics.Gauge:
		otelMeter, err := r.meticMeter.Int64Gauge(r.nameTransformFn(name))
		if err != nil {
			return def, err
		}

		return &wrappedGauge{Gauge: meterDef, otelMeter: otelMeter}, nil
	case metrics.Histogram:
		otelMeter, err := r.meticMeter.Int64Histogram(r.nameTransformFn(name))
		if err != nil {
			r.errorHandler(err)
			break
		}

		return &wrappedHistogram{Histogram: meterDef, otelMeter: otelMeter}, nil
	default:
		// this is just a safety net, as we should have handled all the cases above (based on the lib)
		r.errorHandler(fmt.Errorf("unsupported metric type (name=%s): %v", name, def))
	}

	return def, nil
}

type wrappedMeter struct {
	metrics.Meter
	otelMeter otelmetric.Int64Counter
}

func (m *wrappedMeter) Mark(n int64) {
	m.otelMeter.Add(context.Background(), n)
	m.Meter.Mark(n)
}

type wrappedCounter struct {
	metrics.Counter
	otelMeter otelmetric.Int64UpDownCounter
}

func (m *wrappedCounter) Inc(n int64) {
	m.otelMeter.Add(context.Background(), n)
	m.Counter.Inc(n)
}

func (m *wrappedCounter) Dec(n int64) {
	m.otelMeter.Add(context.Background(), -n)
	m.Counter.Dec(n)
}

type wrappedGaugeFloat64 struct {
	metrics.GaugeFloat64
	otelMeter otelmetric.Float64Gauge
}

func (m *wrappedGaugeFloat64) Update(newVal float64) {
	m.otelMeter.Record(context.Background(), newVal)
	m.GaugeFloat64.Update(newVal)
}

type wrappedGauge struct {
	metrics.Gauge
	otelMeter otelmetric.Int64Gauge
}

func (m *wrappedGauge) Update(newVal int64) {
	m.otelMeter.Record(context.Background(), newVal)
	m.Gauge.Update(newVal)
}

type wrappedHistogram struct {
	metrics.Histogram
	otelMeter otelmetric.Int64Histogram
}

func (m *wrappedHistogram) Update(newVal int64) {
	m.otelMeter.Record(context.Background(), newVal)
	m.Histogram.Update(newVal)
}
