package subscription

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSubscriptionSpecValidateCurrencies(t *testing.T) {
	managedCustom := currencies.Currency{
		NamespacedID: models.NamespacedID{Namespace: "default", ID: "01J00000000000000000000000"},
		Code:         "CREDITS",
		Name:         "Credits",
	}

	newSpec := func(invoiceCurrency currencyx.Code, itemCurrency currencyx.CurrencyIdentity, priced bool) SubscriptionSpec {
		meta := productcatalog.RateCardMeta{Key: "fee", Name: "Fee", Currency: itemCurrency}
		if priced {
			meta.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: alpacadecimal.NewFromInt(1)})
		}

		return SubscriptionSpec{
			CreateSubscriptionCustomerInput: CreateSubscriptionCustomerInput{InvoiceCurrency: invoiceCurrency},
			CreateSubscriptionPlanInput: CreateSubscriptionPlanInput{
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			},
			Phases: map[string]*SubscriptionPhaseSpec{
				"default": {
					CreateSubscriptionPhasePlanInput: CreateSubscriptionPhasePlanInput{PhaseKey: "default"},
					ItemsByKey: map[string][]*SubscriptionItemSpec{
						"fee": {{
							CreateSubscriptionItemInput: CreateSubscriptionItemInput{
								CreateSubscriptionItemPlanInput: CreateSubscriptionItemPlanInput{
									PhaseKey: "default",
									ItemKey:  "fee",
									RateCard: &productcatalog.FlatFeeRateCard{RateCardMeta: meta},
								},
							},
						}},
					},
				},
			},
		}
	}

	tests := []struct {
		name           string
		spec           SubscriptionSpec
		costBasisMode  CostBasisMode
		settlementMode productcatalog.SettlementMode
		wantErr        bool
	}{
		{
			name: "matching fiat item",
			spec: newSpec("USD", currencyx.Code("USD"), true),
		},
		{
			name:    "invoice currency must be fiat",
			spec:    newSpec("CREDITS", managedCustom, true),
			wantErr: true,
		},
		{
			name:    "priced item must have materialized currency",
			spec:    newSpec("USD", nil, true),
			wantErr: true,
		},
		{
			name:    "fiat item must match invoice currency",
			spec:    newSpec("USD", currencyx.Code("EUR"), true),
			wantErr: true,
		},
		{
			name: "managed custom item",
			spec: newSpec("USD", managedCustom, true),
		},
		{
			name:    "code-only custom item is not persisted identity",
			spec:    newSpec("USD", currencyx.Code("CREDITS"), true),
			wantErr: true,
		},
		{
			name:    "unpriced item cannot have currency",
			spec:    newSpec("USD", currencyx.Code("USD"), false),
			wantErr: true,
		},
		{
			name:           "credit only accepts dynamic mode",
			spec:           newSpec("USD", managedCustom, true),
			settlementMode: productcatalog.CreditOnlySettlementMode,
		},
		{
			name:           "credit only rejects pinned mode",
			spec:           newSpec("USD", managedCustom, true),
			settlementMode: productcatalog.CreditOnlySettlementMode,
			costBasisMode:  CostBasisModePinned,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given:
			// - a subscription invoice fiat, settlement mode, and materialized item currency
			// when:
			// - currency-only spec invariants are validated locally
			// then:
			// - persisted identities and fiat compatibility are enforced without DB access
			tt.spec.CostBasisMode = tt.costBasisMode
			if tt.settlementMode != "" {
				tt.spec.CreateSubscriptionPlanInput.SettlementMode = tt.settlementMode
			}

			err := tt.spec.validateCurrencies()
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
