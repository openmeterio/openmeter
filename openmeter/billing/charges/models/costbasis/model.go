package costbasis

import (
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CreateInput struct {
	models.NamespacedID

	CurrencyID string
	Intent     Intent
	State      *State
}

func (i CreateInput) Validate() error {
	var errs []error

	if err := i.NamespacedID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.CurrencyID == "" {
		errs = append(errs, errors.New("currency ID is required"))
	}

	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	if i.State != nil {
		if err := i.State.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("state: %w", err))
		}
	}

	switch i.Intent.Kind() {
	case ModeDynamic:
		if i.State != nil && i.State.CostBasisID == nil {
			errs = append(errs, errors.New("resolved dynamic cost basis must reference a currency cost basis"))
		}
	case ModePinned:
		intent, err := i.Intent.AsPinned()
		if err != nil {
			errs = append(errs, err)
		} else if i.State == nil {
			errs = append(errs, errors.New("pinned cost basis state is required"))
		} else if i.State.CostBasisID == nil || *i.State.CostBasisID != intent.CurrencyCostBasisID {
			errs = append(errs, errors.New("pinned cost basis state must reference the intent cost basis"))
		}
	case ModeManual:
		intent, err := i.Intent.AsManual()
		if err != nil {
			errs = append(errs, err)
		} else if i.State != nil {
			if i.State.CostBasisID != nil {
				errs = append(errs, errors.New("manual cost basis state cannot reference a currency cost basis"))
			}

			if !i.State.CostBasis.Equal(intent.Rate) {
				errs = append(errs, errors.New("manual cost basis state must match the intent rate"))
			}
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type SetResolvedCostBasisInput struct {
	models.NamespacedID
	State State
}

func (i SetResolvedCostBasisInput) Validate() error {
	var errs []error

	if err := i.NamespacedID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := i.State.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("state: %w", err))
	}

	if i.State.CostBasisID == nil {
		errs = append(errs, errors.New("resolved currency cost basis ID is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type Setter[T any] interface {
	SetNillableResolvedCostBasisID(resolvedCostBasisID *string) T
	SetResolvedCostBasis(resolvedCostBasis alpacadecimal.Decimal) T
	SetResolvedAt(resolvedAt time.Time) T
}

func Set[T Setter[T]](setter T, state State) (T, error) {
	if err := state.Validate(); err != nil {
		var empty T
		return empty, err
	}

	return setter.
		SetNillableResolvedCostBasisID(state.CostBasisID).
		SetResolvedCostBasis(state.CostBasis).
		SetResolvedAt(state.ResolvedAt.UTC()), nil
}

type Creator[T any] interface {
	entutils.NamespaceMixinCreator[T]
	entutils.IDMixinCreator[T]
	Setter[T]

	SetMode(mode Mode) T
	SetFiatCurrency(fiatCurrency currencyx.FiatCode) T
	SetCurrencyCostBasisID(currencyCostBasisID string) T
	SetCurrencyID(currencyID string) T
	SetManualRate(manualRate alpacadecimal.Decimal) T
}

func Create[T Creator[T]](creator T, input CreateInput) (T, error) {
	if err := input.Validate(); err != nil {
		var empty T
		return empty, err
	}

	creator = creator.
		SetID(input.ID).
		SetNamespace(input.Namespace).
		SetMode(input.Intent.Kind()).
		SetCurrencyID(input.CurrencyID)
	state := input.State

	switch input.Intent.Kind() {
	case ModeDynamic:
		intent, err := input.Intent.AsDynamic()
		if err != nil {
			var empty T
			return empty, err
		}

		creator = creator.SetFiatCurrency(intent.FiatCurrency.GetFiatCode())
	case ModePinned:
		intent, err := input.Intent.AsPinned()
		if err != nil {
			var empty T
			return empty, err
		}

		creator = creator.
			SetFiatCurrency(intent.FiatCurrency.GetFiatCode()).
			SetCurrencyCostBasisID(intent.CurrencyCostBasisID)
	case ModeManual:
		intent, err := input.Intent.AsManual()
		if err != nil {
			var empty T
			return empty, err
		}

		creator = creator.
			SetFiatCurrency(intent.FiatCurrency.GetFiatCode()).
			SetManualRate(intent.Rate)

		if state == nil {
			state = &State{
				CostBasis:  intent.Rate,
				ResolvedAt: clock.Now().UTC(),
			}
		}
	}

	if state != nil {
		return Set(creator, *state)
	}

	return creator, nil
}

type Getter interface {
	entutils.NamespaceMixinGetter
	entutils.IDMixinGetter
	entutils.TimeMixinGetter

	GetMode() Mode
	GetFiatCurrency() currencyx.FiatCode
	GetCurrencyCostBasisID() *string
	GetResolvedCostBasisID() *string
	GetCurrencyID() string
	GetManualRate() *alpacadecimal.Decimal
	GetResolvedCostBasis() *alpacadecimal.Decimal
	GetResolvedAt() *time.Time
}

func Get(dbEntity Getter) (CostBasis, error) {
	intent, err := getIntent(dbEntity)
	if err != nil {
		return CostBasis{}, err
	}

	state, err := getState(dbEntity)
	if err != nil {
		return CostBasis{}, err
	}

	costBasis := CostBasis{
		NamespacedID: models.NamespacedID{
			Namespace: dbEntity.GetNamespace(),
			ID:        dbEntity.GetID(),
		},
		ManagedModel: entutils.MapTimeMixinFromDB(dbEntity),
		CurrencyID:   dbEntity.GetCurrencyID(),
		Intent:       intent,
		State:        state,
	}

	if err := (CreateInput{
		NamespacedID: costBasis.NamespacedID,
		CurrencyID:   costBasis.CurrencyID,
		Intent:       costBasis.Intent,
		State:        costBasis.State,
	}).Validate(); err != nil {
		return CostBasis{}, fmt.Errorf("validate persisted cost basis: %w", err)
	}

	return costBasis, nil
}

func getIntent(dbEntity Getter) (Intent, error) {
	fiatCurrency, err := currencyx.NewFiatCurrency(dbEntity.GetFiatCurrency())
	if err != nil {
		return Intent{}, fmt.Errorf("map fiat currency: %w", err)
	}

	switch dbEntity.GetMode() {
	case ModeDynamic:
		return NewIntent(DynamicIntent{FiatCurrency: fiatCurrency}), nil
	case ModePinned:
		currencyCostBasisID := dbEntity.GetCurrencyCostBasisID()
		if currencyCostBasisID == nil {
			return Intent{}, errors.New("currency cost basis ID is required")
		}

		return NewIntent(PinnedIntent{
			FiatCurrency:        fiatCurrency,
			CurrencyCostBasisID: *currencyCostBasisID,
		}), nil
	case ModeManual:
		manualRate := dbEntity.GetManualRate()
		if manualRate == nil {
			return Intent{}, errors.New("manual rate is required")
		}

		return NewIntent(ManualIntent{
			FiatCurrency: fiatCurrency,
			Rate:         *manualRate,
		}), nil
	default:
		return Intent{}, fmt.Errorf("invalid cost basis mode: %s", dbEntity.GetMode())
	}
}

func getState(dbEntity Getter) (*State, error) {
	resolvedAt := dbEntity.GetResolvedAt()
	resolvedCostBasis := dbEntity.GetResolvedCostBasis()
	resolvedCostBasisID := dbEntity.GetResolvedCostBasisID()

	if resolvedAt == nil && resolvedCostBasis == nil && resolvedCostBasisID == nil {
		return nil, nil
	}

	if resolvedAt == nil {
		return nil, errors.New("resolved at is required")
	}

	if resolvedCostBasis == nil {
		return nil, errors.New("resolved cost basis is required")
	}

	state := State{
		CostBasis:   *resolvedCostBasis,
		CostBasisID: resolvedCostBasisID,
		ResolvedAt:  resolvedAt.UTC(),
	}

	if err := state.Validate(); err != nil {
		return nil, err
	}

	return &state, nil
}
