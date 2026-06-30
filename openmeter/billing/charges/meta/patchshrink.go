package meta

import (
	"errors"
	"fmt"
	"time"

	"github.com/qmuntal/stateless"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	_             Patch = (*PatchShrink)(nil)
	TriggerShrink       = stateless.Trigger("shrink")
)

type PatchShrink struct {
	changeSource           billing.ChangeSource
	newServicePeriodTo     time.Time
	newFullServicePeriodTo time.Time
	newBillingPeriodTo     time.Time
	newInvoiceAt           time.Time
}

type NewPatchShrinkInput struct {
	ChangeSource           billing.ChangeSource
	NewServicePeriodTo     time.Time
	NewFullServicePeriodTo time.Time
	NewBillingPeriodTo     time.Time
	NewInvoiceAt           time.Time
}

func (i NewPatchShrinkInput) Validate() error {
	if err := i.ChangeSource.Require(billing.ChangeSourceSystem); err != nil {
		return fmt.Errorf("change source: %w", err)
	}

	if i.NewServicePeriodTo.IsZero() {
		return models.NewGenericValidationError(fmt.Errorf("new service period to is required"))
	}

	if i.NewFullServicePeriodTo.IsZero() {
		return models.NewGenericValidationError(fmt.Errorf("new full service period to is required"))
	}

	if i.NewBillingPeriodTo.IsZero() {
		return models.NewGenericValidationError(fmt.Errorf("new billing period to is required"))
	}

	if i.NewInvoiceAt.IsZero() {
		return models.NewGenericValidationError(fmt.Errorf("new invoice at is required"))
	}

	return nil
}

func NewPatchShrink(input NewPatchShrinkInput) (PatchShrink, error) {
	if err := input.Validate(); err != nil {
		return PatchShrink{}, err
	}

	patch := PatchShrink{
		changeSource:           input.ChangeSource,
		newServicePeriodTo:     NormalizeTimestamp(input.NewServicePeriodTo),
		newFullServicePeriodTo: NormalizeTimestamp(input.NewFullServicePeriodTo),
		newBillingPeriodTo:     NormalizeTimestamp(input.NewBillingPeriodTo),
		newInvoiceAt:           NormalizeTimestamp(input.NewInvoiceAt),
	}
	if err := patch.Validate(); err != nil {
		return PatchShrink{}, err
	}

	return patch, nil
}

func (p PatchShrink) GetChangeSource() billing.ChangeSource {
	return p.changeSource
}

func (p PatchShrink) GetTargetLayer(LayeredIntentReader) (ChangeTarget, error) {
	if err := p.GetChangeSource().Require(billing.ChangeSourceSystem); err != nil {
		return "", fmt.Errorf("change source: %w", err)
	}

	return ChangeTargetBase, nil
}

func (p PatchShrink) GetNewServicePeriodTo() time.Time {
	return p.newServicePeriodTo
}

func (p PatchShrink) GetNewFullServicePeriodTo() time.Time {
	return p.newFullServicePeriodTo
}

func (p PatchShrink) GetNewBillingPeriodTo() time.Time {
	return p.newBillingPeriodTo
}

func (p PatchShrink) GetNewInvoiceAt() time.Time {
	return p.newInvoiceAt
}

func (p PatchShrink) Op() PatchType {
	return PatchTypeShrink
}

func (p PatchShrink) Trigger() stateless.Trigger {
	return TriggerShrink
}

func (p PatchShrink) Validate() error {
	if err := p.GetChangeSource().Require(billing.ChangeSourceSystem); err != nil {
		return fmt.Errorf("change source: %w", err)
	}

	if p.GetNewServicePeriodTo().IsZero() {
		return models.NewGenericValidationError(fmt.Errorf("new service period to is required"))
	}

	if p.GetNewFullServicePeriodTo().IsZero() {
		return models.NewGenericValidationError(fmt.Errorf("new full service period to is required"))
	}

	if p.GetNewBillingPeriodTo().IsZero() {
		return models.NewGenericValidationError(fmt.Errorf("new billing period to is required"))
	}

	if p.GetNewInvoiceAt().IsZero() {
		return models.NewGenericValidationError(fmt.Errorf("new invoice at is required"))
	}

	return nil
}

func (p PatchShrink) ValidateWith(intent IntentMutableFields) error {
	var errs []error

	if err := p.Validate(); err != nil {
		errs = append(errs, err)
	}

	if !p.GetNewServicePeriodTo().Before(intent.ServicePeriod.To) {
		errs = append(errs, fmt.Errorf("new service period to must be less than existing service period to"))
	}

	if !p.GetNewServicePeriodTo().After(intent.ServicePeriod.From) {
		errs = append(errs, fmt.Errorf("new service period to must be greater than existing service period from"))
	}

	if p.GetNewFullServicePeriodTo().After(intent.FullServicePeriod.To) {
		errs = append(errs, fmt.Errorf("new full service period to must be less than or equal to existing full service period to"))
	}

	if !p.GetNewFullServicePeriodTo().After(intent.FullServicePeriod.From) {
		errs = append(errs, fmt.Errorf("new full service period to must be greater than existing full service period from"))
	}

	if p.GetNewBillingPeriodTo().After(intent.BillingPeriod.To) {
		errs = append(errs, fmt.Errorf("new billing period to must be less than or equal to existing billing period to"))
	}

	if !p.GetNewBillingPeriodTo().After(intent.BillingPeriod.From) {
		errs = append(errs, fmt.Errorf("new billing period to must be greater than existing billing period from"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
