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
	Deactivate(context.Context, CalculationTarget) error
	Invalidate(context.Context, InvalidateInput) error
	// Process reports whether a ready key was handled, including a durably deferred failure.
	Process(context.Context) (bool, error)
}

type EnsureFreshInput struct {
	Target CalculationTarget
	// PullTTL controls how long RTE tracks changes between calls to EnsureFresh.
	PullTTL time.Duration
	// Zero accepts the current completed version without an age bound.
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

type InvalidateInput struct {
	// Source and EventID identify one immutable fact, across all selectors and
	// redeliveries. Reusing an ID for a different fact suppresses that fact.
	Source    string
	EventID   string
	Selectors []DemandSelector
}

func (i InvalidateInput) Validate() error {
	var errs []error
	if i.Source == "" {
		errs = append(errs, errors.New("source is required"))
	}

	if i.EventID == "" {
		errs = append(errs, errors.New("event ID is required"))
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
// its row lock and ignores retry backoff; background processing skips locked rows.
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

type DemandWorkResult struct {
	Processed        bool
	Retired          bool
	CalculationError error
}

type InvalidationResult struct {
	Matched   int
	Duplicate bool
}

type PruneInvalidationsInput struct {
	Before time.Time
	Limit  int
}

func (i PruneInvalidationsInput) Validate() error {
	var errs []error
	if i.Before.IsZero() {
		errs = append(errs, errors.New("receipt retention boundary is required"))
	}

	if i.Limit <= 0 {
		errs = append(errs, errors.New("receipt prune limit must be positive"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// DemandRepository keeps the work lock, successful state/outbox writes and
// completion atomic. A failed callback rolls back to a savepoint before durable
// retry bookkeeping; a process crash rolls back the entire attempt.
type DemandRepository interface {
	Activate(context.Context, CalculationTarget) error
	// EnsurePull returns true when renewal confirms the key is clean. Otherwise the caller must check for work.
	EnsurePull(context.Context, EnsureFreshInput) (clean bool, err error)
	Deactivate(context.Context, CalculationTarget) error
	Invalidate(context.Context, InvalidateInput) (InvalidationResult, error)
	PruneInvalidations(context.Context, PruneInvalidationsInput) (int, error)
	Process(context.Context, ProcessDemandInput, func(context.Context, CalculationTarget) error) (DemandWorkResult, error)
}
