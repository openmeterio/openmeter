package costbasis

import (
	"errors"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Mode string

const (
	ModeDynamic Mode = "dynamic"
	ModePinned  Mode = "pinned"
	ModeManual  Mode = "manual"
)

func (Mode) Values() []string {
	return []string{
		string(ModeDynamic),
		string(ModePinned),
		string(ModeManual),
	}
}

func (m Mode) Validate() error {
	if !slices.Contains(m.Values(), string(m)) {
		return models.NewGenericValidationError(fmt.Errorf("invalid cost basis mode: %s", m))
	}

	return nil
}

type Intent struct {
	kind Mode

	dynamic *DynamicIntent
	pinned  *PinnedIntent
	manual  *ManualIntent
}

func NewIntent[T DynamicIntent | PinnedIntent | ManualIntent](in T) Intent {
	switch v := any(in).(type) {
	case DynamicIntent:
		return Intent{kind: ModeDynamic, dynamic: &v}
	case PinnedIntent:
		return Intent{kind: ModePinned, pinned: &v}
	case ManualIntent:
		return Intent{kind: ModeManual, manual: &v}
	}

	return Intent{}
}

func (i Intent) Kind() Mode {
	return i.kind
}

func (i Intent) GetFiatCurrency() *currencyx.FiatCurrency {
	switch i.kind {
	case ModeDynamic:
		if i.dynamic != nil {
			return i.dynamic.FiatCurrency
		}
	case ModePinned:
		if i.pinned != nil {
			return i.pinned.FiatCurrency
		}
	case ModeManual:
		if i.manual != nil {
			return i.manual.FiatCurrency
		}
	}

	return nil
}

func (i Intent) Clone() Intent {
	out := i

	if i.dynamic != nil {
		dynamic := *i.dynamic
		out.dynamic = &dynamic
	}

	if i.pinned != nil {
		pinned := *i.pinned
		out.pinned = &pinned
	}

	if i.manual != nil {
		manual := *i.manual
		out.manual = &manual
	}

	return out
}

func (i Intent) Validate() error {
	switch i.kind {
	case ModeDynamic:
		return i.dynamic.Validate()
	case ModePinned:
		return i.pinned.Validate()
	case ModeManual:
		return i.manual.Validate()
	default:
		return models.NewGenericValidationError(fmt.Errorf("invalid intent kind: %s", i.kind))
	}
}

func (i Intent) AsPinned() (PinnedIntent, error) {
	if i.kind != ModePinned {
		return PinnedIntent{}, models.NewGenericValidationError(fmt.Errorf("intent is not a pinned intent"))
	}

	if i.pinned == nil {
		return PinnedIntent{}, models.NewGenericValidationError(fmt.Errorf("pinned intent is nil"))
	}

	return *i.pinned, nil
}

func (i Intent) AsManual() (ManualIntent, error) {
	if i.kind != ModeManual {
		return ManualIntent{}, models.NewGenericValidationError(fmt.Errorf("intent is not a manual intent"))
	}

	if i.manual == nil {
		return ManualIntent{}, models.NewGenericValidationError(fmt.Errorf("manual intent is nil"))
	}

	return *i.manual, nil
}

func (i Intent) AsDynamic() (DynamicIntent, error) {
	if i.kind != ModeDynamic {
		return DynamicIntent{}, models.NewGenericValidationError(fmt.Errorf("intent is not a dynamic intent"))
	}

	if i.dynamic == nil {
		return DynamicIntent{}, models.NewGenericValidationError(fmt.Errorf("dynamic intent is nil"))
	}

	return *i.dynamic, nil
}

// GetFiatCurrency returns the fiat currency in which the custom-currency cost
// basis is expressed, regardless of the selected resolution mode.
func (i Intent) GetFiatCurrency() (*currencyx.FiatCurrency, error) {
	switch i.kind {
	case ModeDynamic:
		intent, err := i.AsDynamic()
		if err != nil {
			return nil, err
		}

		return intent.FiatCurrency, nil
	case ModePinned:
		intent, err := i.AsPinned()
		if err != nil {
			return nil, err
		}

		return intent.FiatCurrency, nil
	case ModeManual:
		intent, err := i.AsManual()
		if err != nil {
			return nil, err
		}

		return intent.FiatCurrency, nil
	default:
		return nil, models.NewGenericValidationError(fmt.Errorf("invalid intent kind: %s", i.kind))
	}
}

type DynamicIntent struct {
	FiatCurrency *currencyx.FiatCurrency
}

func (i DynamicIntent) Validate() error {
	var errs []error

	if err := i.FiatCurrency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("fiat currency: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type PinnedIntent struct {
	FiatCurrency        *currencyx.FiatCurrency
	CurrencyCostBasisID string
}

func (i PinnedIntent) Validate() error {
	var errs []error

	if err := i.FiatCurrency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("fiat currency: %w", err))
	}

	if i.CurrencyCostBasisID == "" {
		errs = append(errs, fmt.Errorf("currency cost basis id is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ManualIntent struct {
	FiatCurrency *currencyx.FiatCurrency
	Rate         alpacadecimal.Decimal
}

func (i ManualIntent) Validate() error {
	var errs []error

	if err := i.FiatCurrency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("fiat currency: %w", err))
	}

	if !i.Rate.IsPositive() {
		errs = append(errs, fmt.Errorf("rate must be positive"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
