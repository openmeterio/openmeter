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
	target                 ChangeTarget
	newServicePeriodTo     time.Time
	newFullServicePeriodTo time.Time
	newBillingPeriodTo     time.Time
	newInvoiceAt           time.Time
}

type NewPatchExtendInput struct {
	Target                 ChangeTarget
	NewServicePeriodTo     time.Time
	NewFullServicePeriodTo time.Time
	NewBillingPeriodTo     time.Time
	NewInvoiceAt           time.Time
}

func (i NewPatchExtendInput) Validate() error {
	var errs []error

	if err := i.Target.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("target: %w", err))
	}

	if i.NewServicePeriodTo.IsZero() {
		errs = append(errs, errors.New("new service period to is required"))
	}

	if i.NewFullServicePeriodTo.IsZero() {
		errs = append(errs, errors.New("new full service period to is required"))
	}

	if i.NewBillingPeriodTo.IsZero() {
		errs = append(errs, errors.New("new billing period to is required"))
	}

	if i.NewInvoiceAt.IsZero() {
		errs = append(errs, errors.New("new invoice at is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func NewPatchExtend(input NewPatchExtendInput) (PatchExtend, error) {
	if err := input.Validate(); err != nil {
		return PatchExtend{}, err
	}

	var patch PatchExtend
	patch.SetTarget(input.Target)
	patch.SetNewServicePeriodTo(input.NewServicePeriodTo)
	patch.SetNewFullServicePeriodTo(input.NewFullServicePeriodTo)
	patch.SetNewBillingPeriodTo(input.NewBillingPeriodTo)
	patch.SetNewInvoiceAt(input.NewInvoiceAt)
	return patch, nil
}

func (p *PatchExtend) SetTarget(v ChangeTarget) {
	p.target = v
}

func (p PatchExtend) GetTarget() ChangeTarget {
	return p.target
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

func (p *PatchExtend) SetNewInvoiceAt(v time.Time) {
	p.newInvoiceAt = NormalizeTimestamp(v)
}

func (p PatchExtend) GetNewInvoiceAt() time.Time {
	return p.newInvoiceAt
}

func (p PatchExtend) Op() PatchType {
	return PatchTypeExtend
}

func (p PatchExtend) Trigger() stateless.Trigger {
	return TriggerExtend
}

func (p PatchExtend) Validate() error {
	var errs []error

	if err := p.GetTarget().Validate(); err != nil {
		errs = append(errs, fmt.Errorf("target: %w", err))
	}

	if p.GetNewServicePeriodTo().IsZero() {
		errs = append(errs, errors.New("new service period to is required"))
	}

	if p.GetNewFullServicePeriodTo().IsZero() {
		errs = append(errs, errors.New("new full service period to is required"))
	}

	if p.GetNewBillingPeriodTo().IsZero() {
		errs = append(errs, errors.New("new billing period to is required"))
	}

	if p.GetNewInvoiceAt().IsZero() {
		errs = append(errs, errors.New("new invoice at is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (p PatchExtend) ValidateWith(intent IntentMutableFields) error {
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
