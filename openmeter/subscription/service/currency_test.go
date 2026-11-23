package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type costBasisServiceStub struct {
	get func(context.Context, currencies.GetCostBasisAtInput) (currencies.CostBasis, error)
}

func (costBasisServiceStub) CreateCostBasis(context.Context, currencies.CreateCostBasisInput) (currencies.CostBasis, error) {
	return currencies.CostBasis{}, errors.New("not implemented")
}

func (costBasisServiceStub) GetCostBasis(context.Context, currencies.GetCostBasisInput) (currencies.CostBasis, error) {
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
	customCurrency := currenciestestutils.NewManagedCurrency(t, namespace, customCurrencyID, "CREDITS")

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
							Currency: lo.ToPtr(customCurrency.Reference()),
							Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
								Amount: alpacadecimal.NewFromInt(1),
							}),
						}},
					},
				},
			}
		}

		return subscription.SubscriptionSpec{
			CreateSubscriptionCustomerInput: subscription.CreateSubscriptionCustomerInput{
				ActiveFrom:      at,
				InvoiceCurrency: currencyx.Code("USD"),
			},
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

		resolved, err := svc.resolveCostBasisRequirements(t.Context(), namespace, spec, customCostBasisPairs(spec))
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
				NamespacedID: models.NamespacedID{Namespace: namespace, ID: costBasisID},
				CurrencyID:   customCurrencyID,
				CostBasis: currencyx.CostBasis{
					FiatCode:      "USD",
					EffectiveFrom: at.Add(-time.Hour),
				},
			}, nil
		}}}}
		spec := newSpec(productcatalog.CreditThenInvoiceSettlementMode)

		resolved, err := svc.resolveCostBasisRequirements(t.Context(), namespace, spec, customCostBasisPairs(spec))
		require.NoError(t, err)
		require.Len(t, resolved, 1)
		require.Equal(t, costBasisID, resolved[0].ID)
		require.Equal(t, 1, calls)
	})

	t.Run("two recurring fees share one lookup at the delayed billing start", func(t *testing.T) {
		spec := newSpec(productcatalog.CreditThenInvoiceSettlementMode)
		cutoff := at.Add(24*time.Hour + 500*time.Millisecond)
		spec.BillingStartsAt = &cutoff
		month := datetime.MustParseDuration(t, "P1M")
		for _, items := range spec.Phases["default"].ItemsByKey {
			items[0].RateCard.(*productcatalog.FlatFeeRateCard).BillingCadence = &month
		}

		require.Equal(t, map[costBasisPair]time.Time{
			{customCurrencyID: customCurrencyID, invoiceCurrency: "USD"}: cutoff.Truncate(time.Second),
		}, customCostBasisPairs(spec))
	})

	t.Run("each pair uses its earliest paid phase", func(t *testing.T) {
		// Given two paid phases sharing one currency pair, with another pair
		// introduced only in the later phase.
		spec := newSpec(productcatalog.CreditThenInvoiceSettlementMode)
		spec.Phases["default"].StartAfter = datetime.MustParseDuration(t, "P1M")
		laterSpec := newSpec(productcatalog.CreditThenInvoiceSettlementMode)
		laterPhase := laterSpec.Phases["default"]
		laterPhase.PhaseKey = "later"
		laterPhase.StartAfter = datetime.MustParseDuration(t, "P2M")
		for _, items := range laterPhase.ItemsByKey {
			items[0].PhaseKey = "later"
		}

		points := currenciestestutils.NewManagedCurrency(t, namespace, "01J00000000000000000000002", "POINTS")
		laterPhase.ItemsByKey["two"][0].RateCard.(*productcatalog.FlatFeeRateCard).Currency = lo.ToPtr(points.Reference())
		spec.Phases["later"] = laterPhase

		// When the lookup times are collected, each pair retains its first
		// paid phase regardless of the order in which phases are visited.
		lookupTimes := customCostBasisPairs(spec)

		// Then the shared pair uses the earlier phase, and the new pair uses
		// the later phase.
		require.Equal(t, map[costBasisPair]time.Time{
			{customCurrencyID: customCurrencyID, invoiceCurrency: "USD"}: at.AddDate(0, 1, 0),
			{customCurrencyID: points.ID, invoiceCurrency: "USD"}:        at.AddDate(0, 2, 0),
		}, lookupTimes)
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

		_, err := svc.resolveCostBasisRequirements(t.Context(), namespace, spec, customCostBasisPairs(spec))
		require.Error(t, err)
		require.True(t, models.IsGenericValidationError(err))
		require.ErrorIs(t, err, productcatalog.ErrCurrencyCostBasisNotFound)
	})
}
