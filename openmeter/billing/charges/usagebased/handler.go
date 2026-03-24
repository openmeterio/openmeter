package usagebased

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
)

type AllocateCreditsInput struct {
	Charge           Charge                `json:"charge"`
	AllocateAt       time.Time             `json:"allocateAt"`
	AmountToAllocate alpacadecimal.Decimal `json:"amountToAllocate"`
	CollectionType   RealizationRunType    `json:"collectionType"`
}

func (i AllocateCreditsInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if i.AllocateAt.IsZero() {
		errs = append(errs, fmt.Errorf("as of is required"))
	}

	if !i.AmountToAllocate.IsPositive() {
		errs = append(errs, fmt.Errorf("amount to allocate must be positive"))
	}

	if err := i.CollectionType.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("collection type: %w", err))
	}

	return errors.Join(errs...)
}

type Handler interface {
	// OnCollectionStarted is called when a collection is started for an usage-based charge.
	OnCollectionStarted(ctx context.Context, input AllocateCreditsInput) (creditrealization.CreateInputs, error)

	// OnCollectionFinalized is called when a collection is finalized for an usage-based charge.
	OnCollectionFinalized(ctx context.Context, input AllocateCreditsInput) (creditrealization.CreateInputs, error)

	// OnCollectionFinalizedRollback is called when a collection is finalized for an usage-based charge and the credit allocations need to be rolled back.
	// TODO: implement this after we have decided on who should be responsible for deciding what to roll back.
	// OnCollectionFinalizedRollback(ctx context.Context, input AllocateCreditsInput) error
}

type UnimplementedHandler struct{}

var _ Handler = (*UnimplementedHandler)(nil)

func (h UnimplementedHandler) OnCollectionStarted(ctx context.Context, input AllocateCreditsInput) (creditrealization.CreateInputs, error) {
	return nil, errors.New("not implemented")
}

func (h UnimplementedHandler) OnCollectionFinalized(ctx context.Context, input AllocateCreditsInput) (creditrealization.CreateInputs, error) {
	return nil, errors.New("not implemented")
}
