package usagebased

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
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

	Corrections                  creditrealization.CorrectionRequest   `json:"corrections"`
	LineageSegmentsByRealization lineage.ActiveSegmentsByRealizationID `json:"-"`
}

type OnInvoiceUsageAccruedInput struct {
	Charge        Charge                `json:"charge"`
	Run           RealizationRun        `json:"run"`
	ServicePeriod timeutil.ClosedPeriod `json:"servicePeriod"`
	Amount        alpacadecimal.Decimal `json:"amount"`
}

func (i OnInvoiceUsageAccruedInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.Run.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("run: %w", err))
	}

	if err := i.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if i.Amount.IsNegative() {
		errs = append(errs, fmt.Errorf("amount cannot be negative"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type RunEventInput struct {
	Charge Charge         `json:"charge"`
	Run    RealizationRun `json:"run"`
}

func (i RunEventInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.Run.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("run: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type (
	OnPaymentAuthorizedInput = RunEventInput
	OnPaymentSettledInput    = RunEventInput
)

type Handler interface {
	// OnInvoiceUsageAccrued is called when invoice-settled usage-based usage is sent to the customer.
	OnInvoiceUsageAccrued(ctx context.Context, input OnInvoiceUsageAccruedInput) (ledgertransaction.GroupReference, error)

	// OnPaymentAuthorized is called when an invoice-backed usage-based run receives payment authorization.
	OnPaymentAuthorized(ctx context.Context, input OnPaymentAuthorizedInput) (ledgertransaction.GroupReference, error)

	// OnPaymentSettled is called when an invoice-backed usage-based run payment is settled.
	OnPaymentSettled(ctx context.Context, input OnPaymentSettledInput) (ledgertransaction.GroupReference, error)

	// OnCreditsOnlyUsageAccrued is called when a credit-only usage-based charge needs to be allocated as credits fully.
	OnCreditsOnlyUsageAccrued(ctx context.Context, input CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error)

	// OnCreditsOnlyUsageAccruedCorrection is called when a credit-only usage-based charge needs to be corrected.
	OnCreditsOnlyUsageAccruedCorrection(ctx context.Context, input CreditsOnlyUsageAccruedCorrectionInput) (creditrealization.CreateCorrectionInputs, error)
}

type UnimplementedHandler struct{}

var _ Handler = (*UnimplementedHandler)(nil)

func (h UnimplementedHandler) OnInvoiceUsageAccrued(ctx context.Context, input OnInvoiceUsageAccruedInput) (ledgertransaction.GroupReference, error) {
	return ledgertransaction.GroupReference{}, errors.New("not implemented")
}

func (h UnimplementedHandler) OnPaymentAuthorized(ctx context.Context, input OnPaymentAuthorizedInput) (ledgertransaction.GroupReference, error) {
	return ledgertransaction.GroupReference{}, errors.New("not implemented")
}

func (h UnimplementedHandler) OnPaymentSettled(ctx context.Context, input OnPaymentSettledInput) (ledgertransaction.GroupReference, error) {
	return ledgertransaction.GroupReference{}, errors.New("not implemented")
}

func (h UnimplementedHandler) OnCreditsOnlyUsageAccrued(ctx context.Context, input CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
	return nil, errors.New("not implemented")
}

func (h UnimplementedHandler) OnCreditsOnlyUsageAccruedCorrection(ctx context.Context, input CreditsOnlyUsageAccruedCorrectionInput) (creditrealization.CreateCorrectionInputs, error) {
	return nil, errors.New("not implemented")
}
