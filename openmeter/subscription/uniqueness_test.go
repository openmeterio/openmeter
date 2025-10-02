package subscription_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestValidateUniqueConstraintBySubscriptions(t *testing.T) {
	t.Run("Should not error if they're far apart", func(t *testing.T) {
		s1 := getSimpleSub(models.CadencedModel{
			ActiveFrom: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			ActiveTo:   lo.ToPtr(time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)),
		})
		s2 := getSimpleSub(models.CadencedModel{
			ActiveFrom: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC),
			ActiveTo:   lo.ToPtr(time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC)),
		})

		require.NoError(t, subscription.ValidateUniqueConstraintBySubscriptions([]subscription.SubscriptionSpec{s1, s2}))
	})

	t.Run("Should error if they're overlapping", func(t *testing.T) {
		s1 := getSimpleSub(models.CadencedModel{
			ActiveFrom: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			ActiveTo:   lo.ToPtr(time.Date(2025, 1, 2, 0, 0, 2, 0, time.UTC)),
		})

		s2 := getSimpleSub(models.CadencedModel{
			ActiveFrom: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			ActiveTo:   lo.ToPtr(time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC)),
		})

		err := subscription.ValidateUniqueConstraintBySubscriptions([]subscription.SubscriptionSpec{s1, s2})
		require.Error(t, err)

		// Now let's assert the error is correct
		issues, err := models.AsValidationIssues(err)
		require.NoError(t, err)
		require.Len(t, issues, 1)
		require.Equal(t, subscription.ErrOnlySingleSubscriptionAllowed.Code(), issues[0].Code())

		detail := issues[0].Attributes()["overlaps"].([]models.OverlapDetail[subscription.SubscriptionSpec])
		require.Len(t, detail, 1)
		require.Equal(t, 0, detail[0].Index1)
		require.Equal(t, 1, detail[0].Index2)
		require.Equal(t, s1, detail[0].Item1)
		require.Equal(t, s2, detail[0].Item2)
	})

	t.Run("Should not error if they're touching", func(t *testing.T) {
		s1 := getSimpleSub(models.CadencedModel{
			ActiveFrom: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			ActiveTo:   lo.ToPtr(time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)),
		})

		s2 := getSimpleSub(models.CadencedModel{
			ActiveFrom: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			ActiveTo:   lo.ToPtr(time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC)),
		})

		require.NoError(t, subscription.ValidateUniqueConstraintBySubscriptions([]subscription.SubscriptionSpec{s1, s2}))
	})

	t.Run("Should work for many subscriptions", func(t *testing.T) {
		s1 := getSimpleSub(models.CadencedModel{
			ActiveFrom: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			ActiveTo:   lo.ToPtr(time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)),
		})

		s2 := getSimpleSub(models.CadencedModel{
			ActiveFrom: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			ActiveTo:   lo.ToPtr(time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC)),
		})

		s3 := getSimpleSub(models.CadencedModel{
			ActiveFrom: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC),
			ActiveTo:   lo.ToPtr(time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC)),
		})

		s4 := getSimpleSub(models.CadencedModel{
			ActiveFrom: time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC),
			ActiveTo:   lo.ToPtr(time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)),
		})

		s5 := getSimpleSub(models.CadencedModel{
			ActiveFrom: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			ActiveTo:   lo.ToPtr(time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)),
		})

		require.NoError(t, subscription.ValidateUniqueConstraintBySubscriptions([]subscription.SubscriptionSpec{s1, s2, s3, s4}))

		err := subscription.ValidateUniqueConstraintBySubscriptions([]subscription.SubscriptionSpec{s1, s2, s3, s4, s5})
		require.Error(t, err)
	})
}

func TestValidateUniqueConstraintByFeatures(t *testing.T) {
	t.Run("Should not error if on a single suscription passed in", func(t *testing.T) {
		s1 := getSimpleSub(models.CadencedModel{
			ActiveFrom: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			ActiveTo:   lo.ToPtr(time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)),
		})

		require.NoError(t, subscription.ValidateUniqueConstraintByFeatures([]subscription.SubscriptionSpec{s1}))
	})

	t.Run("Should not error if two empty subscriptions are overlapping", func(t *testing.T) {
		s1 := getSimpleSub(models.CadencedModel{
			ActiveFrom: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			ActiveTo:   lo.ToPtr(time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC)),
		})

		s2 := getSimpleSub(models.CadencedModel{
			ActiveFrom: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			ActiveTo:   lo.ToPtr(time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC)),
		})

		require.NoError(t, subscription.ValidateUniqueConstraintByFeatures([]subscription.SubscriptionSpec{s1, s2}))
	})

	t.Run("Should not error if two if two subscriptions share the same feature without entitlement or price", func(t *testing.T) {
		clock.FreezeTime(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
		defer clock.UnFreeze()

		// First the setup
		overlappingFeatureKey := "feature1"
		overlappingFeatureID := "01K6JCPG631MH1EKEQB2YMDBJW"

		builder1 := subscriptiontestutils.BuildTestSubscriptionSpec(t)
		builder1 = builder1.AddPhase(nil, &productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Name:       "overlapping feature",
				Key:        overlappingFeatureKey,
				FeatureKey: &overlappingFeatureKey,
				FeatureID:  &overlappingFeatureID,
			},
		})
		s1, err := builder1.Build()
		require.NoError(t, err)

		builder2 := subscriptiontestutils.BuildTestSubscriptionSpec(t)
		builder2 = builder2.AddPhase(nil, &productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Name:       "overlapping feature",
				Key:        overlappingFeatureKey,
				FeatureKey: &overlappingFeatureKey,
				FeatureID:  &overlappingFeatureID,
			},
		})
		s2, err := builder2.Build()
		require.NoError(t, err)

		t.Run("Should not error if the subscriptions are overlapping", func(t *testing.T) {
			s1.ActiveFrom = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			s1.ActiveTo = lo.ToPtr(time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC))
			s2.ActiveFrom = time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
			s2.ActiveTo = lo.ToPtr(time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC))

			require.NoError(t, subscription.ValidateUniqueConstraintByFeatures([]subscription.SubscriptionSpec{s1, s2}))
		})

		t.Run("Should not error if the subscriptions are adjacent", func(t *testing.T) {
			s1.ActiveFrom = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			s1.ActiveTo = lo.ToPtr(time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC))
			s2.ActiveFrom = time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
			s2.ActiveTo = lo.ToPtr(time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC))

			require.NoError(t, subscription.ValidateUniqueConstraintByFeatures([]subscription.SubscriptionSpec{s1, s2}))
		})

		t.Run("Should not error if the subscriptions are not overlapping", func(t *testing.T) {
			s1.ActiveFrom = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			s1.ActiveTo = lo.ToPtr(time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC))
			s2.ActiveFrom = time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC)
			s2.ActiveTo = lo.ToPtr(time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC))

			require.NoError(t, subscription.ValidateUniqueConstraintByFeatures([]subscription.SubscriptionSpec{s1, s2}))
		})
	})

	t.Run("Should not error if two subscriptions have overlapping features, where at most one has a price and neither has an entitlement", func(t *testing.T) {
		clock.FreezeTime(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
		defer clock.UnFreeze()

		overlappingFeatureKey := "feature1"
		overlappingFeatureID := "01K6JCPG631MH1EKEQB2YMDBJW"

		builder1 := subscriptiontestutils.BuildTestSubscriptionSpec(t)
		builder1 = builder1.AddPhase(nil, &productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Name: "overlapping feature",
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(int64(100)),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Key:        overlappingFeatureKey,
				FeatureKey: &overlappingFeatureKey,
				FeatureID:  &overlappingFeatureID,
			},
		})
		s1, err := builder1.Build()
		require.NoError(t, err)

		builder2 := subscriptiontestutils.BuildTestSubscriptionSpec(t)
		builder2 = builder2.AddPhase(nil, &productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Name:       "overlapping feature",
				Key:        overlappingFeatureKey,
				FeatureKey: &overlappingFeatureKey,
				FeatureID:  &overlappingFeatureID,
			},
		})

		s2, err := builder2.Build()
		require.NoError(t, err)

		builder3 := subscriptiontestutils.BuildTestSubscriptionSpec(t)
		builder3 = builder3.AddPhase(nil, &productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Name:       "overlapping feature",
				Key:        overlappingFeatureKey,
				FeatureKey: &overlappingFeatureKey,
				FeatureID:  &overlappingFeatureID,
			},
		})
		s3, err := builder3.Build()
		require.NoError(t, err)

		builder4 := subscriptiontestutils.BuildTestSubscriptionSpec(t)
		builder4 = builder4.AddPhase(nil, &productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Name:       "overlapping feature",
				Key:        overlappingFeatureKey,
				FeatureKey: &overlappingFeatureKey,
				FeatureID:  &overlappingFeatureID,
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(int64(100)),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
			},
		})
		s4, err := builder4.Build()
		require.NoError(t, err)

		t.Run("Should not error when overlapping", func(t *testing.T) {
			s1.ActiveFrom = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			s1.ActiveTo = lo.ToPtr(time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC))
			s2.ActiveFrom = time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
			s2.ActiveTo = lo.ToPtr(time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC))
			s3.ActiveFrom = time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
			s3.ActiveTo = lo.ToPtr(time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC))
			s4.ActiveFrom = time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC)
			s4.ActiveTo = lo.ToPtr(time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC))
			// Looks like this
			// | s1    |   s4  |		<- have prices
			//     |   s2  |    		<- no prices
			//     |     s3    |		<- no prices
			// |   |   |   |   |

			require.NoError(t, subscription.ValidateUniqueConstraintByFeatures([]subscription.SubscriptionSpec{s1, s2}), "Should not error for two")
			require.NoError(t, subscription.ValidateUniqueConstraintByFeatures([]subscription.SubscriptionSpec{s1, s2, s3}), "Should not error for three")
			require.NoError(t, subscription.ValidateUniqueConstraintByFeatures([]subscription.SubscriptionSpec{s1, s2, s3, s4}), "Should not error for four")
		})
	})

	t.Run("Should error if two subscriptions have overlapping billable features", func(t *testing.T) {
		clock.FreezeTime(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
		defer clock.UnFreeze()

		overlappingFeatureKey := "feature1"
		overlappingFeatureID := "01K6JCPG631MH1EKEQB2YMDBJW"

		builder1 := subscriptiontestutils.BuildTestSubscriptionSpec(t)
		builder1 = builder1.AddPhase(nil, &productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Name: "overlapping feature",
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(int64(100)),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Key:        overlappingFeatureKey,
				FeatureKey: &overlappingFeatureKey,
				FeatureID:  &overlappingFeatureID,
			},
		})
		s1, err := builder1.Build()
		require.NoError(t, err)

		builder2 := subscriptiontestutils.BuildTestSubscriptionSpec(t)
		builder2 = builder2.AddPhase(nil, &productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Name: "overlapping feature",
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(int64(100)),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Key:        overlappingFeatureKey,
				FeatureKey: &overlappingFeatureKey,
				FeatureID:  &overlappingFeatureID,
			},
		})
		s2, err := builder2.Build()
		require.NoError(t, err)

		t.Run("Should error when overlapping", func(t *testing.T) {
			s1.ActiveFrom = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			s1.ActiveTo = lo.ToPtr(time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC))
			s2.ActiveFrom = time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
			s2.ActiveTo = lo.ToPtr(time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC))

			err := subscription.ValidateUniqueConstraintByFeatures([]subscription.SubscriptionSpec{s1, s2})
			require.Error(t, err)

			// let's assert the error is correct
			issues, err := models.AsValidationIssues(err)
			require.NoError(t, err)
			require.Len(t, issues, 1)
			require.Equal(t, subscription.ErrOnlySingleBillableItemAllowedAtATime.Code(), issues[0].Code())

			attrs := issues[0].Attributes()
			detail, ok := attrs[subscription.ErrCodeOnlySingleBillableItemAllowedAtATime]
			require.True(t, ok)

			require.NotNil(t, detail)

			detailTyped, ok := detail.(subscription.SubscriptionUniqueConstraintErrorDetail)
			require.True(t, ok)

			require.Equal(t, subscription.SpecPath("/phases/test_phase_1/items/feature1/idx/0"), detailTyped.Left.Path)
			require.Equal(t, subscription.SpecPath("/phases/test_phase_1/items/feature1/idx/0"), detailTyped.Right.Path)
		})
	})

	t.Run("Should error if two subscriptions have overlapping features with entitlements", func(t *testing.T) {
		t.Fatal("Not implemented")
	})

	t.Run("Should error if two subscriptions have overlapping features with entitlements or are billable", func(t *testing.T) {
		t.Fatal("Not implemented")
	})

	t.Run("Should error if multiple subscriptions have overlaps", func(t *testing.T) {
		t.Fatal("Not implemented")
	})

	t.Run("Should not error if multiple subscriptions have overlapping timelines but we don't double charge or have doubled entitlements", func(t *testing.T) {
		t.Fatal("Not implemented")
	})
}

// builds an empty subscription without phases or items, used for testing the subscription level uniqueness constraint
func getSimpleSub(cad models.CadencedModel) subscription.SubscriptionSpec {
	sp := subscription.SubscriptionSpec{
		CreateSubscriptionPlanInput: subscription.CreateSubscriptionPlanInput{
			Plan: &subscription.PlanRef{
				Key: "test_plan",
			},
			BillingCadence: datetime.NewISODuration(0, 1, 0, 0, 0, 0, 0),
		},
		CreateSubscriptionCustomerInput: subscription.CreateSubscriptionCustomerInput{
			CustomerId:    "test_customer",
			ActiveFrom:    cad.ActiveFrom,
			ActiveTo:      cad.ActiveTo,
			Name:          "test_subscription",
			BillingAnchor: cad.ActiveFrom,
			Currency:      currencyx.Code(currency.USD),
		},
	}

	return sp
}
