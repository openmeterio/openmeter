package meta

import (
	"errors"
	"fmt"
	"time"

	"github.com/qmuntal/stateless"
)

var (
	_             Patch = (*PatchExtend)(nil)
	TriggerExtend       = stateless.Trigger("extend")
)

type PatchExtend struct {
	NewServicePeriodTo     time.Time
	NewFullServicePeriodTo time.Time
	NewBillingPeriodTo     time.Time
}

func (p PatchExtend) Trigger() stateless.Trigger {
	return TriggerExtend
}

func (p PatchExtend) TriggerParams() any {
	return p
}

func (p PatchExtend) Validate() error {
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

func (p PatchExtend) ValidateWith(intent Intent) error {
	var errs []error

	if err := p.Validate(); err != nil {
		errs = append(errs, err)
	}

	if p.NewServicePeriodTo.Before(intent.ServicePeriod.To) {
		errs = append(errs, fmt.Errorf("new service period to must be greater than or equal to existing service period to"))
	}

	if p.NewFullServicePeriodTo.Before(intent.FullServicePeriod.To) {
		errs = append(errs, fmt.Errorf("new full service period to must be greater than or equal to existing full service period to"))
	}

	if p.NewBillingPeriodTo.Before(intent.BillingPeriod.To) {
		errs = append(errs, fmt.Errorf("new billing period to must be greater than or equal to existing billing period to"))
	}

	return errors.Join(errs...)
}
