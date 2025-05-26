package productcatalog

import (
	"errors"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

var (
	_ models.Validator                        = (*EffectivePeriod)(nil)
	_ models.Equaler[EffectivePeriod]         = (*EffectivePeriod)(nil)
	_ models.CustomValidator[EffectivePeriod] = (*EffectivePeriod)(nil)
)

// EffectivePeriod describes lifecycle of resource based on the time period defined by it.
type EffectivePeriod struct {
	// EffectiveFrom defines the time from the Plan or Addon becomes active.
	EffectiveFrom *time.Time `json:"effectiveFrom,omitempty"`

	// EffectiveTo defines the time from the Plan or Addon becomes archived.
	EffectiveTo *time.Time `json:"effectiveTo,omitempty"`
}

func (p EffectivePeriod) ValidateWith(v ...models.ValidatorFunc[EffectivePeriod]) error {
	return models.Validate(p, v...)
}

func (p EffectivePeriod) AsPeriod() timeutil.OpenPeriod {
	return timeutil.OpenPeriod{
		From: p.EffectiveFrom,
		To:   p.EffectiveTo,
	}
}

func (p EffectivePeriod) Validate() error {
	return p.ValidateWith(ValidateEffectivePeriod())
}

func ValidateEffectivePeriod() models.ValidatorFunc[EffectivePeriod] {
	return func(p EffectivePeriod) error {
		var errs []error

		from := lo.FromPtr(p.EffectiveFrom)
		to := lo.FromPtr(p.EffectiveTo)

		if !from.IsZero() && !to.IsZero() && from.After(to) {
			errs = append(errs, ErrEffectivePeriodFromAfterTo.
				WithAttrs(models.Attributes{
					"effectiveFrom": p.EffectiveFrom,
					"effectiveTo":   p.EffectiveTo,
				}))
		}

		if from.IsZero() && !to.IsZero() {
			errs = append(errs, ErrEffectivePeriodFromNotSet.
				WithAttrs(models.Attributes{
					"effectiveFrom": nil,
					"effectiveTo":   p.EffectiveTo,
				}))
		}

		return errors.Join(errs...)
	}
}

// Equal returns true if the two EffectivePeriod objects are equal.
func (p EffectivePeriod) Equal(o EffectivePeriod) bool {
	return lo.FromPtr(p.EffectiveFrom).Equal(lo.FromPtr(o.EffectiveFrom)) &&
		lo.FromPtr(p.EffectiveTo).Equal(lo.FromPtr(o.EffectiveTo))
}
