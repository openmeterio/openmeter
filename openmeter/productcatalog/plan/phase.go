package plan

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	_ models.Validator                   = (*PhaseManagedFields)(nil)
	_ models.Equaler[PhaseManagedFields] = (*PhaseManagedFields)(nil)
)

type PhaseManagedFields struct {
	models.ManagedModel
	models.NamespacedID

	// PlanID
	PlanID string `json:"planId"`
}

func (m PhaseManagedFields) Equal(v PhaseManagedFields) bool {
	if m.Namespace != v.Namespace {
		return false
	}

	if m.ID != v.ID {
		return false
	}

	return m.PlanID == v.PlanID
}

func (m PhaseManagedFields) Validate() error {
	var errs []error

	if m.Namespace == "" {
		errs = append(errs, productcatalog.ErrNamespaceEmpty)
	}

	if m.ID == "" {
		errs = append(errs, productcatalog.ErrIDEmpty)
	}

	if m.PlanID == "" {
		errs = append(errs, errors.New("managed plan phase must have plan reference set"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ManagedPhase interface {
	ManagedFields() PhaseManagedFields
}

var (
	_ models.Validator              = (*Phase)(nil)
	_ models.Equaler[Phase]         = (*Phase)(nil)
	_ models.CustomValidator[Phase] = (*Phase)(nil)
)

type Phase struct {
	PhaseManagedFields
	productcatalog.Phase
}

func (p Phase) Equal(v Phase) bool {
	switch any(v).(type) {
	case Phase:
		vv := any(v).(Phase)

		if !p.PhaseManagedFields.Equal(vv.PhaseManagedFields) {
			return false
		}

		if p.PlanID != vv.PlanID {
			return false
		}

		return p.Phase.Equal(vv.Phase)
	case productcatalog.Phase:
		vv := any(v).(productcatalog.Phase)

		return p.Phase.Equal(vv)
	default:
		return false
	}
}

func (p Phase) ValidateWith(validators ...models.ValidatorFunc[Phase]) error {
	return models.Validate(p, validators...)
}

func (p Phase) Validate() error {
	return p.ValidateWith(
		ValidatePhaseManagedFields(),
		ValidatePhase(),
	)
}

func (p Phase) AsProductCatalogPhase() productcatalog.Phase {
	return p.Phase
}

func ValidatePhaseManagedFields() models.ValidatorFunc[Phase] {
	return func(p Phase) error {
		return p.PhaseManagedFields.Validate()
	}
}

func ValidatePhase() models.ValidatorFunc[Phase] {
	return func(p Phase) error {
		return p.Phase.Validate()
	}
}
