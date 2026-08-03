package meta

import (
	"errors"
	"fmt"
	"time"

	"github.com/qmuntal/stateless"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ Patch = (*PatchShrinkToRealizedPeriod)(nil)

type PatchShrinkToRealizedPeriod struct {
	changeSource        billing.ChangeSource
	newServicePeriodEnd time.Time
}

type NewPatchShrinkToRealizedPeriodInput struct {
	ChangeSource        billing.ChangeSource
	NewServicePeriodEnd time.Time
}

func (i NewPatchShrinkToRealizedPeriodInput) Validate() error {
	var errs []error

	if err := i.ChangeSource.Require(billing.ChangeSourceAPIRequest); err != nil {
		errs = append(errs, fmt.Errorf("change source: %w", err))
	}

	if i.NewServicePeriodEnd.IsZero() {
		errs = append(errs, fmt.Errorf("new service period end is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func NewPatchShrinkToRealizedPeriod(input NewPatchShrinkToRealizedPeriodInput) (PatchShrinkToRealizedPeriod, error) {
	if err := input.Validate(); err != nil {
		return PatchShrinkToRealizedPeriod{}, err
	}

	patch := PatchShrinkToRealizedPeriod{
		changeSource:        input.ChangeSource,
		newServicePeriodEnd: NormalizeTimestamp(input.NewServicePeriodEnd),
	}
	if err := patch.Validate(); err != nil {
		return PatchShrinkToRealizedPeriod{}, err
	}

	return patch, nil
}

func (p PatchShrinkToRealizedPeriod) GetChangeSource() billing.ChangeSource {
	return p.changeSource
}

func (p PatchShrinkToRealizedPeriod) GetTargetLayer(intent LayeredIntentReader) (ChangeTarget, error) {
	if err := p.GetChangeSource().Require(billing.ChangeSourceAPIRequest); err != nil {
		return "", fmt.Errorf("change source: %w", err)
	}

	return apiPatchTargetLayer(intent)
}

func (p PatchShrinkToRealizedPeriod) GetNewServicePeriodEnd() time.Time {
	return p.newServicePeriodEnd
}

func (p PatchShrinkToRealizedPeriod) Op() PatchType {
	return PatchTypeShrinkToRealizedPeriod
}

func (p PatchShrinkToRealizedPeriod) Trigger() stateless.Trigger {
	return TriggerShrinkToRealizedPeriod
}

func (p PatchShrinkToRealizedPeriod) Validate() error {
	return NewPatchShrinkToRealizedPeriodInput{
		ChangeSource:        p.GetChangeSource(),
		NewServicePeriodEnd: p.GetNewServicePeriodEnd(),
	}.Validate()
}

func (p PatchShrinkToRealizedPeriod) ValidateWith(intent IntentMutableFields) error {
	var errs []error

	if err := p.Validate(); err != nil {
		errs = append(errs, err)
	}

	if !p.GetNewServicePeriodEnd().Before(intent.ServicePeriod.To) {
		errs = append(errs, fmt.Errorf("new service period end must be less than existing service period to"))
	}

	if !p.GetNewServicePeriodEnd().After(intent.ServicePeriod.From) {
		errs = append(errs, fmt.Errorf("new service period end must be greater than existing service period from"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
