package service

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
)

type currencyValidationItem struct {
	currency *currencies.CurrencyReference
	priced   bool
}

func newCurrencyValidationSpec(phases map[string][]currencyValidationItem) subscription.SubscriptionSpec {
	spec := subscription.SubscriptionSpec{
		Phases: make(map[string]*subscription.SubscriptionPhaseSpec, len(phases)),
	}

	for phaseKey, itemDefs := range phases {
		items := make([]*subscription.SubscriptionItemSpec, 0, len(itemDefs))
		for _, itemDef := range itemDefs {
			meta := productcatalog.RateCardMeta{
				Key:      "fee",
				Name:     "Fee",
				Currency: itemDef.currency,
			}
			if itemDef.priced {
				meta.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount: alpacadecimal.NewFromInt(1),
				})
			}

			items = append(items, &subscription.SubscriptionItemSpec{
				CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
					CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
						PhaseKey: phaseKey,
						ItemKey:  "fee",
						RateCard: &productcatalog.FlatFeeRateCard{RateCardMeta: meta},
					},
				},
			})
		}

		spec.Phases[phaseKey] = &subscription.SubscriptionPhaseSpec{
			ItemsByKey: map[string][]*subscription.SubscriptionItemSpec{
				"fee": items,
			},
		}
	}

	return spec
}

func TestValidateMaterializedItemCurrenciesUnchanged(t *testing.T) {
	customCurrency := currenciestestutils.NewManagedCurrency(t, "test", "01J00000000000000000000000", "CREDITS")

	newSpec := func(identity *currencies.CurrencyReference) subscription.SubscriptionSpec {
		return newCurrencyValidationSpec(map[string][]currencyValidationItem{
			"default": {{currency: identity, priced: true}},
		})
	}

	tests := []struct {
		name    string
		current *currencies.CurrencyReference
		updated *currencies.CurrencyReference
		wantErr bool
	}{
		{
			name:    "same fiat currency",
			current: lo.ToPtr(currencies.NewCurrencyReference("USD")),
			updated: lo.ToPtr(currencies.NewCurrencyReference("USD")),
		},
		{
			name:    "same managed custom currency",
			current: lo.ToPtr(customCurrency.Reference()),
			updated: lo.ToPtr(customCurrency.Reference()),
		},
		{
			name:    "fiat currency changed",
			current: lo.ToPtr(currencies.NewCurrencyReference("USD")),
			updated: lo.ToPtr(currencies.NewCurrencyReference("EUR")),
			wantErr: true,
		},
		{
			name:    "materialized currency removed",
			current: lo.ToPtr(customCurrency.Reference()),
			wantErr: true,
		},
		{
			name:    "legacy item can acquire materialized currency",
			updated: lo.ToPtr(currencies.NewCurrencyReference("USD")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given:
			// - an existing subscription item and its updated representation
			// when:
			// - materialized currencies are compared across the update
			// then:
			// - an existing identity is immutable, while a legacy missing value can be filled
			err := validateMaterializedItemCurrenciesUnchanged(newSpec(tt.current), newSpec(tt.updated))
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestValidateMaterializedItemCurrenciesUnchangedForAppendedVersions(t *testing.T) {
	tests := []struct {
		name    string
		current map[string][]currencyValidationItem
		updated map[string][]currencyValidationItem
		wantErr bool
	}{
		{
			name: "appended version keeps established currency",
			current: map[string][]currencyValidationItem{
				"default": {{currency: lo.ToPtr(currencies.NewCurrencyReference("USD")), priced: true}},
			},
			updated: map[string][]currencyValidationItem{
				"default": {
					{currency: lo.ToPtr(currencies.NewCurrencyReference("USD")), priced: true},
					{currency: lo.ToPtr(currencies.NewCurrencyReference("USD")), priced: true},
				},
			},
		},
		{
			name: "appended version cannot switch established currency",
			current: map[string][]currencyValidationItem{
				"default": {{currency: lo.ToPtr(currencies.NewCurrencyReference("USD")), priced: true}},
			},
			updated: map[string][]currencyValidationItem{
				"default": {
					{currency: lo.ToPtr(currencies.NewCurrencyReference("USD")), priced: true},
					{currency: lo.ToPtr(currencies.NewCurrencyReference("EUR")), priced: true},
				},
			},
			wantErr: true,
		},
		{
			name: "unpriced key can establish currency when it becomes priced",
			current: map[string][]currencyValidationItem{
				"default": {{priced: false}},
			},
			updated: map[string][]currencyValidationItem{
				"default": {
					{priced: false},
					{currency: lo.ToPtr(currencies.NewCurrencyReference("EUR")), priced: true},
				},
			},
		},
		{
			name: "same item key has independent currency history per phase",
			current: map[string][]currencyValidationItem{
				"first":  {{currency: lo.ToPtr(currencies.NewCurrencyReference("USD")), priced: true}},
				"second": {{currency: lo.ToPtr(currencies.NewCurrencyReference("EUR")), priced: true}},
			},
			updated: map[string][]currencyValidationItem{
				"first": {
					{currency: lo.ToPtr(currencies.NewCurrencyReference("USD")), priced: true},
					{currency: lo.ToPtr(currencies.NewCurrencyReference("USD")), priced: true},
				},
				"second": {
					{currency: lo.ToPtr(currencies.NewCurrencyReference("EUR")), priced: true},
					{currency: lo.ToPtr(currencies.NewCurrencyReference("EUR")), priced: true},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given:
			// - an existing item-key timeline and an updated timeline
			// when:
			// - later item versions are checked against the key's established currency
			// then:
			// - priced versions keep that identity, while an unpriced key may establish one
			err := validateMaterializedItemCurrenciesUnchanged(
				newCurrencyValidationSpec(tt.current),
				newCurrencyValidationSpec(tt.updated),
			)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
