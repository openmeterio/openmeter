package payment

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

var _ models.Validator = (*ExternalCreateInput)(nil)

type ExternalCreateInput struct {
	Base

	Namespace string `json:"namespace"`
}

func (i ExternalCreateInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace is required"))
	}

	if err := i.Base.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("payment settlement base: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type External struct {
	Payment
}

func (r External) ErrorAttributes() models.Attributes {
	return models.Attributes{
		PaymentSettlementStatusAttributeKey: string(r.Status),
		PaymentSettlementTypeAttributeKey:   string(PaymentSettlementTypeExternal),
		paymentSettlementIDAttributeKey:     r.ID,
	}
}

type ExternalMixin = Mixin

func CreateExternal[T Creator[T]](creator Creator[T], payment ExternalCreateInput) T {
	return Create(creator, payment.Namespace, payment.Base)
}

func MapExternalFromDB(dbEntity Getter) External {
	payment := mapPaymentFromDB(dbEntity)
	return External{
		Payment: payment,
	}
}

func UpdateExternal[T Updater[T]](updater Updater[T], in External) T {
	return Update(updater, in.Payment)
}
