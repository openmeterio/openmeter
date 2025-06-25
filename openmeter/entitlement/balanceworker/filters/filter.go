package filters

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
)

type EntitlementFilterRequest struct {
	Entitlement entitlement.Entitlement
	EventAt     time.Time
	Operation   snapshot.ValueOperationType
}

func (r EntitlementFilterRequest) Validate() error {
	errs := []error{}

	if r.Entitlement.ID == "" {
		errs = append(errs, fmt.Errorf("id is required"))
	}

	if r.Entitlement.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace is required"))
	}

	if err := r.Operation.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("operation: %w", err))
	}

	if r.EventAt.IsZero() {
		errs = append(errs, fmt.Errorf("eventAt is required"))
	}

	return errors.Join(errs...)
}

type Filter interface {
	IsNamespaceInScope(ctx context.Context, namespace string) (bool, error)
	IsEntitlementInScope(ctx context.Context, request EntitlementFilterRequest) (bool, error)
}

type NamedFilter interface {
	Filter

	Name() string
}

type RecordLastCalculationRequest struct {
	Entitlement  entitlement.Entitlement
	CalculatedAt time.Time
	IsDeleted    bool
}

type CalculationTimeRecorder interface {
	Filter

	RecordLastCalculation(context.Context, RecordLastCalculationRequest) error
}
