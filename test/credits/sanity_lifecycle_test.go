package credits

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func TestSanityLifecycleSuite(t *testing.T) {
	suite.Run(t, new(SanityLifecycleSuite))
}

type SanityLifecycleSuite struct {
	BaseSuite
}

type usageBasedPartialBackfillLifecycleState struct {
	customerID             customer.CustomerID
	usageChargeID          meta.ChargeID
	creditPurchaseChargeID meta.ChargeID
	purchaseAmount         alpacadecimal.Decimal
	costBasis              alpacadecimal.Decimal
}

func (s *SanityLifecycleSuite) TestUsageBasedCreditOnlyLifecyclePartialBackfillCorrectionThenDeleteSanity() {
	ctx := s.T().Context()
	state := s.setupUsageBasedCreditOnlyLifecyclePartialBackfillCorrection(ctx, "charges-sanity-usagebased-credit-only-lifecycle-partial-backfill-correction-delete")

	// When the now-corrected charge is deleted with refund-as-credits, the delete path has to use
	// the already-written-back lineage state rather than the original pre-correction split.
	err := s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
		CustomerID: state.customerID,
		PatchesByChargeID: map[string]charges.Patch{
			state.usageChargeID.ID: meta.NewPatchDelete(meta.RefundAsCreditsDeletePolicy),
		},
	})
	s.NoError(err)

	// Then the corrected usage is fully unwound. The only remaining open receivable is the still-unsettled
	// purchase-side obligation in the purchased cost-basis bucket.
	s.Equal(alpacadecimal.Zero, s.MustCustomerReceivableBalance(state.customerID, USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.Zero, s.MustCustomerAccruedBalance(state.customerID, USD, mo.Some[*alpacadecimal.Decimal](nil)))
	s.Equal(state.purchaseAmount.Neg(), s.MustCustomerReceivableBalance(state.customerID, USD, mo.Some(&state.costBasis), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.Zero, s.MustCustomerAccruedBalance(state.customerID, USD, mo.Some(&state.costBasis)))
	s.Equal(state.purchaseAmount, s.MustCustomerFBOBalance(state.customerID, USD, mo.Some(&state.costBasis)))
	s.Equal(alpacadecimal.Zero, s.MustCustomerFBOBalance(state.customerID, USD, mo.Some[*alpacadecimal.Decimal](nil)))

	// When we close the later credit purchase payment lifecycle too.
	s.mustSettleExternalCreditPurchase(ctx, state.creditPurchaseChargeID)

	// Then the purchased-cost-basis receivable is fully cleaned up, while the refunded purchased
	// credits stay available in FBO. The remaining nil-cost-basis receivable is also netted out here.
	s.Equal(alpacadecimal.Zero, s.MustCustomerReceivableBalance(state.customerID, USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.Zero, s.MustCustomerAccruedBalance(state.customerID, USD, mo.Some[*alpacadecimal.Decimal](nil)))
	s.Equal(alpacadecimal.Zero, s.MustCustomerReceivableBalance(state.customerID, USD, mo.Some(&state.costBasis), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.Zero, s.MustCustomerAccruedBalance(state.customerID, USD, mo.Some(&state.costBasis)))
	s.Equal(state.purchaseAmount, s.MustCustomerFBOBalance(state.customerID, USD, mo.Some(&state.costBasis)))
}

func (s *SanityLifecycleSuite) TestUsageBasedCreditOnlyLifecyclePartialBackfillCorrectionSettleBeforeDeleteSanity() {
	ctx := s.T().Context()
	state := s.setupUsageBasedCreditOnlyLifecyclePartialBackfillCorrection(ctx, "charges-sanity-usagebased-credit-only-lifecycle-partial-backfill-correction-settle-before-delete")

	// When we close the later credit purchase payment lifecycle before refunding the original charge.
	s.mustSettleExternalCreditPurchase(ctx, state.creditPurchaseChargeID)

	// Then the purchased receivable is already cleaned up, but the corrected purchased-credit-backed
	// usage is still split between accrued and available FBO.
	s.Equal(alpacadecimal.NewFromInt(-5), s.MustCustomerReceivableBalance(state.customerID, USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.NewFromInt(5), s.MustCustomerAccruedBalance(state.customerID, USD, mo.Some[*alpacadecimal.Decimal](nil)))
	s.Equal(alpacadecimal.Zero, s.MustCustomerReceivableBalance(state.customerID, USD, mo.Some(&state.costBasis), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.NewFromInt(6), s.MustCustomerAccruedBalance(state.customerID, USD, mo.Some(&state.costBasis)))
	s.Equal(alpacadecimal.NewFromInt(9), s.MustCustomerFBOBalance(state.customerID, USD, mo.Some(&state.costBasis)))

	// When the original charge is deleted with refund-as-credits afterwards.
	err := s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
		CustomerID: state.customerID,
		PatchesByChargeID: map[string]charges.Patch{
			state.usageChargeID.ID: meta.NewPatchDelete(meta.RefundAsCreditsDeletePolicy),
		},
	})
	s.NoError(err)

	// Then the end state is fully cleaned up: the purchase is settled, the corrected usage is refunded,
	// and no receivable remains open on either route.
	s.Equal(alpacadecimal.Zero, s.MustCustomerReceivableBalance(state.customerID, USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.Zero, s.MustCustomerAccruedBalance(state.customerID, USD, mo.Some[*alpacadecimal.Decimal](nil)))
	s.Equal(alpacadecimal.Zero, s.MustCustomerReceivableBalance(state.customerID, USD, mo.Some(&state.costBasis), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.Zero, s.MustCustomerAccruedBalance(state.customerID, USD, mo.Some(&state.costBasis)))
	s.Equal(state.purchaseAmount, s.MustCustomerFBOBalance(state.customerID, USD, mo.Some(&state.costBasis)))
}

func (s *SanityLifecycleSuite) TestUsageBasedCreditOnlyLifecycleTwoChargesTwoPurchasesSanity() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-sanity-usagebased-credit-only-lifecycle-two-charges-two-purchases")

	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	meterSlug := apiRequestsTotal.Feature.Key

	servicePeriodA := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	servicePeriodB := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime(),
	}
	chargeAFinalAt := datetime.MustParseTimeInLocation(s.T(), "2026-02-03T00:01:00Z", time.UTC).AsTime()
	chargeBCreateAt := datetime.MustParseTimeInLocation(s.T(), "2026-02-15T00:00:00Z", time.UTC).AsTime()
	chargeBStartFinalizationAt := datetime.MustParseTimeInLocation(s.T(), "2026-03-01T12:00:00Z", time.UTC).AsTime()
	chargeBFinalizeAt := datetime.MustParseTimeInLocation(s.T(), "2026-03-03T00:01:00Z", time.UTC).AsTime()
	purchase1Amount := alpacadecimal.NewFromInt(25)
	purchase2Amount := alpacadecimal.NewFromInt(10)
	costBasis1 := alpacadecimal.NewFromFloat(0.5)
	costBasis2 := alpacadecimal.NewFromFloat(0.8)

	clock.FreezeTime(chargeAFinalAt)
	defer clock.UnFreeze()

	// Given Charge A belongs to an older service period and is created after its collection window.
	// When it is created with 20 units at $1/unit, it finalizes immediately as advance-backed usage.
	s.MockStreamingConnector.AddSimpleEvent(
		meterSlug,
		20,
		datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
	)
	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:          cust.GetID(),
				Currency:          USD,
				ServicePeriod:     servicePeriodA,
				SettlementMode:    productcatalog.CreditOnlySettlementMode,
				Price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
				Name:              "usage-based-charge-a",
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: "usage-based-charge-a",
				FeatureKey:        meterSlug,
			}),
		},
	})
	s.NoError(err)
	s.Len(res, 1)

	chargeA, err := res[0].AsUsageBasedCharge()
	s.NoError(err)
	s.Equal(meta.ChargeStatusFinal, meta.ChargeStatus(chargeA.Status))
	s.Equal(alpacadecimal.NewFromInt(-20), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.NewFromInt(20), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)))

	// Given Charge B belongs to the next service period.
	// When it is created while that service period is already active, it starts in Active with no allocation yet.
	clock.FreezeTime(chargeBCreateAt)
	priceB := productcatalog.NewPriceFrom(productcatalog.TieredPrice{
		Mode: productcatalog.VolumeTieredPrice,
		Tiers: []productcatalog.PriceTier{
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(10)),
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromInt(2),
				},
			},
			{
				UpToAmount: nil,
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromInt(1),
				},
			},
		},
	})
	res, err = s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:          cust.GetID(),
				Currency:          USD,
				ServicePeriod:     servicePeriodB,
				SettlementMode:    productcatalog.CreditOnlySettlementMode,
				Price:             priceB,
				Name:              "usage-based-charge-b",
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: "usage-based-charge-b",
				FeatureKey:        meterSlug,
			}),
		},
	})
	s.NoError(err)
	s.Len(res, 1)

	chargeB, err := res[0].AsUsageBasedCharge()
	s.NoError(err)
	s.Equal(meta.ChargeStatusActive, meta.ChargeStatus(chargeB.Status))
	s.Equal(alpacadecimal.NewFromInt(-20), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.NewFromInt(20), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)))

	// Given Charge B records 10 units during its own service period.
	s.MockStreamingConnector.AddSimpleEvent(
		meterSlug,
		10,
		datetime.MustParseTimeInLocation(s.T(), "2026-02-20T00:00:00Z", time.UTC).AsTime(),
	)

	// When Charge B starts finalization, it allocates 20 more advance-backed credits.
	clock.FreezeTime(chargeBStartFinalizationAt)
	advancedChargeB := s.mustAdvanceUsageBasedChargeByID(ctx, cust.GetID(), chargeB.GetChargeID())
	s.Require().NotNil(advancedChargeB)
	s.Equal(usagebased.StatusActiveFinalRealizationWaitingForCollection, advancedChargeB.Status)
	s.Equal(alpacadecimal.NewFromInt(-40), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.NewFromInt(40), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)))

	// Given the first later credit purchase arrives while both charges still contribute uncovered advance.
	// When the customer buys 25 credits at cost basis 0.5, it backfills the older uncovered usage first.
	res, err = s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.CreateCreditPurchaseIntent(CreateCreditPurchaseIntentInput{
				Customer: cust.GetID(),
				Currency: USD,
				Amount:   purchase1Amount,
				ServicePeriod: timeutil.ClosedPeriod{
					From: chargeBStartFinalizationAt,
					To:   chargeBStartFinalizationAt,
				},
				Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
					GenericSettlement: creditpurchase.GenericSettlement{
						Currency:  USD,
						CostBasis: costBasis1,
					},
					InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
				}),
			}),
		},
	})
	s.NoError(err)
	s.Len(res, 1)

	purchase1Charge, err := res[0].AsCreditPurchaseCharge()
	s.NoError(err)
	s.Equal(alpacadecimal.NewFromInt(-40), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.NewFromInt(-15), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.NewFromInt(15), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)))
	s.Equal(purchase1Amount.Neg(), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis1), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(purchase1Amount, s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&costBasis1)))
	s.Equal(alpacadecimal.Zero, s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis1)))

	// Given one more unit becomes visible for Charge B before the final cutoff.
	// This reduces Charge B's priced amount from 20 down to 11, so part of Purchase 1 is released again.
	s.MockStreamingConnector.AddSimpleEvent(
		meterSlug,
		1,
		datetime.MustParseTimeInLocation(s.T(), "2026-02-21T00:00:00Z", time.UTC).AsTime(),
		streamingtestutils.WithStoredAt(datetime.MustParseTimeInLocation(s.T(), "2026-03-02T00:00:00Z", time.UTC).AsTime()),
	)

	// When Charge B finalizes, the lifecycle-driven correction should free the 5 cost-basis-backed
	// part first and only then reduce the still-uncovered remainder.
	// That 5 is the portion of Purchase 1 that had already been attributed to Charge B after
	// fully backfilling Charge A's older 20 first.
	// !!! Released purchased credit goes back to FBO here. It does not immediately snap onto
	// Charge B's or any other charge's remaining uncovered advance. Only a later purchase/initiation
	// pass will backfill uncovered advance again.
	clock.FreezeTime(chargeBFinalizeAt)
	advancedChargeB = s.mustAdvanceUsageBasedChargeByID(ctx, cust.GetID(), chargeB.GetChargeID())
	s.Require().NotNil(advancedChargeB)
	s.Equal(meta.ChargeStatusFinal, meta.ChargeStatus(advancedChargeB.Status))
	s.Equal(alpacadecimal.NewFromInt(-36), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen))
	// After the correction, Charge A still accounts for the full 20 costBasis1-backed usage,
	// while Charge B drops back to 11 uncovered usage and releases those 5 purchased credits to FBO.
	s.Equal(alpacadecimal.NewFromInt(-11), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.NewFromInt(11), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)))
	s.Equal(purchase1Amount.Neg(), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis1), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.NewFromInt(20), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&costBasis1)))
	s.Equal(alpacadecimal.NewFromInt(5), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis1)))

	// Given a second later credit purchase now sees only Charge B's remaining uncovered amount.
	// !!! The released 5 from Purchase 1 stayed as available purchased credit in FBO; it did not
	// auto-cover this remaining uncovered advance on its own.
	// When the customer buys another 10 credits at a different cost basis, it should backfill only Charge B.
	clock.FreezeTime(chargeBFinalizeAt.Add(time.Minute))
	res, err = s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.CreateCreditPurchaseIntent(CreateCreditPurchaseIntentInput{
				Customer: cust.GetID(),
				Currency: USD,
				Amount:   purchase2Amount,
				ServicePeriod: timeutil.ClosedPeriod{
					From: clock.Now(),
					To:   clock.Now(),
				},
				Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
					GenericSettlement: creditpurchase.GenericSettlement{
						Currency:  USD,
						CostBasis: costBasis2,
					},
					InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
				}),
			}),
		},
	})
	s.NoError(err)
	s.Len(res, 1)

	purchase2Charge, err := res[0].AsCreditPurchaseCharge()
	s.NoError(err)
	s.Equal(alpacadecimal.NewFromInt(-36), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.NewFromInt(-1), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.NewFromInt(1), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)))
	s.Equal(purchase1Amount.Neg(), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis1), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.NewFromInt(20), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&costBasis1)))
	s.Equal(alpacadecimal.NewFromInt(5), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis1)))
	s.Equal(purchase2Amount.Neg(), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis2), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(purchase2Amount, s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&costBasis2)))
	s.Equal(alpacadecimal.Zero, s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis2)))

	// When Charge B is refunded, only its current backing should be released.
	s.MustRefundCharge(ctx, cust.GetID(), chargeB.GetChargeID())
	s.Equal(alpacadecimal.NewFromInt(-35), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.Zero, s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)))
	s.Equal(purchase1Amount.Neg(), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis1), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.NewFromInt(20), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&costBasis1)))
	s.Equal(alpacadecimal.NewFromInt(5), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis1)))
	s.Equal(purchase2Amount.Neg(), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis2), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.Zero, s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&costBasis2)))
	s.Equal(purchase2Amount, s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis2)))

	// When both later purchases complete their payment lifecycle too.
	s.mustSettleExternalCreditPurchase(ctx, purchase1Charge.GetChargeID())
	s.mustSettleExternalCreditPurchase(ctx, purchase2Charge.GetChargeID())
	s.Equal(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.Zero, s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)))
	s.Equal(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis1), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.NewFromInt(20), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&costBasis1)))
	s.Equal(alpacadecimal.NewFromInt(5), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis1)))
	s.Equal(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis2), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.Zero, s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&costBasis2)))
	s.Equal(purchase2Amount, s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis2)))
}

// Use this helper for the shared single-charge lifecycle setup that stops after
// the later correction has already been applied.
func (s *SanityLifecycleSuite) setupUsageBasedCreditOnlyLifecyclePartialBackfillCorrection(ctx context.Context, namespacePrefix string) usageBasedPartialBackfillLifecycleState {
	ns := s.GetUniqueNamespace(namespacePrefix)

	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	meterSlug := apiRequestsTotal.Feature.Key

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	startFinalizationAt := datetime.MustParseTimeInLocation(s.T(), "2026-02-01T12:00:00Z", time.UTC).AsTime()
	finalizeAt := datetime.MustParseTimeInLocation(s.T(), "2026-02-03T00:01:00Z", time.UTC).AsTime()
	purchaseAmount := alpacadecimal.NewFromInt(15)
	costBasis := alpacadecimal.NewFromFloat(0.5)

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	price := productcatalog.NewPriceFrom(productcatalog.TieredPrice{
		Mode: productcatalog.VolumeTieredPrice,
		Tiers: []productcatalog.PriceTier{
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(10)),
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromInt(2),
				},
			},
			{
				UpToAmount: nil,
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromInt(1),
				},
			},
		},
	})

	// Given current wall clock is 2025-12-01T00:00:00Z, well before the service period.
	// When creating a credit-only usage-based charge with a tiered price.
	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:          cust.GetID(),
				Currency:          USD,
				ServicePeriod:     servicePeriod,
				SettlementMode:    productcatalog.CreditOnlySettlementMode,
				Price:             price,
				Name:              "usage-based-credit-only-lifecycle-partial-backfill-correction-delete",
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: namespacePrefix,
				FeatureKey:        meterSlug,
			}),
		},
	})
	s.NoError(err)
	s.Len(res, 1)

	usageCharge, err := res[0].AsUsageBasedCharge()
	s.NoError(err)

	// Then the first advance at service period start only moves the charge into Active.
	clock.FreezeTime(servicePeriod.From)
	advancedCharge := s.mustAdvanceSingleUsageBasedCharge(ctx, cust.GetID())
	s.Require().NotNil(advancedCharge)
	s.Equal(meta.ChargeStatusActive, meta.ChargeStatus(advancedCharge.Status))
	s.Equal(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.Zero, s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)))
	s.Equal(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.Zero, s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&costBasis)))
	s.Equal(alpacadecimal.Zero, s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis)))

	// Given the customer records 10 units during the service period.
	s.MockStreamingConnector.AddSimpleEvent(
		meterSlug,
		10,
		datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
	)

	// When we advance after the service period, the final realization starts and allocates the
	// initial 20 credits, but the charge still waits for the collection window to close.
	clock.FreezeTime(startFinalizationAt)
	advancedCharge = s.mustAdvanceSingleUsageBasedCharge(ctx, cust.GetID())
	s.Require().NotNil(advancedCharge)
	s.Equal(usagebased.StatusActiveFinalRealizationWaitingForCollection, advancedCharge.Status)
	s.Equal(alpacadecimal.NewFromInt(-20), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.NewFromInt(20), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)))
	s.Equal(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.Zero, s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&costBasis)))
	s.Equal(alpacadecimal.Zero, s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis)))

	// Given a later external credit purchase partially backfills that earlier advance-backed usage.
	res, err = s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.CreateCreditPurchaseIntent(CreateCreditPurchaseIntentInput{
				Customer: cust.GetID(),
				Currency: USD,
				Amount:   purchaseAmount,
				ServicePeriod: timeutil.ClosedPeriod{
					From: startFinalizationAt,
					To:   startFinalizationAt,
				},
				Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
					GenericSettlement: creditpurchase.GenericSettlement{
						Currency:  USD,
						CostBasis: costBasis,
					},
					InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
				}),
			}),
		},
	})
	s.NoError(err)
	s.Len(res, 1)

	creditPurchaseCharge, err := res[0].AsCreditPurchaseCharge()
	s.NoError(err)
	s.Equal(alpacadecimal.NewFromInt(-20), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.NewFromInt(-5), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.NewFromInt(5), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)))
	s.Equal(purchaseAmount.Neg(), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(purchaseAmount, s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&costBasis)))
	s.Equal(alpacadecimal.Zero, s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis)))

	// Given one more unit becomes visible before the final stored_at cutoff.
	// This shrinks the previously allocated amount from 20 down to 11 during finalization.
	s.MockStreamingConnector.AddSimpleEvent(
		meterSlug,
		1,
		datetime.MustParseTimeInLocation(s.T(), "2026-01-20T00:00:00Z", time.UTC).AsTime(),
		streamingtestutils.WithStoredAt(datetime.MustParseTimeInLocation(s.T(), "2026-02-02T00:00:00Z", time.UTC).AsTime()),
	)

	// When we advance after the collection window, the normal lifecycle issues the correction.
	clock.FreezeTime(finalizeAt)
	advancedCharge = s.mustAdvanceSingleUsageBasedCharge(ctx, cust.GetID())
	s.Require().NotNil(advancedCharge)
	s.Equal(meta.ChargeStatusFinal, meta.ChargeStatus(advancedCharge.Status))
	s.Equal(alpacadecimal.NewFromInt(-20), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.NewFromInt(-5), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.NewFromInt(5), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)))
	s.Equal(purchaseAmount.Neg(), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusOpen))
	s.Equal(alpacadecimal.NewFromInt(6), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&costBasis)))
	s.Equal(alpacadecimal.NewFromInt(9), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis)))

	return usageBasedPartialBackfillLifecycleState{
		customerID:             cust.GetID(),
		usageChargeID:          usageCharge.GetChargeID(),
		creditPurchaseChargeID: creditPurchaseCharge.GetChargeID(),
		purchaseAmount:         purchaseAmount,
		costBasis:              costBasis,
	}
}

// Use this helper when the test wants to drive a purchase through the normal
// external-payment authorized -> settled lifecycle.
func (s *SanityLifecycleSuite) mustSettleExternalCreditPurchase(ctx context.Context, chargeID meta.ChargeID) {
	s.T().Helper()

	updatedCharge, err := s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
		ChargeID:           chargeID,
		TargetPaymentState: payment.StatusAuthorized,
	})
	s.NoError(err)
	s.Equal(payment.StatusAuthorized, updatedCharge.Realizations.ExternalPaymentSettlement.Status)

	updatedCharge, err = s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
		ChargeID:           chargeID,
		TargetPaymentState: payment.StatusSettled,
	})
	s.NoError(err)
	s.Equal(payment.StatusSettled, updatedCharge.Realizations.ExternalPaymentSettlement.Status)
}

// Use this helper when one advance call may return multiple usage-based charges
// and the test cares about the transition for one specific charge.
func (s *SanityLifecycleSuite) mustAdvanceUsageBasedChargeByID(ctx context.Context, customerID customer.CustomerID, chargeID meta.ChargeID) *usagebased.Charge {
	s.T().Helper()

	advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: customerID,
	})
	s.NoError(err)

	for _, charge := range advancedCharges {
		if charge.Type() != meta.ChargeTypeUsageBased {
			continue
		}

		advancedCharge, err := charge.AsUsageBasedCharge()
		s.NoError(err)

		if advancedCharge.GetChargeID() == chargeID {
			return &advancedCharge
		}
	}

	return nil
}

// Use this helper when the test expects exactly one usage-based charge to advance.
func (s *SanityLifecycleSuite) mustAdvanceSingleUsageBasedCharge(ctx context.Context, customerID customer.CustomerID) *usagebased.Charge {
	s.T().Helper()

	advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: customerID,
	})
	s.NoError(err)

	if len(advancedCharges) == 0 {
		return nil
	}

	s.Len(advancedCharges, 1)
	s.Equal(meta.ChargeTypeUsageBased, advancedCharges[0].Type())

	advancedCharge, err := advancedCharges[0].AsUsageBasedCharge()
	s.NoError(err)

	return &advancedCharge
}
