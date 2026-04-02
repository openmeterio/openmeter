package meta

import (
	"errors"
	"fmt"
	"time"

	"github.com/qmuntal/stateless"

	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	_             Patch = (*PatchShrink)(nil)
	TriggerShrink       = stateless.Trigger("shrink")
)

type PatchShrink struct {
	newServicePeriodTo     time.Time
	newFullServicePeriodTo time.Time
	newBillingPeriodTo     time.Time
}

type NewPatchShrinkInput struct {
	NewServicePeriodTo     time.Time
	NewFullServicePeriodTo time.Time
	NewBillingPeriodTo     time.Time
}

func (i NewPatchShrinkInput) Validate() error {
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

func NewPatchShrink(input NewPatchShrinkInput) (PatchShrink, error) {
	if err := input.Validate(); err != nil {
		return PatchShrink{}, err
	}

	var patch PatchShrink
	patch.SetNewServicePeriodTo(input.NewServicePeriodTo)
	patch.SetNewFullServicePeriodTo(input.NewFullServicePeriodTo)
	patch.SetNewBillingPeriodTo(input.NewBillingPeriodTo)
	return patch, nil
}

func (p *PatchShrink) SetNewServicePeriodTo(v time.Time) {
	p.newServicePeriodTo = NormalizeTimestamp(v)
}

func (p PatchShrink) GetNewServicePeriodTo() time.Time {
	return p.newServicePeriodTo
}

func (p *PatchShrink) SetNewFullServicePeriodTo(v time.Time) {
	p.newFullServicePeriodTo = NormalizeTimestamp(v)
}

func (p PatchShrink) GetNewFullServicePeriodTo() time.Time {
	return p.newFullServicePeriodTo
}

func (p *PatchShrink) SetNewBillingPeriodTo(v time.Time) {
	p.newBillingPeriodTo = NormalizeTimestamp(v)
}

func (p PatchShrink) GetNewBillingPeriodTo() time.Time {
	return p.newBillingPeriodTo
}

func (p PatchShrink) Trigger() stateless.Trigger {
	return TriggerShrink
}

func (p PatchShrink) TriggerParams() any {
	return p
}

func (p PatchShrink) Validate() error {
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

func (p PatchShrink) ValidateWith(intent Intent) error {
	var errs []error

	if err := p.Validate(); err != nil {
		errs = append(errs, err)
	}

	if p.GetNewServicePeriodTo().After(intent.ServicePeriod.To) {
		errs = append(errs, fmt.Errorf("new service period to must be less than or equal to existing service period to"))
	}

	if p.GetNewFullServicePeriodTo().After(intent.FullServicePeriod.To) {
		errs = append(errs, fmt.Errorf("new full service period to must be less than or equal to existing full service period to"))
	}

	if p.GetNewBillingPeriodTo().After(intent.BillingPeriod.To) {
		errs = append(errs, fmt.Errorf("new billing period to must be less than or equal to existing billing period to"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
