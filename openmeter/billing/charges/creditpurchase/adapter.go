package creditpurchase

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	entutils.TxCreator

	UpdateCharge(ctx context.Context, charge Charge) (Charge, error)
	CreateCharge(ctx context.Context, in CreateChargeInput) (Charge, error)
	GetByIDs(ctx context.Context, ids GetByIDsInput) ([]Charge, error)

	CreateExternalPayment(ctx context.Context, chargeID meta.ChargeID, payment payment.ExternalCreateInput) (payment.External, error)
	UpdateExternalPayment(ctx context.Context, payment payment.External) (payment.External, error)
}

type CreateChargeInput struct {
	Namespace string
	Intent    Intent
}

func (i CreateChargeInput) Validate() error {
	var errs []error
	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	return errors.Join(errs...)
}
