package rte

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

// DemandService tracks changed inputs and refreshes balances in the background
// (Activate) or on request (EnsureFresh). Get reads the last stored result.
type DemandService interface {
	Activate(context.Context, CalculationTarget) error
	EnsureFresh(context.Context, EnsureFreshInput) error
	// ResetSourceOffset clears a checkpoint beyond the current partition log end before consumption starts.
	ResetSourceOffset(ctx context.Context, logEnd SourcePosition) error

	Deactivate(context.Context, CalculationTarget) error
	Invalidate(context.Context, InvalidateInput) error

	// Process reports whether a ready key was handled, including a durably deferred failure.
	Process(context.Context) (bool, error)
}

type EnsureFreshInput struct {
	Target CalculationTarget

	// PullTTL controls how long RTE tracks changes between calls to EnsureFresh.
	PullTTL time.Duration

	// MaxAge limits the age of a completed calculation. Zero accepts any age.
	MaxAge time.Duration
}

func (i EnsureFreshInput) Validate() error {
	var errs []error
	errs = append(errs, i.Target.Validate())
	if i.MaxAge < 0 {
		errs = append(errs, errors.New("max age must not be negative"))
	}

	if i.PullTTL <= 0 {
		errs = append(errs, errors.New("pull TTL must be positive"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// DemandSelector selects tracked identities. An empty owner/target list selects
// the whole kind in the namespace; producers must choose this deliberately.
type DemandSelector struct {
	Namespace string
	Kind      CalculationKind
	OwnerIDs  []string
	Targets   []CalculationTarget
}

// SourcePosition identifies one record in a Kafka topic partition.
type SourcePosition struct {
	Topic     string
	Partition int32
	Offset    int64
}

func (p SourcePosition) Validate() error {
	var errs []error
	if p.Topic == "" {
		errs = append(errs, errors.New("topic is required"))
	}

	if p.Partition < 0 {
		errs = append(errs, errors.New("partition must be non-negative"))
	}

	if p.Offset < 0 {
		errs = append(errs, errors.New("offset must be non-negative"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type InvalidationResult struct {
	Matched   int64
	Duplicate bool
}

type InvalidateInput struct {
	// In-process and non-Kafka deliveries omit Position and always apply.
	Position  *SourcePosition
	Selectors []DemandSelector
}

func (i InvalidateInput) Validate() error {
	var errs []error
	if i.Position != nil {
		errs = append(errs, i.Position.Validate())
	}

	for n, selector := range i.Selectors {
		if selector.Namespace == "" {
			errs = append(errs, fmt.Errorf("selector %d: namespace is required", n))
		}

		if selector.Kind != CalculationKindCreditBalance && selector.Kind != CalculationKindMeteredEntitlement {
			errs = append(errs, fmt.Errorf("selector %d: unsupported calculation kind", n))
		}

		for _, target := range selector.Targets {
			identity, err := target.Identity()
			if err != nil {
				errs = append(errs, err)
				continue
			}

			if identity.Namespace != selector.Namespace || target.Kind != selector.Kind {
				errs = append(errs, fmt.Errorf("selector %d: target scope mismatch", n))
			}
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// ProcessDemandInput optionally targets one key. Targeted processing waits for
// its calculation lock and can retry immediately. Background workers skip busy keys.
type ProcessDemandInput struct {
	Target             *CalculationTarget
	CalculationTimeout time.Duration
	RetryMinDelay      time.Duration
	RetryMaxDelay      time.Duration
}

func (i ProcessDemandInput) Validate() error {
	var errs []error
	if i.Target != nil {
		errs = append(errs, i.Target.Validate())
	}

	if i.RetryMinDelay <= 0 || i.RetryMaxDelay < i.RetryMinDelay {
		errs = append(errs, errors.New("retry delays must be positive and max must be at least min"))
	}

	if i.CalculationTimeout <= 0 {
		errs = append(errs, errors.New("calculation timeout must be positive"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// DemandAttemptTiming is set only after a background worker takes the calculation
// lock and confirms work is due. StartedAt measures elapsed processing locally.
type DemandAttemptTiming struct {
	Kind      CalculationKind
	QueueWait time.Duration
	StartedAt time.Time
}

type DemandWorkResult struct {
	Timing           *DemandAttemptTiming
	Processed        bool
	Retired          bool
	CalculationError error
}

// DemandRepository keeps the work lock, successful state/outbox writes and
// completion atomic. A failed callback rolls back to a savepoint before durable
// retry bookkeeping; a process crash rolls back the entire attempt.
type DemandRepository interface {
	Activate(context.Context, CalculationTarget) error

	// EnsurePull returns true when tracking was continuous, the versions match,
	// the age is acceptable and no source deletion is due.
	EnsurePull(context.Context, EnsureFreshInput) (clean bool, err error)

	// ResetSourceOffset clears a checkpoint beyond the current partition log end before consumption starts.
	ResetSourceOffset(ctx context.Context, logEnd SourcePosition) error

	Deactivate(context.Context, CalculationTarget) error
	Invalidate(context.Context, InvalidateInput) (InvalidationResult, error)
	Process(context.Context, ProcessDemandInput, func(context.Context, CalculationTarget) error) (DemandWorkResult, error)
}
