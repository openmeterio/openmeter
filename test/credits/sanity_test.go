package credits

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
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
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
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
	s.assertUnfundedCreditOnlyRealization(setup.customer.GetID(), setup.amount)

	// When the original charge is deleted with refund-as-credits.
	s.deleteChargeWithRefundAsCredits(setup.ctx, setup.customer.GetID(), chargeID)

	// Then the unfunded receivable/accrued route is fully cleared.
	s.assertUnfundedCreditOnlyDeleted(setup.customer.GetID())
}

func (s *SanitySuite) TestUsageBasedCreditOnlyDeleteCorrectionSanity() {
	setup := s.setupUsageBasedCreditOnlyDeleteCorrection("charges-sanity-usagebased-credit-only-delete")

	clock.FreezeTime(setup.createAt)
	defer clock.UnFreeze()

	// Given usage occurred in the already-closed service period.
	s.recordUsageInClosedServicePeriod(setup)

	// When the credit-only usage charge is created after the service period, it finalizes immediately.
	chargeID := s.createFinalizedUsageBasedCreditOnlyCharge(setup)

	// Then the unfunded realization sits on the nil-cost-basis receivable/accrued route.
	s.assertUnfundedCreditOnlyRealization(setup.customer.GetID(), setup.amount)

	// When the original charge is deleted with refund-as-credits.
	s.deleteChargeWithRefundAsCredits(setup.ctx, setup.customer.GetID(), chargeID)

	// Then the unfunded receivable/accrued route is fully cleared.
	s.assertUnfundedCreditOnlyDeleted(setup.customer.GetID())
}

func (s *SanitySuite) TestFlatFeeFundedCreditOnlyRecognizedRevenueDeleteCorrectionSanity() {
	setup := s.setupFlatFeeCreditOnlyDeleteCorrection("charges-sanity-flatfee-funded-credit-only-recognized-delete")
	zeroCostBasis := alpacadecimal.Zero

	clock.FreezeTime(setup.createAt)
	defer clock.UnFreeze()

	// Given zero-cost-basis promotional credits fund the customer before the charge is realized.
	startOpenReceivable := s.createPromotionalCreditFunding(setup, zeroCostBasis)

	// Given a credit-only flat fee that will be corrected by deleting the charge.
	chargeID := s.createAndAdvanceFlatFeeCreditOnlyCharge(setup)

	// Then the funded credits move from FBO to accrued, without changing the grant's receivable.
	s.assertFundedCreditOnlyAccrued(setup.customer.GetID(), setup.amount, zeroCostBasis, startOpenReceivable)

	// When revenue recognition runs, the accrued funded amount is moved into earnings.
	s.recognizeFundedCreditOnlyRevenue(setup.namespace, setup.customer.GetID(), setup.amount, zeroCostBasis)

	// When the original charge is deleted with refund-as-credits.
	s.deleteChargeWithRefundAsCredits(setup.ctx, setup.customer.GetID(), chargeID)

	// Then the recognized earnings are corrected back out and the funded credits return to FBO.
	s.assertFundedRecognizedCreditOnlyDeleted(setup.namespace, setup.customer.GetID(), setup.amount, zeroCostBasis, startOpenReceivable)
}

func (s *SanitySuite) TestUsageBasedFundedCreditOnlyRecognizedRevenueDeleteCorrectionSanity() {
	setup := s.setupUsageBasedCreditOnlyDeleteCorrection("charges-sanity-usagebased-funded-credit-only-recognized-delete")
	zeroCostBasis := alpacadecimal.Zero

	clock.FreezeTime(setup.createAt)
	defer clock.UnFreeze()

	// Given zero-cost-basis promotional credits fund the customer before the charge is realized.
	startOpenReceivable := s.createPromotionalCreditFunding(setup, zeroCostBasis)

	// Given usage occurred in the already-closed service period.
	s.recordUsageInClosedServicePeriod(setup)

	// When the credit-only usage charge is created after the service period, it finalizes immediately.
	chargeID := s.createFinalizedUsageBasedCreditOnlyCharge(setup)

	// Then the funded credits move from FBO to accrued, without changing the grant's receivable.
	s.assertFundedCreditOnlyAccrued(setup.customer.GetID(), setup.amount, zeroCostBasis, startOpenReceivable)

	// When revenue recognition runs, the accrued funded amount is moved into earnings.
	s.recognizeFundedCreditOnlyRevenue(setup.namespace, setup.customer.GetID(), setup.amount, zeroCostBasis)

	// When the original charge is deleted with refund-as-credits.
	s.deleteChargeWithRefundAsCredits(setup.ctx, setup.customer.GetID(), chargeID)

	// Then the recognized earnings are corrected back out and the funded credits return to FBO.
	s.assertFundedRecognizedCreditOnlyDeleted(setup.namespace, setup.customer.GetID(), setup.amount, zeroCostBasis, startOpenReceivable)
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

func (s *SanitySuite) setupFlatFeeCreditOnlyDeleteCorrection(namespaceSuffix string) creditOnlyDeleteCorrectionSetup {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace(namespaceSuffix)

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

func (s *SanitySuite) setupUsageBasedCreditOnlyDeleteCorrection(namespaceSuffix string) creditOnlyDeleteCorrectionSetup {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace(namespaceSuffix)

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

func (s *SanitySuite) createPromotionalCreditFunding(setup creditOnlyDeleteCorrectionSetup, costBasis alpacadecimal.Decimal) alpacadecimal.Decimal {
	s.T().Helper()

	result := s.CreatePromotionalCreditFunding(setup.ctx, CreatePromotionalCreditFundingInput{
		Namespace: setup.namespace,
		Customer:  setup.customer.GetID(),
		Amount:    setup.amount,
		At:        setup.createAt,
		CostBasis: costBasis,
	})

	return result.OpenReceivable
}

func (s *SanitySuite) createAndAdvanceFlatFeeCreditOnlyCharge(setup creditOnlyDeleteCorrectionSetup) string {
	s.T().Helper()

	res, err := s.Charges.Create(setup.ctx, charges.CreateInput{
		Namespace: setup.namespace,
		Intents: charges.ChargeIntents{
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:       setup.customer.GetID(),
				Currency:       USD,
				ServicePeriod:  setup.servicePeriod,
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      setup.amount,
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Name:              setup.namespace,
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: setup.namespace,
			}),
		},
	})
	s.NoError(err)
	s.Len(res, 1)

	flatFeeChargeID, err := res[0].GetChargeID()
	s.NoError(err)

	clock.FreezeTime(setup.servicePeriod.From)

	advancedCharges, err := s.Charges.AdvanceCharges(setup.ctx, charges.AdvanceChargesInput{
		Customer: setup.customer.GetID(),
	})
	s.NoError(err)
	s.Len(advancedCharges, 1)

	advancedCharge, err := advancedCharges[0].AsFlatFeeCharge()
	s.NoError(err)
	s.Equal(flatfee.StatusFinal, advancedCharge.Status)
	s.Require().NotNil(advancedCharge.Realizations.CurrentRun)
	s.Len(advancedCharge.Realizations.CurrentRun.CreditRealizations, 1)

	return flatFeeChargeID.ID
}

func (s *SanitySuite) recordUsageInClosedServicePeriod(setup creditOnlyDeleteCorrectionSetup) {
	s.T().Helper()

	s.MockStreamingConnector.AddSimpleEvent(
		setup.featureKey,
		setup.amount.InexactFloat64(),
		datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
	)
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
			chargeID: meta.NewPatchDelete(meta.RefundAsCreditsDeletePolicy),
		},
	})
	s.NoError(err)
}

func (s *SanitySuite) assertUnfundedCreditOnlyRealization(customerID customer.CustomerID, amount alpacadecimal.Decimal) {
	s.T().Helper()

	s.True(s.MustCustomerReceivableBalance(customerID, USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).Equal(amount.Neg()))
	s.True(s.MustCustomerAccruedBalance(customerID, USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(amount))
}

func (s *SanitySuite) assertUnfundedCreditOnlyDeleted(customerID customer.CustomerID) {
	s.T().Helper()

	s.True(s.MustCustomerReceivableBalance(customerID, USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.Zero))
	s.True(s.MustCustomerAccruedBalance(customerID, USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(alpacadecimal.Zero))
	s.True(s.MustCustomerFBOBalance(customerID, USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(alpacadecimal.Zero))
}

func (s *SanitySuite) assertFundedCreditOnlyAccrued(customerID customer.CustomerID, amount alpacadecimal.Decimal, costBasis alpacadecimal.Decimal, startOpenReceivable alpacadecimal.Decimal) {
	s.T().Helper()

	s.True(s.MustCustomerFBOBalance(customerID, USD, mo.Some(&costBasis)).Equal(alpacadecimal.Zero))
	s.True(s.MustCustomerReceivableBalance(customerID, USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).Equal(startOpenReceivable))
	s.True(s.MustCustomerAccruedBalance(customerID, USD, mo.Some(&costBasis)).Equal(amount))
}

func (s *SanitySuite) recognizeFundedCreditOnlyRevenue(namespace string, customerID customer.CustomerID, amount alpacadecimal.Decimal, costBasis alpacadecimal.Decimal) {
	s.T().Helper()

	s.MustRecognizeRevenue(customerID, USD, amount)
	s.True(s.MustCustomerAccruedBalance(customerID, USD, mo.Some(&costBasis)).Equal(alpacadecimal.Zero))
	s.True(s.MustCustomerAccruedBalance(customerID, USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(alpacadecimal.Zero))
	s.True(s.MustCustomerFBOBalance(customerID, USD, mo.Some(&costBasis)).Equal(alpacadecimal.Zero))
	s.True(s.MustEarningsBalanceForCostBasis(namespace, USD, mo.Some(&costBasis)).Equal(amount))
	s.True(s.MustEarningsBalanceForCostBasis(namespace, USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(alpacadecimal.Zero))
	s.True(s.MustEarningsBalance(namespace, USD).Equal(amount))
}

func (s *SanitySuite) assertFundedRecognizedCreditOnlyDeleted(namespace string, customerID customer.CustomerID, amount alpacadecimal.Decimal, costBasis alpacadecimal.Decimal, startOpenReceivable alpacadecimal.Decimal) {
	s.T().Helper()

	s.True(s.MustCustomerReceivableBalance(customerID, USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).Equal(startOpenReceivable))
	s.True(s.MustCustomerAccruedBalance(customerID, USD, mo.Some(&costBasis)).Equal(alpacadecimal.Zero))
	s.True(s.MustCustomerAccruedBalance(customerID, USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(alpacadecimal.Zero))
	s.True(s.MustCustomerFBOBalance(customerID, USD, mo.Some(&costBasis)).Equal(amount))
	s.True(s.MustCustomerFBOBalance(customerID, USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(alpacadecimal.Zero))
	s.True(s.MustEarningsBalanceForCostBasis(namespace, USD, mo.Some(&costBasis)).Equal(alpacadecimal.Zero))
	s.True(s.MustEarningsBalanceForCostBasis(namespace, USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(alpacadecimal.Zero))
	s.True(s.MustEarningsBalance(namespace, USD).Equal(alpacadecimal.Zero))
}

func (s *SanitySuite) TestUsageBasedCreditOnlyDeleteCorrectionWithPartialBackfillSanity() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-sanity-usagebased-credit-only-delete-partial-backfill")

	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

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
				Currency:  USD,
				CostBasis: alpacadecimal.NewFromFloat(0.5),
			},
			InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
		}),
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

	// When the original charge is deleted with refund-as-credits.
	err = s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
		CustomerID: cust.GetID(),
		PatchesByChargeID: map[string]charges.Patch{
			usageBasedCharge.ID: meta.NewPatchDelete(meta.RefundAsCreditsDeletePolicy),
		},
	})
	s.NoError(err)

	// Then the purchased part is returned as available credit and the original accrued usage is cleared.
	s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).Equal(purchaseAmount.Neg()))
	s.True(s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(alpacadecimal.Zero))
	s.True(s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&costBasis)).Equal(alpacadecimal.Zero))
	s.True(s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some[*alpacadecimal.Decimal](nil)).Equal(alpacadecimal.Zero))
	s.True(s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis)).Equal(purchaseAmount))
	s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusOpen).Equal(purchaseAmount.Neg()))
}

func (s *SanitySuite) TestFlatFeeCreditThenInvoiceSanity() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-sanity-test")

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

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	setupAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-31T00:00:00Z", time.UTC).AsTime()

	clock.SetTime(setupAt)

	s.Run("the customer receives a promotional credit grant", func() {
		result := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Amount:    alpacadecimal.NewFromFloat(30),
			At:        setupAt,
			CostBasis: alpacadecimal.Zero,
		})

		// This should match the ledger's transaction group ID
		s.NotEmpty(result.Charge.Realizations.CreditGrantRealization.TransactionGroupID)

		// LEDGER:
		// - OnPromotionalCreditPurchase is called
		// - At this point the customer must have 30 USD promotional credits

		// Validate balances
		purchasedCostBasis := alpacadecimal.NewFromFloat(0.5)
		s.Equal(float64(0), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&purchasedCostBasis)).InexactFloat64())
	})

	var externalCreditPurchaseChargeID meta.ChargeID
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
					Currency:  USD,
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
		s.Equal(float64(30), promotionalCreditRealization.Amount.InexactFloat64())

		customerCreditRealization := updatedFlatFeeCharge.Realizations.CurrentRun.CreditRealizations[1]
		s.Equal(float64(50), customerCreditRealization.Amount.InexactFloat64())

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
		s.Equal(float64(20), accruedUsage.Totals.Total.InexactFloat64(), "totals should be the same as the input")
		s.Equal(float64(80), accruedUsage.Totals.CreditsTotal.InexactFloat64(), "totals should be the same as the input")

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
}

func (s *SanitySuite) TestCreditPurchasePersistsPriority() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-creditpurchase-persists-priority")

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
	s.True(s.MustCustomerFBOBalanceWithPriority(cust.GetID(), USD, mo.Some(&zeroCostBasis), priority).Equal(alpacadecimal.NewFromInt(25)))
	s.True(s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&zeroCostBasis)).Equal(alpacadecimal.Zero))
}

func (s *SanitySuite) TestUsageBasedCreditThenInvoicePaymentLifecycle() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-credits-usagebased-credit-then-invoice-payment-lifecycle")

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

	var (
		usageBasedChargeID meta.ChargeID
		invoice            billing.StandardInvoice
	)

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	s.Run("the customer receives a promotional credit grant", func() {
		s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Amount:    alpacadecimal.NewFromInt(5),
			At:        createAt,
			CostBasis: promoCostBasis,
		})
	})

	s.Run("a credit-then-invoice usage based charge is created with initial metered usage", func() {
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
		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			25,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-20T00:00:00Z", time.UTC).AsTime(),
			streamingtestutils.WithStoredAt(datetime.MustParseTimeInLocation(s.T(), "2026-02-02T12:00:00Z", time.UTC).AsTime()),
		)
	})

	s.Run("the invoice is advanced and approved into payment pending", func() {
		clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())

		var err error
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Len(invoice.Lines.OrEmpty(), 1)
		stdLine := invoice.Lines.OrEmpty()[0]
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       12.5,
			Total:        7.5,
			CreditsTotal: 5,
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
		s.Equal(float64(7.5), updatedCharge.Realizations[0].InvoiceUsage.Totals.Total.InexactFloat64())

		// Promotional grants settle immediately through wash, so only the
		// invoice-backed receivable remains open at this point.
		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&promoCostBasis), ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.Zero))
		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&invoiceCostBasis), ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.NewFromFloat(-7.5)))
		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.NewFromFloat(-7.5)))
		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized).Equal(alpacadecimal.Zero))
		s.True(s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]()).Equal(alpacadecimal.NewFromFloat(12.5)))
		s.True(s.MustWashBalance(ns, USD, mo.None[*alpacadecimal.Decimal]()).Equal(alpacadecimal.NewFromFloat(-5)))
	})

	s.Run("the payment is authorized", func() {
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

		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&promoCostBasis), ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.Zero))
		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&invoiceCostBasis), ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.Zero))
		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.Zero))
		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&invoiceCostBasis), ledger.TransactionAuthorizationStatusAuthorized).Equal(alpacadecimal.NewFromFloat(-7.5)))
		s.True(s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized).Equal(alpacadecimal.NewFromFloat(-7.5)))
		s.True(s.MustWashBalance(ns, USD, mo.None[*alpacadecimal.Decimal]()).Equal(alpacadecimal.NewFromFloat(-5)))
	})

	s.Run("the payment is settled and the charge reaches final", func() {
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
		s.True(s.MustWashBalance(ns, USD, mo.None[*alpacadecimal.Decimal]()).Equal(alpacadecimal.NewFromFloat(-12.5)))
	})
}

func (s *SanitySuite) TestFlatFeeCreditOnlySanity() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-sanity-test-credit-only")

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

	s.Run("the customer receives a promotional credit grant", func() {
		result := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Amount:    alpacadecimal.NewFromFloat(30),
			At:        setupAt,
			CostBasis: alpacadecimal.Zero,
		})
		s.NotEmpty(result.Charge.Realizations.CreditGrantRealization.TransactionGroupID)

		purchasedCostBasis := alpacadecimal.NewFromFloat(0.5)
		s.Equal(float64(0), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&purchasedCostBasis)).InexactFloat64())
	})

	var externalCreditPurchaseChargeID meta.ChargeID
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
					Currency:  USD,
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
			Currencies: []currencyx.Code{USD},
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
			Currencies: []currencyx.Code{USD},
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
					Currency:  USD,
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
	})
}

// TestTaxCodeFlowsFromCreditPurchaseToEarnings verifies the end-to-end routing of
// TaxCode: credit purchase → FBO sub-account → accrued → earnings.
// Credits funded with a TaxCode must land in a TaxCode-keyed earnings sub-account
// after charge collection and revenue recognition.
func (s *SanitySuite) TestTaxCodeFlowsFromCreditPurchaseToEarnings() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("taxcode-earnings-flow")

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
			TaxConfig: &productcatalog.TaxCodeConfig{TaxCodeID: &tc.ID},
		})
		s.NotEmpty(result.Charge.Realizations.CreditGrantRealization.TransactionGroupID)

		// FBO sub-account is keyed by TaxCode; nil-TaxCode FBO must be zero
		nilCostBasis := alpacadecimal.Zero
		s.Equal(float64(amount), s.MustCustomerFBOBalanceForTaxCode(cust.GetID(), USD, mo.Some(&nilCostBasis), mo.Some(&tc.ID)).InexactFloat64(),
			"FBO balance in TaxCode sub-account must equal funded amount")
		s.Equal(float64(0), s.MustCustomerFBOBalanceForTaxCode(cust.GetID(), USD, mo.Some(&nilCostBasis), mo.Some[*string](nil)).InexactFloat64(),
			"nil-TaxCode FBO sub-account must be untouched")
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
		s.Equal(float64(0), s.MustCustomerFBOBalanceForTaxCode(cust.GetID(), USD, mo.Some(&nilCostBasis), mo.Some(&tc.ID)).InexactFloat64(),
			"TaxCode FBO sub-account must be drained after charge collection")
		s.Equal(float64(0), s.MustCustomerFBOBalanceForTaxCode(cust.GetID(), USD, mo.Some(&nilCostBasis), mo.Some[*string](nil)).InexactFloat64(),
			"nil-TaxCode FBO sub-account must remain zero")
		s.Equal(float64(amount), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&nilCostBasis)).InexactFloat64(),
			"accrued must hold the collected amount")
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
