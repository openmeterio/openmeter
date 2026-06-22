package linerouter

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/featuregate"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestRouterGetLineEngineForCreateLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		creditsEnabled    bool
		creditThenInvoice bool
		featureGate       featuregate.Gate
		line              billing.GenericInvoiceLineReader
		expectedEngine    billing.LineEngineType
		expectedErr       string
	}{
		{
			name:              "credits disabled falls back to invoice",
			creditsEnabled:    false,
			creditThenInvoice: true,
			featureGate:       featuregate.NewNoop(),
			line:              newRouterTestLine(productcatalog.FlatPriceType, ""),
			expectedEngine:    billing.LineEngineTypeInvoice,
		},
		{
			name:              "feature gate disabled falls back to invoice",
			creditsEnabled:    true,
			creditThenInvoice: true,
			featureGate:       alwaysFalseGate{},
			line:              newRouterTestLine(productcatalog.FlatPriceType, ""),
			expectedEngine:    billing.LineEngineTypeInvoice,
		},
		{
			name:              "credit then invoice disabled falls back to invoice",
			creditsEnabled:    true,
			creditThenInvoice: false,
			featureGate:       featuregate.NewNoop(),
			line:              newRouterTestLine(productcatalog.FlatPriceType, ""),
			expectedEngine:    billing.LineEngineTypeInvoice,
		},
		{
			name:              "enabled flat price routes to flat fee engine",
			creditsEnabled:    true,
			creditThenInvoice: true,
			featureGate:       featuregate.NewNoop(),
			line:              newRouterTestLine(productcatalog.FlatPriceType, ""),
			expectedEngine:    billing.LineEngineTypeChargeFlatFee,
		},
		{
			name:              "enabled unit price routes to usage based engine",
			creditsEnabled:    true,
			creditThenInvoice: true,
			featureGate:       featuregate.NewNoop(),
			line:              newRouterTestLine(productcatalog.UnitPriceType, ""),
			expectedEngine:    billing.LineEngineTypeChargeUsageBased,
		},
		{
			name:              "credits disabled ignores existing engine and falls back to invoice",
			creditsEnabled:    false,
			creditThenInvoice: true,
			featureGate:       featuregate.NewNoop(),
			line:              newRouterTestLine(productcatalog.FlatPriceType, billing.LineEngineTypeChargeFlatFee),
			expectedEngine:    billing.LineEngineTypeInvoice,
		},
		{
			name:              "enabled existing engine is replaced by price route",
			creditsEnabled:    true,
			creditThenInvoice: true,
			featureGate:       featuregate.NewNoop(),
			line:              newRouterTestLine(productcatalog.FlatPriceType, billing.LineEngineTypeChargeUsageBased),
			expectedEngine:    billing.LineEngineTypeChargeFlatFee,
		},
		{
			name:              "enabled nil price errors",
			creditsEnabled:    true,
			creditThenInvoice: true,
			featureGate:       featuregate.NewNoop(),
			line:              newRouterTestLineWithPrice(nil, ""),
			expectedErr:       "price is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := newRouterForTest(t, tt.creditsEnabled, tt.creditThenInvoice, tt.featureGate)
			engine, err := router.GetLineEngineForCreateLine(tt.line)

			if tt.expectedErr != "" {
				require.ErrorContains(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expectedEngine, engine)
		})
	}
}

func newRouterForTest(t testing.TB, creditsEnabled, creditThenInvoice bool, gate featuregate.Gate) *Router {
	t.Helper()

	router, err := New(Config{
		CreditsEnabled:           creditsEnabled,
		CreditThenInvoiceEnabled: creditThenInvoice,
		FeatureGate: featuregate.NewFeatureGateChecker(gate, featuregate.Flags{
			featuregate.CtxKeyCredits: string(featuregate.CtxKeyCredits),
		}, map[featuregate.FeatureFlag]bool{featuregate.CtxKeyCredits: true}),
	})
	require.NoError(t, err)

	return router
}

func newRouterTestLine(priceType productcatalog.PriceType, engine billing.LineEngineType) *billing.StandardLine {
	switch priceType {
	case productcatalog.FlatPriceType:
		return newRouterTestLineWithPrice(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount: alpacadecimal.NewFromInt(100),
		}), engine)
	default:
		return newRouterTestLineWithPrice(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(100),
		}), engine)
	}
}

func newRouterTestLineWithPrice(price *productcatalog.Price, engine billing.LineEngineType) *billing.StandardLine {
	return &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "ns"},
				ID:              "line-1",
				Name:            "line-1",
			},
			Engine: engine,
		},
		UsageBased: &billing.UsageBasedLine{
			Price: price,
		},
	}
}

type alwaysFalseGate struct{}

func (alwaysFalseGate) EvaluateBool(_, _ string, _ bool) (bool, error) {
	return false, nil
}
