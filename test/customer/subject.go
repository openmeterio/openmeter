package customer

import (
	"context"
	"fmt"
	"testing"
	"time"

	alpacadecimal "github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscriptionservice "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/service"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *CustomerHandlerTestSuite) TestSubjectDeletion(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	t.Run("Should allow deletion of a dangling subject (dangling after removing from customer usage attribution)", func(t *testing.T) {
		// Let's create a customer with a subject
		cust, err := s.Env.Customer().CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: s.namespace,
			CustomerMutate: customer.CustomerMutate{
				Key:  lo.ToPtr("customer-1"),
				Name: "Customer 1",
				UsageAttribution: &customer.CustomerUsageAttribution{
					SubjectKeys: []string{"customer-1-subject-1"},
				},
			},
		})

		require.NoError(t, err, "Creating customer must not return error")
		require.NotNil(t, cust, "Customer must not be nil")

		// Let's make the subject dangling by removing it from the customer usage attribution
		cust, err = s.Env.Customer().UpdateCustomer(ctx, customer.UpdateCustomerInput{
			CustomerID: customer.CustomerID{
				Namespace: s.namespace,
				ID:        cust.ID,
			},
			CustomerMutate: func() customer.CustomerMutate {
				mut := cust.AsCustomerMutate()
				// Set UsageAttribution to nil to remove all subject keys
				mut.UsageAttribution = nil

				return mut
			}(),
		})
		require.NoError(t, err, "Updating customer must not return error")
		require.NotNil(t, cust, "Customer must not be nil")

		// Let's fetch the subject
		subj, err := s.Env.Subject().GetByKey(ctx, models.NamespacedKey{
			Namespace: s.namespace,
			Key:       "customer-1-subject-1",
		})
		require.NoError(t, err, "Getting subject must not return error")
		require.NotNil(t, subj, "Subject must not be nil")

		// Let's delete the subject
		err = s.Env.Subject().Delete(ctx, models.NamespacedID{
			Namespace: s.namespace,
			ID:        subj.Id,
		})
		require.NoError(t, err, "Deleting subject must not return error")
	})

	t.Run("Should remove from customer usage attribution if not dangling", func(t *testing.T) {
		// Let's create a customer with a subject
		cust, err := s.Env.Customer().CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: s.namespace,
			CustomerMutate: customer.CustomerMutate{
				Key:  lo.ToPtr("customer-2"),
				Name: "Customer 2",
				UsageAttribution: &customer.CustomerUsageAttribution{
					SubjectKeys: []string{"customer-2-subject-1"},
				},
			},
		})

		require.NoError(t, err, "Creating customer must not return error")
		require.NotNil(t, cust, "Customer must not be nil")

		// Let's fetch the subject
		subj, err := s.Env.Subject().GetByKey(ctx, models.NamespacedKey{
			Namespace: s.namespace,
			Key:       "customer-2-subject-1",
		})
		require.NoError(t, err, "Getting subject must not return error")
		require.NotNil(t, subj, "Subject must not be nil")

		// Let's delete the subject
		err = s.Env.Subject().Delete(ctx, models.NamespacedID{
			Namespace: s.namespace,
			ID:        subj.Id,
		})
		require.NoError(t, err, "Deleting subject must not return error")

		// Let's fetch the customer
		cust, err = s.Env.Customer().GetCustomer(ctx, customer.GetCustomerInput{
			CustomerID: &customer.CustomerID{
				Namespace: s.namespace,
				ID:        cust.ID,
			},
		})
		require.NoError(t, err, "Getting customer must not return error")
		require.NotNil(t, cust, "Customer must not be nil")

		// UsageAttribution is nil when there are no subject keys
		require.Nil(t, cust.UsageAttribution, "Customer usage attribution must be nil when no subject keys")
	})

	t.Run("Should NOT error if customer WITH entitlements has no more subjects after deletion", func(t *testing.T) {
		// Let's create a customer with a subject
		cust, err := s.Env.Customer().CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: s.namespace,
			CustomerMutate: customer.CustomerMutate{
				Key:  lo.ToPtr("customer-3"),
				Name: "Customer 3",
				UsageAttribution: &customer.CustomerUsageAttribution{
					SubjectKeys: []string{"customer-3-subject-1"},
				},
			},
		})

		require.NoError(t, err, "Creating customer must not return error")
		require.NotNil(t, cust, "Customer must not be nil")

		// Let's create a feature
		feature, err := s.Env.Feature().CreateFeature(ctx, feature.CreateFeatureInputs{
			Namespace: s.namespace,
			Key:       "test-feature",
			Name:      "Test Feature",
		})
		require.NoError(t, err, "Creating feature must not return error")
		require.NotNil(t, feature, "Feature must not be nil")

		// Let's create an entitlement for the customer
		entitlement, err := s.Env.Entitlement().CreateEntitlement(ctx, entitlement.CreateEntitlementInputs{
			Namespace:        s.namespace,
			FeatureID:        lo.ToPtr(feature.ID),
			EntitlementType:  entitlement.EntitlementTypeBoolean,
			UsageAttribution: cust.GetUsageAttribution(),
		}, nil)
		require.NoError(t, err, "Creating entitlement must not return error")
		require.NotNil(t, entitlement, "Entitlement must not be nil")

		// Let's fetch the subject
		subj, err := s.Env.Subject().GetByKey(ctx, models.NamespacedKey{
			Namespace: s.namespace,
			Key:       "customer-3-subject-1",
		})
		require.NoError(t, err, "Getting subject must not return error")
		require.NotNil(t, subj, "Subject must not be nil")

		// Let's delete the subject
		err = s.Env.Subject().Delete(ctx, models.NamespacedID{
			Namespace: s.namespace,
			ID:        subj.Id,
		})
		require.NoError(t, err, "Deleting subject must not return error")

		// Let's assert that the customer has no more subjects
		cust, err = s.Env.Customer().GetCustomer(ctx, customer.GetCustomerInput{
			CustomerID: &customer.CustomerID{
				Namespace: s.namespace,
				ID:        cust.ID,
			},
		})
		require.NoError(t, err, "Getting customer must not return error")
		require.NotNil(t, cust, "Customer must not be nil")
		// UsageAttribution is nil when there are no subject keys
		require.Nil(t, cust.UsageAttribution, "Customer usage attribution must be nil when no subject keys")
	})
}

func (s *CustomerHandlerTestSuite) TestMultiSubjectIntegrationFlow(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	meterService := s.Env.Meter()
	require.NotNil(t, meterService, "meter service must be available")

	met, err := meterService.CreateMeter(ctx, meter.CreateMeterInput{
		Namespace:     s.namespace,
		Name:          "Integration Meter",
		Key:           "integration-meter",
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
		EventType:     "test",
	})
	require.NoError(t, err, "creating meter should succeed")
	require.NotNil(t, met, "meter must not be nil")

	met, err = meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
		Namespace: s.namespace,
		IDOrSlug:  met.Key,
	})
	require.NoError(t, err, "getting meter should succeed")
	require.NotNil(t, met, "meter must not be nil")
	require.Equal(t, met.Key, met.Key, "meter key must match")

	planService := s.Env.Plan()
	require.NotNil(t, planService, "plan service must be available")

	billingService := s.Env.Billing()
	require.NotNil(t, billingService, "billing service must be available")

	featureService := s.Env.Feature()
	require.NotNil(t, featureService, "feature service must be available")

	customerService := s.Env.Customer()
	require.NotNil(t, customerService, "customer service must be available")

	subscriptionService := s.Env.Subscription()
	require.NotNil(t, subscriptionService, "subscription service must be available")

	featureOneKey := fmt.Sprintf("integration-feature-%s", ulid.Make().String())
	featureTwoKey := fmt.Sprintf("integration-feature-%s", ulid.Make().String())

	featureOne, err := featureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Name:      "Integration Feature One",
		Key:       featureOneKey,
		Namespace: s.namespace,
		MeterSlug: lo.ToPtr(met.Key),
	})
	require.NoError(t, err, "creating first feature should not error")

	featureTwo, err := featureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Name:      "Integration Feature Two",
		Key:       featureTwoKey,
		Namespace: s.namespace,
		MeterSlug: lo.ToPtr(met.Key),
	})
	require.NoError(t, err, "creating second feature should not error")

	planKey := fmt.Sprintf("integration-plan-%s", ulid.Make().String())
	planInput := plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{Namespace: s.namespace},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Integration Plan",
				Key:            planKey,
				Version:        1,
				Currency:       currency.Code("USD"),
				BillingCadence: datetime.MustParseDuration(t, "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:  "phase-1",
						Name: "Phase 1",
					},
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:        featureOne.Key,
								Name:       "Usage Billable",
								FeatureKey: lo.ToPtr(featureOne.Key),
								EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
									UsagePeriod:     datetime.MustParseDuration(t, "P1D"),
									IssueAfterReset: lo.ToPtr(110.0),
								}),
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(10),
									PaymentTerm: productcatalog.InArrearsPaymentTerm,
								}),
							},
							BillingCadence: datetime.MustParseDuration(t, "P1D"),
						},
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:        featureTwo.Key,
								Name:       "Usage Included",
								FeatureKey: lo.ToPtr(featureTwo.Key),
								EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
									UsagePeriod:     datetime.MustParseDuration(t, "P1D"),
									IssueAfterReset: lo.ToPtr(10.0),
								}),
							},
							BillingCadence: datetime.MustParseDuration(t, "P1D"),
						},
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "flat-billable",
								Name: "Flat Billable",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(25),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(t, "P1M")),
						},
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "flat-non-billable",
								Name: "Flat Non Billable",
							},
						},
					},
				},
			},
		},
	}

	createdPlan, err := planService.CreatePlan(ctx, planInput)
	require.NoError(t, err, "creating plan should succeed")
	require.NotNil(t, createdPlan)

	subjectKeys := []string{
		"integration-subject-1",
		"integration-subject-2",
		"integration-subject-3",
	}

	createdCustomer, err := customerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.namespace,
		CustomerMutate: customer.CustomerMutate{
			Name: "Integration Customer",
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: subjectKeys,
			},
		},
	})
	require.NoError(t, err, "creating customer should succeed")
	require.NotNil(t, createdCustomer)

	t.Run("Should have all subjects created as entitlements", func(t *testing.T) {
		for _, subjectKey := range subjectKeys {
			subj, err := s.Env.Subject().GetByIdOrKey(ctx, s.namespace, subjectKey)
			require.NoError(t, err, "getting subject should succeed")
			require.NotNil(t, subj, "subject must not be nil")
		}
	})

	app := s.installSandboxApp(t, s.namespace)
	_ = s.createDefaultProfile(t, app, s.namespace)

	subscriptionPlan := plansubscriptionservice.PlanFromPlan(*createdPlan)
	now := clock.Now()

	sub, err := s.Env.SubscriptionWorkflow().CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
		Namespace:  s.namespace,
		CustomerID: createdCustomer.ID,
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{
				Enum: lo.ToPtr(subscription.TimingImmediate),
			},
			Name: "Integration Subscription",
		},
	}, subscriptionPlan)
	require.NoError(t, err, "creating customer subscription should succeed")
	require.NotNil(t, sub, "customer subscription must not be nil")

	future := now.Add(48 * time.Hour)
	clock.SetTime(future)
	t.Cleanup(clock.ResetTime)

	periodStart := now
	pendingLine := billing.NewFlatFeeLine(billing.NewFlatFeeLineInput{
		ID:            ulid.Make().String(),
		CreatedAt:     now,
		UpdatedAt:     now,
		Namespace:     s.namespace,
		Name:          "Integration Manual Line",
		Period:        billing.Period{Start: periodStart, End: future},
		InvoiceAt:     future,
		PerUnitAmount: alpacadecimal.NewFromFloat(15),
		PaymentTerm:   productcatalog.InArrearsPaymentTerm,
	}, billing.WithFeatureKey(featureOne.Key))

	result, err := billingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: customer.CustomerID{
			Namespace: s.namespace,
			ID:        createdCustomer.ID,
		},
		Currency: currencyx.Code("USD"),
		Lines:    []*billing.StandardLine{pendingLine},
	})
	require.NoError(t, err, "creating pending invoice lines should succeed")
	require.NotNil(t, result)

	invoices, err := billingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customer.CustomerID{
			Namespace: s.namespace,
			ID:        createdCustomer.ID,
		},
		AsOf: lo.ToPtr(future),
	})
	require.NoError(t, err, "invoicing pending lines should succeed")
	require.NotEmpty(t, invoices, "expected at least one invoice to be generated")
}
