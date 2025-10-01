package subscription_test

import (
	"testing"
	"time"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestValidateUniqueConstraintBySubscriptions(t *testing.T) {
	getSimpleSub := func(cad models.CadencedModel) subscription.SubscriptionSpec {
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
