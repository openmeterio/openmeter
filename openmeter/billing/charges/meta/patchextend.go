package meta

import (
	"errors"
	"fmt"
	"time"

	"github.com/qmuntal/stateless"

	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	_             Patch = (*PatchExtend)(nil)
	TriggerExtend       = stateless.Trigger("extend")
)

type PatchExtend struct {
	newServicePeriodTo     time.Time
	newFullServicePeriodTo time.Time
	newBillingPeriodTo     time.Time
}

type NewPatchExtendInput struct {
	NewServicePeriodTo     time.Time
	NewFullServicePeriodTo time.Time
	NewBillingPeriodTo     time.Time
}

func (i NewPatchExtendInput) Validate() error {
	if i.NewServicePeriodTo.IsZero() {
		return models.NewGenericValidationError(fmt.Errorf("new service period to is required"))
	}

	if i.NewFullServicePeriodTo.IsZero() {
		return models.NewGenericValidationError(fmt.Errorf("new full service period to is required"))
	}

	if i.NewBillingPeriodTo.IsZero() {
		return models.NewGenericValidationError(fmt.Errorf("new billing period to is required"))
	}

	return nil
}

func NewPatchExtend(input NewPatchExtendInput) (PatchExtend, error) {
	if err := input.Validate(); err != nil {
		return PatchExtend{}, err
	}

	var patch PatchExtend
	patch.SetNewServicePeriodTo(input.NewServicePeriodTo)
	patch.SetNewFullServicePeriodTo(input.NewFullServicePeriodTo)
	patch.SetNewBillingPeriodTo(input.NewBillingPeriodTo)
	return patch, nil
}

func (p *PatchExtend) SetNewServicePeriodTo(v time.Time) {
	p.newServicePeriodTo = NormalizeTimestamp(v)
}

func (p PatchExtend) GetNewServicePeriodTo() time.Time {
	return p.newServicePeriodTo
}

func (p *PatchExtend) SetNewFullServicePeriodTo(v time.Time) {
	p.newFullServicePeriodTo = NormalizeTimestamp(v)
}

func (p PatchExtend) GetNewFullServicePeriodTo() time.Time {
	return p.newFullServicePeriodTo
}

func (p *PatchExtend) SetNewBillingPeriodTo(v time.Time) {
	p.newBillingPeriodTo = NormalizeTimestamp(v)
}

func (p PatchExtend) GetNewBillingPeriodTo() time.Time {
	return p.newBillingPeriodTo
}

func (p PatchExtend) Trigger() stateless.Trigger {
	return TriggerExtend
}

func (p PatchExtend) TriggerParams() any {
	return p
}

func (p PatchExtend) Validate() error {
	if p.GetNewServicePeriodTo().IsZero() {
		return models.NewGenericValidationError(fmt.Errorf("new service period to is required"))
	}

	if p.GetNewFullServicePeriodTo().IsZero() {
		return models.NewGenericValidationError(fmt.Errorf("new full service period to is required"))
	}

	if p.GetNewBillingPeriodTo().IsZero() {
		return models.NewGenericValidationError(fmt.Errorf("new billing period to is required"))
	}

	return nil
}

func (p PatchExtend) ValidateWith(intent Intent) error {
	var errs []error

	if err := p.Validate(); err != nil {
		errs = append(errs, err)
	}

	if !p.GetNewServicePeriodTo().After(intent.ServicePeriod.To) {
		errs = append(errs, fmt.Errorf("new service period to must be greater than existing service period to"))
	}

	if p.GetNewFullServicePeriodTo().Before(intent.FullServicePeriod.To) {
		errs = append(errs, fmt.Errorf("new full service period to must be greater than or equal to existing full service period to"))
	}

	if p.GetNewBillingPeriodTo().Before(intent.BillingPeriod.To) {
		errs = append(errs, fmt.Errorf("new billing period to must be greater than or equal to existing billing period to"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
