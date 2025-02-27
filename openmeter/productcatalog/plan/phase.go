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
		errs = append(errs, errors.New("namespace must not be empty"))
	}

	if m.ID == "" {
		errs = append(errs, errors.New("unique identifier must not be empty"))
	}

	if m.PlanID == "" {
		errs = append(errs, errors.New("planID must not be empty"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ManagedPhase interface {
	ManagedFields() PhaseManagedFields
}

var (
	_ models.Validator      = (*Phase)(nil)
	_ models.Equaler[Phase] = (*Phase)(nil)
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

func (p Phase) Validate() error {
	var errs []error

	if err := p.PhaseManagedFields.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := p.Phase.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (p Phase) AsProductCatalogPhase() productcatalog.Phase {
	return p.Phase
}
