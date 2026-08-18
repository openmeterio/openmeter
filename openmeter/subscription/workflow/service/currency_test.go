package service

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type invoiceCurrencyPlan struct {
	currency currencies.CurrencyReference
}

func (p invoiceCurrencyPlan) ToCreateSubscriptionPlanInput() subscription.CreateSubscriptionPlanInput {
	return subscription.CreateSubscriptionPlanInput{}
}

func (p invoiceCurrencyPlan) GetName() string {
	return ""
}

func (p invoiceCurrencyPlan) GetPhases() []subscription.PlanPhase {
	return nil
}

func (p invoiceCurrencyPlan) Currency() currencies.CurrencyReference {
	return p.currency
}

func TestResolveSubscriptionInvoiceCurrency(t *testing.T) {
	tests := []struct {
		name             string
		planCurrency     currencies.CurrencyReference
		customerCurrency *currencyx.Code
		expected         currencyx.Code
		wantErr          bool
	}{
		{
			name:         "fiat plan defaults customer without currency",
			planCurrency: currencies.NewCurrencyReference("USD"),
			expected:     currencyx.Code("USD"),
		},
		{
			name:             "matching fiat customer currency",
			planCurrency:     currencies.NewCurrencyReference("USD"),
			customerCurrency: lo.ToPtr(currencyx.Code("USD")),
			expected:         currencyx.Code("USD"),
		},
		{
			name:             "fiat plan defines invoice currency despite customer mismatch",
			planCurrency:     currencies.NewCurrencyReference("USD"),
			customerCurrency: lo.ToPtr(currencyx.Code("EUR")),
			expected:         currencyx.Code("USD"),
		},
		{
			name:         "custom plan requires customer currency",
			planCurrency: currencies.NewCurrencyReference("CREDITS"),
			wantErr:      true,
		},
		{
			name:             "custom plan uses customer fiat",
			planCurrency:     currencies.NewCurrencyReference("CREDITS"),
			customerCurrency: lo.ToPtr(currencyx.Code("USD")),
			expected:         currencyx.Code("USD"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invoiceCurrency, err := resolveSubscriptionInvoiceCurrency(customer.Customer{
				Currency: tt.customerCurrency,
			}, invoiceCurrencyPlan{currency: tt.planCurrency})
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expected, invoiceCurrency)
		})
	}
}
