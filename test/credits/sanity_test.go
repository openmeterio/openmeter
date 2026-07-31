package credits

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbledgerbreakagerecord "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerbreakagerecord"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	pcfeature "github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func TestSanitySuite(t *testing.T) {
	suite.Run(t, new(SanitySuite))
}

type SanitySuite struct {
	BaseSuite
}

func (s *SanitySuite) TestFlatFeeCreditOnlyDeleteCorrectionSanity() {
	setup := s.setupFlatFeeCreditOnlyDeleteCorrection("charges-sanity-flatfee-credit-only-delete")

	clock.FreezeTime(setup.createAt)
	defer clock.UnFreeze()

	// Given a credit-only flat fee that will be corrected by deleting the charge.
	chargeID := s.createAndAdvanceFlatFeeCreditOnlyCharge(setup)

	// Then the unfunded realization sits on the nil-cost-basis receivable/accrued route.
	s.assertUnfundedCreditOnlyRealization(setup.customer.GetID(), setup.amount, chargeID)

	// When the original charge is deleted with refund-as-credits.
	s.deleteChargeWithRefundAsCredits(setup.ctx, setup.customer.GetID(), chargeID)

	// Then the unfunded receivable/accrued route is fully cleared.
	s.assertUnfundedCreditOnlyDeleted(setup.customer.GetID())
}

func (s *SanitySuite) TestUsageBasedCreditOnlyDeleteCorrectionSanity() {
	setup := s.setupClosedPeriodUsageBasedCreditOnlyCollection("charges-sanity-usagebased-credit-only-delete")

	clock.FreezeTime(setup.createAt)
	defer clock.UnFreeze()

	// Given usage occurred in the already-closed service period.
	s.recordUsageInClosedServicePeriod(setup)

	// When the credit-only usage charge is created after the service period, it finalizes immediately.
	chargeID := s.createFinalizedUsageBasedCreditOnlyCharge(setup)

	// Then the unfunded realization sits on the nil-cost-basis receivable/accrued route.
	s.assertUnfundedCreditOnlyRealization(setup.customer.GetID(), setup.amount, chargeID)

	// When the original charge is deleted with refund-as-credits.
	s.deleteChargeWithRefundAsCredits(setup.ctx, setup.customer.GetID(), chargeID)

	// Then the unfunded receivable/accrued route is fully cleared.
	s.assertUnfundedCreditOnlyDeleted(setup.customer.GetID())
}

func (s *SanitySuite) TestUsageBasedCreditOnlyCollectionDoesNotUseCreditsGrantedAfterServicePeriodSanity() {
	setup := s.setupClosedPeriodUsageBasedCreditOnlyCollection("charges-sanity-usagebased-credit-only-post-period-grant")
	customerID := setup.customer.GetID()
	costBasis := alpacadecimal.Zero
	costBasisFilter := mo.Some(&costBasis)
	servicePeriodFrom := datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime()
	usageAt := datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime()
	servicePeriodTo := datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime()
	grantAndCollectionAt := datetime.MustParseTimeInLocation(s.T(), "2026-02-03T00:00:00Z", time.UTC).AsTime()

	s.Equal(servicePeriodFrom, setup.servicePeriod.From)
	s.Equal(servicePeriodTo, setup.servicePeriod.To)
	s.Equal(grantAndCollectionAt, setup.createAt)

	clock.FreezeTime(setup.createAt)
	defer clock.UnFreeze()

	// given:
	// - usage exists inside the already-closed service period
	// - a grant is created after the service period, before final collection runs
	s.Equal(usageAt, s.recordUsageInClosedServicePeriod(setup))
	funding := s.CreatePromotionalCreditFunding(setup.ctx, CreatePromotionalCreditFundingInput{
		Namespace: setup.namespace,
		Customer:  customerID,
		Amount:    setup.amount,
		At:        setup.createAt,
		CostBasis: costBasis,
	})

	// when:
	// - the usage-based credit-only charge finalizes for the closed service period
	chargeID := s.createFinalizedUsageBasedCreditOnlyCharge(setup)

	// then:
	// - the post-period grant should remain available
	// - the closed-period usage should be booked as unattributed advance-backed usage
	sourceChargeID := funding.Charge.ID
	s.AssertDecimalEqual(setup.amount, s.MustCustomerFBOBalanceAsOf(customerID, USD, costBasisFilter, setup.createAt), "post-period grant FBO after collection")
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerAccruedBalance(customerID, USD, costBasisFilter), "post-period grant cost-basis accrued after collection")
	s.AssertDecimalEqual(setup.amount, s.MustCustomerAccruedBalance(customerID, USD, mo.Some[*alpacadecimal.Decimal](nil)), "unattributed accrued after collection")
	s.AssertDecimalEqual(setup.amount.Neg(), s.MustCustomerReceivableBalance(customerID, USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen), "unattributed open receivable after collection")
	s.requireCustomerFBOSourceBalanceBucketsAsOf(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasisFilter,
	}, setup.createAt, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, nil): setup.amount.InexactFloat64(), // 8 = the post-period grant is still available.
	})
	s.requireCustomerAccruedSourceSpendBalanceBuckets(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: mo.Some[*alpacadecimal.Decimal](nil),
	}, map[string]float64{
		sourceSpendChargeBucketKey(nil, &chargeID): setup.amount.InexactFloat64(), // 8 = closed-period usage is not tied to the future grant.
	})
	s.requireCustomerAccruedSourceSpendBalanceBuckets(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasisFilter,
	}, map[string]float64{})
}

func (s *SanitySuite) TestFlatFeeFundedCreditOnlyRecognizedRevenueDeleteCorrectionSanity() {
	setup := s.setupFlatFeeCreditOnlyDeleteCorrection("charges-sanity-flatfee-funded-credit-only-recognized-delete")
	zeroCostBasis := alpacadecimal.Zero

	clock.FreezeTime(setup.createAt)
	defer clock.UnFreeze()

	// given:
	// - zero-cost-basis promotional credits fund the customer before the charge is realized
	funding := s.createPromotionalCreditFunding(setup, zeroCostBasis)
	startOpenReceivable := funding.OpenReceivable

	// given:
	// - a credit-only flat fee that will be corrected by deleting the charge
	chargeID := s.createAndAdvanceFlatFeeCreditOnlyCharge(setup)

	// then:
	// - the funded credits move from FBO to accrued, without changing the grant's receivable
	s.assertFundedCreditOnlyAccrued(setup.customer.GetID(), setup.amount, zeroCostBasis, startOpenReceivable, funding.Charge.ID, chargeID)

	// when:
	// - revenue recognition moves the accrued funded amount into earnings
	s.recognizeFundedCreditOnlyRevenue(setup.namespace, setup.customer.GetID(), setup.amount, zeroCostBasis, funding.Charge.ID, chargeID)

	// when:
	// - the original charge is deleted with refund-as-credits
	s.deleteChargeWithRefundAsCredits(setup.ctx, setup.customer.GetID(), chargeID)

	// then:
	// - the recognized earnings are corrected back out and the funded credits return to FBO
	s.assertFundedRecognizedCreditOnlyDeleted(setup.namespace, setup.customer.GetID(), setup.amount, zeroCostBasis, startOpenReceivable, funding.Charge.ID)
}

func (s *SanitySuite) TestUsageBasedFundedCreditOnlyRecognizedRevenueDeleteCorrectionSanity() {
	setup := s.setupClosedPeriodUsageBasedCreditOnlyCollection("charges-sanity-usagebased-funded-credit-only-recognized-delete")
	zeroCostBasis := alpacadecimal.Zero
	fundingAt := setup.servicePeriod.From

	clock.FreezeTime(setup.createAt)
	defer clock.UnFreeze()

	// given:
	// - zero-cost-basis promotional credits are effective before the service period closes
	funding := s.createPromotionalCreditFundingAt(setup, zeroCostBasis, fundingAt)
	startOpenReceivable := funding.OpenReceivable

	// given:
	// - usage occurred in the already-closed service period
	s.recordUsageInClosedServicePeriod(setup)

	// when:
	// - the credit-only usage charge is created after the service period, it finalizes immediately
	chargeID := s.createFinalizedUsageBasedCreditOnlyCharge(setup)

	// then:
	// - the funded credits move from FBO to accrued, without changing the grant's receivable
	s.assertFundedCreditOnlyAccrued(setup.customer.GetID(), setup.amount, zeroCostBasis, startOpenReceivable, funding.Charge.ID, chargeID)

	// when:
	// - revenue recognition moves the accrued funded amount into earnings
	s.recognizeFundedCreditOnlyRevenue(setup.namespace, setup.customer.GetID(), setup.amount, zeroCostBasis, funding.Charge.ID, chargeID)

	// when:
	// - the original charge is deleted with refund-as-credits
	s.deleteChargeWithRefundAsCredits(setup.ctx, setup.customer.GetID(), chargeID)

	// then:
	// - the recognized earnings are corrected back out and the funded credits return to FBO
	s.assertFundedRecognizedCreditOnlyDeleted(setup.namespace, setup.customer.GetID(), setup.amount, zeroCostBasis, startOpenReceivable, funding.Charge.ID)
}

func (s *SanitySuite) TestExpiringCreditBreakagePlanReleaseAndExpirySanity() {
	setup := s.setupExpiringCreditBreakage("charges-sanity-expiring-credit-breakage")
	defer clock.UnFreeze()

	// Given an expiring promotional grant.
	clock.FreezeTime(setup.grantAt)
	funding := s.CreatePromotionalCreditFunding(setup.ctx, CreatePromotionalCreditFundingInput{
		Namespace: setup.namespace,
		Customer:  setup.customer.GetID(),
		Amount:    setup.grantAmount,
		At:        setup.grantAt,
		ExpiresAt: &setup.expiresAt,
		CostBasis: setup.costBasis,
	})
	s.NotEmpty(funding.Charge.Realizations.CreditGrantRealization.TransactionGroupID)

	// Then expiry is pre-booked as planned breakage.
	costBasis := mo.Some(&setup.costBasis)
	customerID := setup.customer.GetID()
	s.assertPlannedBreakage(plannedBreakageAssertionInput{
		ctx:       setup.ctx,
		namespace: setup.namespace,
		customer:  customerID,
		currency:  USD,
		costBasis: costBasis,
		amount:    setup.grantAmount,
		createdAt: setup.grantAt,
		expiresAt: setup.expiresAt,
	})

	// When usage consumes part of the expiring credit before expiry.
	charge := s.createAndAdvanceCreditOnlyFlatFeeCharge(createCreditOnlyFlatFeeChargeInput{
		ctx:           setup.ctx,
		namespace:     setup.namespace,
		customer:      setup.customer.GetID(),
		servicePeriod: timeutil.ClosedPeriod{From: setup.usageAt, To: setup.usageAt.Add(time.Hour)},
		createAt:      setup.usageAt.Add(-time.Hour),
		advanceAt:     setup.usageAt,
		amount:        setup.usedAmount,
		name:          "expiring-credit-breakage-usage",
	}).charge
	s.Require().Len(s.mustFlatFeeCreditRealizations(charge), 1)
	s.AssertDecimalEqual(setup.usedAmount, s.mustFlatFeeCreditRealizations(charge)[0].Amount, "used credit realization amount")

	// Then the used portion releases planned breakage, and only the unused remainder breaks at expiry.
	sourceChargeID := funding.Charge.ID
	spendChargeID := charge.ID
	s.assertReleasedBreakage(releasedBreakageAssertionInput{
		ctx:             setup.ctx,
		namespace:       setup.namespace,
		customer:        customerID,
		currency:        USD,
		costBasis:       costBasis,
		planAmount:      setup.grantAmount,
		releaseAmount:   setup.usedAmount,
		asOf:            setup.usageAt,
		expectedFBO:     setup.unusedAmount,
		expectedAccrued: setup.usedAmount,
	})
	s.requireCustomerFBOSourceBalanceBucketsAsOf(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, setup.usageAt, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, nil): setup.unusedAmount.InexactFloat64(), // 4 = 10 grant - 6 used by the charge.
	})
	s.requireCustomerAccruedSourceSpendBalanceBucketsAsOf(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, setup.usageAt, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, &spendChargeID): setup.usedAmount.InexactFloat64(), // 6 = the used grant amount remains tied to the spend charge.
	})
	s.assertBreakageBalancesAt(breakageBalanceAssertionInput{
		namespace:        setup.namespace,
		customer:         customerID,
		currency:         USD,
		costBasis:        costBasis,
		asOf:             setup.expiresAt,
		expectedFBO:      alpacadecimal.Zero,
		expectedBreakage: setup.unusedAmount,
		label:            "at expiry after usage",
	})
	s.requireBreakageSourceBalanceBucketsAsOf(setup.namespace, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, setup.expiresAt, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, nil): setup.unusedAmount.InexactFloat64(), // 4 = only unused credit breaks; spend provenance is not meaningful on breakage.
	})
}

func (s *SanitySuite) TestExpiringCreditBreakageImmediatelyReleasesAdvanceBackfillSanity() {
	setup := s.setupClosedPeriodUsageBasedCreditOnlyCollection("charges-sanity-expiring-credit-breakage-advance-backfill")
	defer clock.UnFreeze()

	costBasisValue := alpacadecimal.Zero
	costBasis := mo.Some(&costBasisValue)
	customerID := setup.customer.GetID()
	advanceAmount := setup.amount
	grantAmount := alpacadecimal.NewFromInt(12)
	unusedPurchaseAmount := grantAmount.Sub(advanceAmount)
	expiresAt := setup.createAt.Add(7 * 24 * time.Hour)
	usageAt := setup.servicePeriod.From.Add(24 * time.Hour)
	chargeCreateAt := setup.createAt
	backfillAt := chargeCreateAt.Add(time.Hour)

	// Given usage consumes advance because no real credit exists yet.
	clock.FreezeTime(chargeCreateAt)
	s.MockStreamingConnector.AddSimpleEvent(
		setup.featureKey,
		advanceAmount.InexactFloat64(),
		usageAt,
	)
	chargeRes, err := s.Charges.Create(setup.ctx, charges.CreateInput{
		Namespace: setup.namespace,
		Intents: charges.ChargeIntents{
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:       customerID,
				Currency:       USD,
				ServicePeriod:  setup.servicePeriod,
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(1),
				}),
				Name:              "expiring-credit-breakage-advance-usage",
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: "expiring-credit-breakage-advance-usage",
				FeatureKey:        setup.featureKey,
			}),
		},
	})
	s.Require().NoError(err)
	s.Require().Len(chargeRes, 1)

	usageCharge, err := chargeRes[0].AsUsageBasedCharge()
	s.Require().NoError(err)
	s.Equal(meta.ChargeStatusFinal, meta.ChargeStatus(usageCharge.Status))
	s.AssertDecimalEqual(advanceAmount.Neg(), s.MustCustomerReceivableBalance(customerID, USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen), "advance receivable before backfill")
	s.AssertDecimalEqual(advanceAmount, s.MustCustomerAccruedBalance(customerID, USD, mo.Some[*alpacadecimal.Decimal](nil)), "advance accrued before backfill")
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerFBOBalance(customerID, USD, costBasis), "FBO before backfill")

	// When a later expiring grant covers the advance and has extra unused value.
	clock.FreezeTime(backfillAt)
	funding := s.createPromotionalCreditGrant(setup.ctx, CreatePromotionalCreditFundingInput{
		Namespace: setup.namespace,
		Customer:  customerID,
		Amount:    grantAmount,
		At:        backfillAt,
		ExpiresAt: &expiresAt,
		CostBasis: costBasisValue,
	})

	// Then the covered advance slice is planned and immediately released from breakage.
	s.assertAdvanceBackfillBreakageRows(setup.ctx, setup.namespace, customerID, grantAmount, advanceAmount, expiresAt)
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerAccruedBalance(customerID, USD, mo.Some[*alpacadecimal.Decimal](nil)), "advance accrued after backfill")
	s.AssertDecimalEqual(advanceAmount, s.MustCustomerAccruedBalance(customerID, USD, costBasis), "attributed accrued after backfill")
	s.AssertDecimalEqual(unusedPurchaseAmount, s.MustCustomerFBOBalanceAsOf(customerID, USD, costBasis, backfillAt), "available FBO after backfill")
	sourceChargeID := funding.ID
	spendChargeID := usageCharge.ID
	s.requireCustomerAccruedSourceSpendBalanceBucketsAsOf(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, backfillAt, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, &spendChargeID): advanceAmount.InexactFloat64(), // 8 = the advance usage is now backed by the grant.
	})
	s.requireCustomerFBOSourceBalanceBucketsAsOf(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, backfillAt, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, nil): unusedPurchaseAmount.InexactFloat64(), // 4 = 12 grant - 8 immediately backfilled advance.
	})
	s.assertBreakageBalancesAt(breakageBalanceAssertionInput{
		namespace:        setup.namespace,
		customer:         customerID,
		currency:         USD,
		costBasis:        costBasis,
		asOf:             backfillAt,
		expectedFBO:      unusedPurchaseAmount,
		expectedBreakage: alpacadecimal.Zero,
		label:            "after advance backfill before expiry",
	})
	s.assertBreakageBalancesAt(breakageBalanceAssertionInput{
		namespace:        setup.namespace,
		customer:         customerID,
		currency:         USD,
		costBasis:        costBasis,
		asOf:             expiresAt,
		expectedFBO:      alpacadecimal.Zero,
		expectedBreakage: unusedPurchaseAmount,
		label:            "at expiry after advance backfill",
	})
	s.requireBreakageSourceBalanceBucketsAsOf(setup.namespace, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, expiresAt, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, nil): unusedPurchaseAmount.InexactFloat64(), // 4 = only the unused grant surplus expires.
	})
}

func (s *SanitySuite) TestExpiringCreditBreakageReopensAdvanceBackfillReleaseOnUsageCorrectionSanity() {
	setup := s.setupClosedPeriodUsageBasedCreditOnlyCollection("charges-sanity-expiring-credit-breakage-advance-backfill-correction")
	defer clock.UnFreeze()

	costBasisValue := alpacadecimal.Zero
	costBasis := mo.Some(&costBasisValue)
	customerID := setup.customer.GetID()
	advanceAmount := setup.amount
	grantAmount := alpacadecimal.NewFromInt(12)
	unusedPurchaseAmount := grantAmount.Sub(advanceAmount)
	expiresAt := setup.createAt.Add(7 * 24 * time.Hour)
	usageAt := setup.servicePeriod.From.Add(24 * time.Hour)
	chargeCreateAt := setup.createAt
	backfillAt := chargeCreateAt.Add(time.Hour)
	correctionAt := backfillAt.Add(time.Hour)

	// Given usage consumes advance because no real credit exists yet.
	clock.FreezeTime(chargeCreateAt)
	s.MockStreamingConnector.AddSimpleEvent(
		setup.featureKey,
		advanceAmount.InexactFloat64(),
		usageAt,
	)
	chargeRes, err := s.Charges.Create(setup.ctx, charges.CreateInput{
		Namespace: setup.namespace,
		Intents: charges.ChargeIntents{
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:       customerID,
				Currency:       USD,
				ServicePeriod:  setup.servicePeriod,
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(1),
				}),
				Name:              "expiring-credit-breakage-advance-correction-usage",
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: "expiring-credit-breakage-advance-correction-usage",
				FeatureKey:        setup.featureKey,
			}),
		},
	})
	s.Require().NoError(err)
	s.Require().Len(chargeRes, 1)

	usageCharge, err := chargeRes[0].AsUsageBasedCharge()
	s.Require().NoError(err)
	s.Equal(meta.ChargeStatusFinal, meta.ChargeStatus(usageCharge.Status))

	// And a later expiring grant covers that advance and has extra unused value.
	clock.FreezeTime(backfillAt)
	funding := s.createPromotionalCreditGrant(setup.ctx, CreatePromotionalCreditFundingInput{
		Namespace: setup.namespace,
		Customer:  customerID,
		Amount:    grantAmount,
		At:        backfillAt,
		ExpiresAt: &expiresAt,
		CostBasis: costBasisValue,
	})

	s.assertAdvanceBackfillBreakageRows(setup.ctx, setup.namespace, customerID, grantAmount, advanceAmount, expiresAt)
	s.AssertDecimalEqual(unusedPurchaseAmount, s.MustCustomerFBOBalanceAsOf(customerID, USD, costBasis, backfillAt), "available FBO after backfill before correction")
	s.AssertDecimalEqual(advanceAmount, s.MustCustomerAccruedBalance(customerID, USD, costBasis), "attributed accrued after backfill before correction")
	sourceChargeID := funding.ID
	spendChargeID := usageCharge.ID
	s.requireCustomerAccruedSourceSpendBalanceBucketsAsOf(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, backfillAt, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, &spendChargeID): advanceAmount.InexactFloat64(), // 8 = the advance is covered by the grant before correction.
	})
	s.assertBreakageBalancesAt(breakageBalanceAssertionInput{
		namespace:        setup.namespace,
		customer:         customerID,
		currency:         USD,
		costBasis:        costBasis,
		asOf:             expiresAt,
		expectedFBO:      alpacadecimal.Zero,
		expectedBreakage: unusedPurchaseAmount,
		label:            "at expiry after advance backfill before correction",
	})

	// When the original usage is deleted with refund-as-credits.
	clock.FreezeTime(correctionAt)
	s.deleteChargeWithRefundAsCredits(setup.ctx, customerID, usageCharge.ID)

	// Then the advance-backed release reopens because the covered credit is unused again.
	s.assertAdvanceBackfillBreakageReopenedRows(setup.ctx, setup.namespace, customerID, grantAmount, advanceAmount, expiresAt)
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerAccruedBalance(customerID, USD, mo.Some[*alpacadecimal.Decimal](nil)), "unattributed accrued after advance correction")
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerAccruedBalance(customerID, USD, costBasis), "attributed accrued after advance correction")
	s.AssertDecimalEqual(grantAmount, s.MustCustomerFBOBalanceAsOf(customerID, USD, costBasis, correctionAt), "available FBO after advance correction")

	sourceOnlyGrantAmount := grantAmount.InexactFloat64() // 12 = 10 corrected advance + 2 never-spent surplus from the grant.
	s.requireCustomerFBOSourceBalanceBucketsAsOf(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, correctionAt, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, nil): sourceOnlyGrantAmount, // freed credit is source-attributed again, but no longer tied to the corrected spend.
	})
	s.requireCustomerAccruedSourceSpendBalanceBucketsAsOf(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, correctionAt, map[string]float64{}) // 0 = the corrected spend no longer has active accrued value.
	s.requireBreakageSourceBalanceBucketsAsOf(setup.namespace, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, expiresAt, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, nil): sourceOnlyGrantAmount, // 12 = the full grant is unused by expiry after correction, and breakage keeps source-only provenance.
	})

	s.assertBreakageBalancesAt(breakageBalanceAssertionInput{
		namespace:        setup.namespace,
		customer:         customerID,
		currency:         USD,
		costBasis:        costBasis,
		asOf:             correctionAt,
		expectedFBO:      grantAmount,
		expectedBreakage: alpacadecimal.Zero,
		label:            "after advance correction before expiry",
	})
	s.assertBreakageBalancesAt(breakageBalanceAssertionInput{
		namespace:        setup.namespace,
		customer:         customerID,
		currency:         USD,
		costBasis:        costBasis,
		asOf:             expiresAt,
		expectedFBO:      alpacadecimal.Zero,
		expectedBreakage: grantAmount,
		label:            "at expiry after advance correction",
	})
}

func (s *SanitySuite) TestExpiringCreditBreakageReopensOnUsageCorrectionSanity() {
	setup := s.setupExpiringCreditBreakage(
		"charges-sanity-expiring-credit-breakage-correction",
		withExpiringCreditBreakageAmounts(alpacadecimal.NewFromInt(12), alpacadecimal.NewFromInt(5)),
	)
	defer clock.UnFreeze()

	costBasis := mo.Some(&setup.costBasis)
	customerID := setup.customer.GetID()

	// Given an expiring promotional grant.
	clock.FreezeTime(setup.grantAt)
	funding := s.CreatePromotionalCreditFunding(setup.ctx, CreatePromotionalCreditFundingInput{
		Namespace: setup.namespace,
		Customer:  customerID,
		Amount:    setup.grantAmount,
		At:        setup.grantAt,
		ExpiresAt: &setup.expiresAt,
		CostBasis: setup.costBasis,
	})

	// Then expiry is pre-booked as planned breakage.
	s.assertPlannedBreakage(plannedBreakageAssertionInput{
		ctx:       setup.ctx,
		namespace: setup.namespace,
		customer:  customerID,
		currency:  USD,
		costBasis: costBasis,
		amount:    setup.grantAmount,
		createdAt: setup.grantAt,
		expiresAt: setup.expiresAt,
	})

	// When usage consumes part of the expiring credit before expiry.
	charge := s.createAndAdvanceCreditOnlyFlatFeeCharge(createCreditOnlyFlatFeeChargeInput{
		ctx:           setup.ctx,
		namespace:     setup.namespace,
		customer:      customerID,
		servicePeriod: timeutil.ClosedPeriod{From: setup.usageAt, To: setup.usageAt.Add(time.Hour)},
		createAt:      setup.usageAt.Add(-time.Hour),
		advanceAt:     setup.usageAt,
		amount:        setup.usedAmount,
		name:          "expiring-credit-breakage-correction-usage",
	}).charge
	s.Require().Len(s.mustFlatFeeCreditRealizations(charge), 1)
	s.AssertDecimalEqual(setup.usedAmount, s.mustFlatFeeCreditRealizations(charge)[0].Amount, "used credit realization amount before correction")

	// Then the used portion releases planned breakage.
	s.assertReleasedBreakage(releasedBreakageAssertionInput{
		ctx:             setup.ctx,
		namespace:       setup.namespace,
		customer:        customerID,
		currency:        USD,
		costBasis:       costBasis,
		planAmount:      setup.grantAmount,
		releaseAmount:   setup.usedAmount,
		asOf:            setup.usageAt,
		expectedFBO:     setup.unusedAmount,
		expectedAccrued: setup.usedAmount,
	})

	// When the full usage charge is deleted with refund-as-credits.
	correctionAt := setup.usageAt.Add(time.Hour)
	clock.FreezeTime(correctionAt)
	s.deleteChargeWithRefundAsCredits(setup.ctx, customerID, charge.ID)

	// Then the correction reopens the released breakage and restores FBO.
	sourceChargeID := funding.Charge.ID
	s.assertReopenedBreakage(reopenedBreakageAssertionInput{
		ctx:             setup.ctx,
		namespace:       setup.namespace,
		customer:        customerID,
		currency:        USD,
		costBasis:       costBasis,
		planAmount:      setup.grantAmount,
		releaseAmount:   setup.usedAmount,
		reopenAmount:    setup.usedAmount,
		asOf:            correctionAt,
		expectedFBO:     setup.grantAmount,
		expectedAccrued: alpacadecimal.Zero,
	})
	s.requireCustomerFBOSourceBalanceBucketsAsOf(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, correctionAt, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, nil): setup.grantAmount.InexactFloat64(), // 12 = the full corrected grant is available again.
	})
	s.requireCustomerAccruedSourceSpendBalanceBucketsAsOf(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, correctionAt, map[string]float64{}) // 0 = deleting the usage fully clears spend-attributed accrued.

	// Then the full restored amount breaks at expiry.
	s.assertBreakageBalancesAt(breakageBalanceAssertionInput{
		namespace:        setup.namespace,
		customer:         customerID,
		currency:         USD,
		costBasis:        costBasis,
		asOf:             setup.expiresAt,
		expectedFBO:      alpacadecimal.Zero,
		expectedBreakage: setup.grantAmount,
		label:            "at expiry after usage correction",
	})
	s.requireBreakageSourceBalanceBucketsAsOf(setup.namespace, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, setup.expiresAt, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, nil): setup.grantAmount.InexactFloat64(), // 12 = the restored source expires in full.
	})
}

func (s *SanitySuite) TestExpiringCreditBreakagePartiallyReopensOnUsageShrinkSanity() {
	setup := s.setupExpiringCreditBreakage(
		"charges-sanity-expiring-credit-breakage-partial-correction",
		withExpiringCreditBreakageAmounts(alpacadecimal.NewFromInt(12), alpacadecimal.NewFromInt(8)),
	)
	defer clock.UnFreeze()

	costBasis := mo.Some(&setup.costBasis)
	customerID := setup.customer.GetID()
	retainedUsage := alpacadecimal.NewFromInt(5)
	correctedUsage := setup.usedAmount.Sub(retainedUsage)

	// Given an expiring promotional grant.
	clock.FreezeTime(setup.grantAt)
	funding := s.CreatePromotionalCreditFunding(setup.ctx, CreatePromotionalCreditFundingInput{
		Namespace: setup.namespace,
		Customer:  customerID,
		Amount:    setup.grantAmount,
		At:        setup.grantAt,
		ExpiresAt: &setup.expiresAt,
		CostBasis: setup.costBasis,
	})

	// When usage consumes part of the expiring credit before expiry.
	charge := s.createAndAdvanceCreditOnlyFlatFeeCharge(createCreditOnlyFlatFeeChargeInput{
		ctx:           setup.ctx,
		namespace:     setup.namespace,
		customer:      customerID,
		servicePeriod: timeutil.ClosedPeriod{From: setup.usageAt, To: setup.usageAt.Add(time.Hour)},
		createAt:      setup.usageAt.Add(-time.Hour),
		advanceAt:     setup.usageAt,
		amount:        setup.usedAmount,
		name:          "expiring-credit-breakage-partial-correction-usage",
	}).charge
	s.Require().Len(s.mustFlatFeeCreditRealizations(charge), 1)
	s.assertReleasedBreakage(releasedBreakageAssertionInput{
		ctx:             setup.ctx,
		namespace:       setup.namespace,
		customer:        customerID,
		currency:        USD,
		costBasis:       costBasis,
		planAmount:      setup.grantAmount,
		releaseAmount:   setup.usedAmount,
		asOf:            setup.usageAt,
		expectedFBO:     setup.grantAmount.Sub(setup.usedAmount),
		expectedAccrued: setup.usedAmount,
	})

	// When only part of the usage allocation is corrected.
	correctionAt := setup.usageAt.Add(time.Hour)
	clock.FreezeTime(correctionAt)
	s.correctCreditUsageAllocation(setup.ctx, charge, s.mustFlatFeeCreditRealizations(charge)[0], correctedUsage, correctionAt)

	// Then only the corrected part reopens breakage.
	sourceChargeID := funding.Charge.ID
	spendChargeID := charge.ID
	s.assertReopenedBreakage(reopenedBreakageAssertionInput{
		ctx:             setup.ctx,
		namespace:       setup.namespace,
		customer:        customerID,
		currency:        USD,
		costBasis:       costBasis,
		planAmount:      setup.grantAmount,
		releaseAmount:   setup.usedAmount,
		reopenAmount:    correctedUsage,
		asOf:            correctionAt,
		expectedFBO:     setup.grantAmount.Sub(retainedUsage),
		expectedAccrued: retainedUsage,
	})
	s.AssertDecimalEqual(retainedUsage, s.MustCustomerAccruedBalance(customerID, USD, costBasis), "accrued after partial reopen")
	s.requireCustomerFBOSourceBalanceBucketsAsOf(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, correctionAt, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, nil): setup.grantAmount.Sub(retainedUsage).InexactFloat64(), // 7 = 12 grant - 5 retained usage.
	})
	s.requireCustomerAccruedSourceSpendBalanceBucketsAsOf(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, correctionAt, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, &spendChargeID): retainedUsage.InexactFloat64(), // 5 = usage still retained after shrinking from 8.
	})
	s.assertBreakageBalancesAt(breakageBalanceAssertionInput{
		namespace:        setup.namespace,
		customer:         customerID,
		currency:         USD,
		costBasis:        costBasis,
		asOf:             setup.expiresAt,
		expectedFBO:      alpacadecimal.Zero,
		expectedBreakage: setup.grantAmount.Sub(retainedUsage),
		label:            "at expiry after partial usage correction",
	})
	s.requireBreakageSourceBalanceBucketsAsOf(setup.namespace, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, setup.expiresAt, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, nil): setup.grantAmount.Sub(retainedUsage).InexactFloat64(), // 7 = only the unused part of the source breaks.
	})
}

func (s *SanitySuite) TestExpiringCreditBreakageReopensLatestExpirationFirstOnUsageShrinkSanity() {
	setup := s.setupExpiringCreditBreakage(
		"charges-sanity-expiring-credit-breakage-multi-expiry-correction",
		withExpiringCreditBreakageAmounts(alpacadecimal.NewFromInt(10), alpacadecimal.NewFromInt(8)),
	)
	defer clock.UnFreeze()

	costBasis := mo.Some(&setup.costBasis)
	customerID := setup.customer.GetID()
	firstExpiresAt := setup.usageAt.Add(3 * 24 * time.Hour)
	secondExpiresAt := setup.expiresAt
	firstGrantAmount := alpacadecimal.NewFromInt(5)
	secondGrantAmount := alpacadecimal.NewFromInt(5)
	retainedUsage := alpacadecimal.NewFromInt(4)
	correctedUsage := setup.usedAmount.Sub(retainedUsage)

	// Given two expiring grants with the same FBO route but different expiration dates.
	clock.FreezeTime(setup.grantAt)
	firstFunding := s.createPromotionalCreditGrant(setup.ctx, CreatePromotionalCreditFundingInput{
		Namespace: setup.namespace,
		Customer:  customerID,
		Amount:    firstGrantAmount,
		At:        setup.grantAt,
		ExpiresAt: &firstExpiresAt,
		CostBasis: setup.costBasis,
	})
	secondFunding := s.createPromotionalCreditGrant(setup.ctx, CreatePromotionalCreditFundingInput{
		Namespace: setup.namespace,
		Customer:  customerID,
		Amount:    secondGrantAmount,
		At:        setup.grantAt,
		ExpiresAt: &secondExpiresAt,
		CostBasis: setup.costBasis,
	})

	// When usage consumes across both expirations, breakage is released in expiration order.
	charge := s.createAndAdvanceCreditOnlyFlatFeeCharge(createCreditOnlyFlatFeeChargeInput{
		ctx:           setup.ctx,
		namespace:     setup.namespace,
		customer:      customerID,
		servicePeriod: timeutil.ClosedPeriod{From: setup.usageAt, To: setup.usageAt.Add(time.Hour)},
		createAt:      setup.usageAt.Add(-time.Hour),
		advanceAt:     setup.usageAt,
		amount:        setup.usedAmount,
		name:          "expiring-credit-breakage-multi-expiry-correction-usage",
	}).charge
	s.Require().Len(s.mustFlatFeeCreditRealizations(charge), 1)
	s.assertBreakageRowsByExpiry(setup.ctx, setup.namespace, customerID, []breakageRowsByExpiryAssertion{
		{expiresAt: firstExpiresAt, planAmount: firstGrantAmount, releaseAmount: firstGrantAmount},
		{expiresAt: secondExpiresAt, planAmount: secondGrantAmount, releaseAmount: alpacadecimal.NewFromInt(3)},
	})

	// When the charge allocation is partially corrected, correction unwinds the latest consumed expiration first.
	correctionAt := setup.usageAt.Add(time.Hour)
	clock.FreezeTime(correctionAt)
	s.correctCreditUsageAllocation(setup.ctx, charge, s.mustFlatFeeCreditRealizations(charge)[0], correctedUsage, correctionAt)

	// Then the later expiration is fully reopened before the earlier expiration is partially reopened.
	firstSourceChargeID := firstFunding.ID
	secondSourceChargeID := secondFunding.ID
	spendChargeID := charge.ID
	s.assertBreakageRowsByExpiry(setup.ctx, setup.namespace, customerID, []breakageRowsByExpiryAssertion{
		{expiresAt: firstExpiresAt, planAmount: firstGrantAmount, releaseAmount: firstGrantAmount, reopenAmount: alpacadecimal.NewFromInt(1)},
		{expiresAt: secondExpiresAt, planAmount: secondGrantAmount, releaseAmount: alpacadecimal.NewFromInt(3), reopenAmount: alpacadecimal.NewFromInt(3)},
	})
	s.requireCustomerFBOSourceBalanceBucketsAsOf(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, correctionAt, map[string]float64{
		sourceSpendChargeBucketKey(&firstSourceChargeID, nil):  1, // 1 = first source had 5 used, then 1 reopened.
		sourceSpendChargeBucketKey(&secondSourceChargeID, nil): 5, // 5 = second source had 3 used, then all 3 reopened plus 2 unused.
	})
	s.requireCustomerAccruedSourceSpendBalanceBucketsAsOf(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, correctionAt, map[string]float64{
		sourceSpendChargeBucketKey(&firstSourceChargeID, &spendChargeID): retainedUsage.InexactFloat64(), // 4 = retained usage stays on the earliest consumed source.
	})
	s.assertBreakageBalancesAt(breakageBalanceAssertionInput{
		namespace:        setup.namespace,
		customer:         customerID,
		currency:         USD,
		costBasis:        costBasis,
		asOf:             correctionAt,
		expectedFBO:      setup.grantAmount.Sub(retainedUsage),
		expectedBreakage: alpacadecimal.Zero,
		label:            "after multi-expiry partial correction before expiry",
	})
	s.AssertDecimalEqual(retainedUsage, s.MustCustomerAccruedBalance(customerID, USD, costBasis), "accrued after multi-expiry partial correction")
	s.assertBreakageBalancesAt(breakageBalanceAssertionInput{
		namespace:        setup.namespace,
		customer:         customerID,
		currency:         USD,
		costBasis:        costBasis,
		asOf:             secondExpiresAt,
		expectedFBO:      alpacadecimal.Zero,
		expectedBreakage: setup.grantAmount.Sub(retainedUsage),
		label:            "after all expirations",
	})
	s.requireBreakageSourceBalanceBucketsAsOf(setup.namespace, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, secondExpiresAt, map[string]float64{
		sourceSpendChargeBucketKey(&firstSourceChargeID, nil):  1, // 1 = first source reopened amount eventually expires.
		sourceSpendChargeBucketKey(&secondSourceChargeID, nil): 5, // 5 = second source is fully unused by its expiry.
	})
}

func (s *SanitySuite) TestExpiringCreditBreakageIgnoresNonExpiringSourceOnUsageShrinkSanity() {
	setup := s.setupExpiringCreditBreakage(
		"charges-sanity-expiring-credit-breakage-non-expiring-correction",
		withExpiringCreditBreakageAmounts(alpacadecimal.NewFromInt(10), alpacadecimal.NewFromInt(8)),
	)
	defer clock.UnFreeze()

	costBasis := mo.Some(&setup.costBasis)
	customerID := setup.customer.GetID()
	expiringAmount := alpacadecimal.NewFromInt(5)
	nonExpiringAmount := alpacadecimal.NewFromInt(5)
	retainedUsage := alpacadecimal.NewFromInt(4)
	correctedUsage := setup.usedAmount.Sub(retainedUsage)
	nonExpiringCorrectedUsage := setup.usedAmount.Sub(expiringAmount)
	expiringReopenAmount := correctedUsage.Sub(nonExpiringCorrectedUsage)

	// Given an expiring grant and a non-expiring grant on the same FBO route.
	clock.FreezeTime(setup.grantAt)
	expiringFunding := s.createPromotionalCreditGrant(setup.ctx, CreatePromotionalCreditFundingInput{
		Namespace: setup.namespace,
		Customer:  customerID,
		Amount:    expiringAmount,
		At:        setup.grantAt,
		ExpiresAt: &setup.expiresAt,
		CostBasis: setup.costBasis,
	})
	nonExpiringFunding := s.createPromotionalCreditGrant(setup.ctx, CreatePromotionalCreditFundingInput{
		Namespace: setup.namespace,
		Customer:  customerID,
		Amount:    nonExpiringAmount,
		At:        setup.grantAt,
		CostBasis: setup.costBasis,
	})

	// When usage consumes all expiring credit plus some non-expiring credit.
	charge := s.createAndAdvanceCreditOnlyFlatFeeCharge(createCreditOnlyFlatFeeChargeInput{
		ctx:           setup.ctx,
		namespace:     setup.namespace,
		customer:      customerID,
		servicePeriod: timeutil.ClosedPeriod{From: setup.usageAt, To: setup.usageAt.Add(time.Hour)},
		createAt:      setup.usageAt.Add(-time.Hour),
		advanceAt:     setup.usageAt,
		amount:        setup.usedAmount,
		name:          "expiring-credit-breakage-non-expiring-correction-usage",
	}).charge
	s.Require().Len(s.mustFlatFeeCreditRealizations(charge), 1)
	s.assertBreakageRowsByExpiry(setup.ctx, setup.namespace, customerID, []breakageRowsByExpiryAssertion{
		{expiresAt: setup.expiresAt, planAmount: expiringAmount, releaseAmount: expiringAmount},
	})

	// When the allocation is partially corrected, correction first unwinds the non-expiring source.
	correctionAt := setup.usageAt.Add(time.Hour)
	clock.FreezeTime(correctionAt)
	s.correctCreditUsageAllocation(setup.ctx, charge, s.mustFlatFeeCreditRealizations(charge)[0], correctedUsage, correctionAt)

	// Then only the corrected amount that came from expiring credit reopens breakage.
	expiringSourceChargeID := expiringFunding.ID
	nonExpiringSourceChargeID := nonExpiringFunding.ID
	spendChargeID := charge.ID
	s.assertBreakageRowsByExpiry(setup.ctx, setup.namespace, customerID, []breakageRowsByExpiryAssertion{
		{expiresAt: setup.expiresAt, planAmount: expiringAmount, releaseAmount: expiringAmount, reopenAmount: expiringReopenAmount},
	})
	s.requireCustomerFBOSourceBalanceBucketsAsOf(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, correctionAt, map[string]float64{
		sourceSpendChargeBucketKey(&expiringSourceChargeID, nil):    1, // 1 = 5 expiring grant - 4 retained expiring usage.
		sourceSpendChargeBucketKey(&nonExpiringSourceChargeID, nil): 5, // 5 = non-expiring source is fully restored before expiring source is reopened.
	})
	s.requireCustomerAccruedSourceSpendBalanceBucketsAsOf(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, correctionAt, map[string]float64{
		sourceSpendChargeBucketKey(&expiringSourceChargeID, &spendChargeID): retainedUsage.InexactFloat64(), // 4 = retained usage stays on the expiring source.
	})
	s.assertBreakageBalancesAt(breakageBalanceAssertionInput{
		namespace:        setup.namespace,
		customer:         customerID,
		currency:         USD,
		costBasis:        costBasis,
		asOf:             correctionAt,
		expectedFBO:      setup.grantAmount.Sub(retainedUsage),
		expectedBreakage: alpacadecimal.Zero,
		label:            "after non-expiring partial correction before expiry",
	})
	s.AssertDecimalEqual(retainedUsage, s.MustCustomerAccruedBalance(customerID, USD, costBasis), "accrued after non-expiring partial correction")
	s.assertBreakageBalancesAt(breakageBalanceAssertionInput{
		namespace:        setup.namespace,
		customer:         customerID,
		currency:         USD,
		costBasis:        costBasis,
		asOf:             setup.expiresAt,
		expectedFBO:      nonExpiringAmount,
		expectedBreakage: alpacadecimal.NewFromInt(1),
		label:            "at expiry after non-expiring partial correction",
	})
	s.requireCustomerFBOSourceBalanceBucketsAsOf(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, setup.expiresAt, map[string]float64{
		sourceSpendChargeBucketKey(&nonExpiringSourceChargeID, nil): nonExpiringAmount.InexactFloat64(), // 5 = non-expiring credit remains customer credit after expiry.
	})
	s.requireBreakageSourceBalanceBucketsAsOf(setup.namespace, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasis,
	}, setup.expiresAt, map[string]float64{
		sourceSpendChargeBucketKey(&expiringSourceChargeID, nil): 1, // 1 = only reopened unused expiring credit breaks.
	})
}

func (s *SanitySuite) TestFeatureRestrictedCreditCollectionCorrectionThenCollectionSanity() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-sanity-feature-credit-correction-collection")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	feature := s.SetupApiRequestsTotalFeature(ctx, ns)
	featureKey := feature.Feature.Key
	costBasis := alpacadecimal.Zero
	costBasisFilter := mo.Some(&costBasis)
	featureRoute := mo.Some([]string{featureKey})
	generalRoute := mo.Some[[]string](nil)
	restrictedPriority := 1
	generalPriority := 2
	grantAt := datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime()
	firstUsageAt := datetime.MustParseTimeInLocation(s.T(), "2026-01-02T00:00:00Z", time.UTC).AsTime()
	correctionAt := firstUsageAt.Add(time.Hour)
	secondUsageAt := correctionAt.Add(time.Hour)

	defer clock.UnFreeze()
	clock.FreezeTime(grantAt)

	// Given feature-restricted credit and general-purpose credit are both available.
	restrictedFunding := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
		Namespace:      ns,
		Customer:       cust.GetID(),
		Amount:         alpacadecimal.NewFromInt(4),
		At:             grantAt,
		CostBasis:      costBasis,
		Priority:       &restrictedPriority,
		FeatureFilters: creditpurchase.FeatureFilters{featureKey},
	})
	generalFunding := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Amount:    alpacadecimal.NewFromInt(6),
		At:        grantAt,
		CostBasis: costBasis,
		Priority:  &generalPriority,
	})
	s.AssertDecimalEqual(alpacadecimal.NewFromInt(4), s.MustCustomerFBOBalanceWithPriorityForFeatures(cust.GetID(), USD, costBasisFilter, restrictedPriority, featureRoute), "feature-restricted FBO before first usage")
	s.AssertDecimalEqual(alpacadecimal.NewFromInt(6), s.MustCustomerFBOBalanceWithPriorityForFeatures(cust.GetID(), USD, costBasisFilter, generalPriority, generalRoute), "general FBO before first usage")
	restrictedSourceChargeID := restrictedFunding.Charge.ID
	generalSourceChargeID := generalFunding.Charge.ID

	// When feature-keyed usage consumes more than the restricted credit alone can cover.
	firstCharge := s.createAndAdvanceCreditOnlyFlatFeeCharge(createCreditOnlyFlatFeeChargeInput{
		ctx:           ctx,
		namespace:     ns,
		customer:      cust.GetID(),
		servicePeriod: timeutil.ClosedPeriod{From: firstUsageAt, To: firstUsageAt.Add(time.Hour)},
		createAt:      firstUsageAt.Add(-time.Hour),
		advanceAt:     firstUsageAt,
		amount:        alpacadecimal.NewFromInt(7),
		name:          "feature-credit-correction-first-usage",
		featureKey:    featureKey,
	}).charge
	firstRealizations := s.mustFlatFeeCreditRealizations(firstCharge)
	s.Require().Len(firstRealizations, 2)
	s.AssertDecimalEqual(alpacadecimal.NewFromInt(4), firstRealizations[0].Amount, "feature-restricted realization amount")
	s.AssertDecimalEqual(alpacadecimal.NewFromInt(3), firstRealizations[1].Amount, "general realization amount")
	s.AssertDecimalEqual(alpacadecimal.NewFromInt(7), firstRealizations.Sum(), "first usage credit realizations")
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerFBOBalanceWithPriorityForFeatures(cust.GetID(), USD, costBasisFilter, restrictedPriority, featureRoute), "feature-restricted FBO after first usage")
	s.AssertDecimalEqual(alpacadecimal.NewFromInt(3), s.MustCustomerFBOBalanceWithPriorityForFeatures(cust.GetID(), USD, costBasisFilter, generalPriority, generalRoute), "general FBO after first usage")
	s.AssertDecimalEqual(alpacadecimal.NewFromInt(7), s.MustCustomerAccruedBalance(cust.GetID(), USD, costBasisFilter), "accrued after first usage")
	s.requireCustomerFBOSourceBalanceBucketsAsOf(cust.GetID(), ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasisFilter,
	}, firstUsageAt, map[string]float64{
		sourceSpendChargeBucketKey(&generalSourceChargeID, nil): 3, // 3 = 6 general-purpose grant - 3 used by first charge.
	})
	s.requireCustomerAccruedSourceSpendBalanceBucketsAsOf(cust.GetID(), ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasisFilter,
	}, firstUsageAt, map[string]float64{
		sourceSpendChargeBucketKey(&restrictedSourceChargeID, &firstCharge.ID): 4, // 4 = restricted grant is fully consumed first.
		sourceSpendChargeBucketKey(&generalSourceChargeID, &firstCharge.ID):    3, // 3 = first charge spills into general-purpose grant.
	})

	generalRealization, ok := lo.Find(firstRealizations, func(realization creditrealization.Realization) bool {
		return realization.Amount.Equal(alpacadecimal.NewFromInt(3))
	})
	s.Require().True(ok, "first usage should include a general-purpose allocation")

	// When part of the last allocation is corrected.
	clock.FreezeTime(correctionAt)
	s.correctCreditUsageAllocation(ctx, firstCharge, generalRealization, alpacadecimal.NewFromInt(2), correctionAt)

	// Then the corrected amount returns according to the reverse collection order.
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerFBOBalanceWithPriorityForFeatures(cust.GetID(), USD, costBasisFilter, restrictedPriority, featureRoute), "feature-restricted FBO after correction")
	s.AssertDecimalEqual(alpacadecimal.NewFromInt(5), s.MustCustomerFBOBalanceWithPriorityForFeatures(cust.GetID(), USD, costBasisFilter, generalPriority, generalRoute), "general FBO after correction")
	s.AssertDecimalEqual(alpacadecimal.NewFromInt(5), s.MustCustomerAccruedBalance(cust.GetID(), USD, costBasisFilter), "accrued after correction")
	s.requireCustomerFBOSourceBalanceBucketsAsOf(cust.GetID(), ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasisFilter,
	}, correctionAt, map[string]float64{
		sourceSpendChargeBucketKey(&generalSourceChargeID, nil): 5, // 5 = 3 remaining + 2 corrected from first charge.
	})
	s.requireCustomerAccruedSourceSpendBalanceBucketsAsOf(cust.GetID(), ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasisFilter,
	}, correctionAt, map[string]float64{
		sourceSpendChargeBucketKey(&restrictedSourceChargeID, &firstCharge.ID): 4, // 4 = restricted source remains tied to first charge.
		sourceSpendChargeBucketKey(&generalSourceChargeID, &firstCharge.ID):    1, // 1 = only the uncorrected general-source slice remains on first charge.
	})

	// When another charge for the same feature consumes again.
	secondCharge := s.createAndAdvanceCreditOnlyFlatFeeCharge(createCreditOnlyFlatFeeChargeInput{
		ctx:           ctx,
		namespace:     ns,
		customer:      cust.GetID(),
		servicePeriod: timeutil.ClosedPeriod{From: secondUsageAt, To: secondUsageAt.Add(time.Hour)},
		createAt:      secondUsageAt.Add(-time.Hour),
		advanceAt:     secondUsageAt,
		amount:        alpacadecimal.NewFromInt(5),
		name:          "feature-credit-correction-second-usage",
		featureKey:    featureKey,
	}).charge
	secondRealizations := s.mustFlatFeeCreditRealizations(secondCharge)
	s.Require().Len(secondRealizations, 1)
	s.AssertDecimalEqual(alpacadecimal.NewFromInt(5), secondRealizations.Sum(), "second usage credit realizations")
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerFBOBalanceWithPriorityForFeatures(cust.GetID(), USD, costBasisFilter, restrictedPriority, featureRoute), "feature-restricted FBO after second usage")
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerFBOBalanceWithPriorityForFeatures(cust.GetID(), USD, costBasisFilter, generalPriority, generalRoute), "general FBO after second usage")
	s.AssertDecimalEqual(alpacadecimal.NewFromInt(10), s.MustCustomerAccruedBalance(cust.GetID(), USD, costBasisFilter), "accrued after second usage")
	s.requireCustomerFBOSourceBalanceBucketsAsOf(cust.GetID(), ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasisFilter,
	}, secondUsageAt, map[string]float64{})
	s.requireCustomerAccruedSourceSpendBalanceBucketsAsOf(cust.GetID(), ledger.RouteFilter{
		Currency:  USD,
		CostBasis: costBasisFilter,
	}, secondUsageAt, map[string]float64{
		sourceSpendChargeBucketKey(&restrictedSourceChargeID, &firstCharge.ID): 4, // 4 = first charge keeps the restricted-source slice.
		sourceSpendChargeBucketKey(&generalSourceChargeID, &firstCharge.ID):    1, // 1 = first charge keeps its uncorrected general-source slice.
		sourceSpendChargeBucketKey(&generalSourceChargeID, &secondCharge.ID):   5, // 5 = second charge consumes the general-source credit reopened by correction.
	})
}

type expiringCreditBreakageSetup struct {
	ctx          context.Context
	namespace    string
	customer     *customer.Customer
	grantAt      time.Time
	usageAt      time.Time
	expiresAt    time.Time
	grantAmount  alpacadecimal.Decimal
	usedAmount   alpacadecimal.Decimal
	unusedAmount alpacadecimal.Decimal
	costBasis    alpacadecimal.Decimal
}

type expiringCreditBreakageSetupOption func(*expiringCreditBreakageSetup)

func withExpiringCreditBreakageAmounts(grantAmount, usedAmount alpacadecimal.Decimal) expiringCreditBreakageSetupOption {
	return func(setup *expiringCreditBreakageSetup) {
		setup.grantAmount = grantAmount
		setup.usedAmount = usedAmount
	}
}

type createCreditOnlyFlatFeeChargeInput struct {
	ctx           context.Context
	namespace     string
	customer      customer.CustomerID
	servicePeriod timeutil.ClosedPeriod
	createAt      time.Time
	advanceAt     time.Time
	amount        alpacadecimal.Decimal
	name          string
	featureKey    string
}

type createdCreditOnlyFlatFeeCharge struct {
	id     string
	charge flatfee.Charge
}

func (s *SanitySuite) mustFlatFeeCreditRealizations(charge flatfee.Charge) creditrealization.Realizations {
	s.T().Helper()

	s.Require().NotNil(charge.Realizations.CurrentRun)

	return charge.Realizations.CurrentRun.CreditRealizations
}

type breakageRowsByExpiryAssertion struct {
	expiresAt     time.Time
	planAmount    alpacadecimal.Decimal
	releaseAmount alpacadecimal.Decimal
	reopenAmount  alpacadecimal.Decimal
}

type plannedBreakageAssertionInput struct {
	ctx       context.Context
	namespace string
	customer  customer.CustomerID
	currency  currencyx.Code
	costBasis mo.Option[*alpacadecimal.Decimal]
	amount    alpacadecimal.Decimal
	createdAt time.Time
	expiresAt time.Time
}

type releasedBreakageAssertionInput struct {
	ctx             context.Context
	namespace       string
	customer        customer.CustomerID
	currency        currencyx.Code
	costBasis       mo.Option[*alpacadecimal.Decimal]
	planAmount      alpacadecimal.Decimal
	releaseAmount   alpacadecimal.Decimal
	asOf            time.Time
	expectedFBO     alpacadecimal.Decimal
	expectedAccrued alpacadecimal.Decimal
}

type reopenedBreakageAssertionInput struct {
	ctx             context.Context
	namespace       string
	customer        customer.CustomerID
	currency        currencyx.Code
	costBasis       mo.Option[*alpacadecimal.Decimal]
	planAmount      alpacadecimal.Decimal
	releaseAmount   alpacadecimal.Decimal
	reopenAmount    alpacadecimal.Decimal
	asOf            time.Time
	expectedFBO     alpacadecimal.Decimal
	expectedAccrued alpacadecimal.Decimal
}

type breakageBalanceAssertionInput struct {
	namespace        string
	customer         customer.CustomerID
	currency         currencyx.Code
	costBasis        mo.Option[*alpacadecimal.Decimal]
	asOf             time.Time
	expectedFBO      alpacadecimal.Decimal
	expectedBreakage alpacadecimal.Decimal
	label            string
}

type creditOnlyDeleteCorrectionSetup struct {
	ctx           context.Context
	namespace     string
	customer      *customer.Customer
	servicePeriod timeutil.ClosedPeriod
	createAt      time.Time
	amount        alpacadecimal.Decimal
	featureKey    string
}

func (s *SanitySuite) mustBreakageRows(ctx context.Context, namespace string, customerID customer.CustomerID) []*entdb.LedgerBreakageRecord {
	s.T().Helper()

	rows, err := s.DBClient.LedgerBreakageRecord.Query().
		Where(
			dbledgerbreakagerecord.NamespaceEQ(namespace),
			dbledgerbreakagerecord.CustomerIDEQ(customerID.ID),
		).
		Order(
			dbledgerbreakagerecord.ByCreatedAt(),
			dbledgerbreakagerecord.ByID(),
		).
		All(ctx)
	s.NoError(err)

	return rows
}

func (s *SanitySuite) assertBreakagePlan(ctx context.Context, namespace string, customerID customer.CustomerID, amount alpacadecimal.Decimal, expiresAt time.Time) string {
	s.T().Helper()

	rows := s.mustBreakageRows(ctx, namespace, customerID)
	s.Require().Len(rows, 1)

	row := rows[0]
	s.Equal(ledger.BreakageKindPlan, row.Kind)
	s.AssertDecimalEqual(amount, row.Amount, "planned breakage row amount")
	s.True(row.ExpiresAt.Equal(expiresAt), "planned breakage expires_at: %s", row.ExpiresAt)

	return row.ID
}

func (s *SanitySuite) assertBreakagePlanAndRelease(ctx context.Context, namespace string, customerID customer.CustomerID, planAmount alpacadecimal.Decimal, releaseAmount alpacadecimal.Decimal) {
	s.T().Helper()

	rows := s.mustBreakageRows(ctx, namespace, customerID)
	s.Require().Len(rows, 2)

	var planRowID string
	var releasePlanID *string
	var releaseSourceEntryID *string

	for _, row := range rows {
		switch row.Kind {
		case ledger.BreakageKindPlan:
			planRowID = row.ID
			s.AssertDecimalEqual(planAmount, row.Amount, "planned breakage row amount after usage")
		case ledger.BreakageKindRelease:
			releasePlanID = row.PlanID
			releaseSourceEntryID = row.SourceEntryID
			s.AssertDecimalEqual(releaseAmount, row.Amount, "released breakage row amount")
		default:
			s.Failf("unexpected breakage row kind", "kind=%s", row.Kind)
		}
	}

	s.Require().NotEmpty(planRowID)
	s.Require().NotNil(releasePlanID)
	s.Require().NotNil(releaseSourceEntryID)
	s.NotEmpty(*releaseSourceEntryID)
	s.Equal(planRowID, *releasePlanID)
}

func (s *SanitySuite) assertAdvanceBackfillBreakageRows(ctx context.Context, namespace string, customerID customer.CustomerID, planAmount alpacadecimal.Decimal, releaseAmount alpacadecimal.Decimal, expiresAt time.Time) {
	s.T().Helper()

	rows := s.mustBreakageRows(ctx, namespace, customerID)
	s.Require().Len(rows, 2)

	var planRowID string
	var releasePlanID *string

	for _, row := range rows {
		s.True(row.ExpiresAt.Equal(expiresAt), "breakage row expires_at: %s", row.ExpiresAt)

		switch row.Kind {
		case ledger.BreakageKindPlan:
			planRowID = row.ID
			s.Equal(ledger.BreakageSourceKindCreditPurchase, row.SourceKind)
			s.AssertDecimalEqual(planAmount, row.Amount, "planned breakage row amount after advance backfill")
		case ledger.BreakageKindRelease:
			releasePlanID = row.PlanID
			s.Equal(ledger.BreakageSourceKindAdvanceBackfill, row.SourceKind)
			s.Nil(row.SourceEntryID)
			s.AssertDecimalEqual(releaseAmount, row.Amount, "released breakage row amount after advance backfill")
		default:
			s.Failf("unexpected breakage row kind", "kind=%s", row.Kind)
		}
	}

	s.Require().NotEmpty(planRowID)
	s.Require().NotNil(releasePlanID)
	s.Equal(planRowID, *releasePlanID)
}

func (s *SanitySuite) assertAdvanceBackfillBreakageReopenedRows(ctx context.Context, namespace string, customerID customer.CustomerID, planAmount alpacadecimal.Decimal, releaseAmount alpacadecimal.Decimal, expiresAt time.Time) {
	s.T().Helper()

	rows := s.mustBreakageRows(ctx, namespace, customerID)
	s.Require().Len(rows, 3)

	var planRowID string
	var releaseRowID string
	var releasePlanID *string
	var reopenPlanID *string
	var reopenReleaseID *string

	for _, row := range rows {
		s.True(row.ExpiresAt.Equal(expiresAt), "breakage row expires_at: %s", row.ExpiresAt)

		switch row.Kind {
		case ledger.BreakageKindPlan:
			planRowID = row.ID
			s.Equal(ledger.BreakageSourceKindCreditPurchase, row.SourceKind)
			s.AssertDecimalEqual(planAmount, row.Amount, "planned breakage row amount after advance correction")
		case ledger.BreakageKindRelease:
			releaseRowID = row.ID
			releasePlanID = row.PlanID
			s.Equal(ledger.BreakageSourceKindAdvanceBackfill, row.SourceKind)
			s.Nil(row.SourceEntryID)
			s.AssertDecimalEqual(releaseAmount, row.Amount, "advance-backfill release amount after correction")
		case ledger.BreakageKindReopen:
			reopenPlanID = row.PlanID
			reopenReleaseID = row.ReleaseID
			s.Equal(ledger.BreakageSourceKindUsageCorrection, row.SourceKind)
			s.AssertDecimalEqual(releaseAmount, row.Amount, "advance-backfill reopen amount")
		default:
			s.Failf("unexpected breakage row kind", "kind=%s", row.Kind)
		}
	}

	s.Require().NotEmpty(planRowID)
	s.Require().NotEmpty(releaseRowID)
	s.Require().NotNil(releasePlanID)
	s.Require().NotNil(reopenPlanID)
	s.Require().NotNil(reopenReleaseID)
	s.Equal(planRowID, *releasePlanID)
	s.Equal(planRowID, *reopenPlanID)
	s.Equal(releaseRowID, *reopenReleaseID)
}

func (s *SanitySuite) assertBreakagePlanReleaseAndReopen(ctx context.Context, namespace string, customerID customer.CustomerID, planAmount alpacadecimal.Decimal, releaseAmount alpacadecimal.Decimal, reopenAmount alpacadecimal.Decimal) {
	s.T().Helper()

	rows := s.mustBreakageRows(ctx, namespace, customerID)
	s.Require().Len(rows, 3)

	var planRowID string
	var releaseRowID string
	var releasePlanID *string
	var reopenPlanID *string
	var reopenReleaseID *string

	for _, row := range rows {
		switch row.Kind {
		case ledger.BreakageKindPlan:
			planRowID = row.ID
			s.AssertDecimalEqual(planAmount, row.Amount, "planned breakage row amount after correction")
		case ledger.BreakageKindRelease:
			releaseRowID = row.ID
			releasePlanID = row.PlanID
			s.AssertDecimalEqual(releaseAmount, row.Amount, "released breakage row amount after correction")
			s.Require().NotNil(row.SourceEntryID)
			s.NotEmpty(*row.SourceEntryID)
		case ledger.BreakageKindReopen:
			reopenPlanID = row.PlanID
			reopenReleaseID = row.ReleaseID
			s.AssertDecimalEqual(reopenAmount, row.Amount, "reopened breakage row amount")
		default:
			s.Failf("unexpected breakage row kind", "kind=%s", row.Kind)
		}
	}

	s.Require().NotEmpty(planRowID)
	s.Require().NotEmpty(releaseRowID)
	s.Require().NotNil(releasePlanID)
	s.Require().NotNil(reopenPlanID)
	s.Require().NotNil(reopenReleaseID)
	s.Equal(planRowID, *releasePlanID)
	s.Equal(planRowID, *reopenPlanID)
	s.Equal(releaseRowID, *reopenReleaseID)
}

func (s *SanitySuite) assertBreakageRowsByExpiry(ctx context.Context, namespace string, customerID customer.CustomerID, expected []breakageRowsByExpiryAssertion) {
	s.T().Helper()

	rows := s.mustBreakageRows(ctx, namespace, customerID)

	expectedByExpiry := make(map[string]breakageRowsByExpiryAssertion, len(expected))
	for _, item := range expected {
		expectedByExpiry[item.expiresAt.UTC().Format(time.RFC3339Nano)] = item
	}

	planIDByExpiry := make(map[string]string, len(expected))
	releaseIDByExpiry := make(map[string]string, len(expected))
	releasePlanIDByExpiry := make(map[string]*string, len(expected))
	reopenPlanIDByExpiry := make(map[string]*string, len(expected))
	reopenReleaseIDByExpiry := make(map[string]*string, len(expected))
	actualPlanAmountByExpiry := make(map[string]alpacadecimal.Decimal, len(expected))
	actualReleaseAmountByExpiry := make(map[string]alpacadecimal.Decimal, len(expected))
	actualReopenAmountByExpiry := make(map[string]alpacadecimal.Decimal, len(expected))

	for _, row := range rows {
		key := row.ExpiresAt.UTC().Format(time.RFC3339Nano)
		if _, ok := expectedByExpiry[key]; !ok {
			s.Failf("unexpected breakage expiry", "expires_at=%s kind=%s amount=%s", row.ExpiresAt, row.Kind, row.Amount)
			continue
		}

		switch row.Kind {
		case ledger.BreakageKindPlan:
			planIDByExpiry[key] = row.ID
			actualPlanAmountByExpiry[key] = actualPlanAmountByExpiry[key].Add(row.Amount)
		case ledger.BreakageKindRelease:
			releaseIDByExpiry[key] = row.ID
			releasePlanIDByExpiry[key] = row.PlanID
			actualReleaseAmountByExpiry[key] = actualReleaseAmountByExpiry[key].Add(row.Amount)
			s.Require().NotNil(row.SourceEntryID)
			s.NotEmpty(*row.SourceEntryID)
		case ledger.BreakageKindReopen:
			reopenPlanIDByExpiry[key] = row.PlanID
			reopenReleaseIDByExpiry[key] = row.ReleaseID
			actualReopenAmountByExpiry[key] = actualReopenAmountByExpiry[key].Add(row.Amount)
		default:
			s.Failf("unexpected breakage row kind", "kind=%s", row.Kind)
		}
	}

	for key, expectedItem := range expectedByExpiry {
		s.AssertDecimalEqual(expectedItem.planAmount, actualPlanAmountByExpiry[key], "planned breakage amount at "+key)
		s.AssertDecimalEqual(expectedItem.releaseAmount, actualReleaseAmountByExpiry[key], "released breakage amount at "+key)
		s.AssertDecimalEqual(expectedItem.reopenAmount, actualReopenAmountByExpiry[key], "reopened breakage amount at "+key)

		s.Require().NotEmpty(planIDByExpiry[key])
		if expectedItem.releaseAmount.IsPositive() {
			s.Require().NotEmpty(releaseIDByExpiry[key])
			s.Require().NotNil(releasePlanIDByExpiry[key])
			s.Equal(planIDByExpiry[key], *releasePlanIDByExpiry[key])
		}
		if expectedItem.reopenAmount.IsPositive() {
			s.Require().NotNil(reopenPlanIDByExpiry[key])
			s.Require().NotNil(reopenReleaseIDByExpiry[key])
			s.Equal(planIDByExpiry[key], *reopenPlanIDByExpiry[key])
			s.Equal(releaseIDByExpiry[key], *reopenReleaseIDByExpiry[key])
		}
	}
}

func (s *SanitySuite) setupExpiringCreditBreakage(namespaceSuffix string, opts ...expiringCreditBreakageSetupOption) expiringCreditBreakageSetup {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace(namespaceSuffix)
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	grantAmount := alpacadecimal.NewFromInt(10)
	usedAmount := alpacadecimal.NewFromInt(6)

	setup := expiringCreditBreakageSetup{
		ctx:          ctx,
		namespace:    ns,
		customer:     cust,
		grantAt:      datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		usageAt:      datetime.MustParseTimeInLocation(s.T(), "2026-01-02T00:00:00Z", time.UTC).AsTime(),
		expiresAt:    datetime.MustParseTimeInLocation(s.T(), "2026-01-10T00:00:00Z", time.UTC).AsTime(),
		grantAmount:  grantAmount,
		usedAmount:   usedAmount,
		unusedAmount: grantAmount.Sub(usedAmount),
		costBasis:    alpacadecimal.Zero,
	}

	for _, opt := range opts {
		opt(&setup)
	}
	setup.unusedAmount = setup.grantAmount.Sub(setup.usedAmount)

	return setup
}

func (s *SanitySuite) createAndAdvanceCreditOnlyFlatFeeCharge(input createCreditOnlyFlatFeeChargeInput) createdCreditOnlyFlatFeeCharge {
	s.T().Helper()

	clock.FreezeTime(input.createAt)

	res, err := s.Charges.Create(input.ctx, charges.CreateInput{
		Namespace: input.namespace,
		Intents: charges.ChargeIntents{
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:       input.customer,
				Currency:       USD,
				ServicePeriod:  input.servicePeriod,
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      input.amount,
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Name:              input.name,
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: input.name,
				FeatureKey:        input.featureKey,
			}),
		},
	})
	s.Require().NoError(err)
	s.Require().Len(res, 1)

	chargeID, err := res[0].GetChargeID()
	s.NoError(err)

	clock.FreezeTime(input.advanceAt)

	advancedCharges, err := s.Charges.AdvanceCharges(input.ctx, charges.AdvanceChargesInput{
		Customer: input.customer,
	})
	s.Require().NoError(err)
	s.Require().Len(advancedCharges, 1)

	charge, err := advancedCharges[0].AsFlatFeeCharge()
	s.NoError(err)
	s.Equal(flatfee.StatusFinal, charge.Status)

	return createdCreditOnlyFlatFeeCharge{
		id:     chargeID.ID,
		charge: charge,
	}
}

func (s *SanitySuite) createPromotionalCreditGrant(ctx context.Context, input CreatePromotionalCreditFundingInput) creditpurchase.Charge {
	s.T().Helper()

	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: input.Namespace,
		Intents: charges.ChargeIntents{
			s.CreateCreditPurchaseIntent(CreateCreditPurchaseIntentInput{
				Customer:       input.Customer,
				Currency:       USD,
				Amount:         input.Amount,
				ExpiresAt:      input.ExpiresAt,
				Priority:       input.Priority,
				ServicePeriod:  timeutil.ClosedPeriod{From: input.At, To: input.At},
				Settlement:     creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
				FeatureFilters: input.FeatureFilters,
			}),
		},
	})
	s.Require().NoError(err)
	s.Require().Len(res, 1)
	s.Equal(meta.ChargeTypeCreditPurchase, res[0].Type())

	charge, err := res[0].AsCreditPurchaseCharge()
	s.Require().NoError(err)

	return charge
}

func (s *SanitySuite) correctCreditUsageAllocation(ctx context.Context, charge flatfee.Charge, allocation creditrealization.Realization, amount alpacadecimal.Decimal, bookedAt time.Time) {
	s.T().Helper()

	lineageSegmentsByRealization, err := s.LineageService.LoadActiveSegmentsByRealizationID(ctx, charge.Namespace, []string{allocation.ID})
	s.Require().NoError(err)

	corrections, err := s.FlatFeeHandler.OnCorrectCreditAllocations(ctx, flatfee.CorrectCreditAllocationsInput{
		Charge:                       charge,
		BookedAt:                     bookedAt,
		LineageSegmentsByRealization: lineageSegmentsByRealization,
		Corrections: creditrealization.CorrectionRequest{
			{
				Allocation: allocation,
				Amount:     amount.Neg(),
			},
		},
	})
	s.Require().NoError(err)
	s.Require().Len(corrections, 1)
	s.AssertDecimalEqual(amount.Neg(), corrections[0].Amount, "credit usage correction amount")
	s.Equal(allocation.ID, corrections[0].CorrectsRealizationID)
}

func (s *SanitySuite) assertPlannedBreakage(input plannedBreakageAssertionInput) {
	s.T().Helper()

	s.assertBreakagePlan(input.ctx, input.namespace, input.customer, input.amount, input.expiresAt)
	s.assertBreakageBalancesAt(breakageBalanceAssertionInput{
		namespace:        input.namespace,
		customer:         input.customer,
		currency:         input.currency,
		costBasis:        input.costBasis,
		asOf:             input.createdAt,
		expectedFBO:      input.amount,
		expectedBreakage: alpacadecimal.Zero,
		label:            "at creation",
	})
	s.assertBreakageBalancesAt(breakageBalanceAssertionInput{
		namespace:        input.namespace,
		customer:         input.customer,
		currency:         input.currency,
		costBasis:        input.costBasis,
		asOf:             input.expiresAt,
		expectedFBO:      alpacadecimal.Zero,
		expectedBreakage: input.amount,
		label:            "at expiry before release",
	})
}

func (s *SanitySuite) assertReleasedBreakage(input releasedBreakageAssertionInput) {
	s.T().Helper()

	s.assertBreakagePlanAndRelease(input.ctx, input.namespace, input.customer, input.planAmount, input.releaseAmount)
	s.assertBreakageBalancesAt(breakageBalanceAssertionInput{
		namespace:        input.namespace,
		customer:         input.customer,
		currency:         input.currency,
		costBasis:        input.costBasis,
		asOf:             input.asOf,
		expectedFBO:      input.expectedFBO,
		expectedBreakage: alpacadecimal.Zero,
		label:            "after release before expiry",
	})
	s.AssertDecimalEqual(input.expectedAccrued, s.MustCustomerAccruedBalance(input.customer, input.currency, input.costBasis), "accrued after release")
}

func (s *SanitySuite) assertReopenedBreakage(input reopenedBreakageAssertionInput) {
	s.T().Helper()

	s.assertBreakagePlanReleaseAndReopen(input.ctx, input.namespace, input.customer, input.planAmount, input.releaseAmount, input.reopenAmount)
	s.assertBreakageBalancesAt(breakageBalanceAssertionInput{
		namespace:        input.namespace,
		customer:         input.customer,
		currency:         input.currency,
		costBasis:        input.costBasis,
		asOf:             input.asOf,
		expectedFBO:      input.expectedFBO,
		expectedBreakage: alpacadecimal.Zero,
		label:            "after reopen before expiry",
	})
	s.AssertDecimalEqual(input.expectedAccrued, s.MustCustomerAccruedBalance(input.customer, input.currency, input.costBasis), "accrued after reopen")
}

func (s *SanitySuite) assertBreakageBalancesAt(input breakageBalanceAssertionInput) {
	s.T().Helper()

	s.AssertDecimalEqual(input.expectedFBO, s.MustCustomerFBOBalanceAsOf(input.customer, input.currency, input.costBasis, input.asOf), "FBO "+input.label)
	s.AssertDecimalEqual(input.expectedBreakage, s.MustBreakageBalanceAsOf(input.namespace, input.currency, input.costBasis, input.asOf), "breakage "+input.label)
}

func (s *SanitySuite) setupFlatFeeCreditOnlyDeleteCorrection(namespaceSuffix string) creditOnlyDeleteCorrectionSetup {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace(namespaceSuffix)
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	return creditOnlyDeleteCorrectionSetup{
		ctx:       ctx,
		namespace: ns,
		customer:  cust,
		servicePeriod: timeutil.ClosedPeriod{
			From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
			To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
		},
		createAt: datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime(),
		amount:   alpacadecimal.NewFromInt(30),
	}
}

func (s *SanitySuite) setupClosedPeriodUsageBasedCreditOnlyCollection(namespaceSuffix string) creditOnlyDeleteCorrectionSetup {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace(namespaceSuffix)
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	return creditOnlyDeleteCorrectionSetup{
		ctx:       ctx,
		namespace: ns,
		customer:  cust,
		servicePeriod: timeutil.ClosedPeriod{
			From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
			To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
		},
		createAt:   datetime.MustParseTimeInLocation(s.T(), "2026-02-03T00:00:00Z", time.UTC).AsTime(),
		amount:     alpacadecimal.NewFromInt(8),
		featureKey: apiRequestsTotal.Feature.Key,
	}
}

func (s *SanitySuite) createPromotionalCreditFunding(setup creditOnlyDeleteCorrectionSetup, costBasis alpacadecimal.Decimal) CreatePromotionalCreditFundingResult {
	s.T().Helper()

	return s.createPromotionalCreditFundingAt(setup, costBasis, setup.createAt)
}

func (s *SanitySuite) createPromotionalCreditFundingAt(setup creditOnlyDeleteCorrectionSetup, costBasis alpacadecimal.Decimal, at time.Time) CreatePromotionalCreditFundingResult {
	s.T().Helper()

	result := s.CreatePromotionalCreditFunding(setup.ctx, CreatePromotionalCreditFundingInput{
		Namespace: setup.namespace,
		Customer:  setup.customer.GetID(),
		Amount:    setup.amount,
		At:        at,
		CostBasis: costBasis,
	})

	return result
}

func (s *SanitySuite) createAndAdvanceFlatFeeCreditOnlyCharge(setup creditOnlyDeleteCorrectionSetup) string {
	s.T().Helper()

	created := s.createAndAdvanceCreditOnlyFlatFeeCharge(createCreditOnlyFlatFeeChargeInput{
		ctx:           setup.ctx,
		namespace:     setup.namespace,
		customer:      setup.customer.GetID(),
		servicePeriod: setup.servicePeriod,
		createAt:      setup.createAt,
		advanceAt:     setup.servicePeriod.From,
		amount:        setup.amount,
		name:          setup.namespace,
	})
	s.Len(s.mustFlatFeeCreditRealizations(created.charge), 1)

	return created.id
}

func (s *SanitySuite) recordUsageInClosedServicePeriod(setup creditOnlyDeleteCorrectionSetup) time.Time {
	s.T().Helper()

	usageAt := datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime()
	s.MockStreamingConnector.AddSimpleEvent(
		setup.featureKey,
		setup.amount.InexactFloat64(),
		usageAt,
	)

	return usageAt
}

func (s *SanitySuite) createFinalizedUsageBasedCreditOnlyCharge(setup creditOnlyDeleteCorrectionSetup) string {
	s.T().Helper()

	res, err := s.Charges.Create(setup.ctx, charges.CreateInput{
		Namespace: setup.namespace,
		Intents: charges.ChargeIntents{
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:       setup.customer.GetID(),
				Currency:       USD,
				ServicePeriod:  setup.servicePeriod,
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(1),
				}),
				Name:              setup.namespace,
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: setup.namespace,
				FeatureKey:        setup.featureKey,
			}),
		},
	})
	s.NoError(err)
	s.Len(res, 1)

	usageBasedCharge, err := res[0].AsUsageBasedCharge()
	s.NoError(err)
	s.Equal(meta.ChargeStatusFinal, meta.ChargeStatus(usageBasedCharge.Status))
	s.Len(usageBasedCharge.Realizations, 1)
	s.True(usageBasedCharge.Realizations[0].NoFiatTransactionRequired)
	s.Len(usageBasedCharge.Realizations[0].CreditsAllocated, 1)
	s.True(usageBasedCharge.Realizations[0].CreditsAllocated[0].Amount.Equal(setup.amount))

	return usageBasedCharge.ID
}

func (s *SanitySuite) deleteChargeWithRefundAsCredits(ctx context.Context, customerID customer.CustomerID, chargeID string) {
	s.T().Helper()

	err := s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
		CustomerID: customerID,
		PatchesByChargeID: map[string]charges.Patch{
			chargeID: lo.Must(meta.NewPatchDelete(meta.NewPatchDeleteInput{
				ChangeSource: billing.ChangeSourceSystem,
				Policy:       meta.RefundAsCreditsDeletePolicy,
			})),
		},
	})
	s.NoError(err)
}

func (s *SanitySuite) assertUnfundedCreditOnlyRealization(customerID customer.CustomerID, amount alpacadecimal.Decimal, spendChargeID string) {
	s.T().Helper()

	s.True(s.MustCustomerReceivableBalance(customerID, USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).Equal(amount.Neg()))
	s.True(s.MustCustomerAccruedBalance(customerID, USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(amount))

	openStatus := ledger.TransactionAuthorizationStatusOpen
	nilCostBasis := mo.Some[*alpacadecimal.Decimal](nil)
	s.requireCustomerReceivableSourceSpendBalanceBuckets(customerID, ledger.RouteFilter{
		Currency:                       USD,
		CostBasis:                      nilCostBasis,
		TransactionAuthorizationStatus: &openStatus,
	}, map[string]float64{
		sourceSpendChargeBucketKey(nil, &spendChargeID): amount.Neg().InexactFloat64(),
	})
	s.requireCustomerAccruedSourceSpendBalanceBuckets(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: nilCostBasis,
	}, map[string]float64{
		sourceSpendChargeBucketKey(nil, &spendChargeID): amount.InexactFloat64(),
	})
}

func (s *SanitySuite) assertUnfundedCreditOnlyDeleted(customerID customer.CustomerID) {
	s.T().Helper()

	s.True(s.MustCustomerReceivableBalance(customerID, USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.Zero))
	s.True(s.MustCustomerAccruedBalance(customerID, USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(alpacadecimal.Zero))
	s.True(s.MustCustomerFBOBalance(customerID, USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(alpacadecimal.Zero))
}

func (s *SanitySuite) assertFundedCreditOnlyAccrued(customerID customer.CustomerID, amount alpacadecimal.Decimal, costBasis alpacadecimal.Decimal, startOpenReceivable alpacadecimal.Decimal, sourceChargeID string, spendChargeID string) {
	s.T().Helper()

	s.True(s.MustCustomerFBOBalance(customerID, USD, mo.Some(&costBasis)).Equal(alpacadecimal.Zero))
	s.True(s.MustCustomerReceivableBalance(customerID, USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).Equal(startOpenReceivable))
	s.True(s.MustCustomerAccruedBalance(customerID, USD, mo.Some(&costBasis)).Equal(amount))
	s.requireCustomerAccruedSourceSpendBalanceBuckets(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: mo.Some(&costBasis),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, &spendChargeID): amount.InexactFloat64(),
	})
}

func (s *SanitySuite) recognizeFundedCreditOnlyRevenue(namespace string, customerID customer.CustomerID, amount alpacadecimal.Decimal, costBasis alpacadecimal.Decimal, sourceChargeID string, spendChargeID string) {
	s.T().Helper()

	expectedAccruedAfterRecognition := alpacadecimal.Zero        // 0 = all funded accrued value is moved into earnings.
	expectedUnknownCostBasisAccrued := alpacadecimal.Zero        // 0 = this funded flow should not leave unattributed accrued value.
	expectedFBOAfterRecognition := alpacadecimal.Zero            // 0 = the credit was already consumed during accrual.
	expectedEarningsAmount := amount                             // full amount = all funded accrued value is recognized.
	expectedUnknownCostBasisEarnings := alpacadecimal.Zero       // 0 = recognized earnings should keep the known credit cost basis.
	expectedEarningsSourceSpendAmount := amount.InexactFloat64() // full amount = earnings preserve the funding and spend charge provenance.

	s.MustRecognizeRevenue(customerID, USD, amount)
	s.True(s.MustCustomerAccruedBalance(customerID, USD, mo.Some(&costBasis)).Equal(expectedAccruedAfterRecognition))
	s.True(s.MustCustomerAccruedBalance(customerID, USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(expectedUnknownCostBasisAccrued))
	s.True(s.MustCustomerFBOBalance(customerID, USD, mo.Some(&costBasis)).Equal(expectedFBOAfterRecognition))
	s.True(s.MustEarningsBalanceForCostBasis(namespace, USD, mo.Some(&costBasis)).Equal(expectedEarningsAmount))
	s.True(s.MustEarningsBalanceForCostBasis(namespace, USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(expectedUnknownCostBasisEarnings))
	s.True(s.MustEarningsBalance(namespace, USD).Equal(expectedEarningsAmount))
	s.requireEarningsSourceSpendBalanceBuckets(namespace, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: mo.Some(&costBasis),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, &spendChargeID): expectedEarningsSourceSpendAmount,
	})
}

func (s *SanitySuite) assertFundedRecognizedCreditOnlyDeleted(namespace string, customerID customer.CustomerID, amount alpacadecimal.Decimal, costBasis alpacadecimal.Decimal, startOpenReceivable alpacadecimal.Decimal, sourceChargeID string) {
	s.T().Helper()

	expectedAccruedAfterDelete := alpacadecimal.Zero               // 0 = recognized spend is fully corrected out of accrued.
	expectedUnknownCostBasisAccrued := alpacadecimal.Zero          // 0 = no unattributed accrued value is created by the correction.
	expectedReissuedFBOAmount := amount                            // full amount = refund-as-credits reissues the consumed credit.
	expectedUnknownCostBasisFBO := alpacadecimal.Zero              // 0 = reissued credit keeps the original cost basis.
	expectedEarningsAfterDelete := alpacadecimal.Zero              // 0 = recognition is fully reversed.
	expectedUnknownCostBasisEarnings := alpacadecimal.Zero         // 0 = no unattributed earnings remain.
	expectedEarningsSourceSpendBalances := map[string]float64{}    // empty = corrected recognition leaves no source/spend earnings bucket.
	expectedReissuedSourceBalanceAmount := amount.InexactFloat64() // full amount = FBO keeps the original source charge after reissue.

	s.True(s.MustCustomerReceivableBalance(customerID, USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).Equal(startOpenReceivable))
	s.True(s.MustCustomerAccruedBalance(customerID, USD, mo.Some(&costBasis)).Equal(expectedAccruedAfterDelete))
	s.True(s.MustCustomerAccruedBalance(customerID, USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(expectedUnknownCostBasisAccrued))
	s.True(s.MustCustomerFBOBalance(customerID, USD, mo.Some(&costBasis)).Equal(expectedReissuedFBOAmount))
	s.True(s.MustCustomerFBOBalance(customerID, USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(expectedUnknownCostBasisFBO))
	s.True(s.MustEarningsBalanceForCostBasis(namespace, USD, mo.Some(&costBasis)).Equal(expectedEarningsAfterDelete))
	s.True(s.MustEarningsBalanceForCostBasis(namespace, USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(expectedUnknownCostBasisEarnings))
	s.True(s.MustEarningsBalance(namespace, USD).Equal(expectedEarningsAfterDelete))
	s.requireEarningsSourceSpendBalanceBuckets(namespace, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: mo.Some(&costBasis),
	}, expectedEarningsSourceSpendBalances)
	s.requireCustomerFBOSourceBalanceBuckets(customerID, ledger.RouteFilter{
		Currency:  USD,
		CostBasis: mo.Some(&costBasis),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, nil): expectedReissuedSourceBalanceAmount,
	})
}

func (s *SanitySuite) TestUsageBasedCreditOnlyDeleteCorrectionWithPartialBackfillSanity() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-sanity-usagebased-credit-only-delete-partial-backfill")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	featureRoute := mo.Some([]string{apiRequestsTotal.Feature.Key})

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	createAt := datetime.MustParseTimeInLocation(s.T(), "2026-02-03T00:00:00Z", time.UTC).AsTime()

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	// Given a usage-based credit-only charge that is created after the service period, so it
	// finalizes immediately with 50 units of unattributed advance-backed usage.
	s.MockStreamingConnector.AddSimpleEvent(
		apiRequestsTotal.Feature.Key,
		50,
		datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
	)

	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:       cust.GetID(),
				Currency:       USD,
				ServicePeriod:  servicePeriod,
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(1),
				}),
				Name:              "usage-based-credit-only-delete-partial-backfill",
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: "usage-based-credit-only-delete-partial-backfill",
				FeatureKey:        apiRequestsTotal.Feature.Key,
			}),
		},
	})
	s.NoError(err)
	s.Len(res, 1)

	usageBasedCharge, err := res[0].AsUsageBasedCharge()
	s.NoError(err)
	s.Equal(meta.ChargeStatusFinal, meta.ChargeStatus(usageBasedCharge.Status))
	s.Len(usageBasedCharge.Realizations, 1)
	s.Len(usageBasedCharge.Realizations[0].CreditsAllocated, 1)
	allocatedAmount := usageBasedCharge.Realizations[0].CreditsAllocated[0].Amount
	purchaseAmount := alpacadecimal.NewFromInt(20)
	remainingUncovered := allocatedAmount.Sub(purchaseAmount)

	// Then the full amount sits on the nil-cost-basis receivable/accrued path.
	s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).Equal(allocatedAmount.Neg()))
	s.True(s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(allocatedAmount))
	openStatus := ledger.TransactionAuthorizationStatusOpen
	s.requireCustomerReceivableSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
		Currency:                       USD,
		CostBasis:                      mo.Some[*alpacadecimal.Decimal](nil),
		TransactionAuthorizationStatus: &openStatus,
	}, map[string]float64{
		sourceSpendChargeBucketKey(nil, &usageBasedCharge.ID): allocatedAmount.Neg().InexactFloat64(), // -50 = advance receivable created by the usage charge.
	})
	s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
		Currency:  USD,
		CostBasis: mo.Some[*alpacadecimal.Decimal](nil),
	}, map[string]float64{
		sourceSpendChargeBucketKey(nil, &usageBasedCharge.ID): allocatedAmount.InexactFloat64(), // 50 = source-less accrued usage before purchase backfill.
	})

	creditPurchaseIntent := s.CreateCreditPurchaseIntent(CreateCreditPurchaseIntentInput{
		Customer: cust.GetID(),
		Currency: USD,
		Amount:   purchaseAmount,
		ServicePeriod: timeutil.ClosedPeriod{
			From: createAt,
			To:   createAt,
		},
		Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
			GenericSettlement: creditpurchase.GenericSettlement{
				Currency:  currencyx.FiatCode(USD),
				CostBasis: alpacadecimal.NewFromFloat(0.5),
			},
			InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
		}),
		FeatureFilters: creditpurchase.FeatureFilters{apiRequestsTotal.Feature.Key},
	})

	// When a later external credit purchase backfills part of that earlier advance-backed usage.
	creditPurchaseRes, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			creditPurchaseIntent,
		},
	})
	s.NoError(err)
	s.Len(creditPurchaseRes, 1)
	creditPurchaseCharge, err := creditPurchaseRes[0].AsCreditPurchaseCharge()
	s.NoError(err)

	costBasis := alpacadecimal.NewFromFloat(0.5)
	backingGroup, err := s.Ledger.GetTransactionGroup(ctx, models.NamespacedID{
		Namespace: ns,
		ID:        creditPurchaseCharge.Realizations.CreditGrantRealization.TransactionGroupID,
	})
	s.NoError(err)
	s.Len(backingGroup.Transactions(), 2)

	// Then only the purchased portion moves onto the purchased-credit cost-basis route.
	s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).Equal(allocatedAmount.Neg()))
	s.True(s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(remainingUncovered))
	s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusOpen).Equal(purchaseAmount.Neg()))
	s.True(s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&costBasis)).Equal(purchaseAmount))
	s.True(s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis)).Equal(alpacadecimal.Zero))
	sourceChargeID := creditPurchaseCharge.ID
	s.requireCustomerReceivableSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
		Currency:                       USD,
		CostBasis:                      mo.Some[*alpacadecimal.Decimal](nil),
		TransactionAuthorizationStatus: &openStatus,
	}, map[string]float64{
		sourceSpendChargeBucketKey(nil, &usageBasedCharge.ID): remainingUncovered.Neg().InexactFloat64(), // -30 = 50 advance - 20 backfilled by purchase.
	})
	s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
		Currency:  USD,
		CostBasis: mo.Some[*alpacadecimal.Decimal](nil),
	}, map[string]float64{
		sourceSpendChargeBucketKey(nil, &usageBasedCharge.ID): remainingUncovered.InexactFloat64(), // 30 = source-less usage still uncovered.
	})
	s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
		Currency:  USD,
		CostBasis: mo.Some(&costBasis),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, &usageBasedCharge.ID): purchaseAmount.InexactFloat64(), // 20 = purchased source attributed to the usage charge.
	})

	// When the original charge is deleted with refund-as-credits.
	err = s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
		CustomerID: cust.GetID(),
		PatchesByChargeID: map[string]charges.Patch{
			usageBasedCharge.ID: lo.Must(meta.NewPatchDelete(meta.NewPatchDeleteInput{
				ChangeSource: billing.ChangeSourceSystem,
				Policy:       meta.RefundAsCreditsDeletePolicy,
			})),
		},
	})
	s.NoError(err)

	// Then the purchased part is returned as available credit and the original accrued usage is cleared.
	s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).Equal(purchaseAmount.Neg()))
	s.True(s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(alpacadecimal.Zero))
	s.True(s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&costBasis)).Equal(alpacadecimal.Zero))
	s.True(s.MustCustomerFBOBalanceForFeatures(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), featureRoute).Equal(alpacadecimal.Zero))
	s.True(s.MustCustomerFBOBalanceForFeatures(cust.GetID(), USD, mo.Some(&costBasis), featureRoute).Equal(purchaseAmount))
	s.True(s.MustCustomerFBOBalanceForFeatures(cust.GetID(), USD, mo.Some(&costBasis), mo.Some[[]string](nil)).Equal(alpacadecimal.Zero))
	s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusOpen).Equal(purchaseAmount.Neg()))
	s.requireCustomerFBOSourceBalanceBuckets(cust.GetID(), ledger.RouteFilter{
		Currency:  USD,
		CostBasis: mo.Some(&costBasis),
		Features:  featureRoute,
	}, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, nil): purchaseAmount.InexactFloat64(), // 20 = corrected purchased backing is spend-free available credit again.
	})
	s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
		Currency: USD,
	}, map[string]float64{})
}

func (s *SanitySuite) TestUsageBasedCreditOnlyDeleteCorrectionWithMixedFeatureAdvanceBackfillSanity() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-sanity-usagebased-credit-only-delete-mixed-feature-backfill")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	meteredFeatures := s.setupMeteredFeatures(ctx, ns,
		meteredFeatureSetup{key: "api-requests-total", name: "API Requests Total"},
		meteredFeatureSetup{key: "storage-gb-total", name: "Storage GB Total"},
	)
	apiRequestsFeature := meteredFeatures["api-requests-total"]
	storageFeature := meteredFeatures["storage-gb-total"]

	apiRequestsRoute := mo.Some([]string{apiRequestsFeature.Key})
	storageRoute := mo.Some([]string{storageFeature.Key})

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	createAt := datetime.MustParseTimeInLocation(s.T(), "2026-02-03T00:00:00Z", time.UTC).AsTime()

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	apiRequestsAmount := alpacadecimal.NewFromInt(30)
	storageAmount := alpacadecimal.NewFromInt(40)
	purchaseAmount := alpacadecimal.NewFromInt(20)
	costBasis := alpacadecimal.NewFromFloat(0.5)

	// Given two feature-specific credit-only charges that finalized as advance-backed usage.
	s.MockStreamingConnector.AddSimpleEvent(
		apiRequestsFeature.Key,
		apiRequestsAmount.InexactFloat64(),
		datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
	)
	s.MockStreamingConnector.AddSimpleEvent(
		storageFeature.Key,
		storageAmount.InexactFloat64(),
		datetime.MustParseTimeInLocation(s.T(), "2026-01-16T00:00:00Z", time.UTC).AsTime(),
	)

	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:       cust.GetID(),
				Currency:       USD,
				ServicePeriod:  servicePeriod,
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(1),
				}),
				Name:              "usage-based-credit-only-delete-mixed-feature-api",
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: "usage-based-credit-only-delete-mixed-feature-api",
				FeatureKey:        apiRequestsFeature.Key,
			}),
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:       cust.GetID(),
				Currency:       USD,
				ServicePeriod:  servicePeriod,
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(1),
				}),
				Name:              "usage-based-credit-only-delete-mixed-feature-storage",
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: "usage-based-credit-only-delete-mixed-feature-storage",
				FeatureKey:        storageFeature.Key,
			}),
		},
	})
	s.NoError(err)
	s.Len(res, 2)

	apiRequestsCharge, err := res[0].AsUsageBasedCharge()
	s.NoError(err)
	storageCharge, err := res[1].AsUsageBasedCharge()
	s.NoError(err)

	s.Equal(meta.ChargeStatusFinal, meta.ChargeStatus(apiRequestsCharge.Status))
	s.Equal(meta.ChargeStatusFinal, meta.ChargeStatus(storageCharge.Status))
	s.Len(apiRequestsCharge.Realizations, 1)
	s.Len(storageCharge.Realizations, 1)
	s.Len(apiRequestsCharge.Realizations[0].CreditsAllocated, 1)
	s.Len(storageCharge.Realizations[0].CreditsAllocated, 1)
	s.AssertDecimalEqual(apiRequestsAmount, apiRequestsCharge.Realizations[0].CreditsAllocated[0].Amount, "feature A allocated amount")
	s.AssertDecimalEqual(storageAmount, storageCharge.Realizations[0].CreditsAllocated[0].Amount, "feature B allocated amount")

	s.AssertDecimalEqual(
		apiRequestsAmount.Add(storageAmount),
		s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)),
		"nil-cost-basis accrued after advances",
	)

	// When feature-A restricted purchased credit is granted after the advances.
	creditPurchaseRes, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.CreateCreditPurchaseIntent(CreateCreditPurchaseIntentInput{
				Customer: cust.GetID(),
				Currency: USD,
				Amount:   purchaseAmount,
				ServicePeriod: timeutil.ClosedPeriod{
					From: createAt,
					To:   createAt,
				},
				Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
					GenericSettlement: creditpurchase.GenericSettlement{
						Currency:  currencyx.FiatCode(USD),
						CostBasis: costBasis,
					},
					InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
				}),
				FeatureFilters: creditpurchase.FeatureFilters{apiRequestsFeature.Key},
			}),
		},
	})
	s.NoError(err)
	s.Len(creditPurchaseRes, 1)
	creditPurchaseCharge, err := creditPurchaseRes[0].AsCreditPurchaseCharge()
	s.NoError(err)
	sourceChargeID := creditPurchaseCharge.ID

	// Then only feature A is backfilled by the purchase.
	s.AssertDecimalEqual(
		apiRequestsAmount.Sub(purchaseAmount).Add(storageAmount),
		s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)),
		"nil-cost-basis accrued after feature A backfill",
	)
	s.AssertDecimalEqual(purchaseAmount, s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&costBasis)), "purchased-cost-basis accrued after feature A backfill")
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerFBOBalanceForFeatures(cust.GetID(), USD, mo.Some(&costBasis), apiRequestsRoute), "feature A purchased-cost-basis FBO after backfill")
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerFBOBalanceForFeatures(cust.GetID(), USD, mo.Some(&costBasis), storageRoute), "feature B purchased-cost-basis FBO after feature A backfill")
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerFBOBalanceForFeatures(cust.GetID(), USD, mo.Some(&costBasis), mo.Some[[]string](nil)), "unrestricted purchased-cost-basis FBO after feature A backfill")
	s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
		Currency:  USD,
		CostBasis: mo.Some[*alpacadecimal.Decimal](nil),
	}, map[string]float64{
		sourceSpendChargeBucketKey(nil, &apiRequestsCharge.ID): apiRequestsAmount.Sub(purchaseAmount).InexactFloat64(), // 10 = feature A advance left uncovered.
		sourceSpendChargeBucketKey(nil, &storageCharge.ID):     storageAmount.InexactFloat64(),                         // 40 = feature B advance is untouched by feature-A purchase.
	})
	s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
		Currency:  USD,
		CostBasis: mo.Some(&costBasis),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, &apiRequestsCharge.ID): purchaseAmount.InexactFloat64(), // 20 = feature-A purchase backfills only feature-A spend.
	})

	// When the unrelated feature-B charge is deleted first.
	err = s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
		CustomerID: cust.GetID(),
		PatchesByChargeID: map[string]charges.Patch{
			storageCharge.ID: lo.Must(meta.NewPatchDelete(meta.NewPatchDeleteInput{
				ChangeSource: billing.ChangeSourceSystem,
				Policy:       meta.RefundAsCreditsDeletePolicy,
			})),
		},
	})
	s.NoError(err)

	// Then the feature-A backfilled credit is still consumed, and no feature-B credit is reopened.
	s.AssertDecimalEqual(
		apiRequestsAmount.Sub(purchaseAmount),
		s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)),
		"nil-cost-basis accrued after deleting feature B",
	)
	s.AssertDecimalEqual(purchaseAmount, s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&costBasis)), "purchased-cost-basis accrued after deleting feature B")
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerFBOBalanceForFeatures(cust.GetID(), USD, mo.Some(&costBasis), apiRequestsRoute), "feature A purchased-cost-basis FBO after deleting feature B")
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerFBOBalanceForFeatures(cust.GetID(), USD, mo.Some(&costBasis), storageRoute), "feature B purchased-cost-basis FBO after deleting feature B")
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerFBOBalanceForFeatures(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), storageRoute), "feature B nil-cost-basis FBO after deleting feature B")
	s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
		Currency:  USD,
		CostBasis: mo.Some[*alpacadecimal.Decimal](nil),
	}, map[string]float64{
		sourceSpendChargeBucketKey(nil, &apiRequestsCharge.ID): apiRequestsAmount.Sub(purchaseAmount).InexactFloat64(), // 10 = feature A uncovered slice remains.
	})
	s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
		Currency:  USD,
		CostBasis: mo.Some(&costBasis),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, &apiRequestsCharge.ID): purchaseAmount.InexactFloat64(), // 20 = feature-A purchased slice survives feature-B correction.
	})

	// When the feature-A charge is deleted.
	err = s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
		CustomerID: cust.GetID(),
		PatchesByChargeID: map[string]charges.Patch{
			apiRequestsCharge.ID: lo.Must(meta.NewPatchDelete(meta.NewPatchDeleteInput{
				ChangeSource: billing.ChangeSourceSystem,
				Policy:       meta.RefundAsCreditsDeletePolicy,
			})),
		},
	})
	s.NoError(err)

	// Then only the feature-A purchased credit is reopened on the feature-A FBO route.
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)), "nil-cost-basis accrued after deleting feature A")
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&costBasis)), "purchased-cost-basis accrued after deleting feature A")
	s.AssertDecimalEqual(purchaseAmount, s.MustCustomerFBOBalanceForFeatures(cust.GetID(), USD, mo.Some(&costBasis), apiRequestsRoute), "feature A purchased-cost-basis FBO after deleting feature A")
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerFBOBalanceForFeatures(cust.GetID(), USD, mo.Some(&costBasis), storageRoute), "feature B purchased-cost-basis FBO after deleting feature A")
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerFBOBalanceForFeatures(cust.GetID(), USD, mo.Some(&costBasis), mo.Some[[]string](nil)), "unrestricted purchased-cost-basis FBO after deleting feature A")
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerFBOBalanceForFeatures(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), apiRequestsRoute), "feature A nil-cost-basis FBO after deleting feature A")
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerFBOBalanceForFeatures(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), storageRoute), "feature B nil-cost-basis FBO after deleting feature A")
	s.requireCustomerFBOSourceBalanceBuckets(cust.GetID(), ledger.RouteFilter{
		Currency:  USD,
		CostBasis: mo.Some(&costBasis),
		Features:  apiRequestsRoute,
	}, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, nil): purchaseAmount.InexactFloat64(), // 20 = feature-A purchased backing returns to source-only FBO.
	})
	s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
		Currency: USD,
	}, map[string]float64{})
}

func (s *SanitySuite) TestFlatFeeCreditThenInvoiceSanity() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-sanity-test")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	const (
		flatFeeName = "flat-fee"
	)
	expectedPromotionalCreditAmount := float64(30) // 30 = promotional source consumed first by the flat-fee charge.
	expectedPurchasedCreditAmount := float64(50)   // 50 = purchased source consumed after promotional credits.
	expectedFlatFeeAmount := float64(100)          // 100 = full flat-fee spend to accrue and eventually recognize.
	// 20 = flat fee less the two available credit sources, so this stays source-less.
	expectedInvoiceBackedFlatFeeAmount := expectedFlatFeeAmount - expectedPromotionalCreditAmount - expectedPurchasedCreditAmount

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	setupAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-31T00:00:00Z", time.UTC).AsTime()

	clock.SetTime(setupAt)

	var promoSourceChargeID string
	s.Run("the customer receives a promotional credit grant", func() {
		result := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Amount:    alpacadecimal.NewFromFloat(expectedPromotionalCreditAmount),
			At:        setupAt,
			CostBasis: alpacadecimal.Zero,
		})

		// This should match the ledger's transaction group ID
		s.NotEmpty(result.Charge.Realizations.CreditGrantRealization.TransactionGroupID)
		promoSourceChargeID = result.Charge.ID

		// LEDGER:
		// - OnPromotionalCreditPurchase is called
		// - At this point the customer must have 30 USD promotional credits

		// Validate balances
		purchasedCostBasis := alpacadecimal.NewFromFloat(0.5)
		s.Equal(float64(0), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&purchasedCostBasis)).InexactFloat64())
	})

	var externalCreditPurchaseChargeID meta.ChargeID
	var externalSourceChargeID string
	s.Run("and customer purchases 50 USD credits as 0.5 costbasis", func() {
		intent := s.CreateCreditPurchaseIntent(CreateCreditPurchaseIntentInput{
			Customer: cust.GetID(),
			Currency: USD,
			Amount:   alpacadecimal.NewFromFloat(expectedPurchasedCreditAmount),
			ServicePeriod: timeutil.ClosedPeriod{
				From: setupAt,
				To:   setupAt,
			},
			Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
				GenericSettlement: creditpurchase.GenericSettlement{
					Currency:  currencyx.FiatCode(USD),
					CostBasis: alpacadecimal.NewFromFloat(0.5),
				},
				InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
			}),
		})

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				intent,
			},
		})
		s.NoError(err)

		s.Len(res, 1)
		s.Equal(meta.ChargeTypeCreditPurchase, res[0].Type())
		cpCharge, err := res[0].AsCreditPurchaseCharge()
		s.NoError(err)

		// This should match the ledger's transaction group ID
		s.NotEmpty(cpCharge.Realizations.CreditGrantRealization.TransactionGroupID)

		// LEDGER:
		// - OnCreditPurchaseInitiated is called
		// - At this point the customer must have 50 USD credits cost basis of 0.5

		// Validate balances
		costBasis := alpacadecimal.NewFromFloat(0.5)
		s.Equal(float64(50), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis)).InexactFloat64())
		s.Equal(float64(-50), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusOpen).InexactFloat64())

		externalCreditPurchaseChargeID = cpCharge.GetChargeID()
		externalSourceChargeID = cpCharge.ID
	})

	s.Run("the customer pays for the credit purchase - authorized", func() {
		updatedCharge, err := s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
			ChargeID:           externalCreditPurchaseChargeID,
			TargetPaymentState: payment.StatusAuthorized,
		})
		s.NoError(err)

		// LEDGER:
		// - OnCreditPurchasePaymentAuthorized is called

		costBasis := alpacadecimal.NewFromFloat(0.5)
		s.Equal(payment.StatusAuthorized, updatedCharge.Realizations.ExternalPaymentSettlement.Status)
		s.Equal(float64(0), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusOpen).InexactFloat64())
		s.Equal(float64(-50), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusAuthorized).InexactFloat64())
	})

	s.Run("the customer settles the credit purchase payment", func() {
		updatedCharge, err := s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
			ChargeID:           externalCreditPurchaseChargeID,
			TargetPaymentState: payment.StatusSettled,
		})
		s.NoError(err)

		// LEDGER:
		// - OnCreditPurchasePaymentSettled is called

		costBasis := alpacadecimal.NewFromFloat(0.5)
		s.Equal(payment.StatusSettled, updatedCharge.Realizations.ExternalPaymentSettlement.Status)
		s.Equal(float64(0), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusOpen).InexactFloat64())
	})

	// TOTAL credits balance: 30 + 50 = 80 USD

	var flatFeeChargeID meta.ChargeID
	promoCostBasis := alpacadecimal.Zero
	externalCostBasis := alpacadecimal.NewFromFloat(0.5)
	type flatFeeLedgerSnapshot struct {
		promoFBO             alpacadecimal.Decimal
		externalFBO          alpacadecimal.Decimal
		promoReceivable      alpacadecimal.Decimal
		externalReceivable   alpacadecimal.Decimal
		totalOpenReceivable  alpacadecimal.Decimal
		accrued              alpacadecimal.Decimal
		authorizedReceivable alpacadecimal.Decimal
		totalWash            alpacadecimal.Decimal
		externalWash         alpacadecimal.Decimal
		earnings             alpacadecimal.Decimal
	}
	flatFeeStart := flatFeeLedgerSnapshot{
		promoFBO:             s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&promoCostBasis)),
		externalFBO:          s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&externalCostBasis)),
		promoReceivable:      s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&promoCostBasis), ledger.TransactionAuthorizationStatusOpen),
		externalReceivable:   s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&externalCostBasis), ledger.TransactionAuthorizationStatusOpen),
		totalOpenReceivable:  s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen),
		accrued:              s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]()),
		authorizedReceivable: s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized),
		totalWash:            s.MustWashBalance(ns, USD, mo.None[*alpacadecimal.Decimal]()),
		externalWash:         s.MustWashBalance(ns, USD, mo.Some(&externalCostBasis)),
		earnings:             s.MustEarningsBalance(ns, USD),
	}
	assertDelta := func(label string, start, delta, actual alpacadecimal.Decimal) {
		s.T().Helper()
		expected := start.Add(delta)
		s.True(actual.Equal(expected), "%s: expected start %s + delta %s = %s, got %s", label, start.String(), delta.String(), expected.String(), actual.String())
	}

	s.Run("create new upcoming charge for flat fee", func() {
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromFloat(expectedFlatFeeAmount),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					Name:              flatFeeName,
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: flatFeeName,
				}),
			},
		})
		s.NoError(err)

		s.Len(res, 1)
		s.Equal(res[0].Type(), meta.ChargeTypeFlatFee)
		flatFeeCharge, err := res[0].AsFlatFeeCharge()
		s.NoError(err)

		// LEDGER:
		// - This is a noop as this is in the future, so it just creates the charge + gathering line

		flatFeeChargeID = flatFeeCharge.GetChargeID()
	})

	var stdInvoiceID billing.InvoiceID
	var stdLineID billing.LineID
	clock.SetTime(servicePeriod.From)
	s.Run("invoice the charge", func() {
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.From),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		invoice := invoices[0]
		s.DebugDumpStandardInvoice("invoice after invoice pending lines", invoice)

		s.Len(invoice.Lines.OrEmpty(), 1)
		stdLine := invoice.Lines.OrEmpty()[0]

		s.Equal(flatFeeChargeID.ID, *stdLine.ChargeID)
		stdLineID = stdLine.GetLineID()

		charge := s.MustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)

		s.Equal(flatFeeChargeID.ID, updatedFlatFeeCharge.ID)
		s.Require().NotNil(updatedFlatFeeCharge.Realizations.CurrentRun)

		// LEDGER:
		// - OnAllocateCredits is called with the pre tax amount to allocate of USD 100
		// - Two credit realizations should happen for the two different credit types

		// Validate the credit realizations
		// The charge should have $80 realized as credits
		s.Len(updatedFlatFeeCharge.Realizations.CurrentRun.CreditRealizations, 2)
		promotionalCreditRealization := updatedFlatFeeCharge.Realizations.CurrentRun.CreditRealizations[0]
		s.Equal(expectedPromotionalCreditAmount, promotionalCreditRealization.Amount.InexactFloat64())

		customerCreditRealization := updatedFlatFeeCharge.Realizations.CurrentRun.CreditRealizations[1]
		s.Equal(expectedPurchasedCreditAmount, customerCreditRealization.Amount.InexactFloat64())

		assertDelta("promo FBO after invoice assignment", flatFeeStart.promoFBO, alpacadecimal.NewFromInt(-30), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&promoCostBasis)))
		assertDelta("external FBO after invoice assignment", flatFeeStart.externalFBO, alpacadecimal.NewFromInt(-50), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&externalCostBasis)))
		assertDelta("promo receivable after invoice assignment", flatFeeStart.promoReceivable, alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&promoCostBasis), ledger.TransactionAuthorizationStatusOpen))
		assertDelta("external receivable after invoice assignment", flatFeeStart.externalReceivable, alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&externalCostBasis), ledger.TransactionAuthorizationStatusOpen))
		assertDelta("total open receivable after invoice assignment", flatFeeStart.totalOpenReceivable, alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen))
		assertDelta("accrued after invoice assignment", flatFeeStart.accrued, alpacadecimal.NewFromInt(80), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]()))
		assertDelta("authorized receivable after invoice assignment", flatFeeStart.authorizedReceivable, alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized))
		assertDelta("total wash after invoice assignment", flatFeeStart.totalWash, alpacadecimal.Zero, s.MustWashBalance(ns, USD, mo.None[*alpacadecimal.Decimal]()))
		assertDelta("external wash after invoice assignment", flatFeeStart.externalWash, alpacadecimal.Zero, s.MustWashBalance(ns, USD, mo.Some(&externalCostBasis)))
		assertDelta("earnings after invoice assignment", flatFeeStart.earnings, alpacadecimal.Zero, s.MustEarningsBalance(ns, USD))
		s.requireCustomerFBOSourceBalanceBuckets(cust.GetID(), ledger.RouteFilter{
			Currency: USD,
		}, map[string]float64{}) // 0 = both available credit sources were consumed by the invoice assignment.
		s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
			Currency: USD,
		}, map[string]float64{
			sourceSpendChargeBucketKey(&promoSourceChargeID, &flatFeeChargeID.ID):    expectedPromotionalCreditAmount,
			sourceSpendChargeBucketKey(&externalSourceChargeID, &flatFeeChargeID.ID): expectedPurchasedCreditAmount,
		})

		stdInvoiceID = invoice.GetInvoiceID()
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
	})

	s.Run("advance the invoice to payment processing", func() {
		invoice, err := s.BillingService.ApproveInvoice(ctx, stdInvoiceID)
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

		charge := s.MustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)

		// LEDGER:
		// - OnFlatFeeStandardInvoiceUsageAccrued is called with the service period and totals of USD 20 to be represented
		//   on the ledger
		// - Payment authorization is deferred until the payment app advances the invoice beyond pending

		// Invoice usage accrued callback should have been invoked
		s.Require().NotNil(updatedFlatFeeCharge.Realizations.CurrentRun)
		accruedUsage := updatedFlatFeeCharge.Realizations.CurrentRun.AccruedUsage
		s.NotNil(accruedUsage)
		s.Equal(flatfee.StatusActiveAwaitingPaymentSettlement, updatedFlatFeeCharge.Status)
		s.Equal(servicePeriod, accruedUsage.ServicePeriod, "service period should be the same as the input")
		s.NotNil(updatedFlatFeeCharge.Realizations.CurrentRun.LineID, "run line ID should be set")
		s.Equal(stdLineID.ID, *updatedFlatFeeCharge.Realizations.CurrentRun.LineID, "run line ID should be the same as the standard line")
		s.Equal(expectedInvoiceBackedFlatFeeAmount, accruedUsage.Totals.Total.InexactFloat64(), "totals should be the same as the input")
		s.Equal(expectedPromotionalCreditAmount+expectedPurchasedCreditAmount, accruedUsage.Totals.CreditsTotal.InexactFloat64(), "totals should be the same as the input")

		assertDelta("promo FBO after payment authorization", flatFeeStart.promoFBO, alpacadecimal.NewFromInt(-30), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&promoCostBasis)))
		assertDelta("external FBO after payment authorization", flatFeeStart.externalFBO, alpacadecimal.NewFromInt(-50), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&externalCostBasis)))
		assertDelta("promo receivable after payment processing pending", flatFeeStart.promoReceivable, alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&promoCostBasis), ledger.TransactionAuthorizationStatusOpen))
		assertDelta("external receivable after payment processing pending", flatFeeStart.externalReceivable, alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&externalCostBasis), ledger.TransactionAuthorizationStatusOpen))
		assertDelta("total open receivable after payment processing pending", flatFeeStart.totalOpenReceivable, alpacadecimal.NewFromInt(-20), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen))
		assertDelta("authorized receivable after payment processing pending", flatFeeStart.authorizedReceivable, alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized))
		assertDelta("accrued after payment processing pending", flatFeeStart.accrued, alpacadecimal.NewFromInt(100), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]()))
		assertDelta("total wash after payment processing pending", flatFeeStart.totalWash, alpacadecimal.Zero, s.MustWashBalance(ns, USD, mo.None[*alpacadecimal.Decimal]()))
		assertDelta("external wash after payment processing pending", flatFeeStart.externalWash, alpacadecimal.Zero, s.MustWashBalance(ns, USD, mo.Some(&externalCostBasis)))
		assertDelta("earnings after payment processing pending", flatFeeStart.earnings, alpacadecimal.Zero, s.MustEarningsBalance(ns, USD))
		s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
			Currency: USD,
		}, map[string]float64{
			sourceSpendChargeBucketKey(&promoSourceChargeID, &flatFeeChargeID.ID):    expectedPromotionalCreditAmount,
			sourceSpendChargeBucketKey(&externalSourceChargeID, &flatFeeChargeID.ID): expectedPurchasedCreditAmount,
			sourceSpendChargeBucketKey(nil, &flatFeeChargeID.ID):                     expectedInvoiceBackedFlatFeeAmount,
		})
	})

	s.Run("payment is authorized", func() {
		invoice, err := s.BillingService.PaymentAuthorized(ctx, stdInvoiceID)
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingAuthorized, invoice.Status)

		// LEDGER:
		// - OnFlatFeePaymentAuthorized is called with the remaining USD 20

		charge := s.MustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)
		s.Equal(flatfee.StatusActiveAwaitingPaymentSettlement, updatedFlatFeeCharge.Status)
		s.Require().NotNil(updatedFlatFeeCharge.Realizations.CurrentRun)
		s.NotNil(updatedFlatFeeCharge.Realizations.CurrentRun.Payment)
		s.Equal(payment.StatusAuthorized, updatedFlatFeeCharge.Realizations.CurrentRun.Payment.Status)
		s.NotNil(updatedFlatFeeCharge.Realizations.CurrentRun.Payment.Authorized)
		s.Nil(updatedFlatFeeCharge.Realizations.CurrentRun.Payment.Settled)

		assertDelta("promo receivable after payment authorization", flatFeeStart.promoReceivable, alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&promoCostBasis), ledger.TransactionAuthorizationStatusOpen))
		assertDelta("external receivable after payment authorization", flatFeeStart.externalReceivable, alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&externalCostBasis), ledger.TransactionAuthorizationStatusOpen))
		assertDelta("total open receivable after payment authorization", flatFeeStart.totalOpenReceivable, alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen))
		assertDelta("authorized receivable after payment authorization", flatFeeStart.authorizedReceivable, alpacadecimal.NewFromInt(-20), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized))
		assertDelta("accrued after payment authorization", flatFeeStart.accrued, alpacadecimal.NewFromInt(100), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]()))
		assertDelta("total wash after payment authorization", flatFeeStart.totalWash, alpacadecimal.Zero, s.MustWashBalance(ns, USD, mo.None[*alpacadecimal.Decimal]()))
		assertDelta("external wash after payment authorization", flatFeeStart.externalWash, alpacadecimal.Zero, s.MustWashBalance(ns, USD, mo.Some(&externalCostBasis)))
		assertDelta("earnings after payment authorization", flatFeeStart.earnings, alpacadecimal.Zero, s.MustEarningsBalance(ns, USD))
	})

	s.Run("payment is settled", func() {
		invoice, err := s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: stdInvoiceID,
			Trigger:   billing.TriggerPaid,
		})
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)

		// LEDGER:
		// - OnFlatFeePaymentSettled is called with the USD 20

		charge := s.MustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)
		s.Equal(flatfee.StatusFinal, updatedFlatFeeCharge.Status)
		s.Require().NotNil(updatedFlatFeeCharge.Realizations.CurrentRun)
		s.NotNil(updatedFlatFeeCharge.Realizations.CurrentRun.Payment)
		s.Equal(payment.StatusSettled, updatedFlatFeeCharge.Realizations.CurrentRun.Payment.Status)
		s.NotNil(updatedFlatFeeCharge.Realizations.CurrentRun.Payment.Authorized)
		s.NotNil(updatedFlatFeeCharge.Realizations.CurrentRun.Payment.Settled)

		assertDelta("promo receivable after payment settlement", flatFeeStart.promoReceivable, alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&promoCostBasis), ledger.TransactionAuthorizationStatusOpen))
		assertDelta("external receivable after payment settlement", flatFeeStart.externalReceivable, alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&externalCostBasis), ledger.TransactionAuthorizationStatusOpen))
		assertDelta("total open receivable after payment settlement", flatFeeStart.totalOpenReceivable, alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen))
		assertDelta("authorized receivable after payment settlement", flatFeeStart.authorizedReceivable, alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized))
		assertDelta("accrued after payment settlement", flatFeeStart.accrued, alpacadecimal.NewFromInt(100), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]()))
		assertDelta("total wash after payment settlement", flatFeeStart.totalWash, alpacadecimal.NewFromInt(-20), s.MustWashBalance(ns, USD, mo.None[*alpacadecimal.Decimal]()))
		assertDelta("external wash after payment settlement", flatFeeStart.externalWash, alpacadecimal.Zero, s.MustWashBalance(ns, USD, mo.Some(&externalCostBasis)))
		assertDelta("earnings after payment settlement", flatFeeStart.earnings, alpacadecimal.Zero, s.MustEarningsBalance(ns, USD))
	})

	s.Run("recognized revenue skips invoice-backed accrued until correction support exists", func() {
		// given:
		// - the flat-fee charge is final with two credited source slices and one invoice-backed slice accrued
		// when:
		// - revenue recognition runs after the service period
		// then:
		// - only credit-backed accrued is recognized
		// - invoice-backed accrued stays accrued until recognition correction support exists for invoice-backed value
		clock.FreezeTime(servicePeriod.To)

		expectedCreditBackedFlatFeeAmount := expectedPromotionalCreditAmount + expectedPurchasedCreditAmount // 80 = 30 promotional + 50 purchased credits have lineage segments eligible for recognition.
		expectedInvoiceBackedAccruedAfterRecognition := expectedInvoiceBackedFlatFeeAmount                   // 20 = invoice-backed accrued intentionally remains unrecognized for now.

		// TODO: invoice-backed accrued has spend_charge_id and cost basis 1, but is
		// intentionally not recognized until invoice-backed recognition correction
		// is implemented.
		s.MustRecognizeRevenue(cust.GetID(), USD, alpacadecimal.NewFromFloat(expectedCreditBackedFlatFeeAmount))
		s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
			Currency: USD,
		}, map[string]float64{
			sourceSpendChargeBucketKey(nil, &flatFeeChargeID.ID): expectedInvoiceBackedAccruedAfterRecognition,
		})
		s.requireEarningsSourceSpendBalanceBuckets(ns, ledger.RouteFilter{
			Currency: USD,
		}, map[string]float64{
			sourceSpendChargeBucketKey(&promoSourceChargeID, &flatFeeChargeID.ID):    expectedPromotionalCreditAmount,
			sourceSpendChargeBucketKey(&externalSourceChargeID, &flatFeeChargeID.ID): expectedPurchasedCreditAmount,
		})
	})
}

func (s *SanitySuite) TestFlatFeeCreditThenInvoiceUsesFreeGrantBeforePaidPriorityTie() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-sanity-free-before-paid")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	setupAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-31T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.SetTime(setupAt)
	defer clock.ResetTime()

	priority := 1
	freeAmount := alpacadecimal.NewFromInt(20)
	paidAmount := alpacadecimal.NewFromInt(20)
	paidCostBasis := alpacadecimal.NewFromFloat(0.5)
	freeCostBasis := alpacadecimal.Zero

	var paidSourceChargeID string
	s.Run("the customer purchases paid credits first", func() {
		intent := s.CreateCreditPurchaseIntent(CreateCreditPurchaseIntentInput{
			Customer:      cust.GetID(),
			Currency:      USD,
			Amount:        paidAmount,
			Priority:      &priority,
			ServicePeriod: timeutil.ClosedPeriod{From: setupAt, To: setupAt},
			Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
				GenericSettlement: creditpurchase.GenericSettlement{
					Currency:  USD,
					CostBasis: paidCostBasis,
				},
				InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
			}),
		})

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents:   charges.ChargeIntents{intent},
		})
		s.NoError(err)
		s.Len(res, 1)
		cpCharge, err := res[0].AsCreditPurchaseCharge()
		s.NoError(err)

		_, err = s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
			ChargeID:           cpCharge.GetChargeID(),
			TargetPaymentState: payment.StatusAuthorized,
		})
		s.NoError(err)
		_, err = s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
			ChargeID:           cpCharge.GetChargeID(),
			TargetPaymentState: payment.StatusSettled,
		})
		s.NoError(err)

		paidSourceChargeID = cpCharge.ID
		s.AssertDecimalEqual(paidAmount, s.MustCustomerFBOBalanceWithPriority(cust.GetID(), USD, mo.Some(&paidCostBasis), priority), "paid FBO before usage")
	})

	var freeSourceChargeID string
	s.Run("the customer receives same-priority free credits second", func() {
		result := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Amount:    freeAmount,
			At:        setupAt,
			CostBasis: freeCostBasis,
			Priority:  &priority,
		})
		freeSourceChargeID = result.Charge.ID
		s.AssertDecimalEqual(freeAmount, s.MustCustomerFBOBalanceWithPriority(cust.GetID(), USD, mo.Some(&freeCostBasis), priority), "free FBO before usage")
	})

	var flatFeeChargeID meta.ChargeID
	s.Run("a flat fee is created", func() {
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromInt(30),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					Name:              "flat-fee-free-before-paid",
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: "flat-fee-free-before-paid",
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		flatFeeCharge, err := res[0].AsFlatFeeCharge()
		s.NoError(err)
		flatFeeChargeID = flatFeeCharge.GetChargeID()
	})

	clock.SetTime(servicePeriod.From)
	s.Run("invoice assignment consumes free credits before paid credits", func() {
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.From),
		})
		s.NoError(err)
		s.Len(invoices, 1)

		charge := s.MustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)
		s.Require().NotNil(updatedFlatFeeCharge.Realizations.CurrentRun)
		s.Len(updatedFlatFeeCharge.Realizations.CurrentRun.CreditRealizations, 2)

		s.requireCustomerFBOSourceBalanceBuckets(cust.GetID(), ledger.RouteFilter{
			Currency:  USD,
			CostBasis: mo.Some(&paidCostBasis),
		}, map[string]float64{
			sourceSpendChargeBucketKey(&paidSourceChargeID, nil): 10, // 10 = paid source remains because free credit tied by priority is collected first.
		})
		s.requireCustomerFBOSourceBalanceBuckets(cust.GetID(), ledger.RouteFilter{
			Currency:  USD,
			CostBasis: mo.Some(&freeCostBasis),
		}, map[string]float64{})
		s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
			Currency: USD,
		}, map[string]float64{
			sourceSpendChargeBucketKey(&freeSourceChargeID, &flatFeeChargeID.ID): 20, // 20 = full free source is used before paid credit.
			sourceSpendChargeBucketKey(&paidSourceChargeID, &flatFeeChargeID.ID): 10, // 10 = remaining spend spills into paid credit.
		})
	})
}

type meteredFeatureSetup struct {
	key  string
	name string
}

func (s *SanitySuite) setupMeteredFeatures(ctx context.Context, ns string, inputs ...meteredFeatureSetup) map[string]pcfeature.Feature {
	s.T().Helper()

	meters := make([]meter.Meter, 0, len(inputs))
	meterIDsByKey := make(map[string]string, len(inputs))
	for _, input := range inputs {
		meterID := ulid.Make().String()
		meterIDsByKey[input.key] = meterID
		meters = append(meters, meter.Meter{
			ManagedResource: models.ManagedResource{
				ID: meterID,
				NamespacedModel: models.NamespacedModel{
					Namespace: ns,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: input.name,
			},
			Key:           input.key,
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		})
	}

	err := s.MeterAdapter.ReplaceMeters(ctx, meters)
	s.NoError(err, "replacing meters must not return error")

	featuresByKey := make(map[string]pcfeature.Feature, len(inputs))
	for _, input := range inputs {
		s.MockStreamingConnector.AddSimpleEvent(input.key, 0, time.Now())

		feature, err := s.FeatureService.CreateFeature(ctx, pcfeature.CreateFeatureInputs{
			Namespace: ns,
			Name:      input.key,
			Key:       input.key,
			MeterID:   lo.ToPtr(meterIDsByKey[input.key]),
		})
		s.NoError(err)
		featuresByKey[input.key] = feature
	}

	return featuresByKey
}

func (s *SanitySuite) TestCreditPurchasePersistsPriority() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-creditpurchase-persists-priority")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	priority := 7
	at := datetime.MustParseTimeInLocation(s.T(), "2026-01-01T12:34:56Z", time.UTC).AsTime()

	result := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Amount:    alpacadecimal.NewFromInt(25),
		At:        at,
		CostBasis: alpacadecimal.Zero,
		Priority:  &priority,
	})

	cpCharge := result.Charge
	s.NotNil(cpCharge.Realizations.CreditGrantRealization)

	fetchedCharge, err := s.MustGetChargeByID(cpCharge.GetChargeID()).AsCreditPurchaseCharge()
	s.NoError(err)
	s.Equal(&priority, fetchedCharge.Intent.Priority)

	zeroCostBasis := alpacadecimal.Zero
	sourceChargeID := cpCharge.ID
	s.True(s.MustCustomerFBOBalanceWithPriority(cust.GetID(), USD, mo.Some(&zeroCostBasis), priority).Equal(alpacadecimal.NewFromInt(25)))
	s.True(s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&zeroCostBasis)).Equal(alpacadecimal.Zero))
	s.requireCustomerFBOSourceBalanceBuckets(cust.GetID(), ledger.RouteFilter{
		Currency:       USD,
		CostBasis:      mo.Some(&zeroCostBasis),
		CreditPriority: &priority,
	}, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, nil): 25, // 25 = the full priority grant is available and source-attributed.
	})
}

func (s *SanitySuite) TestUsageBasedCreditThenInvoicePaymentLifecycle() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-credits-usagebased-credit-then-invoice-payment-lifecycle")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	promoCostBasis := alpacadecimal.Zero
	invoiceCostBasis := alpacadecimal.NewFromInt(1)
	expectedCreditedUsageAmount := float64(5) // 5 = the promotional credit grant can cover the first 5 USD of usage.
	expectedTotalUsageAmount := float64(12.5) // 12.5 = 100 original events + 25 late events, rated at 0.10 USD each.
	// 7.5 = total usage less promotional credit coverage.
	expectedInvoiceBackedUsageAmount := expectedTotalUsageAmount - expectedCreditedUsageAmount

	var (
		usageBasedChargeID meta.ChargeID
		sourceChargeID     string
		invoice            billing.StandardInvoice
	)

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	s.Run("the customer receives a promotional credit grant", func() {
		// given:
		// - a ledger-backed customer exists
		// when:
		// - the customer receives 5 USD promotional credit
		// then:
		// - the funding charge becomes the source provenance for credited usage
		funding := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Amount:    alpacadecimal.NewFromFloat(expectedCreditedUsageAmount),
			At:        createAt,
			CostBasis: promoCostBasis,
		})
		sourceChargeID = funding.Charge.ID
	})

	s.Run("a credit-then-invoice usage based charge is created with initial metered usage", func() {
		// given:
		// - usage exists inside the service period
		// when:
		// - a credit-then-invoice usage charge is created
		// then:
		// - the charge ID becomes the spend provenance for accrued and receivable entries
		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			100,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromFloat(0.1),
					}),
					Name:              "usage-based-credit-then-invoice-payment-lifecycle",
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: "usage-based-credit-then-invoice-payment-lifecycle",
					FeatureKey:        apiRequestsTotal.Feature.Key,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		usageBasedChargeID, err = res[0].GetChargeID()
		s.NoError(err)
	})

	s.Run("the pending invoice is created for the service period", func() {
		// given:
		// - the service period has ended
		// when:
		// - billing invoices pending lines
		// then:
		// - the usage line remains pending until invoice approval
		clock.FreezeTime(servicePeriod.To.Add(time.Second))

		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		invoice = invoices[0]
	})

	s.Run("late arriving usage is included while its stored_at remains before the invoice finalization cutoff", func() {
		// given:
		// - a draft invoice exists before finalization
		// when:
		// - late usage arrives before the invoice cutoff
		// then:
		// - the invoice approval should allocate both original and late usage
		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			25,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-20T00:00:00Z", time.UTC).AsTime(),
			streamingtestutils.WithStoredAt(datetime.MustParseTimeInLocation(s.T(), "2026-02-02T12:00:00Z", time.UTC).AsTime()),
		)
	})

	s.Run("the invoice is advanced and approved into payment pending", func() {
		// given:
		// - total invoice usage exceeds available promotional credits
		// when:
		// - the invoice advances and is approved
		// then:
		// - credited usage keeps source/spend provenance and the fiat remainder keeps spend-only provenance
		clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())

		var err error
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Len(invoice.Lines.OrEmpty(), 1)
		stdLine := invoice.Lines.OrEmpty()[0]
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       expectedTotalUsageAmount,
			Total:        expectedInvoiceBackedUsageAmount,
			CreditsTotal: expectedCreditedUsageAmount,
		}, stdLine.Totals)

		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

		usageBasedCharge := s.MustGetChargeByID(usageBasedChargeID)
		updatedCharge, err := usageBasedCharge.AsUsageBasedCharge()
		s.NoError(err)
		s.Equal(usagebased.StatusActiveAwaitingPaymentSettlement, updatedCharge.Status)
		s.Len(updatedCharge.Realizations, 1)
		s.NotNil(updatedCharge.Realizations[0].InvoiceUsage)
		s.Equal(expectedInvoiceBackedUsageAmount, updatedCharge.Realizations[0].InvoiceUsage.Totals.Total.InexactFloat64())

		// Promotional grants settle immediately through wash, so only the
		// invoice-backed receivable remains open at this point.
		expectedOpenReceivableAmount := -expectedInvoiceBackedUsageAmount // -7.5 = invoice-backed remainder is still open before authorization.
		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&promoCostBasis), ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.Zero))
		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&invoiceCostBasis), ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.NewFromFloat(expectedOpenReceivableAmount)))
		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.NewFromFloat(expectedOpenReceivableAmount)))
		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized).Equal(alpacadecimal.Zero))
		s.True(s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]()).Equal(alpacadecimal.NewFromFloat(expectedTotalUsageAmount)))
		s.True(s.MustWashBalance(ns, USD, mo.None[*alpacadecimal.Decimal]()).Equal(alpacadecimal.NewFromFloat(-expectedCreditedUsageAmount)))
		s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
			Currency: USD,
		}, map[string]float64{
			sourceSpendChargeBucketKey(&sourceChargeID, &usageBasedChargeID.ID): expectedCreditedUsageAmount,
			sourceSpendChargeBucketKey(nil, &usageBasedChargeID.ID):             expectedInvoiceBackedUsageAmount,
		})
		s.requireCustomerReceivableSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
			Currency:                       USD,
			TransactionAuthorizationStatus: lo.ToPtr(ledger.TransactionAuthorizationStatusOpen),
		}, map[string]float64{
			sourceSpendChargeBucketKey(nil, &usageBasedChargeID.ID): expectedOpenReceivableAmount,
		})
	})

	s.Run("the payment is authorized", func() {
		// given:
		// - the invoice-backed remainder is open receivable
		// when:
		// - payment is authorized
		// then:
		// - spend provenance moves from open receivable to authorized receivable
		var err error
		invoice, err = s.BillingService.PaymentAuthorized(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingAuthorized, invoice.Status)

		usageBasedCharge := s.MustGetChargeByID(usageBasedChargeID)
		updatedCharge, err := usageBasedCharge.AsUsageBasedCharge()
		s.NoError(err)
		s.Equal(usagebased.StatusActiveAwaitingPaymentSettlement, updatedCharge.Status)
		s.NotNil(updatedCharge.Realizations[0].Payment)
		s.Equal(payment.StatusAuthorized, updatedCharge.Realizations[0].Payment.Status)
		s.NotNil(updatedCharge.Realizations[0].Payment.Authorized)
		s.Nil(updatedCharge.Realizations[0].Payment.Settled)

		expectedAuthorizedReceivableAmount := -expectedInvoiceBackedUsageAmount // -7.5 = authorization moves the invoice-backed remainder from open to authorized.
		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&promoCostBasis), ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.Zero))
		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&invoiceCostBasis), ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.Zero))
		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.Zero))
		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&invoiceCostBasis), ledger.TransactionAuthorizationStatusAuthorized).Equal(alpacadecimal.NewFromFloat(expectedAuthorizedReceivableAmount)))
		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized).Equal(alpacadecimal.NewFromFloat(expectedAuthorizedReceivableAmount)))
		s.True(s.MustWashBalance(ns, USD, mo.None[*alpacadecimal.Decimal]()).Equal(alpacadecimal.NewFromFloat(-expectedCreditedUsageAmount)))
		s.requireCustomerReceivableSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
			Currency:                       USD,
			TransactionAuthorizationStatus: lo.ToPtr(ledger.TransactionAuthorizationStatusOpen),
		}, map[string]float64{})
		s.requireCustomerReceivableSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
			Currency:                       USD,
			TransactionAuthorizationStatus: lo.ToPtr(ledger.TransactionAuthorizationStatusAuthorized),
		}, map[string]float64{
			sourceSpendChargeBucketKey(nil, &usageBasedChargeID.ID): expectedAuthorizedReceivableAmount,
		})
	})

	s.Run("the payment is settled and the charge reaches final", func() {
		// given:
		// - the invoice-backed remainder is authorized receivable
		// when:
		// - payment settles
		// then:
		// - receivable provenance clears and accrued provenance remains split by source-backed and invoice-backed usage
		var err error
		invoice, err = s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: invoice.GetInvoiceID(),
			Trigger:   billing.TriggerPaid,
		})
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)

		usageBasedCharge := s.MustGetChargeByID(usageBasedChargeID)
		updatedCharge, err := usageBasedCharge.AsUsageBasedCharge()
		s.NoError(err)
		s.Equal(usagebased.StatusFinal, updatedCharge.Status)
		s.NotNil(updatedCharge.Realizations[0].Payment)
		s.Equal(payment.StatusSettled, updatedCharge.Realizations[0].Payment.Status)
		s.NotNil(updatedCharge.Realizations[0].Payment.Settled)

		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&promoCostBasis), ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.Zero))
		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&invoiceCostBasis), ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.Zero))
		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.Zero))
		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&invoiceCostBasis), ledger.TransactionAuthorizationStatusAuthorized).Equal(alpacadecimal.Zero))
		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized).Equal(alpacadecimal.Zero))
		s.True(s.MustWashBalance(ns, USD, mo.None[*alpacadecimal.Decimal]()).Equal(alpacadecimal.NewFromFloat(-expectedTotalUsageAmount)))
		s.requireCustomerReceivableSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
			Currency:                       USD,
			TransactionAuthorizationStatus: lo.ToPtr(ledger.TransactionAuthorizationStatusOpen),
		}, map[string]float64{})
		s.requireCustomerReceivableSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
			Currency:                       USD,
			TransactionAuthorizationStatus: lo.ToPtr(ledger.TransactionAuthorizationStatusAuthorized),
		}, map[string]float64{})
		s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
			Currency: USD,
		}, map[string]float64{
			sourceSpendChargeBucketKey(&sourceChargeID, &usageBasedChargeID.ID): expectedCreditedUsageAmount,
			sourceSpendChargeBucketKey(nil, &usageBasedChargeID.ID):             expectedInvoiceBackedUsageAmount,
		})
	})

	s.Run("only credit-backed accrued usage is recognized as earnings", func() {
		// given:
		// - settlement left credited and invoice-backed usage accrued under separate provenance buckets
		// when:
		// - revenue recognition runs after the service period
		// then:
		// - only credit-backed accrued is recognized
		// - invoice-backed accrued stays accrued until recognition correction support exists for invoice-backed value
		clock.FreezeTime(servicePeriod.To)

		expectedCreditBackedRecognizedAmount := expectedCreditedUsageAmount              // 5 = promotional credit-backed usage has lineage eligible for recognition.
		expectedInvoiceBackedAccruedAfterRecognition := expectedInvoiceBackedUsageAmount // 7.5 = invoice-backed usage intentionally remains unrecognized for now.
		expectedCreditBackedEarningsAmount := expectedCreditedUsageAmount                // 5 = recognized earnings are limited to the credited usage.

		// TODO: invoice-backed accrued has spend_charge_id and cost basis 1, but is
		// intentionally not recognized until invoice-backed recognition correction
		// is implemented.
		s.MustRecognizeRevenue(cust.GetID(), USD, alpacadecimal.NewFromFloat(expectedCreditBackedRecognizedAmount))
		s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
			Currency: USD,
		}, map[string]float64{
			sourceSpendChargeBucketKey(nil, &usageBasedChargeID.ID): expectedInvoiceBackedAccruedAfterRecognition,
		})
		s.requireEarningsSourceSpendBalanceBuckets(ns, ledger.RouteFilter{
			Currency: USD,
		}, map[string]float64{
			sourceSpendChargeBucketKey(&sourceChargeID, &usageBasedChargeID.ID): expectedCreditBackedEarningsAmount,
		})
	})
}

func (s *SanitySuite) TestFlatFeeCreditOnlySanity() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-sanity-test-credit-only")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	const (
		flatFeeName = "flat-fee-credit-only"
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	setupAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-31T00:00:00Z", time.UTC).AsTime()

	clock.SetTime(setupAt)

	var promoSourceChargeID string
	s.Run("the customer receives a promotional credit grant", func() {
		result := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Amount:    alpacadecimal.NewFromFloat(30),
			At:        setupAt,
			CostBasis: alpacadecimal.Zero,
		})
		s.NotEmpty(result.Charge.Realizations.CreditGrantRealization.TransactionGroupID)
		promoSourceChargeID = result.Charge.ID

		purchasedCostBasis := alpacadecimal.NewFromFloat(0.5)
		s.Equal(float64(0), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&purchasedCostBasis)).InexactFloat64())
	})

	var externalCreditPurchaseChargeID meta.ChargeID
	var externalSourceChargeID string
	s.Run("and customer purchases 50 USD credits as 0.5 costbasis", func() {
		intent := s.CreateCreditPurchaseIntent(CreateCreditPurchaseIntentInput{
			Customer: cust.GetID(),
			Currency: USD,
			Amount:   alpacadecimal.NewFromFloat(50),
			ServicePeriod: timeutil.ClosedPeriod{
				From: setupAt,
				To:   setupAt,
			},
			Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
				GenericSettlement: creditpurchase.GenericSettlement{
					Currency:  currencyx.FiatCode(USD),
					CostBasis: alpacadecimal.NewFromFloat(0.5),
				},
				InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
			}),
		})

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				intent,
			},
		})
		s.NoError(err)

		s.Len(res, 1)
		s.Equal(meta.ChargeTypeCreditPurchase, res[0].Type())
		cpCharge, err := res[0].AsCreditPurchaseCharge()
		s.NoError(err)
		s.NotEmpty(cpCharge.Realizations.CreditGrantRealization.TransactionGroupID)

		costBasis := alpacadecimal.NewFromFloat(0.5)
		s.Equal(float64(50), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis)).InexactFloat64())
		s.Equal(float64(-50), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusOpen).InexactFloat64())

		externalCreditPurchaseChargeID = cpCharge.GetChargeID()
		externalSourceChargeID = cpCharge.ID
	})

	s.Run("the customer pays for the credit purchase - authorized", func() {
		updatedCharge, err := s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
			ChargeID:           externalCreditPurchaseChargeID,
			TargetPaymentState: payment.StatusAuthorized,
		})
		s.NoError(err)

		costBasis := alpacadecimal.NewFromFloat(0.5)
		s.Equal(payment.StatusAuthorized, updatedCharge.Realizations.ExternalPaymentSettlement.Status)
		s.Equal(float64(0), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusOpen).InexactFloat64())
		s.Equal(float64(-50), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusAuthorized).InexactFloat64())
	})

	s.Run("the customer settles the credit purchase payment", func() {
		updatedCharge, err := s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
			ChargeID:           externalCreditPurchaseChargeID,
			TargetPaymentState: payment.StatusSettled,
		})
		s.NoError(err)

		costBasis := alpacadecimal.NewFromFloat(0.5)
		s.Equal(payment.StatusSettled, updatedCharge.Realizations.ExternalPaymentSettlement.Status)
		s.Equal(float64(0), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusOpen).InexactFloat64())
	})

	var flatFeeChargeID meta.ChargeID
	promoCostBasis := alpacadecimal.Zero
	externalCostBasis := alpacadecimal.NewFromFloat(0.5)
	type flatFeeLedgerSnapshot struct {
		promoFBO             alpacadecimal.Decimal
		externalFBO          alpacadecimal.Decimal
		unknownFBO           alpacadecimal.Decimal
		promoReceivable      alpacadecimal.Decimal
		externalReceivable   alpacadecimal.Decimal
		totalOpenReceivable  alpacadecimal.Decimal
		accrued              alpacadecimal.Decimal
		authorizedReceivable alpacadecimal.Decimal
		totalWash            alpacadecimal.Decimal
		externalWash         alpacadecimal.Decimal
		earnings             alpacadecimal.Decimal
	}
	flatFeeStart := flatFeeLedgerSnapshot{
		promoFBO:             s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&promoCostBasis)),
		externalFBO:          s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&externalCostBasis)),
		unknownFBO:           s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)),
		promoReceivable:      s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&promoCostBasis), ledger.TransactionAuthorizationStatusOpen),
		externalReceivable:   s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&externalCostBasis), ledger.TransactionAuthorizationStatusOpen),
		totalOpenReceivable:  s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen),
		accrued:              s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]()),
		authorizedReceivable: s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized),
		totalWash:            s.MustWashBalance(ns, USD, mo.None[*alpacadecimal.Decimal]()),
		externalWash:         s.MustWashBalance(ns, USD, mo.Some(&externalCostBasis)),
		earnings:             s.MustEarningsBalance(ns, USD),
	}
	assertDelta := func(label string, start, delta, actual alpacadecimal.Decimal) {
		s.T().Helper()
		expected := start.Add(delta)
		s.True(actual.Equal(expected), "%s: expected start %s + delta %s = %s, got %s", label, start.String(), delta.String(), expected.String(), actual.String())
	}

	s.Run("create new upcoming charge for flat fee", func() {
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditOnlySettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromFloat(100),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					Name:              flatFeeName,
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: flatFeeName,
				}),
			},
		})
		s.NoError(err)

		s.Len(res, 1)
		s.Equal(res[0].Type(), meta.ChargeTypeFlatFee)
		flatFeeCharge, err := res[0].AsFlatFeeCharge()
		s.NoError(err)

		flatFeeChargeID = flatFeeCharge.GetChargeID()
		s.Equal(flatfee.StatusCreated, flatFeeCharge.Status)

		gatheringInvoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
			Namespaces: []string{ns},
			Customers:  []string{cust.ID},
			Currencies: []currencyx.FiatCode{currencyx.FiatCode(USD)},
			Expand:     []billing.GatheringInvoiceExpand{billing.GatheringInvoiceExpandLines},
		})
		s.NoError(err)
		s.Len(gatheringInvoices.Items, 0)

		// Credit-only flat fees bypass invoice creation and are only allocated once the charge advances at InvoiceAt,
		// so creating the charge early should leave every ledger bucket untouched.
		assertDelta("promo FBO after credit-only create", flatFeeStart.promoFBO, alpacadecimal.Zero, s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&promoCostBasis)))
		assertDelta("external FBO after credit-only create", flatFeeStart.externalFBO, alpacadecimal.Zero, s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&externalCostBasis)))
		assertDelta("unknown FBO after credit-only create", flatFeeStart.unknownFBO, alpacadecimal.Zero, s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)))
		assertDelta("authorized receivable after credit-only create", flatFeeStart.authorizedReceivable, alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized))
		assertDelta("total open receivable after credit-only create", flatFeeStart.totalOpenReceivable, alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen))
		assertDelta("accrued after credit-only create", flatFeeStart.accrued, alpacadecimal.Zero, s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]()))
		assertDelta("earnings after credit-only create", flatFeeStart.earnings, alpacadecimal.Zero, s.MustEarningsBalance(ns, USD))
	})

	clock.SetTime(servicePeriod.From)
	s.Run("advance the charge at invoice_at", func() {
		advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
			Customer: cust.GetID(),
		})
		s.NoError(err)
		s.Len(advancedCharges, 1)

		advancedFlatFee, err := advancedCharges[0].AsFlatFeeCharge()
		s.NoError(err)
		s.Equal(flatFeeChargeID.ID, advancedFlatFee.ID)
		s.Equal(flatfee.StatusFinal, advancedFlatFee.Status)
		s.Require().NotNil(advancedFlatFee.Realizations.CurrentRun)
		// We expect three realizations here: promotional credit, purchased credit, and the synthetic shortfall coverage.
		s.Len(advancedFlatFee.Realizations.CurrentRun.CreditRealizations, 3)

		fetchedCharge := s.MustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := fetchedCharge.AsFlatFeeCharge()
		s.NoError(err)
		s.Equal(flatfee.StatusFinal, updatedFlatFeeCharge.Status)
		s.Require().NotNil(updatedFlatFeeCharge.Realizations.CurrentRun)
		s.Len(updatedFlatFeeCharge.Realizations.CurrentRun.CreditRealizations, 3)

		gatheringInvoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
			Namespaces: []string{ns},
			Customers:  []string{cust.ID},
			Currencies: []currencyx.FiatCode{currencyx.FiatCode(USD)},
			Expand:     []billing.GatheringInvoiceExpand{billing.GatheringInvoiceExpandLines},
		})
		s.NoError(err)
		s.Len(gatheringInvoices.Items, 0)

		// Credit-only advancement only performs the allocation step:
		// - existing credit buckets are consumed into accrued
		// - the uncovered remainder becomes open receivable immediately
		// - authorized receivable stays empty because no payment authorization happens
		// - wash and earnings stay unchanged because this flow never enters the invoice payment lifecycle
		assertDelta("promo FBO after credit-only advance", flatFeeStart.promoFBO, alpacadecimal.NewFromInt(-30), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&promoCostBasis)))
		assertDelta("external FBO after credit-only advance", flatFeeStart.externalFBO, alpacadecimal.NewFromInt(-50), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&externalCostBasis)))
		assertDelta("unknown FBO after credit-only advance", flatFeeStart.unknownFBO, alpacadecimal.Zero, s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)))
		assertDelta("promo receivable after credit-only advance", flatFeeStart.promoReceivable, alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&promoCostBasis), ledger.TransactionAuthorizationStatusOpen))
		assertDelta("external receivable after credit-only advance", flatFeeStart.externalReceivable, alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&externalCostBasis), ledger.TransactionAuthorizationStatusOpen))
		assertDelta("total open receivable after credit-only advance", flatFeeStart.totalOpenReceivable, alpacadecimal.NewFromInt(-20), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen))
		assertDelta("authorized receivable after credit-only advance", flatFeeStart.authorizedReceivable, alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized))
		s.True(
			s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.NewFromInt(-20)),
			"the uncovered credit_only shortfall should live in the exact open advance receivable route",
		)
		s.True(
			s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(alpacadecimal.NewFromInt(20)),
			"the uncovered shortfall should also remain in unattributed accrued until a later purchase backfills it",
		)
		assertDelta("accrued after credit-only advance", flatFeeStart.accrued, alpacadecimal.NewFromInt(100), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]()))
		assertDelta("total wash after credit-only advance", flatFeeStart.totalWash, alpacadecimal.Zero, s.MustWashBalance(ns, USD, mo.None[*alpacadecimal.Decimal]()))
		assertDelta("external wash after credit-only advance", flatFeeStart.externalWash, alpacadecimal.Zero, s.MustWashBalance(ns, USD, mo.Some(&externalCostBasis)))
		assertDelta("earnings after credit-only advance", flatFeeStart.earnings, alpacadecimal.Zero, s.MustEarningsBalance(ns, USD))
		s.requireCustomerFBOSourceBalanceBuckets(cust.GetID(), ledger.RouteFilter{
			Currency: USD,
		}, map[string]float64{}) // 0 = both funded sources were consumed by the credit-only allocation.
		s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
			Currency: USD,
		}, map[string]float64{
			sourceSpendChargeBucketKey(&promoSourceChargeID, &flatFeeChargeID.ID):    30, // 30 = promotional source consumed by the flat-fee charge.
			sourceSpendChargeBucketKey(&externalSourceChargeID, &flatFeeChargeID.ID): 50, // 50 = purchased source consumed by the flat-fee charge.
			sourceSpendChargeBucketKey(nil, &flatFeeChargeID.ID):                     20, // 20 = credit-only shortfall is accrued with spend provenance and no source.
		})
		s.requireCustomerReceivableSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
			Currency:                       USD,
			CostBasis:                      mo.Some[*alpacadecimal.Decimal](nil),
			TransactionAuthorizationStatus: lo.ToPtr(ledger.TransactionAuthorizationStatusOpen),
		}, map[string]float64{
			sourceSpendChargeBucketKey(nil, &flatFeeChargeID.ID): -20, // -20 = open advance receivable for the uncovered flat-fee shortfall.
		})
	})

	s.Run("the customer later purchases credits and backfills the prior advance", func() {
		type backfillSnapshot struct {
			externalFBO            alpacadecimal.Decimal
			externalOpenReceivable alpacadecimal.Decimal
			advanceOpenReceivable  alpacadecimal.Decimal
			advanceAuthorized      alpacadecimal.Decimal
			externalAccrued        alpacadecimal.Decimal
			unattributedAccrued    alpacadecimal.Decimal
			totalAccrued           alpacadecimal.Decimal
			externalWash           alpacadecimal.Decimal
		}

		start := backfillSnapshot{
			externalFBO:            s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&externalCostBasis)),
			externalOpenReceivable: s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&externalCostBasis), ledger.TransactionAuthorizationStatusOpen),
			advanceOpenReceivable:  s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen),
			advanceAuthorized:      s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusAuthorized),
			externalAccrued:        s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&externalCostBasis)),
			unattributedAccrued:    s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)),
			totalAccrued:           s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]()),
			externalWash:           s.MustWashBalance(ns, USD, mo.Some(&externalCostBasis)),
		}

		const laterPurchaseAmount = 50
		clock.SetTime(servicePeriod.From.Add(time.Hour))

		intent := s.CreateCreditPurchaseIntent(CreateCreditPurchaseIntentInput{
			Customer: cust.GetID(),
			Currency: USD,
			Amount:   alpacadecimal.NewFromInt(laterPurchaseAmount),
			ServicePeriod: timeutil.ClosedPeriod{
				From: clock.Now(),
				To:   clock.Now(),
			},
			Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
				GenericSettlement: creditpurchase.GenericSettlement{
					Currency:  currencyx.FiatCode(USD),
					CostBasis: externalCostBasis,
				},
				InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
			}),
		})

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				intent,
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		charge, err := res[0].AsCreditPurchaseCharge()
		s.NoError(err)
		s.NotEmpty(charge.Realizations.CreditGrantRealization.TransactionGroupID)
		backfillSourceChargeID := charge.ID

		// Purchase initiation performs the whole attribution decision up front:
		// - the prior advance receivable is re-attributed into the purchased cost-basis bucket
		// - unattributed accrued is translated into the purchased cost-basis bucket
		// - only the remainder becomes newly issued purchased credit
		assertDelta("external FBO after later purchase initiation", start.externalFBO, alpacadecimal.NewFromInt(30), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&externalCostBasis)))
		s.True(
			s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&externalCostBasis), ledger.TransactionAuthorizationStatusOpen).Equal(start.externalOpenReceivable.Sub(alpacadecimal.NewFromInt(50))),
			"the purchased cost-basis open receivable should now represent the full purchase amount",
		)
		s.True(
			s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.Zero),
			"the prior advance receivable should be fully re-attributed out of the nil cost-basis bucket at initiation",
		)
		s.True(
			s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(alpacadecimal.Zero),
			"the unattributed accrued bucket should be translated immediately during attribution",
		)
		s.True(
			s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&externalCostBasis)).Equal(start.externalAccrued.Add(alpacadecimal.NewFromInt(20))),
			"the backfilled portion should already be visible in the purchased cost-basis accrued bucket after initiation",
		)

		updatedCharge, err := s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
			ChargeID:           charge.GetChargeID(),
			TargetPaymentState: payment.StatusAuthorized,
		})
		s.NoError(err)
		s.Equal(payment.StatusAuthorized, updatedCharge.Realizations.ExternalPaymentSettlement.Status)

		// Authorization only moves the purchased receivable into the authorized bucket;
		// attribution already happened during purchase initiation.
		s.True(
			s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&externalCostBasis), ledger.TransactionAuthorizationStatusAuthorized).Equal(alpacadecimal.NewFromInt(-50)),
			"the purchased amount should be visible in the exact authorized receivable route before settlement",
		)
		s.True(
			s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusAuthorized).Equal(start.advanceAuthorized),
			"the legacy advance route should still have no authorized staging",
		)

		updatedCharge, err = s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
			ChargeID:           charge.GetChargeID(),
			TargetPaymentState: payment.StatusSettled,
		})
		s.NoError(err)
		s.Equal(payment.StatusSettled, updatedCharge.Realizations.ExternalPaymentSettlement.Status)

		// Settlement is the cash movement from wash that clears the authorized receivable.
		// The earlier attribution stays intact, and the purchased receivable fully nets out here.
		s.True(
			s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.Zero),
			"the exact open advance receivable bucket should stay cleared after initiation-time attribution",
		)
		s.True(
			s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusAuthorized).Equal(alpacadecimal.Zero),
			"the exact authorized advance bucket should stay empty",
		)
		s.True(
			s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(alpacadecimal.Zero),
			"the unattributed accrued bucket should remain empty after initiation-time translation",
		)
		s.True(
			s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&externalCostBasis)).Equal(start.externalAccrued.Add(alpacadecimal.NewFromInt(20))),
			"the backfilled portion should remain attributed in the purchased cost-basis bucket",
		)
		s.True(
			s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&externalCostBasis)).Equal(start.externalFBO.Add(alpacadecimal.NewFromInt(30))),
			"only the purchase remainder should stay behind as newly available credit",
		)
		s.True(
			s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&externalCostBasis), ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.Zero),
			"the purchased cost-basis receivable should net back to zero after settlement and advance funding",
		)
		s.True(
			s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&externalCostBasis), ledger.TransactionAuthorizationStatusAuthorized).Equal(alpacadecimal.Zero),
			"the purchased authorized receivable route should be cleared by settlement",
		)
		s.True(
			s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]()).Equal(start.totalAccrued),
			"settlement should only translate accrued between buckets, not change the total accrued amount",
		)
		assertDelta("external wash after later purchase settlement", start.externalWash, alpacadecimal.NewFromInt(-50), s.MustWashBalance(ns, USD, mo.Some(&externalCostBasis)))
		s.requireCustomerFBOSourceBalanceBuckets(cust.GetID(), ledger.RouteFilter{
			Currency:  USD,
			CostBasis: mo.Some(&externalCostBasis),
		}, map[string]float64{
			sourceSpendChargeBucketKey(&backfillSourceChargeID, nil): 30, // 30 = 50 later purchase - 20 used to backfill prior advance.
		})
		s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
			Currency: USD,
		}, map[string]float64{
			sourceSpendChargeBucketKey(&promoSourceChargeID, &flatFeeChargeID.ID):    30, // 30 = original promotional source remains tied to the charge.
			sourceSpendChargeBucketKey(&externalSourceChargeID, &flatFeeChargeID.ID): 50, // 50 = original purchased source remains tied to the charge.
			sourceSpendChargeBucketKey(&backfillSourceChargeID, &flatFeeChargeID.ID): 20, // 20 = later purchase replaces the source-less advance slice.
		})
	})
}

func (s *SanitySuite) TestCreditPurchaseAdvanceAttributionAcrossTaxCodeBuckets() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("credit-purchase-multi-taxcode-advance")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	taxA, err := s.TaxCodeService.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       "txcd-40000007",
		Name:      "Advance Tax Code A",
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_40000007"},
		},
	})
	s.Require().NoError(err)

	taxB, err := s.TaxCodeService.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       "txcd-40000008",
		Name:      "Advance Tax Code B",
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_40000008"},
		},
	})
	s.Require().NoError(err)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.SetTime(servicePeriod.From.Add(-time.Hour))
	defer clock.UnFreeze()

	// given:
	// - credit-only charges create advance receivable and accrued exposure in two TaxCode buckets
	var taxASpendChargeID string
	var taxBSpendChargeID string
	for _, input := range []struct {
		name     string
		amount   int64
		taxID    string
		behavior productcatalog.TaxBehavior
	}{
		{name: "tax-a-advance", amount: 10, taxID: taxA.ID, behavior: productcatalog.InclusiveTaxBehavior},
		{name: "tax-b-advance", amount: 5, taxID: taxB.ID, behavior: productcatalog.ExclusiveTaxBehavior},
	} {
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditOnlySettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromInt(input.amount),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					Name:              input.name,
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: input.name,
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: input.taxID,
						Behavior:  lo.ToPtr(input.behavior),
					},
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		chargeID, err := res[0].GetChargeID()
		s.NoError(err)
		switch input.taxID {
		case taxA.ID:
			taxASpendChargeID = chargeID.ID
		case taxB.ID:
			taxBSpendChargeID = chargeID.ID
		}
	}

	clock.FreezeTime(servicePeriod.From)
	advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: cust.GetID(),
	})
	s.NoError(err)
	s.Len(advancedCharges, 2)

	taxABehavior := ledger.TaxBehaviorInclusive
	taxBBehavior := ledger.TaxBehaviorExclusive
	s.Equal(float64(-15), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen).InexactFloat64())
	s.Equal(float64(10), s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), mo.Some(&taxA.ID), mo.Some(&taxABehavior)).InexactFloat64())
	s.Equal(float64(5), s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), mo.Some(&taxB.ID), mo.Some(&taxBBehavior)).InexactFloat64())

	// when:
	// - a smaller purchased grant partially backfills the open advance buckets
	purchaseCostBasis := alpacadecimal.NewFromFloat(0.5)
	purchaseAt := servicePeriod.From.Add(time.Hour)
	purchaseAmount := int64(5)
	clock.FreezeTime(purchaseAt)
	purchaseRes, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.CreateCreditPurchaseIntent(CreateCreditPurchaseIntentInput{
				Customer: cust.GetID(),
				Currency: USD,
				Amount:   alpacadecimal.NewFromInt(purchaseAmount),
				ServicePeriod: timeutil.ClosedPeriod{
					From: purchaseAt,
					To:   purchaseAt,
				},
				Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
					GenericSettlement: creditpurchase.GenericSettlement{
						Currency:  currencyx.FiatCode(USD),
						CostBasis: purchaseCostBasis,
					},
					InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
				}),
				TaxConfig: productcatalog.TaxCodeConfig{
					TaxCodeID: taxA.ID,
					Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
				},
			}),
		},
	})
	s.NoError(err)
	s.Len(purchaseRes, 1)

	purchase, err := purchaseRes[0].AsCreditPurchaseCharge()
	s.NoError(err)
	sourceChargeID := purchase.ID
	backingGroup, err := s.Ledger.GetTransactionGroup(ctx, models.NamespacedID{
		Namespace: ns,
		ID:        purchase.Realizations.CreditGrantRealization.TransactionGroupID,
	})
	s.NoError(err)

	// then:
	// - partial attribution is split proportionally across TaxCode/TaxBehavior buckets
	// - the purchased grant is fully consumed by advance backfill
	templateCounts := map[string]int{}
	for _, tx := range backingGroup.Transactions() {
		code, err := ledger.TransactionTemplateCodeFromAnnotations(tx.Annotations())
		s.NoError(err)
		templateCounts[code]++
	}

	s.Equal(2, templateCounts[transactions.TemplateCode(transactions.AttributeCustomerAdvanceReceivableCostBasisTemplate{})])
	s.Equal(2, templateCounts[transactions.TemplateCode(transactions.TranslateCustomerAccruedCostBasisTemplate{})])
	s.Equal(0, templateCounts[transactions.TemplateCode(transactions.IssueCustomerReceivableTemplate{})])

	s.Equal(float64(-10), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen).InexactFloat64())
	s.Equal(float64(3.33), s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some(&purchaseCostBasis), mo.Some(&taxA.ID), mo.Some(&taxABehavior)).InexactFloat64())
	s.Equal(float64(1.67), s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some(&purchaseCostBasis), mo.Some(&taxB.ID), mo.Some(&taxBBehavior)).InexactFloat64())
	s.Equal(float64(0), s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some(&purchaseCostBasis), mo.Some(&taxA.ID), mo.Some[*ledger.TaxBehavior](nil)).InexactFloat64())
	s.Equal(float64(0), s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some(&purchaseCostBasis), mo.Some(&taxB.ID), mo.Some[*ledger.TaxBehavior](nil)).InexactFloat64())
	s.Equal(float64(6.67), s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), mo.Some(&taxA.ID), mo.Some(&taxABehavior)).InexactFloat64())
	s.Equal(float64(3.33), s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), mo.Some(&taxB.ID), mo.Some(&taxBBehavior)).InexactFloat64())
	s.Equal(float64(0), s.MustCustomerFBOBalanceWithPriority(cust.GetID(), USD, mo.Some(&purchaseCostBasis), ledger.DefaultCustomerFBOPriority).InexactFloat64())
	s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
		Currency:    USD,
		CostBasis:   mo.Some(&purchaseCostBasis),
		TaxCode:     mo.Some(&taxA.ID),
		TaxBehavior: mo.Some(&taxABehavior),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, &taxASpendChargeID): 3.33, // 3.33 = 5 purchase allocated proportionally to the 10/15 tax-A advance bucket.
	})
	s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
		Currency:    USD,
		CostBasis:   mo.Some(&purchaseCostBasis),
		TaxCode:     mo.Some(&taxB.ID),
		TaxBehavior: mo.Some(&taxBBehavior),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, &taxBSpendChargeID): 1.67, // 1.67 = 5 purchase allocated proportionally to the 5/15 tax-B advance bucket.
	})
	s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
		Currency:    USD,
		CostBasis:   mo.Some[*alpacadecimal.Decimal](nil),
		TaxCode:     mo.Some(&taxA.ID),
		TaxBehavior: mo.Some(&taxABehavior),
	}, map[string]float64{
		sourceSpendChargeBucketKey(nil, &taxASpendChargeID): 6.67, // 6.67 = original tax-A advance less the purchased attribution.
	})
	s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
		Currency:    USD,
		CostBasis:   mo.Some[*alpacadecimal.Decimal](nil),
		TaxCode:     mo.Some(&taxB.ID),
		TaxBehavior: mo.Some(&taxBBehavior),
	}, map[string]float64{
		sourceSpendChargeBucketKey(nil, &taxBSpendChargeID): 3.33, // 3.33 = original tax-B advance less the purchased attribution.
	})
}

func (s *SanitySuite) TestCreditPurchaseAdvanceAttributionClearsLegacyNilSpendFeatureBuckets() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("credit-purchase-legacy-feature-advance")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	defer apiRequestsTotal.Cleanup()
	apiRequestsRoute := mo.Some([]string{apiRequestsTotal.Feature.Key})
	unrestrictedRoute := mo.Some[[]string](nil)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.SetTime(servicePeriod.From.Add(-time.Hour))
	defer clock.UnFreeze()

	// given:
	// - two credit-only advances exist in different receivable feature routes
	// - their ledger rows are downgraded to the legacy schema shape where spend provenance was unknowable
	var unrestrictedSpendChargeID string
	var apiRequestsSpendChargeID string
	for _, input := range []struct {
		name       string
		amount     int64
		featureKey string
	}{
		{name: "legacy-unrestricted-advance", amount: 10},
		{name: "legacy-api-requests-advance", amount: 5, featureKey: apiRequestsTotal.Feature.Key},
	} {
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditOnlySettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromInt(input.amount),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					FeatureKey:        input.featureKey,
					Name:              input.name,
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: input.name,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		chargeID, err := res[0].GetChargeID()
		s.NoError(err)
		if input.featureKey == "" {
			unrestrictedSpendChargeID = chargeID.ID
		} else {
			apiRequestsSpendChargeID = chargeID.ID
		}
	}

	clock.FreezeTime(servicePeriod.From)
	advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: cust.GetID(),
	})
	s.NoError(err)
	s.Len(advancedCharges, 2)

	s.markLedgerEntriesLegacyBySpendChargeID(ctx, ns, unrestrictedSpendChargeID, apiRequestsSpendChargeID)

	s.Equal(float64(-10), s.MustCustomerReceivableBalanceForFeatures(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen, unrestrictedRoute).InexactFloat64(),
		"-10 = unrestricted legacy advance receivable before creditpurchase backfill")
	s.Equal(float64(-5), s.MustCustomerReceivableBalanceForFeatures(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen, apiRequestsRoute).InexactFloat64(),
		"-5 = feature-routed legacy advance receivable before creditpurchase backfill")

	// when:
	// - a later creditpurchase fully backfills both legacy nil-spend advance buckets
	purchaseCostBasis := alpacadecimal.NewFromFloat(0.5)
	purchaseAt := servicePeriod.From.Add(time.Hour)
	clock.FreezeTime(purchaseAt)
	purchaseRes, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.CreateCreditPurchaseIntent(CreateCreditPurchaseIntentInput{
				Customer: cust.GetID(),
				Currency: USD,
				Amount:   alpacadecimal.NewFromInt(15),
				ServicePeriod: timeutil.ClosedPeriod{
					From: purchaseAt,
					To:   purchaseAt,
				},
				Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
					GenericSettlement: creditpurchase.GenericSettlement{
						Currency:  currencyx.FiatCode(USD),
						CostBasis: purchaseCostBasis,
					},
					InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
				}),
			}),
		},
	})
	s.NoError(err)
	s.Len(purchaseRes, 1)

	purchase, err := purchaseRes[0].AsCreditPurchaseCharge()
	s.NoError(err)
	sourceChargeID := purchase.ID

	// then:
	// - both original feature routes are cleared, not just the aggregate receivable balance
	// - the new source is assigned while spend remains nil because the legacy rows never had it
	s.Equal(float64(0), s.MustCustomerReceivableBalanceForFeatures(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen, unrestrictedRoute).InexactFloat64(),
		"0 = 10 unrestricted legacy advance receivable fully attributed to the creditpurchase source")
	s.Equal(float64(0), s.MustCustomerReceivableBalanceForFeatures(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen, apiRequestsRoute).InexactFloat64(),
		"0 = 5 feature-routed legacy advance receivable fully attributed to the creditpurchase source")
	s.Equal(float64(15), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&purchaseCostBasis)).InexactFloat64(),
		"15 = 10 unrestricted + 5 feature-routed legacy accrued translated to the purchased cost basis")
	s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
		Currency:  USD,
		CostBasis: mo.Some(&purchaseCostBasis),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, nil): 15, // 15 = legacy spend provenance is unknowable, so only the new source is attributable.
	})
}

func (s *SanitySuite) markLedgerEntriesLegacyBySpendChargeID(ctx context.Context, namespace string, spendChargeIDs ...string) {
	s.T().Helper()

	for _, spendChargeID := range spendChargeIDs {
		result, err := s.DBClient.ExecContext(ctx, `
			UPDATE ledger_entries
			SET schema_version = 1,
				source_charge_id = NULL,
				spend_charge_id = NULL,
				identity_key = ''
			WHERE namespace = $1
				AND spend_charge_id = $2
		`, namespace, spendChargeID)
		s.Require().NoError(err)

		affected, err := result.RowsAffected()
		s.Require().NoError(err)
		s.Require().Positive(affected)
	}
}

func (s *SanitySuite) TestFlatFeeCreditOnlyTaxConfigFlowsToEarnings() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("flatfee-credit-taxconfig-earnings")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	tc := s.createTaxCodeForEarningsFlow(ctx, ns, "txcd-41000001", "Flat Fee Credit Tax")
	taxConfig := productcatalog.TaxCodeConfig{
		TaxCodeID: tc.ID,
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
	}

	const amount = 30
	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-31T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	// given:
	// - a charge is created before tax-configured credit is available
	createdCharges, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:       cust.GetID(),
				Currency:       USD,
				ServicePeriod:  servicePeriod,
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(amount),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Name:              "flatfee-credit-taxconfig-earnings",
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: "flatfee-credit-taxconfig-earnings",
				TaxConfig:         taxConfig,
			}),
		},
	})
	s.NoError(err)
	s.Len(createdCharges, 1)
	createdChargeID, err := createdCharges[0].GetChargeID()
	s.NoError(err)
	spendChargeID := createdChargeID.ID

	funding := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Amount:    alpacadecimal.NewFromInt(amount),
		At:        createAt,
		CostBasis: alpacadecimal.Zero,
		TaxConfig: taxConfig,
	})

	// when:
	// - the flat fee is paid with credit and its accrued amount is recognized
	clock.FreezeTime(servicePeriod.From)
	advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: cust.GetID(),
	})
	s.NoError(err)
	s.Len(advancedCharges, 1)

	advancedCharge, err := advancedCharges[0].AsFlatFeeCharge()
	s.NoError(err)
	s.Equal(flatfee.StatusFinal, advancedCharge.Status)
	s.requireChargeTaxConfig(advancedCharge.Intent.GetTaxConfig(), tc.ID, productcatalog.InclusiveTaxBehavior)
	s.Require().NotNil(advancedCharge.Realizations.CurrentRun)
	s.Len(advancedCharge.Realizations.CurrentRun.CreditRealizations, 1)

	costBasis := alpacadecimal.Zero
	ledgerTaxBehavior := ledger.TaxBehaviorInclusive
	sourceChargeID := funding.Charge.ID
	expectedTaxConfiguredAccruedAmount := float64(amount) // 30 = full flat fee is accrued on the tax-configured route.
	expectedMissingTaxBehaviorAccrued := float64(0)       // 0 = inclusive behavior must not be dropped from accrued.
	expectedAccruedSourceSpendAmount := float64(amount)   // 30 = full flat fee is funded by one promotional source.
	s.Equal(expectedTaxConfiguredAccruedAmount, s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some(&costBasis), mo.Some(&tc.ID), mo.Some(&ledgerTaxBehavior)).InexactFloat64())
	s.Equal(expectedMissingTaxBehaviorAccrued, s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some(&costBasis), mo.Some(&tc.ID), mo.Some[*ledger.TaxBehavior](nil)).InexactFloat64())
	s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
		Currency:    USD,
		CostBasis:   mo.Some(&costBasis),
		TaxCode:     mo.Some(&tc.ID),
		TaxBehavior: mo.Some(&ledgerTaxBehavior),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, &spendChargeID): expectedAccruedSourceSpendAmount,
	})

	clock.FreezeTime(servicePeriod.To)
	s.MustRecognizeRevenue(cust.GetID(), USD, alpacadecimal.NewFromInt(amount))

	// then:
	// - earnings keep the full tax route and source/spend provenance expected by the charge
	expectedTaxCodeEarningsAmount := float64(amount)       // 30 = full flat fee is recognized as earnings.
	expectedMissingTaxCodeEarnings := float64(0)           // 0 = tax code must not be dropped from earnings.
	expectedTaxConfiguredEarningsAmount := float64(amount) // 30 = inclusive tax behavior stays on recognized earnings.
	expectedMissingTaxBehaviorEarnings := float64(0)       // 0 = inclusive behavior must not be dropped from earnings.
	expectedEarningsSourceSpendAmount := float64(amount)   // 30 = recognized earnings preserve source and flat-fee spend charge.
	s.Equal(expectedTaxCodeEarningsAmount, s.MustEarningsBalanceForTaxCode(ns, USD, mo.Some(&costBasis), mo.Some(&tc.ID)).InexactFloat64())
	s.Equal(expectedMissingTaxCodeEarnings, s.MustEarningsBalanceForTaxCode(ns, USD, mo.Some(&costBasis), mo.Some[*string](nil)).InexactFloat64())
	s.Equal(expectedTaxConfiguredEarningsAmount, s.mustEarningsBalanceForTaxConfig(ns, USD, mo.Some(&costBasis), mo.Some(&tc.ID), mo.Some(&ledgerTaxBehavior)).InexactFloat64())
	s.Equal(expectedMissingTaxBehaviorEarnings, s.mustEarningsBalanceForTaxConfig(ns, USD, mo.Some(&costBasis), mo.Some(&tc.ID), mo.Some[*ledger.TaxBehavior](nil)).InexactFloat64())
	s.requireEarningsSourceSpendBalanceBuckets(ns, ledger.RouteFilter{
		Currency:    USD,
		CostBasis:   mo.Some(&costBasis),
		TaxCode:     mo.Some(&tc.ID),
		TaxBehavior: mo.Some(&ledgerTaxBehavior),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, &spendChargeID): expectedEarningsSourceSpendAmount,
	})
}

func (s *SanitySuite) TestFlatFeeCreditThenInvoiceTaxConfigFlowsToEarnings() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("flatfee-invoice-taxconfig-earnings")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	tc := s.createTaxCodeForEarningsFlow(ctx, ns, "txcd-41000002", "Flat Fee Invoice Tax")
	taxConfig := productcatalog.TaxCodeConfig{
		TaxCodeID: tc.ID,
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
	}

	const amount = 30
	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-31T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	invoiceCostBasis := alpacadecimal.NewFromInt(1)
	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	// given:
	// - a tax-configured flat-fee charge starts the invoice flow
	createdCharges, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:       cust.GetID(),
				Currency:       USD,
				ServicePeriod:  servicePeriod,
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(amount),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Name:              "flatfee-invoice-taxconfig-earnings",
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: "flatfee-invoice-taxconfig-earnings",
				TaxConfig:         taxConfig,
			}),
		},
	})
	s.NoError(err)
	s.Len(createdCharges, 1)
	chargeID, err := createdCharges[0].GetChargeID()
	s.NoError(err)

	// when:
	// - the invoice is paid and the accrued usage is recognized
	clock.FreezeTime(servicePeriod.From)
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     lo.ToPtr(servicePeriod.From),
	})
	s.NoError(err)
	s.Len(invoices, 1)
	invoice := invoices[0]

	invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
	s.NoError(err)
	s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

	invoice, err = s.BillingService.PaymentAuthorized(ctx, invoice.GetInvoiceID())
	s.NoError(err)
	s.Equal(billing.StandardInvoiceStatusPaymentProcessingAuthorized, invoice.Status)

	invoice, err = s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
		InvoiceID: invoice.GetInvoiceID(),
		Trigger:   billing.TriggerPaid,
	})
	s.NoError(err)
	s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)

	finalCharge, err := s.MustGetChargeByID(chargeID).AsFlatFeeCharge()
	s.NoError(err)
	s.Equal(flatfee.StatusFinal, finalCharge.Status)
	s.requireChargeTaxConfig(finalCharge.Intent.GetTaxConfig(), tc.ID, productcatalog.InclusiveTaxBehavior)

	ledgerTaxBehavior := ledger.TaxBehaviorInclusive
	s.Equal(float64(amount), s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some(&invoiceCostBasis), mo.Some(&tc.ID), mo.Some(&ledgerTaxBehavior)).InexactFloat64())
	s.Equal(float64(0), s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some(&invoiceCostBasis), mo.Some(&tc.ID), mo.Some[*ledger.TaxBehavior](nil)).InexactFloat64())

	clock.FreezeTime(servicePeriod.To)
	s.mustRecognizeAttributableAccrued(cust.GetID(), USD, alpacadecimal.NewFromInt(amount))

	// then:
	// - earnings keep the full tax route expected by the charge
	s.Equal(float64(amount), s.MustEarningsBalanceForTaxCode(ns, USD, mo.Some(&invoiceCostBasis), mo.Some(&tc.ID)).InexactFloat64())
	s.Equal(float64(0), s.MustEarningsBalanceForTaxCode(ns, USD, mo.Some(&invoiceCostBasis), mo.Some[*string](nil)).InexactFloat64())
	s.Equal(float64(amount), s.mustEarningsBalanceForTaxConfig(ns, USD, mo.Some(&invoiceCostBasis), mo.Some(&tc.ID), mo.Some(&ledgerTaxBehavior)).InexactFloat64())
	s.Equal(float64(0), s.mustEarningsBalanceForTaxConfig(ns, USD, mo.Some(&invoiceCostBasis), mo.Some(&tc.ID), mo.Some[*ledger.TaxBehavior](nil)).InexactFloat64())
}

func (s *SanitySuite) TestUsageBasedCreditOnlyTaxConfigFlowsToEarnings() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("usage-credit-taxconfig-earnings")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)
	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	tc := s.createTaxCodeForEarningsFlow(ctx, ns, "txcd-41000003", "Usage Credit Tax")
	taxConfig := productcatalog.TaxCodeConfig{
		TaxCodeID: tc.ID,
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
	}

	const amount = 10
	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	// given:
	// - a usage-based charge is created before credit funding and meter realization
	createdCharges, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:       cust.GetID(),
				Currency:       USD,
				ServicePeriod:  servicePeriod,
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromFloat(0.1),
				}),
				Name:              "usage-credit-taxconfig-earnings",
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: "usage-credit-taxconfig-earnings",
				FeatureKey:        apiRequestsTotal.Feature.Key,
				TaxConfig:         taxConfig,
			}),
		},
	})
	s.NoError(err)
	s.Len(createdCharges, 1)
	createdChargeID, err := createdCharges[0].GetChargeID()
	s.NoError(err)
	spendChargeID := createdChargeID.ID

	funding := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Amount:    alpacadecimal.NewFromInt(amount),
		At:        createAt,
		CostBasis: alpacadecimal.Zero,
		TaxConfig: taxConfig,
	})
	s.MockStreamingConnector.AddSimpleEvent(
		apiRequestsTotal.Feature.Key,
		100,
		datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
	)

	// when:
	// - usage is paid with credit and recognized as earnings
	clock.FreezeTime(servicePeriod.To.Add(time.Second))
	advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: cust.GetID(),
	})
	s.NoError(err)
	s.Len(advancedCharges, 1)

	advancedCharge, err := advancedCharges[0].AsUsageBasedCharge()
	s.NoError(err)
	s.Equal(usagebased.StatusActiveRealizationWaitingForCollection, advancedCharge.Status)
	s.Require().NotNil(advancedCharge.State.AdvanceAfter)

	clock.FreezeTime(advancedCharge.State.AdvanceAfter.Add(time.Second))
	advancedCharges, err = s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: cust.GetID(),
	})
	s.NoError(err)
	s.Len(advancedCharges, 1)

	advancedCharge, err = advancedCharges[0].AsUsageBasedCharge()
	s.NoError(err)
	s.Equal(usagebased.StatusFinal, advancedCharge.Status)
	s.requireChargeTaxConfig(advancedCharge.Intent.GetTaxConfig(), tc.ID, productcatalog.InclusiveTaxBehavior)
	s.Len(advancedCharge.Realizations, 1)
	s.Len(advancedCharge.Realizations[0].CreditsAllocated, 1)

	costBasis := alpacadecimal.Zero
	ledgerTaxBehavior := ledger.TaxBehaviorInclusive
	sourceChargeID := funding.Charge.ID
	expectedTaxConfiguredAccruedAmount := float64(amount) // 10 = all metered usage is accrued on the tax-configured route.
	expectedMissingTaxBehaviorAccrued := float64(0)       // 0 = inclusive behavior must not be dropped from accrued.
	expectedAccruedSourceSpendAmount := float64(amount)   // 10 = all metered usage is funded by one promotional source.
	s.Equal(expectedTaxConfiguredAccruedAmount, s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some(&costBasis), mo.Some(&tc.ID), mo.Some(&ledgerTaxBehavior)).InexactFloat64())
	s.Equal(expectedMissingTaxBehaviorAccrued, s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some(&costBasis), mo.Some(&tc.ID), mo.Some[*ledger.TaxBehavior](nil)).InexactFloat64())
	s.requireCustomerAccruedSourceSpendBalanceBuckets(cust.GetID(), ledger.RouteFilter{
		Currency:    USD,
		CostBasis:   mo.Some(&costBasis),
		TaxCode:     mo.Some(&tc.ID),
		TaxBehavior: mo.Some(&ledgerTaxBehavior),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, &spendChargeID): expectedAccruedSourceSpendAmount,
	})

	s.MustRecognizeRevenue(cust.GetID(), USD, alpacadecimal.NewFromInt(amount))

	// then:
	// - earnings keep the full tax route and source/spend provenance expected by the charge
	expectedTaxCodeEarningsAmount := float64(amount)       // 10 = all metered usage is recognized as earnings.
	expectedMissingTaxCodeEarnings := float64(0)           // 0 = tax code must not be dropped from earnings.
	expectedTaxConfiguredEarningsAmount := float64(amount) // 10 = inclusive tax behavior stays on recognized earnings.
	expectedMissingTaxBehaviorEarnings := float64(0)       // 0 = inclusive behavior must not be dropped from earnings.
	expectedEarningsSourceSpendAmount := float64(amount)   // 10 = recognized earnings preserve source and usage spend charge.
	s.Equal(expectedTaxCodeEarningsAmount, s.MustEarningsBalanceForTaxCode(ns, USD, mo.Some(&costBasis), mo.Some(&tc.ID)).InexactFloat64())
	s.Equal(expectedMissingTaxCodeEarnings, s.MustEarningsBalanceForTaxCode(ns, USD, mo.Some(&costBasis), mo.Some[*string](nil)).InexactFloat64())
	s.Equal(expectedTaxConfiguredEarningsAmount, s.mustEarningsBalanceForTaxConfig(ns, USD, mo.Some(&costBasis), mo.Some(&tc.ID), mo.Some(&ledgerTaxBehavior)).InexactFloat64())
	s.Equal(expectedMissingTaxBehaviorEarnings, s.mustEarningsBalanceForTaxConfig(ns, USD, mo.Some(&costBasis), mo.Some(&tc.ID), mo.Some[*ledger.TaxBehavior](nil)).InexactFloat64())
	s.requireEarningsSourceSpendBalanceBuckets(ns, ledger.RouteFilter{
		Currency:    USD,
		CostBasis:   mo.Some(&costBasis),
		TaxCode:     mo.Some(&tc.ID),
		TaxBehavior: mo.Some(&ledgerTaxBehavior),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&sourceChargeID, &spendChargeID): expectedEarningsSourceSpendAmount,
	})
}

func (s *SanitySuite) TestUsageBasedCreditThenInvoiceTaxConfigFlowsToEarnings() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("usage-invoice-taxconfig-earnings")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)
	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	tc := s.createTaxCodeForEarningsFlow(ctx, ns, "txcd-41000004", "Usage Invoice Tax")
	taxConfig := productcatalog.TaxCodeConfig{
		TaxCodeID: tc.ID,
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
	}

	const amount = 10
	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	invoiceCostBasis := alpacadecimal.NewFromInt(1)
	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	// given:
	// - a tax-configured usage-based charge starts the invoice flow
	s.MockStreamingConnector.AddSimpleEvent(
		apiRequestsTotal.Feature.Key,
		100,
		datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
	)
	createdCharges, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:       cust.GetID(),
				Currency:       USD,
				ServicePeriod:  servicePeriod,
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromFloat(0.1),
				}),
				Name:              "usage-invoice-taxconfig-earnings",
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: "usage-invoice-taxconfig-earnings",
				FeatureKey:        apiRequestsTotal.Feature.Key,
				TaxConfig:         taxConfig,
			}),
		},
	})
	s.NoError(err)
	s.Len(createdCharges, 1)
	chargeID, err := createdCharges[0].GetChargeID()
	s.NoError(err)

	// when:
	// - the invoice is paid and the accrued usage is recognized
	clock.FreezeTime(servicePeriod.To.Add(time.Second))
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     lo.ToPtr(servicePeriod.To),
	})
	s.NoError(err)
	s.Len(invoices, 1)
	invoice := invoices[0]

	clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())
	invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
	s.NoError(err)
	invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
	s.NoError(err)
	s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

	invoice, err = s.BillingService.PaymentAuthorized(ctx, invoice.GetInvoiceID())
	s.NoError(err)
	s.Equal(billing.StandardInvoiceStatusPaymentProcessingAuthorized, invoice.Status)

	invoice, err = s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
		InvoiceID: invoice.GetInvoiceID(),
		Trigger:   billing.TriggerPaid,
	})
	s.NoError(err)
	s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)

	finalCharge, err := s.MustGetChargeByID(chargeID).AsUsageBasedCharge()
	s.NoError(err)
	s.Equal(usagebased.StatusFinal, finalCharge.Status)
	s.requireChargeTaxConfig(finalCharge.Intent.GetTaxConfig(), tc.ID, productcatalog.InclusiveTaxBehavior)

	ledgerTaxBehavior := ledger.TaxBehaviorInclusive
	s.Equal(float64(amount), s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some(&invoiceCostBasis), mo.Some(&tc.ID), mo.Some(&ledgerTaxBehavior)).InexactFloat64())
	s.Equal(float64(0), s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some(&invoiceCostBasis), mo.Some(&tc.ID), mo.Some[*ledger.TaxBehavior](nil)).InexactFloat64())

	s.mustRecognizeAttributableAccrued(cust.GetID(), USD, alpacadecimal.NewFromInt(amount))

	// then:
	// - earnings keep the full tax route expected by the charge
	s.Equal(float64(amount), s.MustEarningsBalanceForTaxCode(ns, USD, mo.Some(&invoiceCostBasis), mo.Some(&tc.ID)).InexactFloat64())
	s.Equal(float64(0), s.MustEarningsBalanceForTaxCode(ns, USD, mo.Some(&invoiceCostBasis), mo.Some[*string](nil)).InexactFloat64())
	s.Equal(float64(amount), s.mustEarningsBalanceForTaxConfig(ns, USD, mo.Some(&invoiceCostBasis), mo.Some(&tc.ID), mo.Some(&ledgerTaxBehavior)).InexactFloat64())
	s.Equal(float64(0), s.mustEarningsBalanceForTaxConfig(ns, USD, mo.Some(&invoiceCostBasis), mo.Some(&tc.ID), mo.Some[*ledger.TaxBehavior](nil)).InexactFloat64())
}

func (s *SanitySuite) createTaxCodeForEarningsFlow(ctx context.Context, namespace string, key string, name string) taxcode.TaxCode {
	s.T().Helper()

	tc, err := s.TaxCodeService.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
		Namespace: namespace,
		Key:       key,
		Name:      name,
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: strings.Replace(key, "-", "_", 1)},
		},
	})
	s.Require().NoError(err)

	return tc
}

func (s *SanitySuite) requireChargeTaxConfig(config productcatalog.TaxCodeConfig, taxCodeID string, behavior productcatalog.TaxBehavior) {
	s.T().Helper()

	s.Require().NotEmpty(config.TaxCodeID)
	s.Equal(taxCodeID, config.TaxCodeID)
	s.Require().NotNil(config.Behavior)
	s.Equal(behavior, *config.Behavior)
}

func (s *SanitySuite) mustRecognizeAttributableAccrued(customerID customer.CustomerID, currency currencyx.Code, amount alpacadecimal.Decimal) {
	s.T().Helper()

	inputs, err := transactions.ResolveTransactions(
		s.T().Context(),
		transactions.ResolverDependencies{
			AccountService: s.LedgerResolver,
			AccountCatalog: s.LedgerAccountService,
			BalanceQuerier: s.BalanceQuerier,
		},
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  customerID.Namespace,
		},
		transactions.RecognizeEarningsFromAttributableAccruedTemplate{
			At:       clock.Now(),
			Amount:   amount,
			Currency: currency,
		},
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(inputs)

	_, err = s.Ledger.CommitGroup(s.T().Context(), transactions.GroupInputs(customerID.Namespace, nil, inputs...))
	s.Require().NoError(err)
}

func (s *SanitySuite) mustEarningsBalanceForTaxConfig(namespace string, code currencyx.Code, costBasis mo.Option[*alpacadecimal.Decimal], taxCode mo.Option[*string], taxBehavior mo.Option[*ledger.TaxBehavior]) alpacadecimal.Decimal {
	s.T().Helper()

	businessAccounts, err := s.LedgerResolver.GetBusinessAccounts(s.T().Context(), namespace)
	s.Require().NoError(err)

	balance, err := s.BalanceQuerier.GetAccountBalance(s.T().Context(), businessAccounts.EarningsAccount, ledger.RouteFilter{
		Currency:    code,
		CostBasis:   costBasis,
		TaxCode:     taxCode,
		TaxBehavior: taxBehavior,
	}, ledger.BalanceQuery{})
	s.Require().NoError(err)

	return balance
}

// TestTaxCodeFlowsFromCreditPurchaseToEarnings verifies that credits funded by a
// credit purchase can settle a tax-configured charge. Tax routing starts when
// charge usage accrues; customer FBO sources are not tax buckets.
func (s *SanitySuite) TestTaxCodeFlowsFromCreditPurchaseToEarnings() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("taxcode-earnings-flow")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	tc, err := s.TaxCodeService.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       "txcd-40000001",
		Name:      "Test Tax Code",
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_40000001"},
		},
	})
	s.Require().NoError(err)

	const amount = 30

	setupAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-31T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.SetTime(setupAt)

	s.Run("fund customer FBO with TaxCode via promotional credit", func() {
		result := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Amount:    alpacadecimal.NewFromInt(amount),
			At:        setupAt,
			CostBasis: alpacadecimal.Zero,
			TaxConfig: productcatalog.TaxCodeConfig{
				TaxCodeID: tc.ID,
				Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			},
		})
		s.NotEmpty(result.Charge.Realizations.CreditGrantRealization.TransactionGroupID)

		nilCostBasis := alpacadecimal.Zero
		s.Equal(float64(amount), s.MustCustomerFBOBalanceWithPriority(cust.GetID(), USD, mo.Some(&nilCostBasis), ledger.DefaultCustomerFBOPriority).InexactFloat64(),
			"FBO balance must equal funded amount")
	})

	var flatFeeChargeID string
	s.Run("create and advance flat fee credit-only charge", func() {
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditOnlySettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromInt(amount),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					Name:              "flat-fee-taxcode-test",
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: "flat-fee-taxcode-test",
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: tc.ID,
						Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
					},
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		chargeID, err := res[0].GetChargeID()
		s.NoError(err)
		flatFeeChargeID = chargeID.ID

		clock.FreezeTime(servicePeriod.From)

		advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
			Customer: cust.GetID(),
		})
		s.NoError(err)
		s.Len(advancedCharges, 1)

		advancedCharge, err := advancedCharges[0].AsFlatFeeCharge()
		s.NoError(err)
		s.Equal(flatfee.StatusFinal, advancedCharge.Status)
		s.Require().NotNil(advancedCharge.Realizations.CurrentRun)
		s.Len(advancedCharge.Realizations.CurrentRun.CreditRealizations, 1)

		// Accrued must hold the consumed amount; FBO must be drained
		nilCostBasis := alpacadecimal.Zero
		s.Equal(float64(0), s.MustCustomerFBOBalanceWithPriority(cust.GetID(), USD, mo.Some(&nilCostBasis), ledger.DefaultCustomerFBOPriority).InexactFloat64(),
			"FBO balance must be drained after charge collection")
		s.Equal(float64(amount), s.MustCustomerAccruedBalanceForTaxCode(cust.GetID(), USD, mo.Some(&nilCostBasis), mo.Some(&tc.ID)).InexactFloat64(),
			"accrued must hold the collected amount in the charge TaxCode bucket")
	})

	_ = flatFeeChargeID

	s.Run("recognize revenue and assert earnings land in TaxCode sub-account", func() {
		clock.FreezeTime(servicePeriod.To)

		s.MustRecognizeRevenue(cust.GetID(), USD, alpacadecimal.NewFromInt(amount))

		nilCostBasis := alpacadecimal.Zero

		// Earnings keyed by TaxCode must receive the full amount
		taxCodeEarnings := s.MustEarningsBalanceForTaxCode(ns, USD, mo.Some(&nilCostBasis), mo.Some(&tc.ID))
		s.Equal(float64(amount), taxCodeEarnings.InexactFloat64(),
			"earnings for TaxCode sub-account must equal recognized amount")

		// Nil-TaxCode earnings sub-account must remain zero
		nilTaxCodeEarnings := s.MustEarningsBalanceForTaxCode(ns, USD, mo.Some(&nilCostBasis), mo.Some[*string](nil))
		s.Equal(float64(0), nilTaxCodeEarnings.InexactFloat64(),
			"nil-TaxCode earnings sub-account must be untouched")
	})
}

// TestChargeIntentTaxConfigFlowsToEarnings verifies the credit-only charge tax
// config is applied when value leaves customer FBO and reaches accrued and
// earnings.
func (s *SanitySuite) TestChargeIntentTaxConfigFlowsToEarnings() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charge-intent-taxconfig-earnings")

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	tc, err := s.TaxCodeService.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       "txcd-40000002",
		Name:      "Charge Intent Tax Code",
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_40000002"},
		},
	})
	s.Require().NoError(err)

	const amount = 30

	setupAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-31T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.SetTime(setupAt)

	s.Run("fund customer FBO with TaxCode via promotional credit", func() {
		result := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Amount:    alpacadecimal.NewFromInt(amount),
			At:        setupAt,
			CostBasis: alpacadecimal.Zero,
			TaxConfig: productcatalog.TaxCodeConfig{
				TaxCodeID: tc.ID,
				Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			},
		})
		s.NotEmpty(result.Charge.Realizations.CreditGrantRealization.TransactionGroupID)

		nilCostBasis := alpacadecimal.Zero
		s.Equal(float64(amount), s.MustCustomerFBOBalanceWithPriority(cust.GetID(), USD, mo.Some(&nilCostBasis), ledger.DefaultCustomerFBOPriority).InexactFloat64(),
			"FBO balance must equal funded amount")
	})

	s.Run("create and advance flat-fee credit-only charge with matching TaxConfig on intent", func() {
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditOnlySettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromInt(amount),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					Name:              "flat-fee-charge-intent-tax",
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: "flat-fee-charge-intent-tax",
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: tc.ID,
						Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
					}, // tax on the charge intent
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		clock.FreezeTime(servicePeriod.From)

		advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
			Customer: cust.GetID(),
		})
		s.NoError(err)
		s.Len(advancedCharges, 1)

		advancedCharge, err := advancedCharges[0].AsFlatFeeCharge()
		s.NoError(err)
		s.Equal(flatfee.StatusFinal, advancedCharge.Status)

		nilCostBasis := alpacadecimal.Zero
		s.Equal(float64(0), s.MustCustomerFBOBalanceWithPriority(cust.GetID(), USD, mo.Some(&nilCostBasis), ledger.DefaultCustomerFBOPriority).InexactFloat64(),
			"FBO balance must be drained")
		s.Equal(float64(amount), s.MustCustomerAccruedBalanceForTaxCode(cust.GetID(), USD, mo.Some(&nilCostBasis), mo.Some(&tc.ID)).InexactFloat64(),
			"accrued must land in TaxCode bucket")
	})

	s.Run("recognize revenue and assert earnings land in TaxCode bucket", func() {
		clock.FreezeTime(servicePeriod.To)

		s.MustRecognizeRevenue(cust.GetID(), USD, alpacadecimal.NewFromInt(amount))

		nilCostBasis := alpacadecimal.Zero

		taxCodeEarnings := s.MustEarningsBalanceForTaxCode(ns, USD, mo.Some(&nilCostBasis), mo.Some(&tc.ID))
		s.Equal(float64(amount), taxCodeEarnings.InexactFloat64(),
			"earnings must land in TaxCode bucket")

		nilTaxCodeEarnings := s.MustEarningsBalanceForTaxCode(ns, USD, mo.Some(&nilCostBasis), mo.Some[*string](nil))
		s.Equal(float64(0), nilTaxCodeEarnings.InexactFloat64(),
			"nil-TaxCode earnings must remain zero")
	})
}

func (s *SanitySuite) TestChargeIntentTaxBehaviorFlowsToAdvanceAccrualCreditOnly() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charge-intent-taxbehavior-advance")

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	tc, err := s.TaxCodeService.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       "txcd-40000006",
		Name:      "Charge Intent Tax Behavior",
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_40000006"},
		},
	})
	s.Require().NoError(err)

	const amount = 30

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-31T00:00:00Z", time.UTC).AsTime()
	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:       cust.GetID(),
				Currency:       USD,
				ServicePeriod:  servicePeriod,
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(amount),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Name:              "flat-fee-charge-intent-taxbehavior",
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: "flat-fee-charge-intent-taxbehavior",
				TaxConfig: productcatalog.TaxCodeConfig{
					TaxCodeID: tc.ID,
					Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
				},
			}),
		},
	})
	s.NoError(err)
	s.Len(res, 1)

	clock.FreezeTime(servicePeriod.From)

	advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: cust.GetID(),
	})
	s.NoError(err)
	s.Len(advancedCharges, 1)

	advancedCharge, err := advancedCharges[0].AsFlatFeeCharge()
	s.NoError(err)
	s.Equal(flatfee.StatusFinal, advancedCharge.Status)
	s.Require().NotNil(advancedCharge.Intent.GetTaxConfig().Behavior)
	s.Equal(productcatalog.InclusiveTaxBehavior, *advancedCharge.Intent.GetTaxConfig().Behavior)
	s.Require().NotNil(advancedCharge.Realizations.CurrentRun)
	s.Len(advancedCharge.Realizations.CurrentRun.CreditRealizations, 1)

	ledgerTaxBehavior := ledger.TaxBehaviorInclusive
	s.Equal(float64(amount), s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), mo.Some(&tc.ID), mo.Some(&ledgerTaxBehavior)).InexactFloat64(),
		"advance-backed accrued balance must land in charge tax bucket")
	s.Equal(float64(0), s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), mo.Some(&tc.ID), mo.Some[*ledger.TaxBehavior](nil)).InexactFloat64(),
		"advance-backed accrued balance must not use nil-TaxBehavior routes")
}

// TestChargeIntentTaxConfigOverridesFundingTaxCodeCreditOnly locks the
// credit-only contract: customer FBO is not tax-routed. Charge tax config is
// applied when value accrues and is preserved into earnings.
func (s *SanitySuite) TestChargeIntentTaxConfigOverridesFundingTaxCodeCreditOnly() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charge-intent-taxconfig-contract")

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	taxA, err := s.TaxCodeService.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       "txcd-40000004",
		Name:      "Funding Tax Code A",
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_40000004"},
		},
	})
	s.Require().NoError(err)

	taxB, err := s.TaxCodeService.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       "txcd-40000005",
		Name:      "Charge Intent Tax Code B",
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_40000005"},
		},
	})
	s.Require().NoError(err)

	const amount = 30

	setupAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-31T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.SetTime(setupAt)

	s.Run("fund FBO with TaxCode=A", func() {
		s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Amount:    alpacadecimal.NewFromInt(amount),
			At:        setupAt,
			CostBasis: alpacadecimal.Zero,
			TaxConfig: productcatalog.TaxCodeConfig{
				TaxCodeID: taxA.ID,
				Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			},
		})

		nilCostBasis := alpacadecimal.Zero
		s.Equal(float64(amount), s.MustCustomerFBOBalanceWithPriority(cust.GetID(), USD, mo.Some(&nilCostBasis), ledger.DefaultCustomerFBOPriority).InexactFloat64(),
			"FBO balance must equal funded amount")
	})

	s.Run("create and advance flat-fee credit-only charge with TaxConfig=B (different from funding)", func() {
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditOnlySettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromInt(amount),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					Name:              "flat-fee-mismatched-taxconfig",
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: "flat-fee-mismatched-taxconfig",
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: taxB.ID,
						Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
					},
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		clock.FreezeTime(servicePeriod.From)

		advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
			Customer: cust.GetID(),
		})
		s.NoError(err)
		s.Len(advancedCharges, 1)

		advancedCharge, err := advancedCharges[0].AsFlatFeeCharge()
		s.NoError(err)
		s.Equal(flatfee.StatusFinal, advancedCharge.Status)

		// Charge entity preserves Intent.TaxConfig=B (metadata survives)
		s.Require().NotEmpty(advancedCharge.Intent.GetTaxConfig().TaxCodeID)
		s.Equal(taxB.ID, advancedCharge.Intent.GetTaxConfig().TaxCodeID)

		nilCostBasis := alpacadecimal.Zero
		s.Equal(float64(0), s.MustCustomerFBOBalanceWithPriority(cust.GetID(), USD, mo.Some(&nilCostBasis), ledger.DefaultCustomerFBOPriority).InexactFloat64(),
			"FBO balance must be drained")
		s.Equal(float64(0), s.MustCustomerAccruedBalanceForTaxCode(cust.GetID(), USD, mo.Some(&nilCostBasis), mo.Some(&taxA.ID)).InexactFloat64(),
			"funding TaxCode must not define accrued routing")
		s.Equal(float64(amount), s.MustCustomerAccruedBalanceForTaxCode(cust.GetID(), USD, mo.Some(&nilCostBasis), mo.Some(&taxB.ID)).InexactFloat64(),
			"charge TaxCode must define accrued routing")
	})

	s.Run("recognize revenue and confirm earnings follow charge TaxCode=B", func() {
		clock.FreezeTime(servicePeriod.To)

		s.MustRecognizeRevenue(cust.GetID(), USD, alpacadecimal.NewFromInt(amount))

		nilCostBasis := alpacadecimal.Zero

		s.Equal(float64(0), s.MustEarningsBalanceForTaxCode(ns, USD, mo.Some(&nilCostBasis), mo.Some(&taxA.ID)).InexactFloat64(),
			"funding TaxCode must not define earnings routing")
		s.Equal(float64(amount), s.MustEarningsBalanceForTaxCode(ns, USD, mo.Some(&nilCostBasis), mo.Some(&taxB.ID)).InexactFloat64(),
			"charge TaxCode must define earnings routing")
	})
}

// TestTaxCodeFlowsFromInvoicedChargeToAccrued verifies that charge.Intent.GetTaxConfig()
// flows through the credit-then-invoice path: accrual → payment authorization →
// settlement. Receivable and accrued buckets must reconcile cleanly within the
// TaxCode-keyed routes.
//
// This test covers the invoice variant GAlexIHU asked about (comment #3249148560)
// and the route-split bug flagged on chargeadapter/usagebased.go:134
// (#3249298648).
func (s *SanitySuite) TestTaxCodeFlowsFromInvoicedChargeToAccrued() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("invoice-charge-taxconfig-earnings")

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	tc, err := s.TaxCodeService.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       "txcd-40000003",
		Name:      "Invoice Tax Code",
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_40000003"},
		},
	})
	s.Require().NoError(err)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	invoiceCostBasis := alpacadecimal.NewFromInt(1)

	var invoice billing.StandardInvoice

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	s.Run("create credit-then-invoice usage-based charge with TaxConfig on intent", func() {
		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			100,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromFloat(0.1),
					}),
					Name:              "usage-based-invoice-tax",
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: "usage-based-invoice-tax",
					FeatureKey:        apiRequestsTotal.Feature.Key,
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: tc.ID,
						Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
					},
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)
	})

	s.Run("invoice is created, advanced, approved into payment pending", func() {
		clock.FreezeTime(servicePeriod.To.Add(time.Second))

		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		invoice = invoices[0]

		clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())

		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)

		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

		// Accrual leg must place balance in the charge TaxCode/TaxBehavior bucket.
		ledgerTaxBehavior := ledger.TaxBehaviorInclusive
		s.Equal(float64(10), s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some(&invoiceCostBasis), mo.Some(&tc.ID), mo.Some(&ledgerTaxBehavior)).InexactFloat64(),
			"accrued must land in TaxCode/TaxBehavior bucket for invoice-backed usage charge")
		s.Equal(float64(0), s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some(&invoiceCostBasis), mo.Some(&tc.ID), mo.Some[*ledger.TaxBehavior](nil)).InexactFloat64(),
			"nil-TaxBehavior accrued must remain zero")

		// Receivable stays tax-neutral; tax dimensions enter when usage accrues.
		s.Equal(float64(-10), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&invoiceCostBasis), ledger.TransactionAuthorizationStatusOpen).InexactFloat64(),
			"open receivable must sit in tax-neutral route")
	})

	s.Run("payment authorized and settled", func() {
		var err error
		invoice, err = s.BillingService.PaymentAuthorized(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingAuthorized, invoice.Status)

		// After auth, open receivable drains and authorized receivable holds the invoice amount.
		s.Equal(float64(0), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&invoiceCostBasis), ledger.TransactionAuthorizationStatusOpen).InexactFloat64(),
			"open receivable must reconcile to zero after auth")
		s.Equal(float64(-10), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&invoiceCostBasis), ledger.TransactionAuthorizationStatusAuthorized).InexactFloat64(),
			"authorized receivable must sit in tax-neutral route")

		invoice, err = s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: invoice.GetInvoiceID(),
			Trigger:   billing.TriggerPaid,
		})
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)

		// After settle, authorized receivable clears.
		s.Equal(float64(0), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&invoiceCostBasis), ledger.TransactionAuthorizationStatusAuthorized).InexactFloat64(),
			"authorized receivable must clear after settle")
	})

	s.Run("accrued in TaxCode bucket survives through final settlement", func() {
		ledgerTaxBehavior := ledger.TaxBehaviorInclusive
		s.Equal(float64(10), s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some(&invoiceCostBasis), mo.Some(&tc.ID), mo.Some(&ledgerTaxBehavior)).InexactFloat64(),
			"accrued must remain in TaxCode/TaxBehavior bucket after settlement")
		s.Equal(float64(0), s.MustCustomerAccruedBalanceForTaxConfig(cust.GetID(), USD, mo.Some(&invoiceCostBasis), mo.Some(&tc.ID), mo.Some[*ledger.TaxBehavior](nil)).InexactFloat64(),
			"nil-TaxBehavior accrued must remain zero")
	})
}
