package subscription_test

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	pcsubscriptionservice "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/service"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestAlignedBillingPeriodCalculation(t *testing.T) {
	ns := subscriptiontestutils.ExampleNamespace

	planInp := plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: ns,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:     "Test Plan",
				Currency: currency.USD,
				Alignment: productcatalog.Alignment{
					BillablesMustAlign: true,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Name:     "trial",
						Key:      "trial",
						Duration: lo.ToPtr(testutils.GetISODuration(t, "P1M")),
					},
					// TODO[OM-1031]: let's add discount handling (as this could be a 100% discount for the first month)
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  subscriptiontestutils.ExampleFeature.Key,
								Name: subscriptiontestutils.ExampleFeature.Name,
								// feature doesn't have to exist, we never call out to DB
								Feature: &feature.Feature{
									Namespace: subscriptiontestutils.ExampleFeature.Namespace,
									Key:       subscriptiontestutils.ExampleFeature.Key,
									ID:        "test-feature-id",
									Name:      subscriptiontestutils.ExampleFeature.Name,
									MeterSlug: subscriptiontestutils.ExampleFeature.MeterSlug,
									CreatedAt: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
									UpdatedAt: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
								},
							},
							BillingCadence: isodate.MustParse(t, "P1M"),
						},
					},
				},
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Name:     "default",
						Key:      "default",
						Duration: nil,
					},
					// TODO[OM-1031]: 50% discount
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  subscriptiontestutils.ExampleFeature.Key,
								Name: subscriptiontestutils.ExampleFeature.Name,
								Feature: &feature.Feature{
									Namespace: subscriptiontestutils.ExampleFeature.Namespace,
									Key:       subscriptiontestutils.ExampleFeature.Key,
									ID:        "test-feature-id",
									Name:      subscriptiontestutils.ExampleFeature.Name,
									MeterSlug: subscriptiontestutils.ExampleFeature.MeterSlug,
									CreatedAt: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
									UpdatedAt: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
								},
								Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
									Amount: alpacadecimal.NewFromFloat(5),
								}),
							},
							BillingCadence: isodate.MustParse(t, "P1M"),
						},
					},
				},
			},
		},
	}

	subPlan, err := pcsubscriptionservice.PlanFromPlanInput(planInp)
	require.NoError(t, err)

	t.Run("Should error if the subscription is canceled or inactive", func(t *testing.T) {
		spec, err := subscription.NewSpecFromPlan(subPlan, subscription.CreateSubscriptionCustomerInput{
			Name:       "test-customer",
			CustomerId: "test-customer-id",
			Currency:   currencyx.Code(currency.USD),
			// active for one day
			ActiveFrom: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
			ActiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2024-01-02T00:00:00Z")),
		})
		require.NoError(t, err)

		// Let's check the aligned billing period after activeTo for the second phase
		_, err = spec.GetAlignedBillingPeriodAt("default", testutils.GetRFC3339Time(t, "2024-03-02T00:00:00Z"))
		require.Error(t, err)
	})
}
