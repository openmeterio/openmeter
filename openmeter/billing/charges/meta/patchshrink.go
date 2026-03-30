package meta

import (
	"errors"
	"fmt"
	"time"

	"github.com/qmuntal/stateless"
)

var (
	_             Patch = (*PatchShrink)(nil)
	TriggerShrink       = stateless.Trigger("shrink")
)

type PatchShrink struct {
	NewServicePeriodTo     time.Time
	NewFullServicePeriodTo time.Time
	NewBillingPeriodTo     time.Time
}

func (p PatchShrink) Trigger() stateless.Trigger {
	return TriggerShrink
}

func (p PatchShrink) TriggerParams() any {
	return p
}

func (p PatchShrink) Validate() error {
	if p.NewServicePeriodTo.IsZero() {
		return fmt.Errorf("new service period to is required")
	}

	if p.NewFullServicePeriodTo.IsZero() {
		return fmt.Errorf("new full service period to is required")
	}

	if p.NewBillingPeriodTo.IsZero() {
		return fmt.Errorf("new billing period to is required")
	}

	return nil
}

func (p PatchShrink) ValidateWith(intent Intent) error {
	var errs []error

	if err := p.Validate(); err != nil {
		errs = append(errs, err)
	}

	if p.NewServicePeriodTo.After(intent.ServicePeriod.To) {
		errs = append(errs, fmt.Errorf("new service period to must be less than or equal to existing service period to"))
	}

	if p.NewFullServicePeriodTo.After(intent.FullServicePeriod.To) {
		errs = append(errs, fmt.Errorf("new full service period to must be less than or equal to existing full service period to"))
	}

	if p.NewBillingPeriodTo.After(intent.BillingPeriod.To) {
		errs = append(errs, fmt.Errorf("new billing period to must be less than or equal to existing billing period to"))
	}

	return errors.Join(errs...)
}
