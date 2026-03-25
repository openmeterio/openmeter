package meta

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	RegisterCharges(ctx context.Context, in RegisterChargesInput) error
	DeleteRegisteredCharge(ctx context.Context, in DeleteRegisteredChargeInput) error

	entutils.TxCreator
}

type RegisterChargesInput struct {
	Namespace string
	Type      ChargeType

	Charges []IDWithUniqueReferenceID
}

func (i RegisterChargesInput) Validate() error {
	var errs []error
	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if err := i.Type.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("type: %w", err))
	}

	for idx, charge := range i.Charges {
		if charge.ID == "" {
			errs = append(errs, fmt.Errorf("charge [%d]: ID is required", idx))
		}
	}
	return errors.Join(errs...)
}

type IDWithUniqueReferenceID struct {
	ID                string
	UniqueReferenceID *string
}

type DeleteRegisteredChargeInput = ChargeID
