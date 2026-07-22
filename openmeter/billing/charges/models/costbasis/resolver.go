package costbasis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
)

type Resolver interface {
	// ResolveInitialState resolves the initial state of the cost basis for a given intent,
	// dynamic costbasis is resolved once we hit current service period's start so we are not
	// doing the resolution during creation.
	ResolveInitialState(ctx context.Context, input ResolveInitialStateInput) (*State, error)

	// ResolveDynamicState resolves the dynamic state of the cost basis for a given intent,
	// dynamic costbasis is resolved once we hit current service period's start so we are not
	// doing the resolution during creation.
	ResolveDynamicState(ctx context.Context, input ResolveDynamicStateInput) (State, error)
}

type ResolveInitialStateInput struct {
	CurrencyID models.NamespacedID
	Intent     Intent
	ResolvedAt time.Time
}

func (i ResolveInitialStateInput) Validate() error {
	var errs []error

	if err := i.CurrencyID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	if i.ResolvedAt.IsZero() {
		errs = append(errs, errors.New("resolved at is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ResolveDynamicStateInput struct {
	CurrencyID models.NamespacedID
	Intent     Intent

	ServicePeriodFrom time.Time
}

func (i ResolveDynamicStateInput) Validate() error {
	var errs []error

	if err := i.CurrencyID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	} else if i.Intent.Kind() != ModeDynamic {
		errs = append(errs, errors.New("intent is not a dynamic intent"))
	}

	if i.ServicePeriodFrom.IsZero() {
		errs = append(errs, errors.New("service period from is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type resolver struct {
	currencies currencies.Service
}

var _ Resolver = (*resolver)(nil)

type ResolverConfig struct {
	Currencies currencies.Service
}

func (c ResolverConfig) Validate() error {
	if c.Currencies == nil {
		return errors.New("currencies is required")
	}

	return nil
}

func NewResolver(config ResolverConfig) (Resolver, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &resolver{
		currencies: config.Currencies,
	}, nil
}

func (r *resolver) ResolveInitialState(ctx context.Context, input ResolveInitialStateInput) (*State, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	switch input.Intent.kind {
	case ModeDynamic:
		return nil, nil
	case ModePinned:
		return r.resolvePinnedState(ctx, input.CurrencyID, input.Intent, input.ResolvedAt)
	case ModeManual:
		return r.resolveManualState(ctx, input.Intent, input.ResolvedAt)
	default:
		return nil, errors.New("invalid intent kind")
	}
}

func (r *resolver) ResolveDynamicState(ctx context.Context, input ResolveDynamicStateInput) (State, error) {
	if err := input.Validate(); err != nil {
		return State{}, err
	}

	effectiveCostBasis, err := r.currencies.GetCostBasisAt(ctx, currencies.GetCostBasisAtInput{
		Namespace:  input.CurrencyID.Namespace,
		CurrencyID: input.CurrencyID.ID,
		FiatCode:   input.Intent.dynamic.FiatCurrency.Details().Code,
		At:         input.ServicePeriodFrom,
	})
	if err != nil {
		return State{}, err
	}

	return State{
		CostBasis:   effectiveCostBasis.Rate,
		CostBasisID: lo.ToPtr(effectiveCostBasis.ID),
		ResolvedAt:  input.ServicePeriodFrom.UTC(),
	}, nil
}

func (r *resolver) resolveManualState(_ context.Context, intent Intent, at time.Time) (*State, error) {
	return &State{
		CostBasis:  intent.manual.Rate,
		ResolvedAt: at,
	}, nil
}

func (r *resolver) resolvePinnedState(ctx context.Context, currencyID models.NamespacedID, intent Intent, at time.Time) (*State, error) {
	currencyCostBasis, err := r.currencies.GetCostBasis(ctx, currencies.GetCostBasisInput{
		NamespacedID: models.NamespacedID{
			Namespace: currencyID.Namespace,
			ID:        intent.pinned.CurrencyCostBasisID,
		},
	})
	if err != nil {
		return nil, err
	}

	if currencyCostBasis.CurrencyID != currencyID.ID {
		return nil, fmt.Errorf("currency cost basis currency mismatch: %s != %s", currencyCostBasis.CurrencyID, currencyID.ID)
	}

	if currencyCostBasis.FiatCode != intent.pinned.FiatCurrency.Details().Code {
		return nil, fmt.Errorf("currency cost basis fiat currency mismatch: %s != %s", currencyCostBasis.FiatCode, intent.pinned.FiatCurrency.Details().Code)
	}

	return &State{
		CostBasis:   currencyCostBasis.Rate,
		CostBasisID: lo.ToPtr(currencyCostBasis.ID),
		ResolvedAt:  at,
	}, nil
}
