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
	// Create creates a new credit purchase charge. It can only handle a single intent at a time
	// as based on current state of credits we are not going to create multiple credit purchases at once.
	Create(ctx context.Context, input CreateInput) (ChargeWithGatheringLine, error)

	GetByIDs(ctx context.Context, input GetByIDsInput) ([]Charge, error)
	List(ctx context.Context, input ListChargesInput) (pagination.Result[Charge], error)
}

type ChargeWithGatheringLine struct {
	Charge                Charge
	GatheringLineToCreate *billing.GatheringLine
}

type ExternalPaymentLifecycle interface {
	HandleExternalPaymentAuthorized(ctx context.Context, charge Charge) (Charge, error)
	HandleExternalPaymentSettled(ctx context.Context, charge Charge) (Charge, error)
}

type InvoicePaymentLifecycle interface {
	PostInvoicePaymentAuthorized(ctx context.Context, charge Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error
	PostInvoicePaymentSettled(ctx context.Context, charge Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error
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
