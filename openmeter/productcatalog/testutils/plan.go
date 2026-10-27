package testutils

import (
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

var MonthPeriod = datetime.NewISODuration(0, 1, 0, 0, 0, 0, 0)

type TransformerFunc[T any] func(*testing.T, *T)

func WithPlanPhases(phases ...productcatalog.Phase) TransformerFunc[productcatalog.Plan] {
	return func(t *testing.T, plan *productcatalog.Plan) {
		t.Helper()

		plan.Phases = phases
	}
}

func WithPlanKey(key string) TransformerFunc[productcatalog.Plan] {
	return func(t *testing.T, plan *productcatalog.Plan) {
		t.Helper()

		plan.PlanMeta.Key = key
	}
}

// WithRateCardTaxCode points every rate card in the plan at tc by ID, keeping any tax behavior.
// Apply it after WithPlanPhases to cover the replacement phases.
func WithRateCardTaxCode(tc taxcode.TaxCode) TransformerFunc[productcatalog.Plan] {
	return func(t *testing.T, plan *productcatalog.Plan) {
		t.Helper()

		for _, phase := range plan.Phases {
			for _, rc := range phase.RateCards {
				require.NoError(t, rc.ChangeMeta(func(m productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
					var behavior *productcatalog.TaxBehavior
					if m.TaxConfig != nil {
						behavior = m.TaxConfig.Behavior
					}

					m.TaxConfig = &productcatalog.TaxConfig{TaxCodeID: &tc.ID, Behavior: behavior}

					return m, nil
				}))
			}
		}
	}
}

// NewTestPlan returns a plan input with one untaxed flat-fee rate card; use WithRateCardTaxCode to
// reference a tax code.
func NewTestPlan(t *testing.T, namespace string, transformers ...TransformerFunc[productcatalog.Plan]) plan.CreatePlanInput {
	t.Helper()

	input := plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Key:            "test",
				Name:           "Test",
				Description:    lo.ToPtr("Test plan"),
				Metadata:       models.Metadata{"name": "test"},
				Currency:       currencies.NewCurrencyReference(currencyx.Code(currency.USD)),
				BillingCadence: MonthPeriod,
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:         "free",
						Name:        "Free",
						Description: lo.ToPtr("Trial phase"),
						Metadata:    models.Metadata{"name": "free"},
						Duration:    nil,
					},
					RateCards: []productcatalog.RateCard{
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:                 "api_requests",
								Name:                "API Requests",
								Description:         lo.ToPtr("API Requests"),
								Metadata:            models.Metadata{"name": "api_requests"},
								EntitlementTemplate: nil,
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      decimal.NewFromInt(0),
									PaymentTerm: productcatalog.InArrearsPaymentTerm,
								}),
							},
							BillingCadence: &MonthPeriod,
						},
					},
				},
			},
		},
	}

	for _, transformer := range transformers {
		transformer(t, &input.Plan)
	}

	return input
}
