package clickhouseotel

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"go.opentelemetry.io/otel/metric"
)

type ConnPoolMetrics struct {
	conn   clickhouse.Conn
	logger *slog.Logger

	pollInterval time.Duration

	meterOpenConnections    metric.Int64Gauge
	meterOpenConnectionsPct metric.Float64Gauge
	meterIdleConnections    metric.Int64Gauge
	meterIdleConnectionsPct metric.Float64Gauge
	pingTime                metric.Int64Histogram
	pingFailures            metric.Int64Counter

	stopChan  chan struct{}
	stopClose func()

	doneChan  chan struct{}
	doneClose func()

	started atomic.Bool
}

type ConnPoolMetricsConfig struct {
	Conn         clickhouse.Conn
	Meter        metric.Meter
	Logger       *slog.Logger
	PollInterval time.Duration
}

func (c ConnPoolMetricsConfig) Validate() error {
	var errs []error

	if c.Conn == nil {
		errs = append(errs, errors.New("conn is required"))
	}

	if c.Meter == nil {
		errs = append(errs, errors.New("meter is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if c.PollInterval <= 0 {
		errs = append(errs, errors.New("poll interval must be > 0"))
	}

	return errors.Join(errs...)
}

func NewConnPoolMetrics(cfg ConnPoolMetricsConfig) (*ConnPoolMetrics, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	openConnections, err := cfg.Meter.Int64Gauge(
		"clickhouse.pool.open_connections",
		metric.WithDescription("Number of open connections in the ClickHouse connection pool"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: clickhouse.pool.open_connections: %w", err)
	}

	openConnectionsPct, err := cfg.Meter.Float64Gauge(
		"clickhouse.pool.open_connections_pct",
		metric.WithDescription("Open connections as a percentage of max open connections"),
		metric.WithUnit("{percent}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: clickhouse.pool.open_connections_pct: %w", err)
	}

	idleConnections, err := cfg.Meter.Int64Gauge(
		"clickhouse.pool.idle_connections",
		metric.WithDescription("Number of idle connections in the ClickHouse connection pool"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: clickhouse.pool.idle_connections: %w", err)
	}

	idleConnectionsPct, err := cfg.Meter.Float64Gauge(
		"clickhouse.pool.idle_connections_pct",
		metric.WithDescription("Idle connections as a percentage of max idle connections"),
		metric.WithUnit("{percent}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: clickhouse.pool.idle_connections_pct: %w", err)
	}

	pingTime, err := cfg.Meter.Int64Histogram(
		"clickhouse.ping_time_ms",
		metric.WithDescription("Time it takes to perform a ClickHouse Ping call"),
		metric.WithUnit("{millisecond}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: clickhouse.ping_time_ms: %w", err)
	}

	pingFailures, err := cfg.Meter.Int64Counter(
		"clickhouse.ping_failures_total",
		metric.WithDescription("Number of failed ClickHouse Ping calls"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: clickhouse.ping_failures_total: %w", err)
	}

	stopChan := make(chan struct{})
	stopClose := sync.OnceFunc(func() { close(stopChan) })

	doneChan := make(chan struct{})
	doneClose := sync.OnceFunc(func() { close(doneChan) })

	return &ConnPoolMetrics{
		conn:                 cfg.Conn,
		logger:               cfg.Logger,
		pollInterval:         cfg.PollInterval,
		meterOpenConnections: openConnections, meterOpenConnectionsPct: openConnectionsPct,
		meterIdleConnections: idleConnections, meterIdleConnectionsPct: idleConnectionsPct,
		pingTime:     pingTime,
		pingFailures: pingFailures,
		stopChan:     stopChan,
		stopClose:    stopClose,
		doneChan:     doneChan,
		doneClose:    doneClose,
	}, nil
}

func (m *ConnPoolMetrics) Start(ctx context.Context) error {
	if m.started.Swap(true) {
		return errors.New("conn pool metrics already started")
	}

	go m.run(ctx)

	return nil
}

func (m *ConnPoolMetrics) Shutdown() error {
	m.stopClose()

	// Nothing to wait for if we were never started.
	if !m.started.Load() {
		return nil
	}

	<-m.doneChan

	return nil
}

func (m *ConnPoolMetrics) run(ctx context.Context) {
	defer m.doneClose()

	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	// Record immediately so the series exists before the first tick.
	m.record(ctx)

	for {
		select {
		case <-ticker.C:
			m.record(ctx)
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		}
	}
}

func (m *ConnPoolMetrics) record(ctx context.Context) {
	stats := m.conn.Stats()

	m.meterOpenConnections.Record(ctx, int64(stats.Open))
	m.meterIdleConnections.Record(ctx, int64(stats.Idle))

	m.meterOpenConnectionsPct.Record(ctx, pct(stats.Open, stats.MaxOpenConns))
	m.meterIdleConnectionsPct.Record(ctx, pct(stats.Idle, stats.MaxIdleConns))

	m.ping(ctx)
}

func (m *ConnPoolMetrics) ping(ctx context.Context) {
	// Ensure ping can't block shutdown forever (Shutdown waits for run loop).
	pingTimeout := m.pollInterval
	if pingTimeout > 5*time.Second {
		pingTimeout = 5 * time.Second
	}
	if pingTimeout <= 0 {
		pingTimeout = 5 * time.Second
	}

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	start := time.Now()
	if err := m.conn.Ping(pingCtx); err != nil {
		m.logger.WarnContext(ctx, "clickhouse ping failed", "error", err)
		m.pingFailures.Add(ctx, 1)
	}
	m.pingTime.Record(ctx, time.Since(start).Milliseconds())
}

func pct(val, max int) float64 {
	if max <= 0 {
		return 0
	}

	return (float64(val) / float64(max))
}
