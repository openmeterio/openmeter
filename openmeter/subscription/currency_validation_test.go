package subscription

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestSubscriptionSpecValidateCurrencies(t *testing.T) {
	managedCustom := currenciestestutils.NewManagedCurrency(t, "default", "01J00000000000000000000000", "CREDITS")

	newSpec := func(invoiceCurrency currencyx.Code, itemCurrency *currencies.CurrencyReference, priced bool) SubscriptionSpec {
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
			spec: newSpec("USD", lo.ToPtr(currencies.NewCurrencyReference("USD")), true),
		},
		{
			name:    "invoice currency must be fiat",
			spec:    newSpec("CREDITS", lo.ToPtr(managedCustom.Reference()), true),
			wantErr: true,
		},
		{
			name:    "priced item must have materialized currency",
			spec:    newSpec("USD", nil, true),
			wantErr: true,
		},
		{
			name:    "fiat item must match invoice currency",
			spec:    newSpec("USD", lo.ToPtr(currencies.NewCurrencyReference("EUR")), true),
			wantErr: true,
		},
		{
			name: "managed custom item",
			spec: newSpec("USD", lo.ToPtr(managedCustom.Reference()), true),
		},
		{
			name:    "code-only custom item is not persisted identity",
			spec:    newSpec("USD", lo.ToPtr(currencies.NewCurrencyReference("CREDITS")), true),
			wantErr: true,
		},
		{
			name:    "unpriced item cannot have currency",
			spec:    newSpec("USD", lo.ToPtr(currencies.NewCurrencyReference("USD")), false),
			wantErr: true,
		},
		{
			name:           "credit only accepts dynamic mode",
			spec:           newSpec("USD", lo.ToPtr(managedCustom.Reference()), true),
			settlementMode: productcatalog.CreditOnlySettlementMode,
		},
		{
			name:           "credit only rejects pinned mode",
			spec:           newSpec("USD", lo.ToPtr(managedCustom.Reference()), true),
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
