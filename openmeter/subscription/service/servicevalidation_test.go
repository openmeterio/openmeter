package service

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type currencyValidationItem struct {
	currency currencyx.CurrencyIdentity
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
	customCurrency := currencies.Currency{
		NamespacedID: models.NamespacedID{Namespace: "test", ID: "01J00000000000000000000000"},
		Code:         "CREDITS",
		Name:         "Credits",
	}

	newSpec := func(identity currencyx.CurrencyIdentity) subscription.SubscriptionSpec {
		return newCurrencyValidationSpec(map[string][]currencyValidationItem{
			"default": {{currency: identity, priced: true}},
		})
	}

	tests := []struct {
		name    string
		current currencyx.CurrencyIdentity
		updated currencyx.CurrencyIdentity
		wantErr bool
	}{
		{
			name:    "same fiat currency",
			current: currencyx.Code("USD"),
			updated: currencyx.Code("USD"),
		},
		{
			name:    "same managed custom currency",
			current: customCurrency,
			updated: customCurrency,
		},
		{
			name:    "fiat currency changed",
			current: currencyx.Code("USD"),
			updated: currencyx.Code("EUR"),
			wantErr: true,
		},
		{
			name:    "materialized currency removed",
			current: customCurrency,
			wantErr: true,
		},
		{
			name:    "legacy item can acquire materialized currency",
			updated: currencyx.Code("USD"),
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
				"default": {{currency: currencyx.Code("USD"), priced: true}},
			},
			updated: map[string][]currencyValidationItem{
				"default": {
					{currency: currencyx.Code("USD"), priced: true},
					{currency: currencyx.Code("USD"), priced: true},
				},
			},
		},
		{
			name: "appended version cannot switch established currency",
			current: map[string][]currencyValidationItem{
				"default": {{currency: currencyx.Code("USD"), priced: true}},
			},
			updated: map[string][]currencyValidationItem{
				"default": {
					{currency: currencyx.Code("USD"), priced: true},
					{currency: currencyx.Code("EUR"), priced: true},
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
					{currency: currencyx.Code("EUR"), priced: true},
				},
			},
		},
		{
			name: "same item key has independent currency history per phase",
			current: map[string][]currencyValidationItem{
				"first":  {{currency: currencyx.Code("USD"), priced: true}},
				"second": {{currency: currencyx.Code("EUR"), priced: true}},
			},
			updated: map[string][]currencyValidationItem{
				"first": {
					{currency: currencyx.Code("USD"), priced: true},
					{currency: currencyx.Code("USD"), priced: true},
				},
				"second": {
					{currency: currencyx.Code("EUR"), priced: true},
					{currency: currencyx.Code("EUR"), priced: true},
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
