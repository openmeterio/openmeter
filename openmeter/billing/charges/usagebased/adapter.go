package usagebased

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Adapter interface {
	RealizationRunAdapter
	RealizationRunCreditAllocationAdapter
	RealizationRunInvoiceUsageAdapter
	RealizationRunPaymentAdapter
	ChargeAdapter
	ChargeCostBasisAdapter

	entutils.TxCreator
}

type ChargeCostBasisAdapter interface {
	SetResolvedCostBasis(ctx context.Context, input costbasis.SetResolvedCostBasisInput) (costbasis.CostBasis, error)
}

type ChargeAdapter interface {
	CreateCharges(ctx context.Context, charges CreateChargesInput) ([]Charge, error)
	UpdateCharge(ctx context.Context, charge ChargeBase) (ChargeBase, error)
	CreateChargeOverride(ctx context.Context, charge ChargeBase, override IntentMutableFields) (ChargeBase, error)
	DeleteChargeOverride(ctx context.Context, charge ChargeBase) (ChargeBase, error)
	UpdateSubscriptionItemID(ctx context.Context, charge Charge, newSubscriptionItemID string) (Charge, error)
	DeleteCharge(ctx context.Context, charge Charge) error
	GetByIDs(ctx context.Context, input GetByIDsInput) ([]Charge, error)
	GetByID(ctx context.Context, input GetByIDInput) (Charge, error)
}

type RealizationRunAdapter interface {
	CreateRealizationRun(ctx context.Context, chargeID meta.ChargeID, input CreateRealizationRunAdapterInput) (RealizationRunBase, error)
	UpdateRealizationRun(ctx context.Context, input UpdateRealizationRunInput) (RealizationRunBase, error)
	UpsertRunDetailedLines(ctx context.Context, input UpsertRunDetailedLinesInput) error
	FetchDetailedLines(ctx context.Context, charge Charge) (Charge, error)
}

type CreateRealizationRunAdapterInput struct {
	CreateRealizationRunInput

	PriorRunID *RealizationRunID
}

var _ models.Validator = (*CreateRealizationRunAdapterInput)(nil)

func (i CreateRealizationRunAdapterInput) Validate() error {
	var errs []error

	if err := i.CreateRealizationRunInput.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.PriorRunID != nil {
		if err := i.PriorRunID.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("prior run id: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type RealizationRunCreditAllocationAdapter interface {
	CreateChargeCurrencyCreditRealizations(ctx context.Context, input CreateCreditRealizationsInput) (creditrealization.Realizations, error)
	CreateFiatOverageCreditRealizations(ctx context.Context, input CreateCreditRealizationsInput) (creditrealization.Realizations, error)
}

type CreateCreditRealizationsInput struct {
	RunID              RealizationRunID
	CreditRealizations creditrealization.CreateInputs
}

func (i CreateCreditRealizationsInput) Validate() error {
	var errs []error

	if err := i.RunID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("run ID: %w", err))
	}

	if err := i.CreditRealizations.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("credit realizations: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type RealizationRunInvoiceUsageAdapter interface {
	CreateRunInvoicedUsage(ctx context.Context, runID RealizationRunID, invoicedUsage invoicedusage.AccruedUsage) (invoicedusage.AccruedUsage, error)
	DeleteRunInvoicedUsage(ctx context.Context, id models.NamespacedID) error
}

type RealizationRunPaymentAdapter interface {
	CreateRunPayment(ctx context.Context, runID RealizationRunID, in payment.InvoicedCreate) (payment.Invoiced, error)
	UpdateRunPayment(ctx context.Context, in payment.Invoiced) (payment.Invoiced, error)
}
