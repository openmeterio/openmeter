// Package shadow publishes sampled balance operations and committed source changes
// for the RTE comparison without making their delivery part of the production operation.
package shadow

import (
	"context"
	"errors"
	"hash/crc32"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	rtemetrics "github.com/openmeterio/openmeter/openmeter/rte/metrics"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/models"
)

const queueCapacity = 128

type Config struct {
	Publish       func(context.Context, marshaler.Event) error
	SamplePercent int
	Meter         metric.Meter
	Logger        *slog.Logger
}

func (c Config) Validate() error {
	var errs []error
	if c.Publish == nil || c.Meter == nil || c.Logger == nil {
		errs = append(errs, errors.New("publisher, meter and logger are required"))
	}

	if c.SamplePercent < 1 || c.SamplePercent > 100 {
		errs = append(errs, errors.New("sample percent must be between 1 and 100"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type publication struct {
	event     marshaler.Event
	kind      rte.CalculationKind
	operation rte.ComparisonOperation
}

type Recorder struct {
	config       Config
	publications *rtemetrics.Publications
	ctx          context.Context
	cancel       context.CancelFunc
	queue        chan publication
	done         chan struct{}
	admission    sync.RWMutex
}

func NewRecorder(ctx context.Context, config Config) (*Recorder, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	publications, err := rtemetrics.NewPublications(config.Meter)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	r := &Recorder{
		config: config, publications: publications,
		ctx: ctx, cancel: cancel,
		queue: make(chan publication, queueCapacity), done: make(chan struct{}),
	}
	go r.run()

	return r, nil
}

func (r *Recorder) Close() {
	r.cancel()
	// A shared Kafka publisher can be in SendMessage until the broker timeout.
	// Let it finish using the publisher before its own cleanup closes the client.
	<-r.done
}

func (r *Recorder) ObserveCreditRead(ctx context.Context, input customerbalance.GetBalanceServiceInput, duration time.Duration) {
	if r == nil || input.BalanceQuery.AsOf != nil || input.BalanceQuery.After != nil {
		return
	}

	key := creditbalance.Key{CustomerID: input.CustomerID, Currency: input.Currency, FeatureFilter: input.FeatureFilter}
	r.observe(ctx, key, rte.CalculationKindCreditBalance, rte.ComparisonOperationRead, duration)
}

func (r *Recorder) ObserveMeteredRead(ctx context.Context, id models.NamespacedID, duration time.Duration) {
	if r != nil {
		r.observe(ctx, meteredentitlement.Key{EntitlementID: id}, rte.CalculationKindMeteredEntitlement, rte.ComparisonOperationRead, duration)
	}
}

func (r *Recorder) ObserveMeteredCalculation(ctx context.Context, id models.NamespacedID, duration time.Duration) {
	if r != nil {
		r.observe(ctx, meteredentitlement.Key{EntitlementID: id}, rte.CalculationKindMeteredEntitlement, rte.ComparisonOperationCalculate, duration)
	}
}

func (r *Recorder) observe(ctx context.Context, key rte.Key, kind rte.CalculationKind, op rte.ComparisonOperation, duration time.Duration) {
	at := time.Now()
	item := publication{kind: kind, operation: op}
	target, err := rte.NewCalculationTarget(key)
	if err != nil {
		r.publications.Record(ctx, item.kind, item.operation, rtemetrics.PublicationOutcomeInvalid)
		return
	}

	// NewCalculationTarget encodes the typed key, so every producer sees the
	// same bytes for the same balance, across replicas and operation types.
	if int(crc32.ChecksumIEEE(target.Key)%100) >= r.config.SamplePercent {
		r.publications.Record(ctx, item.kind, item.operation, rtemetrics.PublicationOutcomeSampledOut)
		return
	}

	item.event = rte.BalanceComparisonEvent{CalculationTarget: target, Operation: op, DirectDuration: duration, ObservedAt: at}
	r.enqueue(ctx, item)
}

// TryPublishSource takes ownership of an immutable, committed source fact.
// All source facts are eligible because one can affect multiple sampled targets.
func (r *Recorder) TryPublishSource(event marshaler.Event) {
	if r != nil && event != nil {
		r.enqueue(r.ctx, publication{event: event})
	}
}

func (r *Recorder) enqueue(ctx context.Context, item publication) {
	r.admission.RLock()
	defer r.admission.RUnlock()

	if r.ctx.Err() != nil {
		r.publications.Record(ctx, item.kind, item.operation, rtemetrics.PublicationOutcomeStopped)
		return
	}

	select {
	case r.queue <- item:
	default:
		r.publications.Record(ctx, item.kind, item.operation, rtemetrics.PublicationOutcomeQueueFull)
	}
}

func (r *Recorder) run() {
	defer func() {
		// Once cancellation is visible no new items can enter. Wait for
		// producers already in enqueue, then account for abandoned events.
		r.admission.Lock()
		close(r.queue)
		r.admission.Unlock()

		for item := range r.queue {
			r.publications.Record(r.ctx, item.kind, item.operation, rtemetrics.PublicationOutcomeStopped)
		}

		close(r.done)
	}()

	for r.ctx.Err() == nil {
		select {
		case <-r.ctx.Done():
			return
		case item := <-r.queue:
			if r.ctx.Err() != nil {
				r.publications.Record(r.ctx, item.kind, item.operation, rtemetrics.PublicationOutcomeStopped)
				return
			}

			if event, ok := item.event.(rte.BalanceComparisonEvent); ok && time.Since(event.ObservedAt) >= rte.ComparisonObservationTTL {
				r.publications.Record(r.ctx, item.kind, item.operation, rtemetrics.PublicationOutcomeExpired)
				continue
			}

			// Never pass request cancellation, transaction handles, or their
			// context values to an asynchronous publication.
			ctx, cancel := context.WithTimeout(r.ctx, 5*time.Second)
			err := r.config.Publish(ctx, item.event)
			cancel()
			if err != nil {
				r.publications.Record(r.ctx, item.kind, item.operation, rtemetrics.PublicationOutcomeFailed)
				r.config.Logger.DebugContext(r.ctx, "publish RTE comparison event", "error", err)

				continue
			}

			r.publications.Record(r.ctx, item.kind, item.operation, rtemetrics.PublicationOutcomePublished)
		}
	}
}
