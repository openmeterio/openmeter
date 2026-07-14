package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/invoicemetrics"
	"github.com/openmeterio/openmeter/pkg/clock"
)

const metricName = "openmeter.billing.worker.pending_invoices"

type Config struct {
	Adapter invoicemetrics.Adapter
	Meter   metric.Meter
	Logger  *slog.Logger

	ReportInterval     time.Duration
	OverdueThreshold   time.Duration
	QueryTimeout       time.Duration
	ExcludedNamespaces []string
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter is required"))
	}

	if c.Meter == nil {
		errs = append(errs, errors.New("meter is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if c.ReportInterval <= 0 {
		errs = append(errs, errors.New("report interval must be greater than zero"))
	}

	if c.OverdueThreshold <= 0 {
		errs = append(errs, errors.New("overdue threshold must be greater than zero"))
	}

	if c.QueryTimeout <= 0 {
		errs = append(errs, errors.New("query timeout must be greater than zero"))
	}

	return errors.Join(errs...)
}

func New(config Config) (invoicemetrics.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	pendingInvoices, err := config.Meter.Int64Gauge(
		metricName,
		metric.WithDescription("Number of invoices overdue for billing worker processing"),
		metric.WithUnit("{invoice}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create pending invoice gauge: %w", err)
	}

	return &Service{
		adapter:            config.Adapter,
		logger:             config.Logger,
		pendingInvoices:    pendingInvoices,
		reportInterval:     config.ReportInterval,
		overdueThreshold:   config.OverdueThreshold,
		queryTimeout:       config.QueryTimeout,
		excludedNamespaces: config.ExcludedNamespaces,
		stopCh:             make(chan struct{}),
	}, nil
}

var _ invoicemetrics.Service = (*Service)(nil)

type Service struct {
	adapter invoicemetrics.Adapter
	logger  *slog.Logger

	pendingInvoices metric.Int64Gauge

	reportInterval     time.Duration
	overdueThreshold   time.Duration
	queryTimeout       time.Duration
	excludedNamespaces []string

	started  atomic.Bool
	stopCh   chan struct{}
	stopOnce sync.Once
}

func (s *Service) Start(ctx context.Context) error {
	if s.started.Swap(true) {
		return errors.New("invoice metrics service is already started")
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		select {
		case <-runCtx.Done():
		case <-s.stopCh:
			cancel()
		}
	}()

	s.report(runCtx)

	ticker := time.NewTicker(s.reportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-runCtx.Done():
			return nil
		case <-ticker.C:
			s.report(runCtx)
		}
	}
}

func (s *Service) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
}

func (s *Service) report(ctx context.Context) {
	queryCtx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	counts, err := s.adapter.CountOverdueInvoices(queryCtx, invoicemetrics.CountOverdueInvoicesInput{
		AsOf:               clock.Now(),
		MinimumAge:         s.overdueThreshold,
		ExcludedNamespaces: s.excludedNamespaces,
	})
	if err != nil {
		s.logger.WarnContext(ctx, "failed to report overdue invoice counts", "error", err)

		return
	}

	s.pendingInvoices.Record(ctx, counts.Collection, metric.WithAttributes(
		attribute.String("operation", "collection"),
	))
	s.pendingInvoices.Record(ctx, counts.Advancement, metric.WithAttributes(
		attribute.String("operation", "advancement"),
	))
}
