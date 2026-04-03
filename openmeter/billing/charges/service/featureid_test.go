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

func (s *ChargeFeatureIDTestSuite) TestCreateResolvesFeatureIDsForUsageBasedAndFlatFeeCharges() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-service-feature-id-create")

	cust := s.CreateTestCustomer(ns, "feature-id-create")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	usageMeter := newTestMeter(ns, "usage-meter")
	flatFeeMeter := newTestMeter(ns, "flat-fee-meter")
	s.installMeters(ctx, usageMeter, flatFeeMeter)

	usageFeature := s.createFeature(ctx, ns, "usage-feature", usageMeter.ID)
	flatFeeFeature := s.createFeature(ctx, ns, "flat-fee-feature", flatFeeMeter.ID)

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
				settlementMode: productcatalog.InvoiceOnlySettlementMode,
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
	s.True(alpacadecimal.NewFromInt(7).Equal(currentTotals.Quantity))

	clock.SetTime(servicePeriod.To.Add(2 * time.Hour))
	s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued = func(ctx context.Context, input usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
		return creditrealization.CreateAllocationInputs{
			{
				Amount:        input.AmountToAllocate,
				ServicePeriod: input.Charge.Intent.ServicePeriod,
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
	s.True(alpacadecimal.NewFromInt(7).Equal(finalCharge.Realizations[0].MeterValue))
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
