package creditpurchase

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Service interface {
	CreditPurchaseService
	ExternalPaymentLifecycle
	InvoicePaymentLifecycle
}

type CreditPurchaseService interface {
	// Create creates one credit-purchase charge. Credit purchases are a separate
	// lifecycle from flat-fee/usage-based overrides and intentionally accept a
	// single intent at a time.
	Create(ctx context.Context, input CreateInput) (ChargeWithGatheringLine, error)

	// GetByIDs loads credit-purchase charges for payment and grant lifecycle
	// checks; credit-purchase charges do not participate in intent overrides.
	GetByIDs(ctx context.Context, input GetByIDsInput) ([]Charge, error)
	// List returns credit-purchase charges for customer/API views, independent
	// from invoice-backed flat-fee and usage-based line-engine ownership.
	List(ctx context.Context, input ListChargesInput) (pagination.Result[Charge], error)
	// ListFundedCreditActivities reports grant-side activity funded by credit
	// purchases, not invoice-line override state.
	ListFundedCreditActivities(ctx context.Context, input ListFundedCreditActivitiesInput) (ListFundedCreditActivitiesResult, error)
	// MarkVoided records the void time on the charge row. Callers run it in the
	// same transaction as the ledger void booking.
	MarkVoided(ctx context.Context, input MarkVoidedInput) (ChargeBase, error)
}

type ChargeWithGatheringLine struct {
	Charge                Charge
	GatheringLineToCreate *billing.GatheringLine
}

type ExternalPaymentLifecycle interface {
	// HandleExternalPaymentAuthorized records authorization for externally paid
	// credit purchases before settlement grants credits.
	HandleExternalPaymentAuthorized(ctx context.Context, charge Charge) (Charge, error)
	// HandleExternalPaymentSettled finalizes externally paid credit purchases
	// and funds the related credit grant.
	HandleExternalPaymentSettled(ctx context.Context, charge Charge) (Charge, error)
}

type InvoicePaymentLifecycle interface {
	// PostInvoicePaymentAuthorized records authorization for invoice-backed
	// credit purchases after billing confirms the standard line payment.
	PostInvoicePaymentAuthorized(ctx context.Context, charge Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error
	// PostInvoicePaymentSettled finalizes invoice-backed credit purchases and
	// funds credits after the standard line payment settles.
	PostInvoicePaymentSettled(ctx context.Context, charge Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error
	// PostInvoiceDraftCreated attaches invoice-backed credit purchases to their
	// persisted standard invoice line before payment lifecycle callbacks run.
	PostInvoiceDraftCreated(ctx context.Context, charge Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error
}

type CreateInput struct {
	Namespace string
	Intent    Intent
}

func (i CreateInput) Validate() error {
	var errs []error
	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
