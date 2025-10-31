package pgdriver

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/XSAM/otelsql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxstdlib "github.com/jackc/pgx/v5/stdlib"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

type Option interface {
	apply(*options)
}

type optionFunc func(c *options)

func (fn optionFunc) apply(c *options) {
	fn(c)
}

func WithTracerProvider(p trace.TracerProvider) Option {
	return optionFunc(func(o *options) {
		o.otelOptions = append(o.otelOptions, otelsql.WithTracerProvider(p))
	})
}

func WithMeterProvider(p metric.MeterProvider) Option {
	return optionFunc(func(o *options) {
		o.otelOptions = append(o.otelOptions, otelsql.WithMeterProvider(p))
	})
}

func WithSpanOptions(opt otelsql.SpanOptions) Option {
	return optionFunc(func(o *options) {
		o.otelOptions = append(o.otelOptions, otelsql.WithSpanOptions(opt))
	})
}

func WithLockTimeout(timeout time.Duration) Option {
	return optionFunc(func(o *options) {
		o.connConfig.ConnConfig.RuntimeParams["lock_timeout"] = fmt.Sprintf("%d", timeout.Milliseconds())
	})
}

func WithInterceptor(i Interceptor) Option {
	return optionFunc(func(o *options) {
		o.interceptors = append(o.interceptors, i)
	})
}

func WithObserver(ob Observer) Option {
	return optionFunc(func(o *options) {
		o.observers = append(o.observers, ob)
	})
}

type options struct {
	connConfig   *pgxpool.Config
	otelOptions  []otelsql.Option
	metricMeter  metric.Meter
	interceptors []Interceptor
	observers    []Observer
}

type Driver struct {
	pool *pgxpool.Pool
	db   *sql.DB
}

func (d *Driver) DB() *sql.DB {
	return d.db
}

func (d *Driver) Close() error {
	d.pool.Close()

	return nil
}

func NewPostgresDriver(ctx context.Context, url string, opts ...Option) (*Driver, error) {
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres url: %w", err)
	}

	o := &options{
		connConfig: config,
		otelOptions: []otelsql.Option{
			otelsql.WithAttributes(
				semconv.DBSystemPostgreSQL,
			),
		},
	}

	for _, opt := range opts {
		opt.apply(o)
	}

	tracers := make([]pgx.QueryTracer, len(o.interceptors))
	for i, interceptor := range o.interceptors {
		tracers[i] = interceptorAsTracer(interceptor)
	}
	o.connConfig.ConnConfig.Tracer = &multiTracer{tracers: tracers}

	pool, err := pgxpool.NewWithConfig(ctx, o.connConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	for _, observer := range o.observers {
		if err := observer.ObservePool(pool); err != nil {
			return nil, fmt.Errorf("failed to observe pool: %w", err)
		}
	}

	db := otelsql.OpenDB(pgxstdlib.GetPoolConnector(pool), o.otelOptions...)

	// Set maximum idle connections to 0 as connections are managed from pgx.Pool.
	// See: https://github.com/jackc/pgx/blob/v5.6.0/stdlib/sql.go#L204-L208
	db.SetMaxIdleConns(0)

	return &Driver{
		pool: pool,
		db:   db,
	}, nil
}
