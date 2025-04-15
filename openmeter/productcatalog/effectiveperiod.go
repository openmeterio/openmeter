package productcatalog

import (
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

var (
	_ models.Validator                = (*EffectivePeriod)(nil)
	_ models.Equaler[EffectivePeriod] = (*EffectivePeriod)(nil)
)

// EffectivePeriod describes lifecycle of resource based on the time period defined by it.
type EffectivePeriod struct {
	// EffectiveFrom defines the time from the Plan or Addon becomes active.
	EffectiveFrom *time.Time `json:"effectiveFrom,omitempty"`

	// EffectiveTo defines the time from the Plan or Addon becomes archived.
	EffectiveTo *time.Time `json:"effectiveTo,omitempty"`
}

func (p EffectivePeriod) AsPeriod() timeutil.OpenPeriod {
	return timeutil.OpenPeriod{
		From: p.EffectiveFrom,
		To:   p.EffectiveTo,
	}
}

func (p EffectivePeriod) Validate() error {
	from := lo.FromPtr(p.EffectiveFrom)
	to := lo.FromPtr(p.EffectiveTo)

	if !from.IsZero() && !to.IsZero() && from.After(to) {
		return models.NewGenericValidationError(fmt.Errorf("invalid effective time range: to is before from"))
	}

	if from.IsZero() && !to.IsZero() {
		return models.NewGenericValidationError(fmt.Errorf("invalid effective time range: to is set while from is not"))
	}

	return nil
}

// Equal returns true if the two EffectivePeriod objects are equal.
func (p EffectivePeriod) Equal(o EffectivePeriod) bool {
	return lo.FromPtr(p.EffectiveFrom).Equal(lo.FromPtr(o.EffectiveFrom)) &&
		lo.FromPtr(p.EffectiveTo).Equal(lo.FromPtr(o.EffectiveTo))
}
