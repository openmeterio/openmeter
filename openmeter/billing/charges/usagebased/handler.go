package usagebased

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CreditsOnlyUsageAccruedInput struct {
	Charge           Charge                `json:"charge"`
	Run              RealizationRun        `json:"run"`
	AllocateAt       time.Time             `json:"allocateAt"`
	AmountToAllocate alpacadecimal.Decimal `json:"amountToAllocate"`
}

func (i CreditsOnlyUsageAccruedInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.Run.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("run: %w", err))
	}

	if i.AllocateAt.IsZero() {
		errs = append(errs, fmt.Errorf("as of is required"))
	}

	if !i.AmountToAllocate.IsPositive() {
		errs = append(errs, fmt.Errorf("amount to allocate must be positive"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CreditsOnlyUsageAccruedCorrectionInput struct {
	Charge     Charge         `json:"charge"`
	Run        RealizationRun `json:"run"`
	AllocateAt time.Time      `json:"allocateAt"`

	Corrections creditrealization.CorrectionRequest `json:"corrections"`
}

type Handler interface {
	// OnCreditsOnlyUsageAccrued is called when a credit-only usage-based charge needs to be allocated as credits fully.
	OnCreditsOnlyUsageAccrued(ctx context.Context, input CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error)

	// OnCreditsOnlyUsageAccruedCorrection is called when a credit-only usage-based charge needs to be corrected.
	OnCreditsOnlyUsageAccruedCorrection(ctx context.Context, input CreditsOnlyUsageAccruedCorrectionInput) (creditrealization.CreateCorrectionInputs, error)
}

type UnimplementedHandler struct{}

var _ Handler = (*UnimplementedHandler)(nil)

func (h UnimplementedHandler) OnCreditsOnlyUsageAccrued(ctx context.Context, input CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
	return nil, errors.New("not implemented")
}

func (h UnimplementedHandler) OnCreditsOnlyUsageAccruedCorrection(ctx context.Context, input CreditsOnlyUsageAccruedCorrectionInput) (creditrealization.CreateCorrectionInputs, error) {
	return nil, errors.New("not implemented")
}
