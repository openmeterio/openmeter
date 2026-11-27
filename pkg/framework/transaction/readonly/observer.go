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

// ErrWriteRejected is returned for a write attempted by a read-only request when
// the observer rejects writes.
var ErrWriteRejected = errors.New("read-only request attempted a database write")

type ObserverConfig struct {
	Logger *slog.Logger
	Meter  metric.Meter

	// RejectWrites fails the writes instead of only reporting them. It is meant
	// for tests: a read-only request that writes then fails where it is introduced.
	RejectWrites bool
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

// Observer records the writes attempted by read-only requests. It only stops
// them when it is configured to reject writes.
type Observer struct {
	logger       *slog.Logger
	rejectWrites bool

	requestCount metric.Int64Counter
	writeCount   metric.Int64Counter

	// logged holds the writes that were already logged, so a hot path logs once
	// per process instead of once per request.
	logged sync.Map
}

func NewObserver(config ObserverConfig) (*Observer, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid read-only observer config: %w", err)
	}

	requestCount, err := config.Meter.Int64Counter(
		metricNameRequests,
		metric.WithDescription("Requests that ran inside a read-only scope"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s counter: %w", metricNameRequests, err)
	}

	writeCount, err := config.Meter.Int64Counter(
		metricNameWrites,
		metric.WithDescription("Database writes attempted inside a read-only scope"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s counter: %w", metricNameWrites, err)
	}

	return &Observer{
		logger:       config.Logger,
		rejectWrites: config.RejectWrites,
		requestCount: requestCount,
		writeCount:   writeCount,
	}, nil
}

// MarkReadOnly marks ctx as read-only for the given operation. Every marked
// request is counted, so an operation that never writes can be told apart from
// one that is never called.
func (o *Observer) MarkReadOnly(ctx context.Context, operation string) context.Context {
	o.requestCount.Add(ctx, 1, metric.WithAttributes(attrKeyOperation.String(operation)))

	// WithWriteObserver only fails without an observer.
	ctx, _ = transaction.WithWriteObserver(ctx, operationObserver{observer: o, operation: operation})

	return ctx
}

type operationObserver struct {
	observer  *Observer
	operation string
}

var _ transaction.WriteObserver = operationObserver{}

func (o operationObserver) ObserveWrite(ctx context.Context, attempt transaction.WriteAttempt) error {
	o.observer.observe(ctx, o.operation, attempt)

	if o.observer.rejectWrites {
		return fmt.Errorf("%w: %s by %q (target %q, operation %q)",
			ErrWriteRejected, attempt.Kind, o.operation, attempt.Target, attempt.Operation)
	}

	return nil
}

func (o operationObserver) RejectsWrites() bool {
	return o.observer.rejectWrites
}

func (o *Observer) observe(ctx context.Context, operation string, attempt transaction.WriteAttempt) {
	// Advisory lock keys embed entity IDs, so they would make the series and the
	// set of logged writes grow without bound.
	target := attempt.Target
	if attempt.Kind == transaction.WriteKindAdvisoryLock {
		target = ""
	}

	o.writeCount.Add(ctx, 1, metric.WithAttributes(
		attrKeyOperation.String(operation),
		attrKeyWriteKind.String(string(attempt.Kind)),
		attrKeyWriteTarget.String(target),
		attrKeyWriteOperation.String(attempt.Operation),
	))

	// Log each distinct write once per process: a hot path would otherwise flood
	// the logs, and the metric already counts every attempt.
	key := strings.Join([]string{operation, string(attempt.Kind), target, attempt.Operation}, "|")
	if _, alreadyLogged := o.logged.LoadOrStore(key, struct{}{}); alreadyLogged {
		return
	}

	// The caller may ignore a rejection and carry on, like a balance read that
	// skips its snapshot when it cannot lock. Then this log is the only trace of
	// it, so it is an error.
	level := slog.LevelWarn
	if o.rejectWrites {
		level = slog.LevelError
	}

	// Capture the stack only for the writes that are logged: walking it for every
	// write of a hot path is too expensive.
	if !o.logger.Enabled(ctx, level) {
		return
	}

	o.logger.Log(ctx, level, "read-only request attempted a database write",
		slog.String("operation", operation),
		slog.String("write_kind", string(attempt.Kind)),
		slog.String("write_target", attempt.Target),
		slog.String("write_operation", attempt.Operation),
		slog.Bool("rejected", o.rejectWrites),
		slog.Any("stack", writeStack()),
	)
}

// writeStack returns the call stack of the write, starting at the code that
// reported it. The whole stack is kept because the nearest frames are usually a
// shared adapter method, which does not tell which code path led to the write.
func writeStack() []string {
	// Skip runtime.Callers, writeStack, observe, ObserveWrite and transaction.ReportWrite.
	pcs := make([]uintptr, 64)

	n := runtime.Callers(5, pcs)
	if n == 0 {
		return nil
	}

	frames := runtime.CallersFrames(pcs[:n])
	stack := make([]string, 0, n)

	for {
		frame, more := frames.Next()
		stack = append(stack, fmt.Sprintf("%s:%d", frame.Function, frame.Line))

		if !more {
			return stack
		}
	}
}
