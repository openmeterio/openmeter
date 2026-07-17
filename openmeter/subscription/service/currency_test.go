package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type costBasisServiceStub struct {
	get func(context.Context, currencies.GetCostBasisAtInput) (currencies.CostBasis, error)
}

func (costBasisServiceStub) CreateCostBasis(context.Context, currencies.CreateCostBasisInput) (currencies.CostBasis, error) {
	return currencies.CostBasis{}, errors.New("not implemented")
}

func (costBasisServiceStub) ListCostBases(context.Context, currencies.ListCostBasesInput) (pagination.Result[currencies.CostBasis], error) {
	return pagination.Result[currencies.CostBasis]{}, errors.New("not implemented")
}

func (s costBasisServiceStub) GetCostBasisAt(ctx context.Context, input currencies.GetCostBasisAtInput) (currencies.CostBasis, error) {
	return s.get(ctx, input)
}

func TestResolveCostBasisRequirements(t *testing.T) {
	const (
		namespace        = "default"
		customCurrencyID = "01J00000000000000000000000"
		costBasisID      = "01J00000000000000000000001"
	)

	at := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	customCurrency := currencies.Currency{
		NamespacedID: models.NamespacedID{Namespace: namespace, ID: customCurrencyID},
		Code:         "CREDITS",
		Name:         "Credits",
	}

	newSpec := func(mode productcatalog.SettlementMode) subscription.SubscriptionSpec {
		newItem := func(key string) *subscription.SubscriptionItemSpec {
			return &subscription.SubscriptionItemSpec{
				CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
					CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
						PhaseKey: "default",
						ItemKey:  key,
						RateCard: &productcatalog.FlatFeeRateCard{RateCardMeta: productcatalog.RateCardMeta{
							Key:      key,
							Name:     key,
							Currency: customCurrency,
							Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
								Amount: alpacadecimal.NewFromInt(1),
							}),
						}},
					},
				},
			}
		}

		return subscription.SubscriptionSpec{
			CreateSubscriptionCustomerInput: subscription.CreateSubscriptionCustomerInput{InvoiceCurrency: currencyx.Code("USD")},
			CreateSubscriptionPlanInput: subscription.CreateSubscriptionPlanInput{
				SettlementMode: mode,
			},
			Phases: map[string]*subscription.SubscriptionPhaseSpec{
				"default": {
					CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{PhaseKey: "default"},
					ItemsByKey: map[string][]*subscription.SubscriptionItemSpec{
						"one": {newItem("one")},
						"two": {newItem("two")},
					},
				},
			},
		}
	}

	t.Run("credit only does not look up cost basis", func(t *testing.T) {
		// given:
		// - a credit-only subscription with custom-currency priced items
		// when:
		// - cost-basis requirements are resolved
		// then:
		// - no fiat conversion lookup is performed
		calls := 0
		svc := service{ServiceConfig: ServiceConfig{CostBasisService: costBasisServiceStub{get: func(context.Context, currencies.GetCostBasisAtInput) (currencies.CostBasis, error) {
			calls++
			return currencies.CostBasis{}, nil
		}}}}
		spec := newSpec(productcatalog.CreditOnlySettlementMode)

		resolved, err := svc.resolveCostBasisRequirements(t.Context(), namespace, spec, at, customCostBasisPairs(spec))
		require.NoError(t, err)
		require.Empty(t, resolved)
		require.Zero(t, calls)
	})

	t.Run("credit then invoice resolves each pair once at the requested time", func(t *testing.T) {
		// given:
		// - two items sharing one custom-currency to invoice-fiat pair
		// when:
		// - conversion eligibility is validated
		// then:
		// - one exact cost-basis resource is selected at the supplied effective time
		calls := 0
		svc := service{ServiceConfig: ServiceConfig{CostBasisService: costBasisServiceStub{get: func(_ context.Context, input currencies.GetCostBasisAtInput) (currencies.CostBasis, error) {
			calls++
			require.Equal(t, namespace, input.Namespace)
			require.Equal(t, customCurrencyID, input.CurrencyID)
			require.Equal(t, currencyx.Code("USD"), input.FiatCode)
			require.Equal(t, at, input.At)
			return currencies.CostBasis{
				NamespacedID:  models.NamespacedID{Namespace: namespace, ID: costBasisID},
				CurrencyID:    customCurrencyID,
				FiatCode:      "USD",
				EffectiveFrom: at.Add(-time.Hour),
			}, nil
		}}}}
		spec := newSpec(productcatalog.CreditThenInvoiceSettlementMode)

		resolved, err := svc.resolveCostBasisRequirements(t.Context(), namespace, spec, at, customCostBasisPairs(spec))
		require.NoError(t, err)
		require.Len(t, resolved, 1)
		require.Equal(t, costBasisID, resolved[0].ID)
		require.Equal(t, 1, calls)
	})

	t.Run("missing cost basis is a validation error", func(t *testing.T) {
		// given:
		// - a credit-then-invoice subscription without an effective mapping
		// when:
		// - conversion eligibility is validated
		// then:
		// - subscription creation fails as invalid input
		svc := service{ServiceConfig: ServiceConfig{CostBasisService: costBasisServiceStub{get: func(context.Context, currencies.GetCostBasisAtInput) (currencies.CostBasis, error) {
			return currencies.CostBasis{}, models.NewGenericNotFoundError(errors.New("missing"))
		}}}}
		spec := newSpec(productcatalog.CreditThenInvoiceSettlementMode)

		_, err := svc.resolveCostBasisRequirements(t.Context(), namespace, spec, at, customCostBasisPairs(spec))
		require.Error(t, err)
		require.True(t, models.IsGenericValidationError(err))
		require.ErrorIs(t, err, productcatalog.ErrCurrencyCostBasisNotFound)
	})
}
