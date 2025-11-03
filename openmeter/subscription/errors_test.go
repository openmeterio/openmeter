package subscription_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func TestSubscriptionErrors(t *testing.T) {
	t.Run("ErrSubscriptionBillingPeriodQueriedBeforeSubscriptionStart", func(t *testing.T) {
		queriedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		subscriptionStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

		err := subscription.NewErrSubscriptionBillingPeriodQueriedBeforeSubscriptionStart(queriedAt, subscriptionStart)

		require.True(t, subscription.IsValidationIssueWithCode(err, subscription.ErrCodeSubscriptionBillingPeriodQueriedBeforeSubscriptionStart))

		issues, err := models.AsValidationIssues(err)
		require.NoError(t, err)

		exts := issues.AsErrorExtensions()
		require.Len(t, exts, 1)

		ext := exts[0]
		require.Equal(t, subscription.ErrCodeSubscriptionBillingPeriodQueriedBeforeSubscriptionStart, ext["code"])
		require.Equal(t, "billing period queried before subscription start", ext["message"])
		require.Equal(t, subscriptionStart, ext["subscription_start"])
		require.Equal(t, queriedAt, ext["queried_at"])
	})
}

func TestSubscriptionSpecValidation(t *testing.T) {
	t.Run("Should be a valid subscription", func(t *testing.T) {
		spec := subscription.SubscriptionSpec{
			CreateSubscriptionPlanInput: subscription.CreateSubscriptionPlanInput{
				Plan: &subscription.PlanRef{
					Key:     "test",
					Version: 1,
				},
				BillingCadence: datetime.MustParseDuration(t, "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Mode: productcatalog.ProRatingModeProratePrices,
				},
			},
			CreateSubscriptionCustomerInput: subscription.CreateSubscriptionCustomerInput{
				Name:          "test",
				CustomerId:    "test",
				Currency:      currencyx.Code("USD"),
				ActiveFrom:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				BillingAnchor: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			Phases: map[string]*subscription.SubscriptionPhaseSpec{
				"phase1": {
					CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
						PhaseKey:   "phase1",
						StartAfter: datetime.MustParseDuration(t, "P0D"),
						Name:       "phase1",
					},
					CreateSubscriptionPhaseCustomerInput: subscription.CreateSubscriptionPhaseCustomerInput{},
					ItemsByKey: map[string][]*subscription.SubscriptionItemSpec{
						"item1": {
							{
								CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
									CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
										PhaseKey: "phase1",
										ItemKey:  "item1",
										RateCard: &productcatalog.UsageBasedRateCard{
											RateCardMeta: productcatalog.RateCardMeta{
												Key:        "item1",
												Name:       "item1",
												FeatureKey: lo.ToPtr("item1"),
												FeatureID:  lo.ToPtr("item1"),
												Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
													Amount: alpacadecimal.NewFromFloat(100),
												}),
												EntitlementTemplate: func() *productcatalog.EntitlementTemplate {
													et := productcatalog.EntitlementTemplate{}
													et.FromMetered(productcatalog.MeteredEntitlementTemplate{
														IssueAfterReset:         lo.ToPtr(10.0),
														IssueAfterResetPriority: lo.ToPtr(uint8(1)),
														UsagePeriod:             datetime.MustParseDuration(t, "P1M"),
													})

													return &et
												}(),
											},
											BillingCadence: datetime.MustParseDuration(t, "P1M"),
										},
									},
									CreateSubscriptionItemCustomerInput: subscription.CreateSubscriptionItemCustomerInput{},
								},
							},
						},
					},
				},
			},
		}

		err := spec.Validate()
		require.NoError(t, err)
	})

	t.Run("Should have expected errors", func(t *testing.T) {
		spec := subscription.SubscriptionSpec{
			CreateSubscriptionPlanInput: subscription.CreateSubscriptionPlanInput{
				Plan: &subscription.PlanRef{
					Key:     "test",
					Version: 1,
				},
				BillingCadence: datetime.MustParseDuration(t, "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Mode: productcatalog.ProRatingModeProratePrices,
				},
			},
			CreateSubscriptionCustomerInput: subscription.CreateSubscriptionCustomerInput{
				Name:          "test",
				CustomerId:    "test",
				Currency:      currencyx.Code("USD"),
				ActiveFrom:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				BillingAnchor: time.Date(2025, 1, 1, 5, 0, 0, 0, time.UTC), // will error
			},
			Phases: map[string]*subscription.SubscriptionPhaseSpec{
				"phase1": {
					CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
						PhaseKey:   "phase1",
						StartAfter: datetime.MustParseDuration(t, "P0D"),
						Name:       "phase1",
					},
					CreateSubscriptionPhaseCustomerInput: subscription.CreateSubscriptionPhaseCustomerInput{},
					ItemsByKey: map[string][]*subscription.SubscriptionItemSpec{
						"item1": {
							{
								CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
									CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
										PhaseKey: "phase1",
										ItemKey:  "item1",
										RateCard: &productcatalog.UsageBasedRateCard{
											RateCardMeta: productcatalog.RateCardMeta{
												Key:        "item1",
												Name:       "item1",
												FeatureKey: lo.ToPtr("badkey"), // will error
												FeatureID:  lo.ToPtr("badid"),
												Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
													Amount: alpacadecimal.NewFromFloat(100),
												}),
												EntitlementTemplate: func() *productcatalog.EntitlementTemplate {
													et := productcatalog.EntitlementTemplate{}
													et.FromMetered(productcatalog.MeteredEntitlementTemplate{
														// IssueAfterReset:         lo.ToPtr(10.0), // will error
														IssueAfterResetPriority: lo.ToPtr(uint8(1)),
														UsagePeriod:             datetime.MustParseDuration(t, "P1M"),
													})

													return &et
												}(),
											},
											BillingCadence: datetime.MustParseDuration(t, "P1M"),
										},
									},
									CreateSubscriptionItemCustomerInput: subscription.CreateSubscriptionItemCustomerInput{},
								},
							},
						},
					},
				},
				"phase2": {
					CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
						PhaseKey:   "phase2",
						StartAfter: datetime.MustParseDuration(t, "-P2D"), // will error
						Name:       "phase2",
					},
					CreateSubscriptionPhaseCustomerInput: subscription.CreateSubscriptionPhaseCustomerInput{},
					ItemsByKey:                           map[string][]*subscription.SubscriptionItemSpec{}, // will error
				},
			},
		}

		specValidateError := spec.Validate()
		err := specValidateError
		require.Error(t, err)

		issues, err := models.AsValidationIssues(err)
		require.NoError(t, err)

		exts := issues.AsErrorExtensions()

		byts, err := json.MarshalIndent(exts, "", "  ")
		require.NoError(t, err)

		require.Len(t, issues, 4, "got %s", string(byts))

		models.RequireValidationIssuesMatch(t, models.ValidationIssues{
			models.NewValidationIssue(
				subscription.ErrCodeSubscriptionPhaseStartAfterIsNegative,
				"subscription phase start after cannot be negative",
				models.WithField(
					models.NewFieldSelectorGroup(
						models.NewFieldSelector("phases"),
						models.NewFieldSelector("phase2"),
						models.NewFieldSelector("startAfter"),
					),
				),
				commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
			),
			models.NewValidationIssue(
				subscription.ErrCodeSubscriptionPhaseHasNoItems,
				"subscription phase must have at least one item",
				models.WithField(
					models.NewFieldSelectorGroup(
						models.NewFieldSelector("phases"),
						models.NewFieldSelector("phase2"),
						models.NewFieldSelector("items"),
					),
				),
				subscription.AllowedDuringApplyingToSpecError(),
				commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
			),
			models.NewValidationIssue(
				productcatalog.ErrCodeEntitlementTemplateInvalidIssueAfterResetWithPriority,
				"invalid entitlement template as issue after reset is required if issue after reset priority is set",
				models.WithField(
					models.NewFieldSelectorGroup(
						models.NewFieldSelector("phases"),
						models.NewFieldSelector("phase1"),
						models.NewFieldSelector("itemsByKey"),
						models.NewFieldSelector("item1").WithExpression(models.NewFieldArrIndex(0)),
						models.NewFieldSelector("entitlementTemplate"),
						models.NewFieldSelector("issueAfterReset"),
					),
				),
				models.WithComponent("rateCard"),
				models.WithWarningSeverity(),
				commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
			),
			models.NewValidationIssue(
				productcatalog.ErrCodeRateCardKeyFeatureKeyMismatch,
				"rate card key must match feature key",
				models.WithField(
					models.NewFieldSelectorGroup(
						models.NewFieldSelector("phases"),
						models.NewFieldSelector("phase1"),
						models.NewFieldSelector("itemsByKey"),
						models.NewFieldSelector("item1").WithExpression(models.NewFieldArrIndex(0)),
						models.NewFieldSelector("key"),
					),
				),
				models.WithComponent("rateCard"),
				commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
			),
		}, issues)

		// Expected issues:
		// - BillingAnchor is a hours after subscription start hours after subscription startfter subscription start
		// - Item FeatureKey does not match item key
		// - RateCard IssueAfterReset is not set
		// - Phase2 startAfter is negative
		// - Phase2 has no items

		t.Run("MapSubscriptionSpecValidationIssueFieldSelectors", func(t *testing.T) {
			ogErr := specValidateError
			require.Error(t, ogErr)
			issues, err := models.AsValidationIssues(ogErr)
			require.NoError(t, err)

			mapped, err := slicesx.MapWithErr(issues, func(issue models.ValidationIssue) (models.ValidationIssue, error) {
				return subscription.MapSubscriptionSpecValidationIssueField(issue)
			})

			require.NoError(t, err)

			models.RequireValidationIssuesMatch(t, models.ValidationIssues{
				models.NewValidationIssue(
					subscription.ErrCodeSubscriptionPhaseStartAfterIsNegative,
					"subscription phase start after cannot be negative",
					models.WithField(
						models.NewFieldSelectorGroup(
							models.NewFieldSelector("phases").WithExpression(models.NewFieldAttrValue("key", "phase2")),
							models.NewFieldSelector("startAfter"),
						),
					),
					commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
				),
				models.NewValidationIssue(
					subscription.ErrCodeSubscriptionPhaseHasNoItems,
					"subscription phase must have at least one item",
					models.WithField(
						models.NewFieldSelectorGroup(
							models.NewFieldSelector("phases").WithExpression(models.NewFieldAttrValue("key", "phase2")),
							models.NewFieldSelector("items"),
						),
					),
					subscription.AllowedDuringApplyingToSpecError(),
					commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
				),
				models.NewValidationIssue(
					productcatalog.ErrCodeEntitlementTemplateInvalidIssueAfterResetWithPriority,
					"invalid entitlement template as issue after reset is required if issue after reset priority is set",
					models.WithField(
						models.NewFieldSelectorGroup(
							models.NewFieldSelector("phases").WithExpression(models.NewFieldAttrValue("key", "phase1")),
							models.NewFieldSelector("itemsByKey"),
							models.NewFieldSelector("item1").WithExpression(models.NewFieldArrIndex(0)),
							models.NewFieldSelector("entitlementTemplate"),
							models.NewFieldSelector("issueAfterReset"),
						),
					),
					models.WithComponent("rateCard"),
					models.WithWarningSeverity(),
					commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
				),
				models.NewValidationIssue(
					productcatalog.ErrCodeRateCardKeyFeatureKeyMismatch,
					"rate card key must match feature key",
					models.WithField(
						models.NewFieldSelectorGroup(
							models.NewFieldSelector("phases").WithExpression(models.NewFieldAttrValue("key", "phase1")),
							models.NewFieldSelector("itemsByKey"),
							models.NewFieldSelector("item1").WithExpression(models.NewFieldArrIndex(0)),
							models.NewFieldSelector("key"),
						),
					),
					models.WithComponent("rateCard"),
					commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
				),
			}, mapped)

			t.Run("Should not map already mapped issue", func(t *testing.T) {
				iss := models.NewValidationIssue(
					productcatalog.ErrCodeRateCardKeyFeatureKeyMismatch,
					"rate card key must match feature key",
					models.WithField(
						models.NewFieldSelector("phases").WithExpression(models.NewFieldAttrValue("key", "phase1")),
						models.NewFieldSelector("itemsByKey"),
						models.NewFieldSelector("item1").WithExpression(models.NewFieldArrIndex(0)),
						models.NewFieldSelector("key"),
					),
				)

				require.Equal(t, "$.phases[?(@.key=='phase1')].itemsByKey.item1[0].key", iss.Field().JSONPath())

				mapped, err := subscription.MapSubscriptionSpecValidationIssueField(iss)
				require.NoError(t, err)

				models.RequireValidationIssuesMatch(t,
					models.ValidationIssues{iss},
					models.ValidationIssues{mapped},
				)
			})
		})
	})
}
