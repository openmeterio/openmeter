package service

import (
	"context"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type invoiceCurrencyPlan struct {
	currency currencyx.CurrencyIdentity
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

func (p invoiceCurrencyPlan) Currency() currencyx.CurrencyIdentity {
	return p.currency
}

type recordingCurrencyResolver struct {
	resolve func(context.Context, string, currencyx.Code) (currencyx.CurrencyIdentity, error)
}

func (r recordingCurrencyResolver) Resolve(ctx context.Context, namespace string, code currencyx.Code) (currencyx.CurrencyIdentity, error) {
	return r.resolve(ctx, namespace, code)
}

func (recordingCurrencyResolver) HasCostBasis(context.Context, string, currencyx.ManagedCurrency, currencyx.CurrencyIdentity) (bool, error) {
	return false, nil
}

func TestResolveSubscriptionInvoiceCurrency(t *testing.T) {
	tests := []struct {
		name             string
		planCurrency     currencyx.CurrencyIdentity
		customerCurrency *currencyx.Code
		expected         currencyx.Code
		wantErr          bool
	}{
		{
			name:         "fiat plan defaults customer without currency",
			planCurrency: currencyx.Code("USD"),
			expected:     currencyx.Code("USD"),
		},
		{
			name:             "matching fiat customer currency",
			planCurrency:     currencyx.Code("USD"),
			customerCurrency: lo.ToPtr(currencyx.Code("USD")),
			expected:         currencyx.Code("USD"),
		},
		{
			name:             "mismatching fiat customer currency",
			planCurrency:     currencyx.Code("USD"),
			customerCurrency: lo.ToPtr(currencyx.Code("EUR")),
			wantErr:          true,
		},
		{
			name:         "custom plan requires customer currency",
			planCurrency: currencyx.Code("CREDITS"),
			wantErr:      true,
		},
		{
			name:             "custom plan uses customer fiat",
			planCurrency:     currencyx.Code("CREDITS"),
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

func TestResolveEditPatchCurrency(t *testing.T) {
	const namespace = "default"

	managedCurrency := &currencies.Currency{
		NamespacedID: models.NamespacedID{
			Namespace: namespace,
			ID:        "currency-id",
		},
		Code: "CREDITS",
		Name: "Credits",
	}
	fiatCurrency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
		WithCode(currencyx.Code("USD")).
		Build()
	require.NoError(t, err)

	tests := []struct {
		name                string
		currency            currencyx.CurrencyIdentity
		resolvedCurrency    currencyx.CurrencyIdentity
		expectedResolveCode *currencyx.Code
		expectedCurrency    currencyx.CurrencyIdentity
		asPointer           bool
	}{
		{
			name:                "resolves code-only custom currency",
			currency:            currencyx.Code("CREDITS"),
			resolvedCurrency:    managedCurrency,
			expectedResolveCode: lo.ToPtr(currencyx.Code("CREDITS")),
			expectedCurrency:    managedCurrency,
		},
		{
			name:                "defaults omitted currency to invoice currency",
			resolvedCurrency:    fiatCurrency,
			expectedResolveCode: lo.ToPtr(currencyx.Code("USD")),
			expectedCurrency:    fiatCurrency,
		},
		{
			name:             "preserves managed custom identity",
			currency:         managedCurrency,
			expectedCurrency: managedCurrency,
			asPointer:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resolvedCodes []currencyx.Code
			resolver := recordingCurrencyResolver{
				resolve: func(_ context.Context, actualNamespace string, code currencyx.Code) (currencyx.CurrencyIdentity, error) {
					require.Equal(t, namespace, actualNamespace)
					resolvedCodes = append(resolvedCodes, code)
					return tt.resolvedCurrency, nil
				},
			}
			svc := service{WorkflowServiceConfig: WorkflowServiceConfig{CurrencyResolver: resolver}}

			addItem := patch.PatchAddItem{
				PhaseKey: "phase",
				ItemKey:  "item",
				CreateInput: subscription.SubscriptionItemSpec{
					CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							PhaseKey: "phase",
							ItemKey:  "item",
							RateCard: &productcatalog.FlatFeeRateCard{RateCardMeta: productcatalog.RateCardMeta{
								Key:      "item",
								Name:     "item",
								Price:    productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: alpacadecimal.NewFromInt(10)}),
								Currency: tt.currency,
							}},
						},
					},
				},
			}

			var customization subscription.Patch = addItem
			if tt.asPointer {
				customization = &addItem
			}

			resolvedPatch, err := svc.resolveEditPatchCurrency(t.Context(), namespace, currencyx.Code("USD"), customization)
			require.NoError(t, err)

			var resolvedAddItem patch.PatchAddItem
			switch actual := resolvedPatch.(type) {
			case patch.PatchAddItem:
				resolvedAddItem = actual
			case *patch.PatchAddItem:
				resolvedAddItem = *actual
			default:
				t.Fatalf("expected add-item patch, got %T", resolvedPatch)
			}

			if tt.expectedResolveCode == nil {
				require.Empty(t, resolvedCodes)
			} else {
				require.Equal(t, []currencyx.Code{*tt.expectedResolveCode}, resolvedCodes)
			}

			actualCurrency := resolvedAddItem.CreateInput.RateCard.AsMeta().Currency
			require.Same(t, tt.expectedCurrency, actualCurrency)
		})
	}
}
