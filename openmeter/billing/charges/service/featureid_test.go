package service

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	featurepkg "github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func TestChargeFeatureIDs(t *testing.T) {
	suite.Run(t, new(ChargeFeatureIDTestSuite))
}

type ChargeFeatureIDTestSuite struct {
	BaseSuite
}

func (s *ChargeFeatureIDTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
}

func (s *ChargeFeatureIDTestSuite) TearDownTest() {
	s.BaseSuite.TearDownTest()
}

func (s *ChargeFeatureIDTestSuite) TestCreateResolvesFeatureIDsForUsageBasedAndMeterlessFlatFeeCharges() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-service-feature-id-create")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "feature-id-create")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	usageMeter := newTestMeter(ns, "usage-meter")
	s.installMeters(ctx, usageMeter)

	usageFeature := s.createFeature(ctx, ns, "usage-feature", usageMeter.ID)
	flatFeeFeature, err := s.FeatureService.CreateFeature(ctx, featurepkg.CreateFeatureInputs{
		Namespace: ns,
		Name:      "flat-fee-feature",
		Key:       "flat-fee-feature",
	})
	s.Require().NoError(err)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.SetTime(servicePeriod.From)

	createdCharges, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(25),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				name:              "flat-fee",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "flat-fee",
				featureKey:        flatFeeFeature.Key,
			}),
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditOnlySettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromFloat(2),
				}),
				name:              "usage-based",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "usage-based",
				featureKey:        usageFeature.Key,
			}),
		},
	})
	s.NoError(err)
	s.Len(createdCharges, 2)

	flatFeeCharge, err := createdCharges[0].AsFlatFeeCharge()
	s.NoError(err)
	s.NotNil(flatFeeCharge.State.FeatureID)
	s.Equal(flatFeeFeature.ID, *flatFeeCharge.State.FeatureID)

	usageBasedCharge, err := createdCharges[1].AsUsageBasedCharge()
	s.NoError(err)
	s.Equal(usageFeature.ID, usageBasedCharge.State.FeatureID)

	fetchedFlatFee, err := s.mustGetChargeByID(flatFeeCharge.GetChargeID()).AsFlatFeeCharge()
	s.NoError(err)
	s.NotNil(fetchedFlatFee.State.FeatureID)
	s.Equal(flatFeeFeature.ID, *fetchedFlatFee.State.FeatureID)

	fetchedUsageBased, err := s.mustGetChargeByID(usageBasedCharge.GetChargeID()).AsUsageBasedCharge()
	s.NoError(err)
	s.Equal(usageFeature.ID, fetchedUsageBased.State.FeatureID)
}

func (s *ChargeFeatureIDTestSuite) TestUsageBasedActivationRecalculatesFeatureIDAndRunsKeepUsingIt() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-service-feature-id-usage")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "feature-id-usage")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(), billingtest.WithCollectionInterval(lo.Must(datetime.ISODurationString("PT1H").Parse())))

	meterV1 := newTestMeter(ns, "versioned-meter-v1")
	meterV2 := newTestMeter(ns, "versioned-meter-v2")
	meterV3 := newTestMeter(ns, "versioned-meter-v3")
	s.installMeters(ctx, meterV1, meterV2, meterV3)

	const featureKey = "versioned-usage-feature"

	featureV1 := s.createFeature(ctx, ns, featureKey, meterV1.ID)

	baseTime := time.Now().UTC().Truncate(time.Minute)
	servicePeriod := timeutil.ClosedPeriod{
		From: baseTime.Add(2 * time.Hour),
		To:   baseTime.Add(26 * time.Hour),
	}

	clock.SetTime(servicePeriod.From.Add(-time.Hour))

	createdCharges, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditOnlySettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromFloat(3),
				}),
				name:              "usage-based-versioned",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "usage-based-versioned",
				featureKey:        featureKey,
			}),
		},
	})
	s.NoError(err)
	s.Len(createdCharges, 1)

	createdCharge, err := createdCharges[0].AsUsageBasedCharge()
	s.NoError(err)
	s.Equal(meta.ChargeStatusCreated, meta.ChargeStatus(createdCharge.Status))
	s.Equal(featureV1.ID, createdCharge.State.FeatureID)

	s.archiveFeature(ctx, ns, featureV1.ID)
	featureV2 := s.createFeature(ctx, ns, featureKey, meterV2.ID)

	s.MockStreamingConnector.AddSimpleEvent(meterV2.Key, 7, servicePeriod.From.Add(30*time.Minute))

	clock.SetTime(servicePeriod.From)
	advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: cust.GetID(),
	})
	s.NoError(err)
	s.Len(advancedCharges, 1)

	activatedCharge, err := s.mustGetChargeByID(createdCharge.GetChargeID()).AsUsageBasedCharge()
	s.NoError(err)
	s.Equal(meta.ChargeStatusActive, meta.ChargeStatus(activatedCharge.Status))
	s.Equal(featureV2.ID, activatedCharge.State.FeatureID)

	s.archiveFeature(ctx, ns, featureV2.ID)
	_ = s.createFeature(ctx, ns, featureKey, meterV3.ID)
	s.MockStreamingConnector.AddSimpleEvent(meterV3.Key, 11, servicePeriod.From.Add(30*time.Minute))

	clock.SetTime(servicePeriod.From.Add(31 * time.Minute))
	currentTotals, err := s.UsageBasedService.GetCurrentTotals(ctx, usagebased.GetCurrentTotalsInput{
		ChargeID: activatedCharge.GetChargeID(),
	})
	s.NoError(err)
	s.True(alpacadecimal.NewFromInt(21).Equal(currentTotals.DueTotals.Total))

	clock.SetTime(servicePeriod.To.Add(2 * time.Hour))
	s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued = func(ctx context.Context, input usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
		return creditrealization.CreateAllocationInputs{
			{
				Amount:        input.AmountToAllocate,
				ServicePeriod: input.Charge.Intent.GetEffectiveServicePeriod(),
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: ulid.Make().String(),
				},
			},
		}, nil
	}
	_, err = s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: cust.GetID(),
	})
	s.NoError(err)

	finalCharge, err := s.mustGetChargeByID(createdCharge.GetChargeID()).AsUsageBasedCharge()
	s.NoError(err)
	s.Equal(featureV2.ID, finalCharge.State.FeatureID)
	s.Len(finalCharge.Realizations, 1)
	s.Equal(featureV2.ID, finalCharge.Realizations[0].FeatureID)
	s.True(alpacadecimal.NewFromInt(7).Equal(finalCharge.Realizations[0].MeteredQuantity))
}

// TestCreateCustomerChargeResolvesFeatureByID covers the v3 API boundary, where the
// caller names a feature by ID instead of key. The charge service must resolve that ID
// and backfill the intent's feature key, because the key is what persistence, rating,
// gathering lines, and activation-time re-resolution all read.
func (s *ChargeFeatureIDTestSuite) TestCreateCustomerChargeResolvesFeatureByID() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-service-feature-id-api")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "feature-id-api")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	usageMeter := newTestMeter(ns, "api-usage-meter")
	flatFeeMeter := newTestMeter(ns, "api-flat-fee-meter")
	s.installMeters(ctx, usageMeter, flatFeeMeter)

	usageFeature := s.createFeature(ctx, ns, "api-usage-feature", usageMeter.ID)
	flatFeeFeature := s.createFeature(ctx, ns, "api-flat-fee-feature", flatFeeMeter.ID)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime(),
	}

	// Stay before the service period so the charges settle in `created` and no
	// auto-advance runs; this test is about create-time feature resolution only.
	clock.SetTime(servicePeriod.From.Add(-time.Hour))
	defer clock.UnFreeze()

	s.Run("flat fee charge created by feature ID", func() {
		// given:
		// - a caller that only knows the feature ID, as the v3 create endpoint does
		discounts := billing.DiscountsFromProductCatalog(productcatalog.Discounts{
			Percentage: &productcatalog.PercentageDiscount{Percentage: models.NewPercentage(10)},
		})
		flatFeeInput := &charges.CreateCustomerChargeFlatFeeInput{
			IntentMutableFields: flatfee.IntentMutableFields{
				IntentMutableFields: meta.IntentMutableFields{
					Name:              "flat fee by feature id",
					ServicePeriod:     servicePeriod,
					FullServicePeriod: servicePeriod,
					BillingPeriod:     servicePeriod,
				},
				InvoiceAt:             servicePeriod.From,
				PaymentTerm:           productcatalog.InAdvancePaymentTerm,
				PercentageDiscounts:   discounts.Percentage,
				AmountBeforeProration: alpacadecimal.NewFromInt(25),
			},
			FeatureID:      lo.ToPtr(flatFeeFeature.ID),
			SettlementMode: productcatalog.CreditOnlySettlementMode,
		}

		// when:
		created, err := s.Charges.CreateCustomerCharge(ctx, charges.CreateCustomerChargeInput{
			Namespace:    ns,
			CustomerID:   cust.ID,
			CurrencyCode: USD,
			FlatFee:      flatFeeInput,
		})
		s.Require().NoError(err)

		// then:
		// - the resolved version is persisted, and the key is backfilled for downstream consumers
		flatFeeCharge, err := created.AsFlatFeeCharge()
		s.Require().NoError(err)
		s.Require().NotNil(flatFeeCharge.State.FeatureID)
		s.Equal(flatFeeFeature.ID, *flatFeeCharge.State.FeatureID)
		s.Equal(flatFeeFeature.Key, flatFeeCharge.Intent.GetFeatureKey())
		s.Require().NotNil(flatFeeCharge.Intent.GetEffectiveIntent().PercentageDiscounts)
		s.NotEmpty(flatFeeCharge.Intent.GetEffectiveIntent().PercentageDiscounts.CorrelationID)

		fetched, err := s.mustGetChargeByID(flatFeeCharge.GetChargeID()).AsFlatFeeCharge()
		s.Require().NoError(err)
		s.Require().NotNil(fetched.State.FeatureID)
		s.Equal(flatFeeFeature.ID, *fetched.State.FeatureID)
		s.Equal(flatFeeFeature.Key, fetched.Intent.GetFeatureKey())
	})

	s.Run("usage based charge created by feature ID", func() {
		discounts := billing.DiscountsFromProductCatalog(productcatalog.Discounts{
			Usage: &productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(1)},
		})
		usageBasedInput := &charges.CreateCustomerChargeUsageBasedInput{
			IntentMutableFields: usagebased.IntentMutableFields{
				IntentMutableFields: meta.IntentMutableFields{
					Name:              "usage based by feature id",
					ServicePeriod:     servicePeriod,
					FullServicePeriod: servicePeriod,
					BillingPeriod:     servicePeriod,
				},
				InvoiceAt: servicePeriod.From,
				Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(2),
				}),
				Discounts: discounts,
			},
			FeatureID:      usageFeature.ID,
			SettlementMode: productcatalog.CreditOnlySettlementMode,
		}

		// when:
		created, err := s.Charges.CreateCustomerCharge(ctx, charges.CreateCustomerChargeInput{
			Namespace:    ns,
			CustomerID:   cust.ID,
			CurrencyCode: USD,
			UsageBased:   usageBasedInput,
		})
		s.Require().NoError(err)

		// then:
		usageBasedCharge, err := created.AsUsageBasedCharge()
		s.Require().NoError(err)
		s.Equal(usageFeature.ID, usageBasedCharge.State.FeatureID)
		s.Equal(usageFeature.Key, usageBasedCharge.Intent.GetFeatureKey())
		s.Require().NotNil(usageBasedCharge.Intent.GetEffectiveIntent().Discounts.Usage)
		s.NotEmpty(usageBasedCharge.Intent.GetEffectiveIntent().Discounts.Usage.CorrelationID)

		fetched, err := s.mustGetChargeByID(usageBasedCharge.GetChargeID()).AsUsageBasedCharge()
		s.Require().NoError(err)
		s.Equal(usageFeature.ID, fetched.State.FeatureID)
		s.Equal(usageFeature.Key, fetched.Intent.GetFeatureKey())
	})

	s.Run("flat fee charge without a feature stays featureless", func() {
		// given:
		// - flat-fee features are optional, unlike usage-based
		created, err := s.Charges.CreateCustomerCharge(ctx, charges.CreateCustomerChargeInput{
			Namespace:    ns,
			CustomerID:   cust.ID,
			CurrencyCode: USD,
			FlatFee: &charges.CreateCustomerChargeFlatFeeInput{
				IntentMutableFields: flatfee.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "flat fee without feature",
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					InvoiceAt:             servicePeriod.From,
					PaymentTerm:           productcatalog.InAdvancePaymentTerm,
					AmountBeforeProration: alpacadecimal.NewFromInt(5),
				},
				SettlementMode: productcatalog.CreditOnlySettlementMode,
			},
		})
		s.Require().NoError(err)

		// then:
		flatFeeCharge, err := created.AsFlatFeeCharge()
		s.Require().NoError(err)
		s.Nil(flatFeeCharge.State.FeatureID)
		s.Empty(flatFeeCharge.Intent.GetFeatureKey())
	})

	s.Run("unknown feature ID is rejected", func() {
		// given:
		// - an ID that resolves to no feature in this namespace
		unknownFeatureID := ulid.Make().String()

		// when:
		_, err := s.Charges.CreateCustomerCharge(ctx, charges.CreateCustomerChargeInput{
			Namespace:    ns,
			CustomerID:   cust.ID,
			CurrencyCode: USD,
			UsageBased: &charges.CreateCustomerChargeUsageBasedInput{
				IntentMutableFields: usagebased.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "usage based with unknown feature",
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					InvoiceAt: servicePeriod.From,
					Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(2),
					}),
				},
				FeatureID:      unknownFeatureID,
				SettlementMode: productcatalog.CreditOnlySettlementMode,
			},
		})

		// then:
		// - the failure surfaces as not-found rather than a downstream "feature key is required"
		s.Require().Error(err)
		s.True(models.IsGenericNotFoundError(err), "expected not found error, got %v", err)
	})
}

// TestCreateFlatFeeWithUnresolvableFeatureKeyReturnsError covers key-only producers
// (subscription sync, pending lines, invoice-line mappers) reaching the shared charge
// creation boundary with a feature that no longer resolves.
func (s *ChargeFeatureIDTestSuite) TestCreateFlatFeeWithUnresolvableFeatureKeyReturnsError() {
	// given:
	// - a flat-fee intent carrying only a feature key, for a feature that does not exist
	// when:
	// - the charge is created
	// then:
	// - creation returns a not-found error without persisting a charge
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-service-feature-key-unresolvable")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "feature-key-unresolvable")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-06-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-07-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.SetTime(servicePeriod.From.Add(-time.Hour))
	defer clock.UnFreeze()

	_, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(25),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				name:              "flat-fee-missing-feature",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "flat-fee-missing-feature",
				featureKey:        "this-feature-does-not-exist",
			}),
		},
	})

	s.Require().Error(err)
	s.ErrorContains(err, "resolve create feature meter")
	s.True(models.IsGenericNotFoundError(err), "expected not found error, got %v", err)
	s.False(models.IsGenericValidationError(err), "not found error must not be wrapped as validation: %v", err)

	listed, listErr := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Namespace:   ns,
		CustomerIDs: []string{cust.ID},
	})
	s.Require().NoError(listErr)
	s.Empty(listed.Items)
}

func (s *ChargeFeatureIDTestSuite) TestCreateUsageBasedWithUnresolvableFeatureKeyReturnsError() {
	// given:
	// - a usage-based intent references a feature key that does not exist
	// when:
	// - the charge is created
	// then:
	// - creation returns a not-found error without persisting a charge
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-usage-feature-key-unresolvable")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "usage-feature-key-unresolvable")
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-06-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-07-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.FreezeTime(servicePeriod.From.Add(-time.Hour))
	defer clock.UnFreeze()

	_, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditOnlySettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(2),
				}),
				name:       "usage-missing-feature",
				managedBy:  billing.ManuallyManagedLine,
				featureKey: "this-usage-feature-does-not-exist",
			}),
		},
	})

	s.Require().Error(err)
	s.True(models.IsGenericNotFoundError(err), "expected not found error, got %v", err)
	s.False(models.IsGenericValidationError(err), "not found error must not be wrapped as validation: %v", err)

	listed, listErr := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Namespace:   ns,
		CustomerIDs: []string{cust.ID},
	})
	s.Require().NoError(listErr)
	s.Empty(listed.Items)
}

func (s *ChargeFeatureIDTestSuite) TestCreateUsageBasedRequiresFeatureMeterBeforeCreatingAnyCharge() {
	// given:
	// - a valid flat-fee intent and a usage-based intent backed by a meterless feature
	// when:
	// - both charges are created in one request
	// then:
	// - validation rejects the whole request before either charge is persisted
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-usage-feature-meter-required")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "usage-feature-meter-required")
	meterlessFeature, err := s.FeatureService.CreateFeature(ctx, featurepkg.CreateFeatureInputs{
		Namespace: ns,
		Name:      "meterless feature",
		Key:       "meterless-feature",
	})
	s.Require().NoError(err)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-06-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-07-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.FreezeTime(servicePeriod.From.Add(-time.Hour))
	defer clock.UnFreeze()

	_, err = s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditOnlySettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(10),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				name:      "valid-featureless-flat-fee",
				managedBy: billing.ManuallyManagedLine,
			}),
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditOnlySettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(2),
				}),
				name:       "usage-without-feature-meter",
				managedBy:  billing.ManuallyManagedLine,
				featureKey: meterlessFeature.Key,
			}),
		},
	})

	s.Require().Error(err)
	s.True(models.IsGenericValidationError(err), "expected validation error, got %v", err)
	s.ErrorContains(err, "has no meter associated")

	listed, listErr := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Namespace:   ns,
		CustomerIDs: []string{cust.ID},
	})
	s.Require().NoError(listErr)
	s.Empty(listed.Items)
}

func (s *ChargeFeatureIDTestSuite) TestCreateUsageBasedMissingFeatureTakesPrecedenceOverMeterValidation() {
	// given:
	// - one usage-based intent references a missing feature and another a meterless feature
	// when:
	// - both charges are created in one request
	// then:
	// - the missing feature rejects the whole request as not-found before anything is persisted
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-usage-mixed-feature-errors")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "usage-mixed-feature-errors")
	meterlessFeature, err := s.FeatureService.CreateFeature(ctx, featurepkg.CreateFeatureInputs{
		Namespace: ns,
		Name:      "meterless feature",
		Key:       "meterless-feature",
	})
	s.Require().NoError(err)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-06-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-07-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.FreezeTime(servicePeriod.From.Add(-time.Hour))
	defer clock.UnFreeze()

	_, err = s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditOnlySettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(2),
				}),
				name:       "usage-missing-feature",
				managedBy:  billing.ManuallyManagedLine,
				featureKey: "this-usage-feature-does-not-exist",
			}),
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditOnlySettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(2),
				}),
				name:       "usage-without-feature-meter",
				managedBy:  billing.ManuallyManagedLine,
				featureKey: meterlessFeature.Key,
			}),
		},
	})

	s.Require().Error(err)
	s.True(models.IsGenericNotFoundError(err), "expected not found error, got %v", err)
	s.False(models.IsGenericValidationError(err), "not found error must take precedence over validation: %v", err)

	listed, listErr := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Namespace:   ns,
		CustomerIDs: []string{cust.ID},
	})
	s.Require().NoError(listErr)
	s.Empty(listed.Items)
}

// TestCreateCustomerChargeByIDResolvesLatestFeatureVersion pins the version semantics of
// feature_id: it names a feature, it does not pin the version the caller passed. Passing
// an archived ID resolves through its key to the current version, matching key-based and
// subscription-synced creates.
func (s *ChargeFeatureIDTestSuite) TestCreateCustomerChargeByIDResolvesLatestFeatureVersion() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-service-feature-id-archived")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "feature-id-archived")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	meterV1 := newTestMeter(ns, "archived-meter-v1")
	meterV2 := newTestMeter(ns, "archived-meter-v2")
	s.installMeters(ctx, meterV1, meterV2)

	const featureKey = "archived-usage-feature"

	// given:
	// - a feature that has been re-versioned: v1 archived, v2 current under the same key
	featureV1 := s.createFeature(ctx, ns, featureKey, meterV1.ID)
	s.archiveFeature(ctx, ns, featureV1.ID)
	featureV2 := s.createFeature(ctx, ns, featureKey, meterV2.ID)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-04-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-05-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.SetTime(servicePeriod.From.Add(-time.Hour))
	defer clock.UnFreeze()

	// when:
	// - the caller creates against the archived v1 ID
	created, err := s.Charges.CreateCustomerCharge(ctx, charges.CreateCustomerChargeInput{
		Namespace:    ns,
		CustomerID:   cust.ID,
		CurrencyCode: USD,
		UsageBased: &charges.CreateCustomerChargeUsageBasedInput{
			IntentMutableFields: usagebased.IntentMutableFields{
				IntentMutableFields: meta.IntentMutableFields{
					Name:              "usage based against archived feature",
					ServicePeriod:     servicePeriod,
					FullServicePeriod: servicePeriod,
					BillingPeriod:     servicePeriod,
				},
				InvoiceAt: servicePeriod.From,
				Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(4),
				}),
			},
			FeatureID:      featureV1.ID,
			SettlementMode: productcatalog.CreditOnlySettlementMode,
		},
	})

	// then:
	// - the create succeeds against the archived ID, and the intent carries the shared key
	s.Require().NoError(err)

	usageBasedCharge, err := created.AsUsageBasedCharge()
	s.Require().NoError(err)
	s.Equal(featureKey, usageBasedCharge.Intent.GetFeatureKey())
	s.Equal(featureV1.ID, usageBasedCharge.State.FeatureID)
	s.NotEqual(featureV2.ID, usageBasedCharge.State.FeatureID)
}

func (s *ChargeFeatureIDTestSuite) installMeters(ctx context.Context, meters ...meter.Meter) {
	s.T().Helper()
	require.NoError(s.T(), s.MeterAdapter.ReplaceMeters(ctx, meters))
}

func (s *ChargeFeatureIDTestSuite) createFeature(ctx context.Context, namespace, key, meterID string) featurepkg.Feature {
	s.T().Helper()

	feat, err := s.FeatureService.CreateFeature(ctx, featurepkg.CreateFeatureInputs{
		Namespace: namespace,
		Name:      key,
		Key:       key,
		MeterID:   lo.ToPtr(meterID),
	})
	require.NoError(s.T(), err)

	return feat
}

func (s *ChargeFeatureIDTestSuite) archiveFeature(ctx context.Context, namespace, featureID string) {
	s.T().Helper()
	require.NoError(s.T(), s.FeatureService.ArchiveFeature(ctx, models.NamespacedID{
		Namespace: namespace,
		ID:        featureID,
	}))
}

func newTestMeter(namespace, key string) meter.Meter {
	now := time.Now()

	return meter.Meter{
		ManagedResource: models.ManagedResource{
			ID: ulid.Make().String(),
			NamespacedModel: models.NamespacedModel{
				Namespace: namespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name: key,
		},
		Key:           key,
		Aggregation:   meter.MeterAggregationSum,
		EventType:     "test",
		ValueProperty: lo.ToPtr("$.value"),
	}
}
