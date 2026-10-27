package credits

import (
	"context"
	"errors"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

// A collecting invoice has already allocated paid credit, but its usage run is still
// mutable. Canceling that charge must restore the allocation whether or not it
// happened to leave a fiat overage. The partial-credit case provides a control:
// the only difference is whether five TOKENS of usage exhaust the initial grant.
// The TOKENS were externally paid at one USD each, so failing to restore five
// TOKENS leaves five USD of purchased credit consumed by a deleted charge.
//
// cancelRealizationRun currently uses ShouldOmitInvoiceLine to decide whether to
// correct credit allocations. A zero overage can occur before collection has
// completed, so it does not establish that the allocation is final.
//
// This applies the system cancellation patch at the charge-service boundary;
// it does not execute subscription ownership validation or full reconciliation.
// The production reconciler.diffItem removes a persisted charge absent from the target state,
// and chargePatchCollection.AddDelete selects RefundAsCreditsDeletePolicy.
// For example, targetstate.Builder.Build returns an empty target for a deleted
// subscription. An ordinary mid-period cancellation can instead shrink a charge
// and is not the entry point covered here.
// The public DeleteCustomerCharge API currently exposes only adjustment=none.
//
// The "Currency identity" section of openmeter/billing/charges/README.md says
// deletion corrects allocations while a run is reversible. The boundary to fix
// is cancelRealizationRun in charges/usagebased/service/creditheninvoice.go;
// its flat-fee counterpart is cancelCurrentRealization in
// charges/flatfee/service/creditheninvoice.go. Also review the zero-overage guard
// in reconcileDeletedStandardLines in charges/usagebased/service/lineengine.go:
// system invoice-line deletion uses that separate cleanup entry point.
//
// Run all refund regressions in this file:
// POSTGRES_HOST=127.0.0.1 go test -tags=dynamic ./test/credits -run 'TestCustomCurrencyCreditsSuite/TestCreditThenInvoiceRefund' -count=1
func (s *CustomCurrencyCreditsSuite) TestCreditThenInvoiceRefundOnUsageCancellation() {
	for _, tc := range []struct {
		name          string
		grant         int64
		allocated     float64
		invoiceAmount float64
	}{
		{
			name:          "partial credit coverage",
			grant:         3,
			allocated:     3,
			invoiceAmount: 2,
		},
		{
			name:          "full credit coverage",
			grant:         10,
			allocated:     5,
			invoiceAmount: 0,
		},
	} {
		s.Run(tc.name, func() {
			// given: five TOKENS of usage has a provisional credit allocation.
			ctx := s.T().Context()
			ns := s.GetUniqueNamespace("audit-cancel-collecting-custom-usage")
			period := timeutil.ClosedPeriod{
				From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			}
			midPeriod := period.From.Add(15 * 24 * time.Hour)
			clock.FreezeTime(period.From.Add(-time.Hour))
			defer clock.UnFreeze()

			defaults := s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), ns)
			customInvoicing := s.SetupCustomInvoicing(ns)
			cust := s.CreateLedgerBackedCustomer(ns, "audit-cancel-collecting")
			s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
				billingtest.WithManualApproval(), billingtest.WithProgressiveBilling())
			feature := s.SetupApiRequestsTotalFeature(ctx, ns)
			defer feature.Cleanup()
			defer s.MockStreamingConnector.Reset()
			tokens := s.createCustomCurrency(ns, "TOKENS")
			manualCostBasis := s.newManualCostBasis(alpacadecimal.NewFromInt(1))
			s.createCustomCurrencyCreditPurchase(ctx, customCurrencyCreditPurchaseInput{
				Namespace: ns,
				Customer:  cust.GetID(),
				Currency:  tokens,
				Amount:    alpacadecimal.NewFromInt(tc.grant),
				At:        clock.Now(),
				Name:      "externally paid TOKENS",
				Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
					InitialStatus: creditpurchase.SettledInitialPaymentSettlementStatus,
				}),
				CostBasis: creditpurchase.NewCostBasis(manualCostBasis),
				TaxConfig: productcatalog.TaxCodeConfig{
					TaxCodeID: defaults.CreditGrantTaxCodeID,
				},
			})
			charge := s.createCustomCurrencyUsageCharge(ctx, customCurrencyUsageChargeInput{
				Namespace:      ns,
				Customer:       cust.GetID(),
				Currency:       tokens,
				ServicePeriod:  period,
				FeatureKey:     feature.Feature.Key,
				UnitPrice:      alpacadecimal.NewFromInt(1),
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				CostBasis:      &manualCostBasis,
				Name:           "cancel collecting custom usage",
				TaxConfig: productcatalog.TaxCodeConfig{
					TaxCodeID: defaults.InvoicingTaxCodeID,
				},
			})
			s.MockStreamingConnector.AddSimpleEvent(feature.Feature.Key, 5, period.From.Add(time.Hour))
			clock.FreezeTime(midPeriod)
			defer clock.UnFreeze()
			invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
				Customer: cust.GetID(),
				AsOf:     lo.ToPtr(midPeriod),
			})
			s.Require().NoError(err)
			s.Require().Len(invoices, 1)
			s.Equal(billing.StandardInvoiceStatusDraftWaitingForCollection, invoices[0].Status)
			s.Equal(tc.invoiceAmount, invoices[0].Totals.Total.InexactFloat64())
			collectingCharge, err := s.MustGetChargeByID(charge.GetChargeID()).AsUsageBasedCharge()
			s.Require().NoError(err)
			s.Equal(usagebased.StatusActiveRealizationWaitingForCollection, collectingCharge.Status)
			run, err := collectingCharge.GetCurrentRealizationRun()
			s.Require().NoError(err)
			s.Require().NotNil(run.LineID)
			s.False(run.Immutable)
			s.Nil(run.InvoiceUsage)
			s.Equal(tc.allocated, run.CreditsAllocated.Sum().InexactFloat64())
			accounts := s.mustCustomerAccounts(cust.GetID())
			route := ledger.RouteFilter{Currency: tokens.Reference()}
			s.requireAccountBalance(accounts.FBOAccount, route, float64(tc.grant)-tc.allocated, "provisional available credit")
			s.requireAccountBalance(accounts.AccruedAccount, route, tc.allocated, "provisional accrued usage")

			// when: the charge is canceled with refund-as-credits before collection.
			s.MustRefundCharge(ctx, cust.GetID(), charge.GetChargeID())

			// then: its provisional credit allocation is released in either case.
			deletedCharge, err := s.MustGetChargeByID(charge.GetChargeID()).AsUsageBasedCharge()
			s.Require().NoError(err)
			s.Equal(usagebased.StatusDeleted, deletedCharge.Status)
			s.Nil(deletedCharge.State.CurrentRealizationRunID)
			fetchedInvoice, err := s.BillingService.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
				Invoice: invoices[0].GetInvoiceID(),
				Expand:  billing.InvoiceExpands{billing.InvoiceExpandLines, billing.InvoiceExpandDeletedLines},
			})
			s.Require().NoError(err)
			standardInvoice, err := fetchedInvoice.AsStandardInvoice()
			s.Require().NoError(err)
			deletedLine := standardInvoice.Lines.GetByID(*run.LineID)
			s.Require().NotNil(deletedLine)
			s.NotNil(deletedLine.DeletedAt)
			s.Zero(standardInvoice.Lines.NonDeletedLineCount())
			history, err := s.Charges.GetByID(ctx, charges.GetByIDInput{
				ChargeID: charge.GetChargeID(),
				Expands:  meta.Expands{meta.ExpandRealizations, meta.ExpandDeletedRealizations},
			})
			s.Require().NoError(err)
			chargeHistory, err := history.AsUsageBasedCharge()
			s.Require().NoError(err)
			canceledRun, err := chargeHistory.Realizations.GetByID(run.ID.ID)
			s.Require().NoError(err)
			s.NotNil(canceledRun.DeletedAt, "a canceled provisional run must not remain eligible billing history")
			s.requireAccountBalance(accounts.FBOAccount, route, float64(tc.grant), "canceling a mutable run restores its initial credit")
			s.requireAccountBalance(accounts.AccruedAccount, route, 0, "canceled provisional usage has no remaining accrual")

			// Repeating the same cancellation must not leave the customer worse off.
			// A deleted charge can reject the trigger; either response must preserve
			// the credit restoration required by the first successful cancellation.
			patch, err := meta.NewPatchDelete(meta.NewPatchDeleteInput{
				ChangeSource: billing.ChangeSourceSystem,
				Policy:       meta.RefundAsCreditsDeletePolicy,
			})
			s.Require().NoError(err)
			err = s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
				CustomerID:        cust.GetID(),
				PatchesByChargeID: map[string]charges.Patch{charge.ID: patch},
			})
			s.T().Logf("repeated charge cancellation returned: %v", err)
			s.requireAccountBalance(accounts.FBOAccount, route, float64(tc.grant), "credit after repeating cancellation")
			s.requireAccountBalance(accounts.AccruedAccount, route, 0, "accrual after repeating cancellation")
		})
	}
}

// A usage sibling keeps this invoice in collection while the fully funded flat
// fee is still mutable. Its zero overage must not prevent refunding the TOKENS
// allocated by the flat fee when subscription reconciliation removes that charge
// using the same system cancellation policy exercised by the usage test above.
// cancelCurrentRealization in charges/flatfee/service/creditheninvoice.go uses
// the same zero-overage guard as usage-based cancellation; the reversible-run
// contract and command for both reproducers are documented above.
func (s *CustomCurrencyCreditsSuite) TestCreditThenInvoiceRefundOnFlatFeeCancellation() {
	// given: a five-TOKEN flat fee and a usage line share a collecting USD invoice.
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("audit-cancel-collecting-custom-flat")
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	midPeriod := period.From.Add(15 * 24 * time.Hour)
	clock.FreezeTime(period.From.Add(-time.Hour))
	defer clock.UnFreeze()
	defaults := s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), ns)
	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "audit-cancel-flat")
	s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithManualApproval(), billingtest.WithProgressiveBilling())
	feature := s.SetupApiRequestsTotalFeature(ctx, ns)
	defer feature.Cleanup()
	defer s.MockStreamingConnector.Reset()
	tokens := s.createCustomCurrency(ns, "TOKENS")
	manualCostBasis := s.newManualCostBasis(alpacadecimal.NewFromInt(1))
	s.createCustomCurrencyCreditPurchase(ctx, customCurrencyCreditPurchaseInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Currency:  tokens,
		Amount:    alpacadecimal.NewFromInt(10),
		At:        clock.Now(),
		Name:      "externally paid TOKENS",
		Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
			InitialStatus: creditpurchase.SettledInitialPaymentSettlementStatus,
		}),
		CostBasis: creditpurchase.NewCostBasis(manualCostBasis),
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.CreditGrantTaxCodeID,
		},
	})
	charge := s.createCustomCurrencyFlatFeeCharge(ctx, customCurrencyFlatFeeChargeInput{
		Namespace:      ns,
		Customer:       cust.GetID(),
		Currency:       tokens,
		ServicePeriod:  period,
		InvoiceAt:      midPeriod,
		Amount:         alpacadecimal.NewFromInt(5),
		PaymentTerm:    productcatalog.InAdvancePaymentTerm,
		SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
		CostBasis:      &manualCostBasis,
		Name:           "cancel collecting custom flat fee",
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.InvoicingTaxCodeID,
		},
	})
	siblingCharges, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.NewCreateChargeIntents(s.CreateMockChargeIntent(CreateMockChargeIntentInput{
			Customer:          cust.GetID(),
			Currency:          USD,
			ServicePeriod:     period,
			Price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
			FeatureKey:        feature.Feature.Key,
			Name:              "usage keeps invoice in collection",
			SettlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
			ManagedBy:         billing.SubscriptionManagedLine,
			UniqueReferenceID: "collecting-usage-sibling",
		})),
	})
	s.Require().NoError(err)
	s.Require().Len(siblingCharges, 1)
	s.MockStreamingConnector.AddSimpleEvent(feature.Feature.Key, 1, period.From.Add(time.Hour))
	clock.FreezeTime(midPeriod)
	defer clock.UnFreeze()
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     lo.ToPtr(midPeriod),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	s.Require().Len(invoices[0].Lines.OrEmpty(), 2)
	s.Equal(billing.StandardInvoiceStatusDraftWaitingForCollection, invoices[0].Status)
	collectingCharge, err := s.MustGetChargeByID(charge.GetChargeID()).AsFlatFeeCharge()
	s.Require().NoError(err)
	s.Equal(flatfee.StatusActiveRealizationWaitingForCollection, collectingCharge.Status)
	s.Require().NotNil(collectingCharge.Realizations.CurrentRun)
	run := collectingCharge.Realizations.CurrentRun
	s.Require().NotNil(run.LineID)
	s.False(run.Immutable)
	s.Nil(run.AccruedUsage)
	s.Equal(float64(5), run.CreditRealizations.Sum().InexactFloat64())
	accounts := s.mustCustomerAccounts(cust.GetID())
	route := ledger.RouteFilter{Currency: tokens.Reference()}
	s.requireAccountBalance(accounts.FBOAccount, route, 5, "provisional available credit")
	s.requireAccountBalance(accounts.AccruedAccount, route, 5, "provisional accrued flat fee")

	// when: the flat-fee charge is canceled before invoice collection completes.
	s.MustRefundCharge(ctx, cust.GetID(), charge.GetChargeID())

	// then: its provisional allocation is released, leaving the usage sibling alone.
	deletedCharge, err := s.MustGetChargeByID(charge.GetChargeID()).AsFlatFeeCharge()
	s.Require().NoError(err)
	s.Equal(flatfee.StatusDeleted, deletedCharge.Status)
	s.Nil(deletedCharge.Realizations.CurrentRun)
	fetchedInvoice, err := s.BillingService.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
		Invoice: invoices[0].GetInvoiceID(),
		Expand:  billing.InvoiceExpands{billing.InvoiceExpandLines, billing.InvoiceExpandDeletedLines},
	})
	s.Require().NoError(err)
	standardInvoice, err := fetchedInvoice.AsStandardInvoice()
	s.Require().NoError(err)
	deletedLine := standardInvoice.Lines.GetByID(*run.LineID)
	s.Require().NotNil(deletedLine)
	s.NotNil(deletedLine.DeletedAt)
	s.Equal(1, standardInvoice.Lines.NonDeletedLineCount())
	siblingChargeID, err := siblingCharges[0].GetChargeID()
	s.Require().NoError(err)
	siblingCharge, err := s.MustGetChargeByID(siblingChargeID).AsUsageBasedCharge()
	s.Require().NoError(err)
	s.Equal(usagebased.StatusActiveRealizationWaitingForCollection, siblingCharge.Status)
	siblingRun, err := siblingCharge.GetCurrentRealizationRun()
	s.Require().NoError(err)
	s.Require().NotNil(siblingRun.LineID)
	siblingLine := standardInvoice.Lines.GetByID(*siblingRun.LineID)
	s.Require().NotNil(siblingLine)
	s.Nil(siblingLine.DeletedAt)
	s.requireAccountBalance(accounts.FBOAccount, route, 10, "canceling the mutable flat fee restores its credit")
	s.requireAccountBalance(accounts.AccruedAccount, route, 0, "canceled flat fee has no remaining accrual")
}

// The public invoice edit and deletion services accept this collecting custom-
// currency line: validateInvoiceLineDeleteViaAPI requires credit_then_invoice,
// one realization, and a matching invoice/line association. No invoice has been
// issued, so reversing the provisional allocation needs no credit note.
//
// Both API paths call handleInvoiceLineDeleteViaAPI in
// openmeter/billing/charges/usagebased/service/lineengine.go, which chooses
// RefundAsCreditsDeletePolicy. cancelRealizationRun in that package's
// creditheninvoice.go currently skips correction when ShouldOmitInvoiceLine is
// true. Five TOKENS that were paid at one USD each must return to the customer
// when their mutable charge is deleted, even when no fiat overage remains.
// The partial-funded cases prove the same API operation restores paid credits
// when the usage also leaves a fiat overage.
//
// This is the reversible-deletion contract described in the "Currency identity"
// section of openmeter/billing/charges/README.md. Also review the zero-overage
// guards in validateDeletedStandardLines and reconcileDeletedStandardLines when
// fixing cleanup; they are separate consumers of the line omission decision.
//
// POSTGRES_HOST=127.0.0.1 go test -tags=dynamic ./test/credits -run 'TestCustomCurrencyCreditsSuite/TestCreditThenInvoiceRefundOnInvoiceDeletion' -count=1
func (s *CustomCurrencyCreditsSuite) TestCreditThenInvoiceRefundOnInvoiceDeletion() {
	for _, operation := range []struct {
		name           string
		delete         func(context.Context, billing.StandardInvoice, string) (billing.StandardInvoice, error)
		assertDeletion func(billing.StandardInvoice, *billing.StandardLine)
	}{
		{
			name: "delete invoice line",
			delete: func(ctx context.Context, invoice billing.StandardInvoice, lineID string) (billing.StandardInvoice, error) {
				return s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
					Invoice:      invoice.GetInvoiceID(),
					ChangeSource: billing.ChangeSourceAPIRequest,
					EditFn: func(editedInvoice *billing.StandardInvoice) error {
						line := editedInvoice.Lines.GetByID(lineID)
						if line == nil {
							return errors.New("collecting custom-currency invoice line is missing")
						}

						line.DeletedAt = lo.ToPtr(clock.Now())

						return nil
					},
				})
			},
			assertDeletion: func(invoice billing.StandardInvoice, line *billing.StandardLine) {
				s.NotNil(line.DeletedAt)
				s.Zero(invoice.Lines.NonDeletedLineCount())
			},
		},
		{
			name: "delete whole invoice",
			delete: func(ctx context.Context, invoice billing.StandardInvoice, _ string) (billing.StandardInvoice, error) {
				return s.BillingService.DeleteInvoice(ctx, billing.DeleteInvoiceInput{
					Invoice:        invoice.GetInvoiceID(),
					DeletionSource: billing.ChangeSourceAPIRequest,
				})
			},
			assertDeletion: func(invoice billing.StandardInvoice, _ *billing.StandardLine) {
				// Whole-invoice deletion keeps its line rows as invoice history.
				s.Equal(billing.StandardInvoiceStatusDeleted, invoice.Status)
				s.NotNil(invoice.DeletedAt)
			},
		},
	} {
		for _, coverage := range []struct {
			name          string
			grant         int64
			allocated     float64
			invoiceAmount float64
		}{
			{
				name:          "partial paid credit coverage",
				grant:         3,
				allocated:     3,
				invoiceAmount: 2,
			},
			{
				name:          "full paid credit coverage",
				grant:         10,
				allocated:     5,
				invoiceAmount: 0,
			},
		} {
			s.Run(operation.name+"/"+coverage.name, func() {
				// given: paid TOKENS provisionally fund five units on a collecting invoice.
				ctx := s.T().Context()
				ns := s.GetUniqueNamespace("audit-public-invoice-delete")
				period := timeutil.ClosedPeriod{
					From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				}
				midPeriod := period.From.Add(15 * 24 * time.Hour)
				clock.FreezeTime(period.From.Add(-time.Hour))
				defer clock.UnFreeze()

				defaults := s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), ns)
				customInvoicing := s.SetupCustomInvoicing(ns)
				cust := s.CreateLedgerBackedCustomer(ns, "audit-public-invoice-delete")
				s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
					billingtest.WithManualApproval(), billingtest.WithProgressiveBilling())
				feature := s.SetupApiRequestsTotalFeature(ctx, ns)
				defer feature.Cleanup()
				defer s.MockStreamingConnector.Reset()
				tokens := s.createCustomCurrency(ns, "TOKENS")
				manualCostBasis := s.newManualCostBasis(alpacadecimal.NewFromInt(1))
				s.createCustomCurrencyCreditPurchase(ctx, customCurrencyCreditPurchaseInput{
					Namespace: ns,
					Customer:  cust.GetID(),
					Currency:  tokens,
					Amount:    alpacadecimal.NewFromInt(coverage.grant),
					At:        clock.Now(),
					Name:      "externally paid TOKENS",
					Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
						InitialStatus: creditpurchase.SettledInitialPaymentSettlementStatus,
					}),
					CostBasis: creditpurchase.NewCostBasis(manualCostBasis),
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: defaults.CreditGrantTaxCodeID,
					},
				})
				charge := s.createCustomCurrencyUsageCharge(ctx, customCurrencyUsageChargeInput{
					Namespace:      ns,
					Customer:       cust.GetID(),
					Currency:       tokens,
					ServicePeriod:  period,
					FeatureKey:     feature.Feature.Key,
					UnitPrice:      alpacadecimal.NewFromInt(1),
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					CostBasis:      &manualCostBasis,
					Name:           "usage on a publicly deleted invoice",
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: defaults.InvoicingTaxCodeID,
					},
				})
				s.MockStreamingConnector.AddSimpleEvent(feature.Feature.Key, 5, period.From.Add(time.Hour))
				clock.FreezeTime(midPeriod)
				defer clock.UnFreeze()
				invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
					Customer: cust.GetID(),
					AsOf:     lo.ToPtr(midPeriod),
				})
				s.Require().NoError(err)
				s.Require().Len(invoices, 1)
				s.Equal(billing.StandardInvoiceStatusDraftWaitingForCollection, invoices[0].Status)
				s.Equal(coverage.invoiceAmount, invoices[0].Totals.Total.InexactFloat64())
				collectingCharge, err := s.MustGetChargeByID(charge.GetChargeID()).AsUsageBasedCharge()
				s.Require().NoError(err)
				s.Equal(usagebased.StatusActiveRealizationWaitingForCollection, collectingCharge.Status)
				s.Require().Len(collectingCharge.Realizations, 1)
				run, err := collectingCharge.GetCurrentRealizationRun()
				s.Require().NoError(err)
				s.Require().NotNil(run.LineID)
				s.False(run.Immutable)
				s.Nil(run.InvoiceUsage)
				s.Equal(coverage.allocated, run.CreditsAllocated.Sum().InexactFloat64())
				accounts := s.mustCustomerAccounts(cust.GetID())
				route := ledger.RouteFilter{Currency: tokens.Reference()}
				s.requireAccountBalance(accounts.FBOAccount, route, float64(coverage.grant)-coverage.allocated, "provisional available paid credit")
				s.requireAccountBalance(accounts.AccruedAccount, route, coverage.allocated, "provisional accrued usage")

				// when: the public billing service deletes the line or its whole draft invoice.
				_, err = operation.delete(ctx, invoices[0], *run.LineID)
				s.Require().NoError(err)

				// then: persisted deletion also releases every provisional paid-credit allocation.
				fetchedInvoice, err := s.BillingService.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
					Invoice: invoices[0].GetInvoiceID(),
					Expand:  billing.InvoiceExpands{billing.InvoiceExpandLines, billing.InvoiceExpandDeletedLines},
				})
				s.Require().NoError(err)
				standardInvoice, err := fetchedInvoice.AsStandardInvoice()
				s.Require().NoError(err)
				deletedLine := standardInvoice.Lines.GetByID(*run.LineID)
				s.Require().NotNil(deletedLine)
				operation.assertDeletion(standardInvoice, deletedLine)
				history, err := s.Charges.GetByID(ctx, charges.GetByIDInput{
					ChargeID: charge.GetChargeID(),
					Expands:  meta.Expands{meta.ExpandRealizations, meta.ExpandDeletedRealizations},
				})
				s.Require().NoError(err)
				deletedCharge, err := history.AsUsageBasedCharge()
				s.Require().NoError(err)
				s.Equal(usagebased.StatusDeleted, deletedCharge.Status)
				s.Nil(deletedCharge.State.CurrentRealizationRunID)
				canceledRun, err := deletedCharge.Realizations.GetByID(run.ID.ID)
				s.Require().NoError(err)
				s.NotNil(canceledRun.DeletedAt, "a deleted mutable invoice run must be excluded from billing history")
				s.requireAccountBalance(accounts.FBOAccount, route, float64(coverage.grant), "public invoice deletion restores paid credit")
				s.requireAccountBalance(accounts.AccruedAccount, route, 0, "deleted mutable usage has no remaining accrual")
			})
		}
	}
}

// Public standard invoice-line deletion rejects charges with multiple non-voided
// runs. Both API operations below must leave three paid partial invoices and a
// fourth collecting run untouched. The system shrink patch has a narrower
// supported operation: end the service period at the third run's end and remove
// only the fourth mutable run. No issued history needs reversing.
//
// usageBasedChargeCollection.AddShrink in the subscription reconciler creates
// this system patch. This fixture applies it at the charge-service boundary;
// it does not execute subscription ownership validation or full reconciliation.
// handleRunsOnShrink cancels the fourth run, then the charge
// invoice updater deletes its empty invoice using ChangeSourceSystem. The test
// deliberately uses this caller rather than directly deleting a nonempty invoice
// as the system. cancelRealizationRun must restore the five paid TOKENS even
// though the fourth run had no fiat overage; its three issued predecessors stay.
//
// POSTGRES_HOST=127.0.0.1 go test -tags=dynamic ./test/credits -run 'TestCustomCurrencyCreditsSuite/TestCreditThenInvoiceRefundAfterThreePaidPartials' -count=1
func (s *CustomCurrencyCreditsSuite) TestCreditThenInvoiceRefundAfterThreePaidPartials() {
	// given: three progressive invoices have each collected five USD of usage.
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("audit-delete-after-three-partials")
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	clock.FreezeTime(period.From.Add(-time.Hour))
	defer clock.UnFreeze()
	defaults := s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), ns)
	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "audit-delete-three-partials")
	s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithManualApproval(), billingtest.WithProgressiveBilling())
	feature := s.SetupApiRequestsTotalFeature(ctx, ns)
	defer feature.Cleanup()
	defer s.MockStreamingConnector.Reset()
	tokens := s.createCustomCurrency(ns, "TOKENS")
	manualCostBasis := s.newManualCostBasis(alpacadecimal.NewFromInt(1))
	charge := s.createCustomCurrencyUsageCharge(ctx, customCurrencyUsageChargeInput{
		Namespace:      ns,
		Customer:       cust.GetID(),
		Currency:       tokens,
		ServicePeriod:  period,
		FeatureKey:     feature.Feature.Key,
		UnitPrice:      alpacadecimal.NewFromInt(1),
		SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
		CostBasis:      &manualCostBasis,
		Name:           "usage across four progressive invoices",
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.InvoicingTaxCodeID,
		},
	})
	paidInvoices := make([]billing.StandardInvoice, 0, 3)
	for checkpoint := 1; checkpoint <= 3; checkpoint++ {
		previousCheckpoint := period.From.Add(time.Duration(checkpoint-1) * 7 * 24 * time.Hour)
		asOf := period.From.Add(time.Duration(checkpoint) * 7 * 24 * time.Hour)
		s.MockStreamingConnector.AddSimpleEvent(feature.Feature.Key, 5, previousCheckpoint.Add(time.Hour))
		clock.FreezeTime(asOf)
		defer clock.UnFreeze()
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     &asOf,
		})
		s.Require().NoError(err)
		s.Require().Len(invoices, 1)
		invoice := invoices[0]
		clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())
		defer clock.UnFreeze()
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(float64(5), invoice.Totals.Total.InexactFloat64())
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
		invoice, err = s.BillingService.PaymentAuthorized(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		invoice, err = s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: invoice.GetInvoiceID(),
			Trigger:   billing.TriggerPaid,
		})
		s.Require().NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)
		s.Require().NotNil(invoice.IssuedAt)
		paidInvoices = append(paidInvoices, invoice)
	}

	// and: a later paid grant fully covers the fourth run, which is still mutable.
	s.createCustomCurrencyCreditPurchase(ctx, customCurrencyCreditPurchaseInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Currency:  tokens,
		Amount:    alpacadecimal.NewFromInt(10),
		At:        clock.Now(),
		Name:      "paid TOKENS for the fourth partial",
		Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
			InitialStatus: creditpurchase.SettledInitialPaymentSettlementStatus,
		}),
		CostBasis: creditpurchase.NewCostBasis(manualCostBasis),
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.CreditGrantTaxCodeID,
		},
	})
	fourthCheckpoint := period.From.Add(28 * 24 * time.Hour)
	s.MockStreamingConnector.AddSimpleEvent(feature.Feature.Key, 5, period.From.Add(21*24*time.Hour+time.Hour))
	clock.FreezeTime(fourthCheckpoint)
	defer clock.UnFreeze()
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     &fourthCheckpoint,
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	currentInvoice := invoices[0]
	s.Equal(billing.StandardInvoiceStatusDraftWaitingForCollection, currentInvoice.Status)
	s.Equal(float64(0), currentInvoice.Totals.Total.InexactFloat64())
	collectingCharge, err := s.MustGetChargeByID(charge.GetChargeID()).AsUsageBasedCharge()
	s.Require().NoError(err)
	s.Require().Len(collectingCharge.Realizations, 4)
	currentRun, err := collectingCharge.GetCurrentRealizationRun()
	s.Require().NoError(err)
	s.Require().NotNil(currentRun.LineID)
	s.False(currentRun.Immutable)
	s.Equal(float64(5), currentRun.CreditsAllocated.Sum().InexactFloat64())
	for _, run := range collectingCharge.Realizations {
		if run.ID.ID != currentRun.ID.ID {
			s.True(run.Immutable, "the first three runs are issued history")
			s.Require().NotNil(run.Payment)
		}
	}

	accounts := s.mustCustomerAccounts(cust.GetID())
	route := ledger.RouteFilter{Currency: tokens.Reference()}
	s.requireAccountBalance(accounts.FBOAccount, route, 5, "fourth run reserves five of the ten purchased TOKENS")
	accruedBefore, err := s.BalanceQuerier.GetAccountBalance(ctx, accounts.AccruedAccount, route, ledger.BalanceQuery{})
	s.Require().NoError(err)
	transactionsBefore, err := s.Ledger.ListTransactions(ctx, ledger.ListTransactionsInput{
		Namespace: ns,
		Limit:     100,
	})
	s.Require().NoError(err)
	s.Require().Nil(transactionsBefore.NextCursor)

	for _, operation := range []struct {
		name   string
		delete func() error
	}{
		{
			name: "delete fourth invoice line",
			delete: func() error {
				_, err := s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
					Invoice:      currentInvoice.GetInvoiceID(),
					ChangeSource: billing.ChangeSourceAPIRequest,
					EditFn: func(editedInvoice *billing.StandardInvoice) error {
						line := editedInvoice.Lines.GetByID(*currentRun.LineID)
						if line == nil {
							return errors.New("fourth progressive invoice line is missing")
						}

						line.DeletedAt = lo.ToPtr(clock.Now())

						return nil
					},
				})

				return err
			},
		},
		{
			name: "delete whole fourth invoice",
			delete: func() error {
				_, err := s.BillingService.DeleteInvoice(ctx, billing.DeleteInvoiceInput{
					Invoice:        currentInvoice.GetInvoiceID(),
					DeletionSource: billing.ChangeSourceAPIRequest,
				})

				return err
			},
		},
	} {
		s.Run(operation.name, func() {
			// when: only the fourth, still collecting invoice is targeted for API deletion.
			err := operation.delete()
			s.Require().ErrorIs(err, billing.ErrCannotEditProgressivelyBilledUsageBasedLine)

			// then: rejection preserves all prior paid history and the current allocation.
			reloadedCharge, err := s.MustGetChargeByID(charge.GetChargeID()).AsUsageBasedCharge()
			s.Require().NoError(err)
			s.Equal(collectingCharge.Status, reloadedCharge.Status)
			s.Equal(collectingCharge.State.CurrentRealizationRunID, reloadedCharge.State.CurrentRealizationRunID)
			s.Equal(collectingCharge.Realizations, reloadedCharge.Realizations)
			reloadedCurrentInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
				Invoice: currentInvoice.GetInvoiceID(),
				Expand:  billing.StandardInvoiceExpandAll,
			})
			s.Require().NoError(err)
			s.Equal(billing.StandardInvoiceStatusDraftWaitingForCollection, reloadedCurrentInvoice.Status)
			s.Nil(reloadedCurrentInvoice.DeletedAt)
			line := reloadedCurrentInvoice.Lines.GetByID(*currentRun.LineID)
			s.Require().NotNil(line)
			s.Nil(line.DeletedAt)
			for _, paidInvoice := range paidInvoices {
				reloadedPaidInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
					Invoice: paidInvoice.GetInvoiceID(),
					Expand:  billing.StandardInvoiceExpandAll,
				})
				s.Require().NoError(err)
				s.Equal(billing.StandardInvoiceStatusPaid, reloadedPaidInvoice.Status)
				s.Equal(paidInvoice.Totals, reloadedPaidInvoice.Totals)
				s.Equal(paidInvoice.IssuedAt, reloadedPaidInvoice.IssuedAt)
				s.Nil(reloadedPaidInvoice.DeletedAt)
			}

			s.requireAccountBalance(accounts.FBOAccount, route, 5, "rejected deletion preserves available paid credit")
			s.requireAccountBalance(accounts.AccruedAccount, route, accruedBefore.InexactFloat64(), "rejected deletion preserves accrued usage")
			transactionsAfter, err := s.Ledger.ListTransactions(ctx, ledger.ListTransactionsInput{
				Namespace: ns,
				Limit:     100,
			})
			s.Require().NoError(err)
			s.Require().Nil(transactionsAfter.NextCursor)
			s.Equal(len(transactionsBefore.Items), len(transactionsAfter.Items), "rejected deletion commits no ledger transactions")
		})
	}

	s.Run("system shrink removes only the fourth mutable invoice", func() {
		// given: both preceding API operations were rejected without changing the fixture.
		// when: the charge service applies the system shrink patch emitted by reconciliation.
		thirdCheckpoint := period.From.Add(21 * 24 * time.Hour)
		patch, err := meta.NewPatchShrink(meta.NewPatchShrinkInput{
			ChangeSource:           billing.ChangeSourceSystem,
			NewServicePeriodTo:     thirdCheckpoint,
			NewFullServicePeriodTo: period.To,
			NewBillingPeriodTo:     thirdCheckpoint,
			NewInvoiceAt:           thirdCheckpoint,
		})
		s.Require().NoError(err)
		err = s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
			CustomerID:        cust.GetID(),
			PatchesByChargeID: map[string]charges.Patch{charge.ID: patch},
		})
		s.Require().NoError(err)

		// then: the fourth invoice is deleted and its paid-credit allocation is released.
		deletedInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: currentInvoice.GetInvoiceID(),
			Expand:  billing.StandardInvoiceExpands{billing.StandardInvoiceExpandLines, billing.StandardInvoiceExpandDeletedLines},
		})
		s.Require().NoError(err)
		s.Equal(billing.StandardInvoiceStatusDeleted, deletedInvoice.Status)
		s.NotNil(deletedInvoice.DeletedAt)
		deletedLine := deletedInvoice.Lines.GetByID(*currentRun.LineID)
		s.Require().NotNil(deletedLine)
		s.NotNil(deletedLine.DeletedAt)
		history, err := s.Charges.GetByID(ctx, charges.GetByIDInput{
			ChargeID: charge.GetChargeID(),
			Expands:  meta.Expands{meta.ExpandRealizations, meta.ExpandDeletedRealizations},
		})
		s.Require().NoError(err)
		shrunkCharge, err := history.AsUsageBasedCharge()
		s.Require().NoError(err)
		s.Equal(thirdCheckpoint, shrunkCharge.Intent.GetEffectiveServicePeriod().To)
		s.Nil(shrunkCharge.State.CurrentRealizationRunID)
		canceledRun, err := shrunkCharge.Realizations.GetByID(currentRun.ID.ID)
		s.Require().NoError(err)
		s.NotNil(canceledRun.DeletedAt, "the fourth mutable run must be voided after shrink")
		s.Len(shrunkCharge.Realizations.WithoutVoidedBillingHistory(), 3)
		s.requireAccountBalance(accounts.FBOAccount, route, 10, "shrinking away the fourth run restores its five paid TOKENS")
		s.requireAccountBalance(accounts.AccruedAccount, route, accruedBefore.Sub(alpacadecimal.NewFromInt(5)).InexactFloat64(), "only the fourth run's accrual is reversed")

		// The earlier invoices remain paid, with the same immutable money and ledger references.
		for _, priorRun := range collectingCharge.Realizations {
			if priorRun.ID.ID == currentRun.ID.ID {
				continue
			}

			preservedRun, err := shrunkCharge.Realizations.GetByID(priorRun.ID.ID)
			s.Require().NoError(err)
			s.True(preservedRun.Immutable)
			s.Nil(preservedRun.DeletedAt)
			s.Equal(priorRun.Totals, preservedRun.Totals)
			s.Equal(priorRun.CreditsAllocated, preservedRun.CreditsAllocated)
			s.Equal(priorRun.InvoiceUsage, preservedRun.InvoiceUsage)
			s.Equal(priorRun.Payment, preservedRun.Payment)
			s.Equal(priorRun.LineID, preservedRun.LineID)
			s.Equal(priorRun.InvoiceID, preservedRun.InvoiceID)
		}

		for _, paidInvoice := range paidInvoices {
			reloadedPaidInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
				Invoice: paidInvoice.GetInvoiceID(),
				Expand:  billing.StandardInvoiceExpandAll,
			})
			s.Require().NoError(err)
			s.Equal(billing.StandardInvoiceStatusPaid, reloadedPaidInvoice.Status)
			s.Equal(paidInvoice.Totals, reloadedPaidInvoice.Totals)
			s.Equal(paidInvoice.IssuedAt, reloadedPaidInvoice.IssuedAt)
			s.Nil(reloadedPaidInvoice.DeletedAt)
		}
	})
}
