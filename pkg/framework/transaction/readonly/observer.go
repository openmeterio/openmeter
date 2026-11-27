// Package readonly observes requests that are expected to only read from the
// database, to find the ones that write before reads are moved to a replica.
package readonly

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	metricNameRequests = "openmeter.database.read_only.requests"
	metricNameWrites   = "openmeter.database.read_only.writes"

	attrKeyOperation      = attribute.Key("operation")
	attrKeyWriteKind      = attribute.Key("write.kind")
	attrKeyWriteTarget    = attribute.Key("write.target")
	attrKeyWriteOperation = attribute.Key("write.operation")
)

type ObserverConfig struct {
	Logger *slog.Logger
	Meter  metric.Meter
}

var _ models.Validator = (*ObserverConfig)(nil)

func (c ObserverConfig) Validate() error {
	var errs []error

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if c.Meter == nil {
		errs = append(errs, errors.New("meter is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Observer records the writes attempted by read-only requests without stopping them.
type Observer struct {
	logger *slog.Logger

	requests metric.Int64Counter
	writes   metric.Int64Counter

	// logged holds the writes that were already logged, so a hot path logs once
	// per process instead of once per request.
	logged sync.Map
}

func NewObserver(config ObserverConfig) (*Observer, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid read-only observer config: %w", err)
	}

	requests, err := config.Meter.Int64Counter(
		metricNameRequests,
		metric.WithDescription("Requests that ran inside a read-only scope"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s counter: %w", metricNameRequests, err)
	}

	writes, err := config.Meter.Int64Counter(
		metricNameWrites,
		metric.WithDescription("Database writes attempted inside a read-only scope"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s counter: %w", metricNameWrites, err)
	}

	return &Observer{
		logger:   config.Logger,
		requests: requests,
		writes:   writes,
	}, nil
}

// MarkReadOnly marks ctx as read-only for the given operation. Every marked
// request is counted, so an operation that never writes can be told apart from
// one that is never called.
func (o *Observer) MarkReadOnly(ctx context.Context, operation string) context.Context {
	o.requests.Add(ctx, 1, metric.WithAttributes(attrKeyOperation.String(operation)))

	// WithWriteObserver only fails without an observer.
	ctx, _ = transaction.WithWriteObserver(ctx, operationObserver{observer: o, operation: operation})

	return ctx
}

type operationObserver struct {
	observer  *Observer
	operation string
}

func (o operationObserver) ObserveWrite(ctx context.Context, attempt transaction.WriteAttempt) error {
	o.observer.observe(ctx, o.operation, attempt)

	return nil
}

func (o *Observer) observe(ctx context.Context, operation string, attempt transaction.WriteAttempt) {
	// Advisory lock keys embed entity IDs, so they would make the series and the
	// set of logged writes grow without bound.
	target := attempt.Target
	if attempt.Kind == transaction.WriteKindAdvisoryLock {
		target = ""
	}

	o.writes.Add(ctx, 1, metric.WithAttributes(
		attrKeyOperation.String(operation),
		attrKeyWriteKind.String(string(attempt.Kind)),
		attrKeyWriteTarget.String(target),
		attrKeyWriteOperation.String(attempt.Operation),
	))

	key := strings.Join([]string{operation, string(attempt.Kind), target, attempt.Operation}, "|")
	if _, alreadyLogged := o.logged.LoadOrStore(key, struct{}{}); alreadyLogged {
		return
	}

	o.logger.WarnContext(ctx, "read-only request wrote to the database",
		slog.String("operation", operation),
		slog.String("write_kind", string(attempt.Kind)),
		slog.String("write_target", attempt.Target),
		slog.String("write_operation", attempt.Operation),
		slog.String("caller", writeSite()),
	)
}

// writeSite returns the first caller outside of the database plumbing, which is
// the adapter or service code that issued the write.
func writeSite() string {
	pcs := make([]uintptr, 64)
	frames := runtime.CallersFrames(pcs[:runtime.Callers(2, pcs)])

	for {
		frame, more := frames.Next()

		if !isDatabasePlumbing(frame.Function) {
			return fmt.Sprintf("%s:%d", frame.Function, frame.Line)
		}

		if !more {
			return ""
		}
	}
}

func isDatabasePlumbing(function string) bool {
	return strings.HasPrefix(function, "entgo.io/ent") ||
		strings.Contains(function, "/pkg/framework/") ||
		// generated Ent packages, including the ones of downstream modules
		strings.Contains(function, "/ent/db.") ||
		strings.Contains(function, "/ent/db/")
}
