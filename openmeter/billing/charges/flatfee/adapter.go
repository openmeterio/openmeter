package flatfee

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Adapter interface {
	ChargeAdapter
	ChargeDetailedLineAdapter
	ChargeCreditAllocationAdapter
	ChargeInvoicedUsageAdapter
	ChargePaymentAdapter

	entutils.TxCreator
}

type ChargeAdapter interface {
	CreateCharges(ctx context.Context, charges CreateChargesInput) ([]Charge, error)
	ProvisionCurrentRun(ctx context.Context, input ProvisionCurrentRunInput) (RealizationRunBase, error)
	UpdateCharge(ctx context.Context, charge ChargeBase) (ChargeBase, error)
	DeleteCharge(ctx context.Context, charge Charge) error
	GetByIDs(ctx context.Context, ids GetByIDsInput) ([]Charge, error)
	GetByID(ctx context.Context, id GetByIDInput) (Charge, error)
}

type ChargeDetailedLineAdapter interface {
	UpsertDetailedLines(ctx context.Context, chargeID meta.ChargeID, lines DetailedLines) error
	FetchCurrentRunDetailedLines(ctx context.Context, charge Charge) (Charge, error)
}

type ChargeInvoicedUsageAdapter interface {
	CreateInvoicedUsage(ctx context.Context, input CreateInvoicedUsageInput) (invoicedusage.AccruedUsage, error)
}

type ChargeCreditAllocationAdapter interface {
	CreateCreditAllocations(ctx context.Context, chargeID meta.ChargeID, creditAllocations creditrealization.CreateInputs) (creditrealization.Realizations, error)
}

type ChargePaymentAdapter interface {
	CreatePayment(ctx context.Context, chargeID meta.ChargeID, paymentSettlement payment.InvoicedCreate) (payment.Invoiced, error)
	UpdatePayment(ctx context.Context, paymentSettlement payment.Invoiced) (payment.Invoiced, error)
}

type ProvisionCurrentRunInput struct {
	Charge                    ChargeBase
	NoFiatTransactionRequired bool
}

func (i ProvisionCurrentRunInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CreateInvoicedUsageInput struct {
	ChargeID      meta.ChargeID
	LineID        string
	InvoiceID     string
	InvoicedUsage invoicedusage.AccruedUsage
}

func (i CreateInvoicedUsageInput) Validate() error {
	var errs []error

	if err := i.ChargeID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge ID: %w", err))
	}

	if i.InvoiceID == "" {
		errs = append(errs, fmt.Errorf("invoice ID is required"))
	}

	if i.LineID == "" {
		errs = append(errs, fmt.Errorf("line ID is required"))
	}

	if err := i.InvoicedUsage.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invoiced usage: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type IntentWithInitialStatus struct {
	Intent
	FeatureID                 *string
	InitialStatus             Status
	InitialAdvanceAfter       *time.Time
	AmountAfterProration      alpacadecimal.Decimal
	NoFiatTransactionRequired bool
}

func (i IntentWithInitialStatus) Validate() error {
	var errs []error
	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	if i.AmountAfterProration.IsNegative() {
		errs = append(errs, fmt.Errorf("amount after proration cannot be negative"))
	}

	if err := i.InitialStatus.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("initial status: %w", err))
	}

	if i.InitialAdvanceAfter != nil && i.InitialAdvanceAfter.IsZero() {
		errs = append(errs, fmt.Errorf("initial advance after cannot be zero"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type GetByIDsInput struct {
	Namespace string
	IDs       []string

	Expands meta.Expands
}

func (i GetByIDsInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	for _, id := range i.IDs {
		if id == "" {
			errs = append(errs, errors.New("id is required"))
		}
	}

	if err := validateExpands(i.Expands); err != nil {
		errs = append(errs, fmt.Errorf("expands: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type GetByIDInput struct {
	ChargeID meta.ChargeID
	Expands  meta.Expands
}

func (i GetByIDInput) Validate() error {
	var errs []error
	if err := i.ChargeID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge ID: %w", err))
	}

	if err := validateExpands(i.Expands); err != nil {
		errs = append(errs, fmt.Errorf("expands: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CreateChargesInput struct {
	Namespace string
	Intents   []IntentWithInitialStatus
}

func (i CreateChargesInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	for idx, intent := range i.Intents {
		if err := intent.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("intent [%d]: %w", idx, err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func validateExpands(expands meta.Expands) error {
	if err := expands.Validate(); err != nil {
		return err
	}

	if expands.Has(meta.ExpandDetailedLines) && !expands.Has(meta.ExpandRealizations) {
		return fmt.Errorf("%q requires %q", meta.ExpandDetailedLines, meta.ExpandRealizations)
	}

	if expands.Has(meta.ExpandDeletedRealizations) && !expands.Has(meta.ExpandRealizations) {
		return fmt.Errorf("%q requires %q", meta.ExpandDeletedRealizations, meta.ExpandRealizations)
	}

	return nil
}
