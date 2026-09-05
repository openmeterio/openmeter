package credits

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/suite"

	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func TestCustomCurrencyCreditsSuite(t *testing.T) {
	suite.Run(t, new(CustomCurrencyCreditsSuite))
}

type CustomCurrencyCreditsSuite struct {
	BaseSuite
}

func (s *CustomCurrencyCreditsSuite) TestUsageBasedCreditOnlyAllocatesEligibleBucketsAndBackfillsAdvance() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("custom-currency-credit-only-backfill")
	defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
	customer := s.CreateLedgerBackedCustomer(ns, "test-subject")
	sandboxApp := s.InstallSandboxApp(t, ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(t, "P2D")),
		billingtest.WithManualApproval(),
	)

	feature := s.SetupApiRequestsTotalFeature(ctx, ns)
	t.Cleanup(feature.Cleanup)
	usageFeature := feature.Feature.Key
	otherFeature := "storage-gigabytes"

	tokens := s.createCustomCurrency(ns, "TOKENS")
	points := s.createCustomCurrency(ns, "POINTS")
	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	usageAt := datetime.MustParseTimeInLocation(t, "2026-01-15T00:00:00Z", time.UTC).AsTime()
	collectionAt := datetime.MustParseTimeInLocation(t, "2026-02-03T00:01:00Z", time.UTC).AsTime()
	backfillAt := collectionAt.Add(time.Minute)
	matchingPriority := 10
	unrestrictedPriority := 20
	wrongFeaturePriority := 1

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	// given:
	// - two TOKENS buckets can fund this feature (one restricted and one unrestricted)
	// - another TOKENS bucket is restricted to a different feature
	// - POINTS and USD balances are also available but are different currencies
	matchingCredit := s.createCustomCurrencyCreditPurchase(ctx, customCurrencyCreditPurchaseInput{
		Namespace:      ns,
		Customer:       customer.GetID(),
		Currency:       tokens,
		Amount:         alpacadecimal.NewFromInt(3),
		At:             setupAt,
		Name:           "matching TOKENS grant",
		Priority:       &matchingPriority,
		FeatureFilters: creditpurchase.FeatureFilters{usageFeature},
		Settlement:     creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.CreditGrantTaxCodeID,
		},
	})
	unrestrictedCredit := s.createCustomCurrencyCreditPurchase(ctx, customCurrencyCreditPurchaseInput{
		Namespace:  ns,
		Customer:   customer.GetID(),
		Currency:   tokens,
		Amount:     alpacadecimal.NewFromInt(4),
		At:         setupAt,
		Name:       "unrestricted TOKENS grant",
		Priority:   &unrestrictedPriority,
		Settlement: creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.CreditGrantTaxCodeID,
		},
	})
	wrongFeatureCredit := s.createCustomCurrencyCreditPurchase(ctx, customCurrencyCreditPurchaseInput{
		Namespace:      ns,
		Customer:       customer.GetID(),
		Currency:       tokens,
		Amount:         alpacadecimal.NewFromInt(5),
		At:             setupAt,
		Name:           "other-feature TOKENS grant",
		Priority:       &wrongFeaturePriority,
		FeatureFilters: creditpurchase.FeatureFilters{otherFeature},
		Settlement:     creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.CreditGrantTaxCodeID,
		},
	})
	pointsCredit := s.createCustomCurrencyCreditPurchase(ctx, customCurrencyCreditPurchaseInput{
		Namespace:  ns,
		Customer:   customer.GetID(),
		Currency:   points,
		Amount:     alpacadecimal.NewFromInt(7),
		At:         setupAt,
		Name:       "POINTS grant",
		Settlement: creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.CreditGrantTaxCodeID,
		},
	})
	fiatCredit := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
		Namespace: ns,
		Customer:  customer.GetID(),
		Amount:    alpacadecimal.NewFromInt(11),
		At:        setupAt,
		CostBasis: alpacadecimal.Zero,
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.CreditGrantTaxCodeID,
		},
	})

	accounts := s.mustCustomerAccounts(customer.GetID())
	s.requireAccountBalance(accounts.FBOAccount, ledger.RouteFilter{Currency: tokens.Reference()}, 12, "initial TOKENS FBO")
	s.requireAccountBalance(accounts.FBOAccount, ledger.RouteFilter{Currency: points.Reference()}, 7, "initial POINTS FBO")
	s.requireAccountBalance(accounts.FBOAccount, ledger.RouteFilter{Currency: currencies.NewCurrencyReference(USD)}, 11, "initial USD FBO")

	s.MockStreamingConnector.AddSimpleEvent(usageFeature, 12, usageAt)
	clock.FreezeTime(collectionAt)

	// when:
	// - the already-closed usage period is collected as a 12 TOKENS credit-only charge
	usageCharge := s.createCustomCurrencyUsageCharge(ctx, customCurrencyUsageChargeInput{
		Namespace:      ns,
		Customer:       customer.GetID(),
		Currency:       tokens,
		ServicePeriod:  servicePeriod,
		FeatureKey:     usageFeature,
		UnitPrice:      alpacadecimal.NewFromInt(1),
		SettlementMode: productcatalog.CreditOnlySettlementMode,
		Name:           "custom-currency credit-only usage",
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.InvoicingTaxCodeID,
		},
	})
	s.Equal(usagebased.StatusFinal, usageCharge.Status)
	s.Require().Len(usageCharge.Realizations, 1)
	s.Equal(float64(12), usageCharge.Realizations[0].CreditsAllocated.Sum().InexactFloat64())

	// then:
	// - only matching and unrestricted TOKENS are consumed
	// - the remaining 5 TOKENS become advance-backed accrued usage
	// - wrong-feature TOKENS, POINTS, and USD remain untouched
	usageChargeID := usageCharge.ID
	matchingCreditID := matchingCredit.ID
	unrestrictedCreditID := unrestrictedCredit.ID
	wrongFeatureCreditID := wrongFeatureCredit.ID
	pointsCreditID := pointsCredit.ID
	fiatCreditID := fiatCredit.Charge.ID
	s.requireCustomerAccruedSourceSpendBalanceBuckets(customer.GetID(), ledger.RouteFilter{
		Currency: tokens.Reference(),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&matchingCreditID, &usageChargeID):     3,
		sourceSpendChargeBucketKey(&unrestrictedCreditID, &usageChargeID): 4,
		sourceSpendChargeBucketKey(nil, &usageChargeID):                   5,
	})
	s.requireCustomerFBOSourceBalanceBuckets(customer.GetID(), ledger.RouteFilter{
		Currency: tokens.Reference(),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&wrongFeatureCreditID, nil): 5,
	})
	s.requireCustomerFBOSourceBalanceBuckets(customer.GetID(), ledger.RouteFilter{
		Currency: points.Reference(),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&pointsCreditID, nil): 7,
	})
	s.requireCustomerFBOSourceBalanceBuckets(customer.GetID(), ledger.RouteFilter{
		Currency: currencies.NewCurrencyReference(USD),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&fiatCreditID, nil): 11,
	})
	s.requireAccountBalance(accounts.ReceivableAccount, ledger.RouteFilter{
		Currency:                       tokens.Reference(),
		CostBasis:                      mo.Some[*alpacadecimal.Decimal](nil),
		TransactionAuthorizationStatus: lo.ToPtr(ledger.TransactionAuthorizationStatusOpen),
	}, -5, "uncovered TOKENS receivable")
	s.requireAccountBalance(accounts.AccruedAccount, ledger.RouteFilter{
		Currency:  tokens.Reference(),
		CostBasis: mo.Some[*alpacadecimal.Decimal](nil),
	}, 12, "initial TOKENS accrued")

	clock.FreezeTime(backfillAt)
	manualCostBasis := s.newManualCostBasis(alpacadecimal.NewFromFloat(0.5))

	// when:
	// - a later paid 8 TOKENS purchase restricted to the usage feature arrives
	backfillPurchase := s.createCustomCurrencyCreditPurchase(ctx, customCurrencyCreditPurchaseInput{
		Namespace:      ns,
		Customer:       customer.GetID(),
		Currency:       tokens,
		Amount:         alpacadecimal.NewFromInt(8),
		At:             backfillAt,
		Name:           "TOKENS advance backfill purchase",
		FeatureFilters: creditpurchase.FeatureFilters{usageFeature},
		Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
			InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
		}),
		CostBasis: creditpurchase.NewCostBasis(manualCostBasis),
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.CreditGrantTaxCodeID,
		},
	})
	s.Require().NotNil(backfillPurchase.Realizations.CreditGrantRealization)
	backfillPurchase = s.settleExternalCreditPurchase(ctx, backfillPurchase.GetChargeID())
	s.Equal(creditpurchase.StatusFinal, backfillPurchase.Status)

	// then:
	// - 5 purchased TOKENS reattribute the uncovered advance
	// - the other 3 purchased TOKENS remain available on the matching-feature route
	// - every ineligible bucket is still untouched
	settlementCurrency := USD
	resolvedRate := alpacadecimal.NewFromFloat(0.5)
	backfillPurchaseID := backfillPurchase.ID
	s.requireCustomerAccruedSourceSpendBalanceBuckets(customer.GetID(), ledger.RouteFilter{
		Currency: tokens.Reference(),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&matchingCreditID, &usageChargeID):     3,
		sourceSpendChargeBucketKey(&unrestrictedCreditID, &usageChargeID): 4,
		sourceSpendChargeBucketKey(&backfillPurchaseID, &usageChargeID):   5,
	})
	s.requireAccountBalance(accounts.ReceivableAccount, ledger.RouteFilter{
		Currency:                       tokens.Reference(),
		TransactionAuthorizationStatus: lo.ToPtr(ledger.TransactionAuthorizationStatusOpen),
	}, 0, "settled TOKENS receivable")
	s.requireAccountBalance(accounts.AccruedAccount, ledger.RouteFilter{
		Currency:          tokens.Reference(),
		CostBasisCurrency: mo.Some(&settlementCurrency),
		CostBasis:         mo.Some(&resolvedRate),
	}, 5, "backfilled TOKENS accrued")
	s.requireAccountBalance(accounts.FBOAccount, ledger.RouteFilter{
		Currency:          tokens.Reference(),
		CostBasisCurrency: mo.Some(&settlementCurrency),
		CostBasis:         mo.Some(&resolvedRate),
		Features:          mo.Some([]string{usageFeature}),
	}, 3, "unused purchased TOKENS")
	s.requireAccountBalance(accounts.FBOAccount, ledger.RouteFilter{
		Currency: tokens.Reference(),
		Features: mo.Some([]string{otherFeature}),
	}, 5, "wrong-feature TOKENS after backfill")
	s.requireAccountBalance(accounts.FBOAccount, ledger.RouteFilter{Currency: points.Reference()}, 7, "POINTS after backfill")
	s.requireAccountBalance(accounts.FBOAccount, ledger.RouteFilter{Currency: currencies.NewCurrencyReference(USD)}, 11, "USD after backfill")

	lineages, err := s.LineageService.LoadLineagesByCustomer(ctx, lineage.LoadLineagesByCustomerInput{
		Namespace:  ns,
		CustomerID: customer.ID,
		Currency:   tokens.Reference(),
	})
	s.Require().NoError(err)
	s.Empty(lineages, "new collections must not create lineage state")
}

func (s *CustomCurrencyCreditsSuite) TestFlatFeeCreditThenInvoiceUsesFiatCreditsAndSettlesRemainder() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("custom-currency-credit-then-invoice-fiat-coverage")
	defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
	customInvoicing := s.SetupCustomInvoicing(ns)
	customer := s.CreateLedgerBackedCustomer(ns, "test-subject")
	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(), billingtest.WithManualApproval())

	tokens := s.createCustomCurrency(ns, "TOKENS")
	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	// given:
	// - the customer owns 3 fully settled USD credits
	// - a 10 TOKENS flat fee converts to a gross 5 USD overage
	fiatCredit := s.createSettledFiatCreditPurchase(ctx, settledFiatCreditPurchaseInput{
		Namespace: ns,
		Customer:  customer.GetID(),
		Amount:    alpacadecimal.NewFromInt(3),
		At:        setupAt,
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.CreditGrantTaxCodeID,
		},
	})
	accounts := s.mustCustomerAccounts(customer.GetID())
	s.requireAccountBalance(accounts.FBOAccount, ledger.RouteFilter{Currency: currencies.NewCurrencyReference(USD)}, 3, "settled USD credit balance")

	manualCostBasis := s.newManualCostBasis(alpacadecimal.NewFromFloat(0.5))
	flatFeeCharge := s.createCustomCurrencyFlatFeeCharge(ctx, customCurrencyFlatFeeChargeInput{
		Namespace:      ns,
		Customer:       customer.GetID(),
		Currency:       tokens,
		ServicePeriod:  servicePeriod,
		InvoiceAt:      servicePeriod.From,
		Amount:         alpacadecimal.NewFromInt(10),
		PaymentTerm:    productcatalog.InAdvancePaymentTerm,
		SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
		CostBasis:      &manualCostBasis,
		Name:           "custom-currency credit-then-invoice flat fee",
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.InvoicingTaxCodeID,
		},
	})
	s.Equal(flatfee.StatusCreated, flatFeeCharge.Status)

	// when:
	// - billing collects and finalizes the line
	// - the existing USD credits settle part of the gross overage
	clock.FreezeTime(servicePeriod.From)
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customer.GetID(),
		AsOf:     lo.ToPtr(servicePeriod.From),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	invoice := invoices[0]
	s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)

	invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
	s.Require().NoError(err)
	s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
	s.requireCustomCurrencyInvoiceTotals(invoice, 5, 3, 2)

	preparedCharge, err := s.MustGetChargeByID(flatFeeCharge.GetChargeID()).AsFlatFeeCharge()
	s.Require().NoError(err)
	s.Equal(flatfee.StatusActiveAwaitingPaymentSettlement, preparedCharge.Status)
	s.Require().NotNil(preparedCharge.Realizations.CurrentRun)
	run := preparedCharge.Realizations.CurrentRun
	s.Require().NotNil(run.AccruedUsage)
	s.Require().NotNil(run.AccruedUsage.LedgerTransaction)
	// Custom-currency CTI enables settlement-fiat overage coverage by default.
	s.True(run.FiatOverageCreditAllocationCompleted)
	s.Require().Len(run.FiatOverageCreditRealizations, 1)
	s.Equal(float64(3), run.FiatOverageCreditRealizations[0].Amount.InexactFloat64())

	s.requireAccountBalance(accounts.FBOAccount, ledger.RouteFilter{Currency: tokens.Reference()}, 0, "prepared TOKENS FBO")
	s.requireAccountBalance(accounts.ReceivableAccount, ledger.RouteFilter{Currency: tokens.Reference()}, 0, "prepared TOKENS receivable")
	s.requireAccountBalance(accounts.AccruedAccount, ledger.RouteFilter{Currency: tokens.Reference()}, 10, "prepared TOKENS accrued")
	s.requireAccountBalance(accounts.FBOAccount, ledger.RouteFilter{Currency: currencies.NewCurrencyReference(USD)}, 0, "prepared USD FBO")
	s.requireAccountBalance(accounts.ReceivableAccount, ledger.RouteFilter{
		Currency:                       currencies.NewCurrencyReference(USD),
		TransactionAuthorizationStatus: lo.ToPtr(ledger.TransactionAuthorizationStatusOpen),
	}, -2, "prepared USD receivable")

	coverageGroup, err := s.Ledger.GetTransactionGroup(ctx, models.NamespacedID{
		Namespace: ns,
		ID:        run.FiatOverageCreditRealizations[0].LedgerTransaction.TransactionGroupID,
	})
	s.Require().NoError(err)
	s.Require().Len(coverageGroup.Transactions(), 1)
	for _, entry := range coverageGroup.Transactions()[0].Entries() {
		s.Require().NotNil(entry.SourceChargeID())
		s.Equal(fiatCredit.ID, *entry.SourceChargeID())
		s.Require().NotNil(entry.SpendChargeID())
		s.Equal(flatFeeCharge.ID, *entry.SpendChargeID())
	}

	// when:
	// - the payment app authorizes and settles the remaining 2 USD
	invoice, err = s.BillingService.PaymentAuthorized(ctx, invoice.GetInvoiceID())
	s.Require().NoError(err)
	s.Equal(billing.StandardInvoiceStatusPaymentProcessingAuthorized, invoice.Status)

	invoice, err = s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
		InvoiceID: invoice.GetInvoiceID(),
		Trigger:   billing.TriggerPaid,
	})
	s.Require().NoError(err)
	s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)
	invoice, err = s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: invoice.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.Require().NoError(err)
	s.requireCustomCurrencyInvoiceTotals(invoice, 5, 3, 2)

	// then:
	// - only the uncovered 2 USD is paid
	// - the gross 10 TOKENS accrual and the fiat-credit provenance remain intact
	finalCharge, err := s.MustGetChargeByID(flatFeeCharge.GetChargeID()).AsFlatFeeCharge()
	s.Require().NoError(err)
	s.Equal(flatfee.StatusFinal, finalCharge.Status)
	s.Require().NotNil(finalCharge.Realizations.CurrentRun)
	s.Require().NotNil(finalCharge.Realizations.CurrentRun.Payment)
	s.Equal(payment.StatusSettled, finalCharge.Realizations.CurrentRun.Payment.Status)
	s.Equal(float64(2), finalCharge.Realizations.CurrentRun.Payment.FiatAmount.InexactFloat64())
	s.requireAccountBalance(accounts.ReceivableAccount, ledger.RouteFilter{Currency: currencies.NewCurrencyReference(USD)}, 0, "settled USD receivable")
	s.requireAccountBalance(accounts.AccruedAccount, ledger.RouteFilter{Currency: tokens.Reference()}, 10, "settled TOKENS accrued")
}

func (s *CustomCurrencyCreditsSuite) TestUsageBasedCreditOnlyBackfillRespectsFeaturesAndLeavesPartialAdvance() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("custom-currency-credit-only-partial-feature-backfill")
	defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
	customer := s.CreateLedgerBackedCustomer(ns, "test-subject")
	sandboxApp := s.InstallSandboxApp(t, ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(t, "P2D")),
		billingtest.WithManualApproval(),
	)

	feature := s.SetupApiRequestsTotalFeature(ctx, ns)
	t.Cleanup(feature.Cleanup)
	usageFeature := feature.Feature.Key
	otherFeature := "storage-gigabytes"
	tokens := s.createCustomCurrency(ns, "TOKENS")
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	usageAt := datetime.MustParseTimeInLocation(t, "2026-01-15T00:00:00Z", time.UTC).AsTime()
	collectionAt := datetime.MustParseTimeInLocation(t, "2026-02-03T00:01:00Z", time.UTC).AsTime()

	clock.FreezeTime(collectionAt)
	defer clock.UnFreeze()

	// given:
	// - 10 TOKENS of already-closed usage has no credit backing
	s.MockStreamingConnector.AddSimpleEvent(usageFeature, 10, usageAt)
	usageCharge := s.createCustomCurrencyUsageCharge(ctx, customCurrencyUsageChargeInput{
		Namespace:      ns,
		Customer:       customer.GetID(),
		Currency:       tokens,
		ServicePeriod:  servicePeriod,
		FeatureKey:     usageFeature,
		UnitPrice:      alpacadecimal.NewFromInt(1),
		SettlementMode: productcatalog.CreditOnlySettlementMode,
		Name:           "partially backfilled custom-currency usage",
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.InvoicingTaxCodeID,
		},
	})
	s.Equal(usagebased.StatusFinal, usageCharge.Status)
	s.Require().Len(usageCharge.Realizations, 1)
	s.Equal(float64(10), usageCharge.Realizations[0].CreditsAllocated.Sum().InexactFloat64())

	accounts := s.mustCustomerAccounts(customer.GetID())
	openStatus := ledger.TransactionAuthorizationStatusOpen
	s.requireAccountBalance(accounts.ReceivableAccount, ledger.RouteFilter{
		Currency:                       tokens.Reference(),
		CostBasis:                      mo.Some[*alpacadecimal.Decimal](nil),
		TransactionAuthorizationStatus: &openStatus,
	}, -10, "initial uncovered TOKENS receivable")
	s.requireAccountBalance(accounts.AccruedAccount, ledger.RouteFilter{
		Currency:  tokens.Reference(),
		CostBasis: mo.Some[*alpacadecimal.Decimal](nil),
	}, 10, "initial advance-backed TOKENS accrued")

	lineages, err := s.LineageService.LoadLineagesByCustomer(ctx, lineage.LoadLineagesByCustomerInput{
		Namespace:  ns,
		CustomerID: customer.ID,
		Currency:   tokens.Reference(),
	})
	s.Require().NoError(err)
	s.Empty(lineages, "new collections must not create lineage state")

	// when:
	// - a paid TOKENS purchase is restricted to another feature
	clock.FreezeTime(collectionAt.Add(time.Minute))
	wrongFeatureCostBasis := s.newManualCostBasis(alpacadecimal.NewFromFloat(0.25))
	wrongFeaturePurchase := s.createCustomCurrencyCreditPurchase(ctx, customCurrencyCreditPurchaseInput{
		Namespace:      ns,
		Customer:       customer.GetID(),
		Currency:       tokens,
		Amount:         alpacadecimal.NewFromInt(4),
		At:             clock.Now(),
		Name:           "wrong-feature TOKENS purchase",
		FeatureFilters: creditpurchase.FeatureFilters{otherFeature},
		Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
			InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
		}),
		CostBasis: creditpurchase.NewCostBasis(wrongFeatureCostBasis),
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.CreditGrantTaxCodeID,
		},
	})
	wrongFeaturePurchase = s.settleExternalCreditPurchase(ctx, wrongFeaturePurchase.GetChargeID())
	s.Equal(creditpurchase.StatusFinal, wrongFeaturePurchase.Status)

	// then:
	// - the purchase remains spendable and does not change the usage advance
	wrongFeatureRate := alpacadecimal.NewFromFloat(0.25)
	settlementCurrency := USD
	s.requireAccountBalance(accounts.FBOAccount, ledger.RouteFilter{
		Currency:          tokens.Reference(),
		CostBasisCurrency: mo.Some(&settlementCurrency),
		CostBasis:         mo.Some(&wrongFeatureRate),
		Features:          mo.Some([]string{otherFeature}),
	}, 4, "wrong-feature TOKENS FBO")
	s.requireAccountBalance(accounts.ReceivableAccount, ledger.RouteFilter{
		Currency:                       tokens.Reference(),
		CostBasis:                      mo.Some[*alpacadecimal.Decimal](nil),
		TransactionAuthorizationStatus: &openStatus,
	}, -10, "uncovered TOKENS after wrong-feature purchase")

	lineages, err = s.LineageService.LoadLineagesByCustomer(ctx, lineage.LoadLineagesByCustomerInput{
		Namespace:  ns,
		CustomerID: customer.ID,
		Currency:   tokens.Reference(),
	})
	s.Require().NoError(err)
	s.Empty(lineages, "new collections must not create lineage state")

	// when:
	// - a matching paid purchase can cover only 6 of the 10 uncovered TOKENS
	clock.FreezeTime(collectionAt.Add(2 * time.Minute))
	matchingCostBasis := s.newManualCostBasis(alpacadecimal.NewFromFloat(0.5))
	matchingPurchase := s.createCustomCurrencyCreditPurchase(ctx, customCurrencyCreditPurchaseInput{
		Namespace:      ns,
		Customer:       customer.GetID(),
		Currency:       tokens,
		Amount:         alpacadecimal.NewFromInt(6),
		At:             clock.Now(),
		Name:           "partial matching TOKENS purchase",
		FeatureFilters: creditpurchase.FeatureFilters{usageFeature},
		Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
			InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
		}),
		CostBasis: creditpurchase.NewCostBasis(matchingCostBasis),
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.CreditGrantTaxCodeID,
		},
	})
	s.Require().NotNil(matchingPurchase.Realizations.CreditGrantRealization)
	matchingPurchase = s.settleExternalCreditPurchase(ctx, matchingPurchase.GetChargeID())
	s.Equal(creditpurchase.StatusFinal, matchingPurchase.Status)

	// then:
	// - 6 TOKENS carry paid-source attribution and 4 remain uncovered
	// - the unrelated 4-token purchase remains available
	matchingRate := alpacadecimal.NewFromFloat(0.5)
	matchingPurchaseID := matchingPurchase.ID
	usageChargeID := usageCharge.ID
	s.requireCustomerAccruedSourceSpendBalanceBuckets(customer.GetID(), ledger.RouteFilter{
		Currency: tokens.Reference(),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&matchingPurchaseID, &usageChargeID): 6,
		sourceSpendChargeBucketKey(nil, &usageChargeID):                 4,
	})
	s.requireAccountBalance(accounts.AccruedAccount, ledger.RouteFilter{
		Currency:          tokens.Reference(),
		CostBasisCurrency: mo.Some(&settlementCurrency),
		CostBasis:         mo.Some(&matchingRate),
	}, 6, "partially backfilled TOKENS accrued")
	s.requireAccountBalance(accounts.ReceivableAccount, ledger.RouteFilter{
		Currency:                       tokens.Reference(),
		CostBasis:                      mo.Some[*alpacadecimal.Decimal](nil),
		TransactionAuthorizationStatus: &openStatus,
	}, -4, "remaining uncovered TOKENS receivable")
	s.requireAccountBalance(accounts.FBOAccount, ledger.RouteFilter{
		Currency:          tokens.Reference(),
		CostBasisCurrency: mo.Some(&settlementCurrency),
		CostBasis:         mo.Some(&wrongFeatureRate),
		Features:          mo.Some([]string{otherFeature}),
	}, 4, "wrong-feature TOKENS after matching backfill")

	lineages, err = s.LineageService.LoadLineagesByCustomer(ctx, lineage.LoadLineagesByCustomerInput{
		Namespace:  ns,
		CustomerID: customer.ID,
		Currency:   tokens.Reference(),
	})
	s.Require().NoError(err)
	s.Empty(lineages, "new collections must not create lineage state")
}

func (s *CustomCurrencyCreditsSuite) TestFlatFeeCreditThenInvoiceAllocatesNativeCreditsBeforeSelectiveFiatCoverage() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("custom-currency-credit-then-invoice-native-and-fiat")
	defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
	customInvoicing := s.SetupCustomInvoicing(ns)
	customer := s.CreateLedgerBackedCustomer(ns, "test-subject")
	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(), billingtest.WithManualApproval())

	feature := s.SetupApiRequestsTotalFeature(ctx, ns)
	t.Cleanup(feature.Cleanup)
	chargeFeature := feature.Feature.Key
	otherFeature := "storage-gigabytes"
	tokens := s.createCustomCurrency(ns, "TOKENS")
	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	matchingPriority := 10
	wrongFeaturePriority := 1

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	// given:
	// - 4 matching and 2 wrong-feature TOKENS are available
	// - 1 matching and 2 wrong-feature USD credits are available
	matchingCustomCredit := s.createCustomCurrencyCreditPurchase(ctx, customCurrencyCreditPurchaseInput{
		Namespace:      ns,
		Customer:       customer.GetID(),
		Currency:       tokens,
		Amount:         alpacadecimal.NewFromInt(4),
		At:             setupAt,
		Name:           "matching TOKENS grant for CTI",
		Priority:       &matchingPriority,
		FeatureFilters: creditpurchase.FeatureFilters{chargeFeature},
		Settlement:     creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.CreditGrantTaxCodeID,
		},
	})
	wrongCustomCredit := s.createCustomCurrencyCreditPurchase(ctx, customCurrencyCreditPurchaseInput{
		Namespace:      ns,
		Customer:       customer.GetID(),
		Currency:       tokens,
		Amount:         alpacadecimal.NewFromInt(2),
		At:             setupAt,
		Name:           "wrong-feature TOKENS grant for CTI",
		Priority:       &wrongFeaturePriority,
		FeatureFilters: creditpurchase.FeatureFilters{otherFeature},
		Settlement:     creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.CreditGrantTaxCodeID,
		},
	})
	matchingFiatCredit := s.createSettledFiatCreditPurchase(ctx, settledFiatCreditPurchaseInput{
		Namespace:      ns,
		Customer:       customer.GetID(),
		Amount:         alpacadecimal.NewFromInt(1),
		At:             setupAt,
		Priority:       &matchingPriority,
		FeatureFilters: creditpurchase.FeatureFilters{chargeFeature},
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.CreditGrantTaxCodeID,
		},
	})
	wrongFiatCredit := s.createSettledFiatCreditPurchase(ctx, settledFiatCreditPurchaseInput{
		Namespace:      ns,
		Customer:       customer.GetID(),
		Amount:         alpacadecimal.NewFromInt(2),
		At:             setupAt,
		Priority:       &wrongFeaturePriority,
		FeatureFilters: creditpurchase.FeatureFilters{otherFeature},
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.CreditGrantTaxCodeID,
		},
	})

	accounts := s.mustCustomerAccounts(customer.GetID())
	s.requireAccountBalance(accounts.FBOAccount, ledger.RouteFilter{Currency: tokens.Reference()}, 6, "initial feature-scoped TOKENS")
	s.requireAccountBalance(accounts.FBOAccount, ledger.RouteFilter{Currency: currencies.NewCurrencyReference(USD)}, 3, "initial feature-scoped USD")

	manualCostBasis := s.newManualCostBasis(alpacadecimal.NewFromFloat(0.5))
	flatFeeCharge := s.createCustomCurrencyFlatFeeCharge(ctx, customCurrencyFlatFeeChargeInput{
		Namespace:      ns,
		Customer:       customer.GetID(),
		Currency:       tokens,
		ServicePeriod:  servicePeriod,
		InvoiceAt:      servicePeriod.From,
		Amount:         alpacadecimal.NewFromInt(10),
		PaymentTerm:    productcatalog.InAdvancePaymentTerm,
		SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
		FeatureKey:     &chargeFeature,
		CostBasis:      &manualCostBasis,
		Name:           "custom-currency CTI native and fiat allocation",
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.InvoicingTaxCodeID,
		},
	})
	s.Equal(flatfee.StatusCreated, flatFeeCharge.Status)

	// when:
	// - invoice finalization first consumes 4 native TOKENS
	// - the remaining 6 TOKENS convert to 3 USD and only 1 matching USD credit applies
	clock.FreezeTime(servicePeriod.From)
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customer.GetID(),
		AsOf:     lo.ToPtr(servicePeriod.From),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	invoice := invoices[0]
	s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)

	invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
	s.Require().NoError(err)
	s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
	s.requireCustomCurrencyInvoiceTotals(invoice, 3, 1, 2)

	preparedCharge, err := s.MustGetChargeByID(flatFeeCharge.GetChargeID()).AsFlatFeeCharge()
	s.Require().NoError(err)
	s.Equal(flatfee.StatusActiveAwaitingPaymentSettlement, preparedCharge.Status)
	s.Require().NotNil(preparedCharge.Realizations.CurrentRun)
	run := preparedCharge.Realizations.CurrentRun
	s.Require().Len(run.CreditRealizations, 1)
	s.Equal(float64(4), run.CreditRealizations.Sum().InexactFloat64())
	// Custom-currency CTI enables settlement-fiat overage coverage by default.
	s.True(run.FiatOverageCreditAllocationCompleted)
	s.Require().Len(run.FiatOverageCreditRealizations, 1)
	s.Equal(float64(1), run.FiatOverageCreditRealizations.Sum().InexactFloat64())

	matchingCustomCreditID := matchingCustomCredit.ID
	wrongCustomCreditID := wrongCustomCredit.ID
	matchingFiatCreditID := matchingFiatCredit.ID
	wrongFiatCreditID := wrongFiatCredit.ID
	flatFeeChargeID := flatFeeCharge.ID
	s.requireCustomerAccruedSourceSpendBalanceBuckets(customer.GetID(), ledger.RouteFilter{
		Currency: tokens.Reference(),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&matchingCustomCreditID, &flatFeeChargeID): 4,
		sourceSpendChargeBucketKey(&flatFeeChargeID, &flatFeeChargeID):        6,
	})
	s.requireCustomerFBOSourceBalanceBuckets(customer.GetID(), ledger.RouteFilter{
		Currency: tokens.Reference(),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&wrongCustomCreditID, nil): 2,
	})
	s.requireCustomerFBOSourceBalanceBuckets(customer.GetID(), ledger.RouteFilter{
		Currency: currencies.NewCurrencyReference(USD),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&wrongFiatCreditID, nil): 2,
	})
	s.requireAccountBalance(accounts.ReceivableAccount, ledger.RouteFilter{
		Currency:                       currencies.NewCurrencyReference(USD),
		TransactionAuthorizationStatus: lo.ToPtr(ledger.TransactionAuthorizationStatusOpen),
	}, -2, "selectively covered USD receivable")

	coverageGroup, err := s.Ledger.GetTransactionGroup(ctx, models.NamespacedID{
		Namespace: ns,
		ID:        run.FiatOverageCreditRealizations[0].LedgerTransaction.TransactionGroupID,
	})
	s.Require().NoError(err)
	s.Require().Len(coverageGroup.Transactions(), 1)
	for _, entry := range coverageGroup.Transactions()[0].Entries() {
		s.Require().NotNil(entry.SourceChargeID())
		s.Equal(matchingFiatCreditID, *entry.SourceChargeID())
		s.Require().NotNil(entry.SpendChargeID())
		s.Equal(flatFeeChargeID, *entry.SpendChargeID())
	}

	// when:
	// - the remaining 2 USD is authorized and settled
	invoice, err = s.BillingService.PaymentAuthorized(ctx, invoice.GetInvoiceID())
	s.Require().NoError(err)
	invoice, err = s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
		InvoiceID: invoice.GetInvoiceID(),
		Trigger:   billing.TriggerPaid,
	})
	s.Require().NoError(err)
	s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)

	// then:
	// - the charge is final and the two wrong-feature sources remain untouched
	finalCharge, err := s.MustGetChargeByID(flatFeeCharge.GetChargeID()).AsFlatFeeCharge()
	s.Require().NoError(err)
	s.Equal(flatfee.StatusFinal, finalCharge.Status)
	s.Require().NotNil(finalCharge.Realizations.CurrentRun)
	s.Require().NotNil(finalCharge.Realizations.CurrentRun.Payment)
	s.Equal(float64(2), finalCharge.Realizations.CurrentRun.Payment.FiatAmount.InexactFloat64())
	s.requireCustomerFBOSourceBalanceBuckets(customer.GetID(), ledger.RouteFilter{
		Currency: tokens.Reference(),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&wrongCustomCreditID, nil): 2,
	})
	s.requireCustomerFBOSourceBalanceBuckets(customer.GetID(), ledger.RouteFilter{
		Currency: currencies.NewCurrencyReference(USD),
	}, map[string]float64{
		sourceSpendChargeBucketKey(&wrongFiatCreditID, nil): 2,
	})
}

type customCurrencyUsageChargeInput struct {
	Namespace      string
	Customer       customer.CustomerID
	Currency       currencies.Currency
	ServicePeriod  timeutil.ClosedPeriod
	FeatureKey     string
	UnitPrice      alpacadecimal.Decimal
	SettlementMode productcatalog.SettlementMode
	CostBasis      *costbasis.Intent
	Name           string
	TaxConfig      productcatalog.TaxCodeConfig
}

func (i customCurrencyUsageChargeInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}
	if err := i.Customer.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}
	if i.Customer.Namespace != i.Namespace {
		errs = append(errs, errors.New("customer namespace must match input namespace"))
	}
	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}
	if !i.Currency.IsCustom() {
		errs = append(errs, errors.New("currency must be custom"))
	}
	if err := i.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}
	if i.FeatureKey == "" {
		errs = append(errs, errors.New("feature key is required"))
	}
	if !i.UnitPrice.IsPositive() {
		errs = append(errs, errors.New("unit price must be positive"))
	}
	if err := i.SettlementMode.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("settlement mode: %w", err))
	}
	if i.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}
	if err := i.TaxConfig.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("tax config: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (s *CustomCurrencyCreditsSuite) createCustomCurrencyUsageCharge(ctx context.Context, input customCurrencyUsageChargeInput) usagebased.Charge {
	s.T().Helper()
	s.Require().NoError(input.Validate())

	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: input.Namespace,
		Intents: charges.ChargeIntents{
			charges.NewChargeIntent(usagebased.Intent{
				Intent: meta.Intent{
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: lo.ToPtr(input.Name),
					CustomerID:        input.Customer.ID,
					Currency:          input.Currency,
					TaxConfig:         input.TaxConfig,
				},
				IntentMutableFields: usagebased.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              input.Name,
						ServicePeriod:     input.ServicePeriod,
						FullServicePeriod: input.ServicePeriod,
						BillingPeriod:     input.ServicePeriod,
					},
					InvoiceAt: input.ServicePeriod.To,
					Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: input.UnitPrice,
					}),
				},
				SettlementMode: input.SettlementMode,
				FeatureKey:     input.FeatureKey,
				CostBasis:      input.CostBasis,
			}),
		},
	})
	s.Require().NoError(err)
	s.Require().Len(created, 1)
	charge, err := created[0].AsUsageBasedCharge()
	s.Require().NoError(err)

	return charge
}

type customCurrencyFlatFeeChargeInput struct {
	Namespace      string
	Customer       customer.CustomerID
	Currency       currencies.Currency
	ServicePeriod  timeutil.ClosedPeriod
	InvoiceAt      time.Time
	Amount         alpacadecimal.Decimal
	PaymentTerm    productcatalog.PaymentTermType
	SettlementMode productcatalog.SettlementMode
	FeatureKey     *string
	CostBasis      *costbasis.Intent
	Name           string
	TaxConfig      productcatalog.TaxCodeConfig
}

func (i customCurrencyFlatFeeChargeInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}
	if err := i.Customer.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}
	if i.Customer.Namespace != i.Namespace {
		errs = append(errs, errors.New("customer namespace must match input namespace"))
	}
	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}
	if !i.Currency.IsCustom() {
		errs = append(errs, errors.New("currency must be custom"))
	}
	if err := i.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}
	if i.InvoiceAt.IsZero() {
		errs = append(errs, errors.New("invoice at is required"))
	}
	if !i.Amount.IsPositive() {
		errs = append(errs, errors.New("amount must be positive"))
	}
	if err := i.PaymentTerm.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("payment term: %w", err))
	}
	if err := i.SettlementMode.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("settlement mode: %w", err))
	}
	if i.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}
	if err := i.TaxConfig.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("tax config: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (s *CustomCurrencyCreditsSuite) createCustomCurrencyFlatFeeCharge(ctx context.Context, input customCurrencyFlatFeeChargeInput) flatfee.Charge {
	s.T().Helper()
	s.Require().NoError(input.Validate())

	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: input.Namespace,
		Intents: charges.ChargeIntents{
			charges.NewChargeIntent(flatfee.Intent{
				Intent: meta.Intent{
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: lo.ToPtr(input.Name),
					CustomerID:        input.Customer.ID,
					Currency:          input.Currency,
					TaxConfig:         input.TaxConfig,
				},
				IntentMutableFields: flatfee.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              input.Name,
						ServicePeriod:     input.ServicePeriod,
						FullServicePeriod: input.ServicePeriod,
						BillingPeriod:     input.ServicePeriod,
					},
					InvoiceAt:             input.InvoiceAt,
					PaymentTerm:           input.PaymentTerm,
					AmountBeforeProration: input.Amount,
				},
				SettlementMode: input.SettlementMode,
				FeatureKey:     input.FeatureKey,
				CostBasis:      input.CostBasis,
			}),
		},
	})
	s.Require().NoError(err)
	s.Require().Len(created, 1)
	charge, err := created[0].AsFlatFeeCharge()
	s.Require().NoError(err)

	return charge
}

type customCurrencyCreditPurchaseInput struct {
	Namespace      string
	Customer       customer.CustomerID
	Currency       currencies.Currency
	Amount         alpacadecimal.Decimal
	At             time.Time
	Name           string
	Priority       *int
	FeatureFilters creditpurchase.FeatureFilters
	Settlement     creditpurchase.Settlement
	CostBasis      creditpurchase.CostBasis
	TaxConfig      productcatalog.TaxCodeConfig
}

func (i customCurrencyCreditPurchaseInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}
	if err := i.Customer.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}
	if i.Customer.Namespace != i.Namespace {
		errs = append(errs, errors.New("customer namespace must match input namespace"))
	}
	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}
	if !i.Currency.IsCustom() {
		errs = append(errs, errors.New("currency must be custom"))
	}
	if !i.Amount.IsPositive() {
		errs = append(errs, errors.New("amount must be positive"))
	}
	if i.At.IsZero() {
		errs = append(errs, errors.New("at is required"))
	}
	if i.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}
	if err := i.Settlement.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("settlement: %w", err))
	}
	if err := i.FeatureFilters.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("feature filters: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (s *CustomCurrencyCreditsSuite) createCustomCurrencyCreditPurchase(ctx context.Context, input customCurrencyCreditPurchaseInput) creditpurchase.Charge {
	s.T().Helper()
	s.Require().NoError(input.Validate())

	servicePeriod := timeutil.ClosedPeriod{From: input.At, To: input.At}
	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: input.Namespace,
		Intents: charges.ChargeIntents{
			charges.NewChargeIntent(creditpurchase.Intent{
				Intent: meta.Intent{
					ManagedBy:  billing.ManuallyManagedLine,
					CustomerID: input.Customer.ID,
					Currency:   input.Currency,
					TaxConfig:  input.TaxConfig,
				},
				IntentMutableFields: creditpurchase.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              input.Name,
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					CreditAmount:   input.Amount,
					EffectiveAt:    &input.At,
					Priority:       input.Priority,
					FeatureFilters: input.FeatureFilters,
					Settlement:     input.Settlement,
				},
				CostBasis: input.CostBasis,
			}),
		},
	})
	s.Require().NoError(err)
	s.Require().Len(created, 1)
	charge, err := created[0].AsCreditPurchaseCharge()
	s.Require().NoError(err)

	return charge
}

type settledFiatCreditPurchaseInput struct {
	Namespace      string
	Customer       customer.CustomerID
	Amount         alpacadecimal.Decimal
	At             time.Time
	Priority       *int
	FeatureFilters creditpurchase.FeatureFilters
	TaxConfig      productcatalog.TaxCodeConfig
}

func (i settledFiatCreditPurchaseInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}
	if err := i.Customer.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}
	if i.Customer.Namespace != i.Namespace {
		errs = append(errs, errors.New("customer namespace must match input namespace"))
	}
	if !i.Amount.IsPositive() {
		errs = append(errs, errors.New("amount must be positive"))
	}
	if i.At.IsZero() {
		errs = append(errs, errors.New("at is required"))
	}
	if err := i.FeatureFilters.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("feature filters: %w", err))
	}
	if err := i.TaxConfig.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("tax config: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (s *CustomCurrencyCreditsSuite) createSettledFiatCreditPurchase(ctx context.Context, input settledFiatCreditPurchaseInput) creditpurchase.Charge {
	s.T().Helper()
	s.Require().NoError(input.Validate())

	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: input.Namespace,
		Intents: charges.ChargeIntents{
			s.CreateCreditPurchaseIntent(CreateCreditPurchaseIntentInput{
				Customer:      input.Customer,
				Currency:      USD,
				Amount:        input.Amount,
				EffectiveAt:   &input.At,
				Priority:      input.Priority,
				ServicePeriod: timeutil.ClosedPeriod{From: input.At, To: input.At},
				Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
					InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
				}),
				CostBasis:      newFiatCreditPurchaseCostBasis(alpacadecimal.NewFromInt(1)),
				FeatureFilters: input.FeatureFilters,
				TaxConfig:      input.TaxConfig,
			}),
		},
	})
	s.Require().NoError(err)
	s.Require().Len(created, 1)
	charge, err := created[0].AsCreditPurchaseCharge()
	s.Require().NoError(err)

	return s.settleExternalCreditPurchase(ctx, charge.GetChargeID())
}

func (s *CustomCurrencyCreditsSuite) settleExternalCreditPurchase(ctx context.Context, chargeID meta.ChargeID) creditpurchase.Charge {
	s.T().Helper()

	charge, err := s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
		ChargeID:           chargeID,
		TargetPaymentState: payment.StatusAuthorized,
	})
	s.Require().NoError(err)
	s.Equal(payment.StatusAuthorized, charge.Realizations.ExternalPaymentSettlement.Status)

	charge, err = s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
		ChargeID:           chargeID,
		TargetPaymentState: payment.StatusSettled,
	})
	s.Require().NoError(err)
	s.Equal(payment.StatusSettled, charge.Realizations.ExternalPaymentSettlement.Status)

	return charge
}

func (s *CustomCurrencyCreditsSuite) createCustomCurrency(namespace string, code currencyx.Code) currencies.Currency {
	s.T().Helper()

	currency, err := s.CurrencyService.CreateCurrency(s.T().Context(), currencies.CreateCurrencyInput{
		Namespace: namespace,
		CurrencyDetails: currencyx.CurrencyDetails{
			Code:               code,
			Name:               code.String(),
			Symbol:             code.String(),
			Precision:          2,
			DecimalMark:        ".",
			ThousandsSeparator: ",",
		},
	})
	s.Require().NoError(err)

	return currency
}

func (s *CustomCurrencyCreditsSuite) newManualCostBasis(rate alpacadecimal.Decimal) costbasis.Intent {
	s.T().Helper()

	fiatCurrency, err := currencyx.NewFiatCurrency(USD)
	s.Require().NoError(err)

	return costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: fiatCurrency,
		Rate:         rate,
	})
}

func (s *CustomCurrencyCreditsSuite) mustCustomerAccounts(customerID customer.CustomerID) ledger.CustomerAccounts {
	s.T().Helper()

	accounts, err := s.LedgerResolver.GetCustomerAccounts(s.T().Context(), customerID)
	s.Require().NoError(err)

	return accounts
}

func (s *CustomCurrencyCreditsSuite) requireAccountBalance(account ledger.Account, route ledger.RouteFilter, expected float64, message string) {
	s.T().Helper()

	balance, err := s.BalanceQuerier.GetAccountBalance(s.T().Context(), account, route, ledger.BalanceQuery{})
	s.Require().NoError(err)
	s.Equal(expected, balance.InexactFloat64(), message)
}

func (s *CustomCurrencyCreditsSuite) requireCustomCurrencyInvoiceTotals(invoice billing.StandardInvoice, amount, credits, total float64) {
	s.T().Helper()

	s.RequireTotals(billingtest.ExpectedTotals{
		Amount:       amount,
		CreditsTotal: credits,
		Total:        total,
	}, invoice.Totals)
	s.Require().Len(invoice.Lines.OrEmpty(), 1)
	fiatCurrency, err := currencyx.NewFiatCurrency(USD)
	s.Require().NoError(err)
	s.Equal(credits, invoice.Lines.OrEmpty()[0].CreditsApplied.SumAmount(fiatCurrency).InexactFloat64())
}
