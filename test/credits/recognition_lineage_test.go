package credits

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/recognizer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func (s *CustomCurrencyCreditsSuite) TestPaidBackfillRecognitionLeavesPromotionalCreditsDeferred() {
	for _, tc := range []struct {
		name    string
		amounts []int64
	}{
		{name: "full backfill", amounts: []int64{8}},
		{name: "partial backfills", amounts: []int64{3, 5}},
	} {
		s.Run(tc.name, func() {
			t := s.T()
			ctx := t.Context()
			ns := s.GetUniqueNamespace("paid-backfill-recognition")
			defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
			customer := s.CreateLedgerBackedCustomer(ns, "test-subject")
			sandbox := s.InstallSandboxApp(t, ns)
			s.ProvisionBillingProfile(ctx, ns, sandbox.GetID(),
				billingtest.WithCollectionInterval(datetime.MustParseDuration(t, "P2D")),
			)
			feature := s.SetupApiRequestsTotalFeature(ctx, ns)
			t.Cleanup(feature.Cleanup)
			tokens := s.createCustomCurrency(ns, "TOKENS")
			start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			end := start.AddDate(0, 1, 0)
			clock.FreezeTime(start)
			defer clock.UnFreeze()

			// given: ten credits of usage consume two promotional credits with no cost
			// basis and create an eight-credit advance in the same managed currency.
			promo := s.createCustomCurrencyCreditPurchase(ctx, customCurrencyCreditPurchaseInput{
				Namespace: ns, Customer: customer.GetID(), Currency: tokens,
				Amount: alpacadecimal.NewFromInt(2), At: start, Name: "Promotional credits",
				Settlement: creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
				TaxConfig:  productcatalog.TaxCodeConfig{TaxCodeID: defaults.CreditGrantTaxCodeID},
			})
			s.MockStreamingConnector.AddSimpleEvent(feature.Feature.Key, 10, start.Add(time.Hour))
			clock.FreezeTime(end.Add(3 * 24 * time.Hour))
			usage := s.createCustomCurrencyUsageCharge(ctx, customCurrencyUsageChargeInput{
				Namespace: ns, Customer: customer.GetID(), Currency: tokens,
				ServicePeriod: timeutil.ClosedPeriod{From: start, To: end}, FeatureKey: feature.Feature.Key,
				UnitPrice: alpacadecimal.NewFromInt(1), SettlementMode: productcatalog.CreditOnlySettlementMode,
				Name: "Credit-only usage", TaxConfig: productcatalog.TaxCodeConfig{TaxCodeID: defaults.InvoicingTaxCodeID},
			})

			// when: paid purchases at USD 0.5 backfill the advance, with recognition
			// and an unchanged retry after every purchase.
			amountByBacking := make(map[string]float64)
			recognitionByBacking := make(map[string]string)
			earningsBySource := make(map[string]float64)
			for _, amount := range tc.amounts {
				clock.FreezeTime(clock.Now().Add(time.Minute))
				purchase := s.createCustomCurrencyCreditPurchase(ctx, customCurrencyCreditPurchaseInput{
					Namespace: ns, Customer: customer.GetID(), Currency: tokens,
					Amount: alpacadecimal.NewFromInt(amount), At: clock.Now(), Name: "Paid backfill",
					Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus}),
					CostBasis:  creditpurchase.NewCostBasis(s.newManualCostBasis(alpacadecimal.NewFromFloat(0.5))),
					TaxConfig:  productcatalog.TaxCodeConfig{TaxCodeID: defaults.CreditGrantTaxCodeID},
				})
				purchase = s.settleExternalCreditPurchase(ctx, purchase.GetChargeID())
				result, err := s.RevenueRecognizer.RecognizeEarnings(ctx, recognizer.RecognizeEarningsInput{
					CustomerID: customer.GetID(), Currency: tokens, At: clock.Now(),
				})
				s.Require().NoError(err)
				s.Equal(float64(amount), result.RecognizedAmount.InexactFloat64())
				backing := purchase.Realizations.CreditGrantRealization.TransactionGroupID
				amountByBacking[backing] = float64(amount)
				recognitionByBacking[backing] = result.LedgerGroupID
				earningsBySource[sourceSpendChargeBucketKey(&purchase.ID, &usage.ID)] = float64(amount)
				retry, err := s.RevenueRecognizer.RecognizeEarnings(ctx, recognizer.RecognizeEarningsInput{
					CustomerID: customer.GetID(), Currency: tokens, At: clock.Now(),
				})
				s.Require().NoError(err)
				s.Zero(retry.RecognizedAmount.InexactFloat64())
				s.Empty(retry.LedgerGroupID)
			}

			// then: journal and lineage agree on the paid source, leaving the entire
			// promotional allocation deferred. Retain the exact 2/8 segment assertion.
			accounts := s.mustCustomerAccounts(customer.GetID())
			business, err := s.LedgerResolver.GetBusinessAccounts(ctx, ns)
			s.Require().NoError(err)
			s.requireAccountBalance(business.EarningsAccount, ledger.RouteFilter{Currency: tokens.Reference()}, 8, "paid earnings")
			s.requireEarningsSourceSpendBalanceBuckets(ns, ledger.RouteFilter{Currency: tokens.Reference()}, earningsBySource)
			s.requireAccountBalance(accounts.AccruedAccount, ledger.RouteFilter{Currency: tokens.Reference()}, 2, "promotional accrued")
			s.requireCustomerAccruedSourceSpendBalanceBuckets(customer.GetID(), ledger.RouteFilter{Currency: tokens.Reference()}, map[string]float64{
				sourceSpendChargeBucketKey(&promo.ID, &usage.ID): 2,
			})
			roots, err := s.LineageService.LoadLineagesByCustomer(ctx, lineage.LoadLineagesByCustomerInput{
				Namespace: ns, CustomerID: customer.ID, Currency: tokens.Reference(),
			})
			s.Require().NoError(err)
			s.Require().Len(roots, 2)
			for _, root := range roots {
				switch root.OriginKind {
				case creditrealization.LineageOriginKindRealCredit:
					s.Require().Len(root.Segments, 1)
					s.Equal(float64(2), root.Segments[0].Amount.InexactFloat64())
					s.Equal(creditrealization.LineageSegmentStateRealCredit, root.Segments[0].State)
				case creditrealization.LineageOriginKindAdvance:
					s.Require().Len(root.Segments, len(tc.amounts))
					seen := make(map[string]float64)
					for _, segment := range root.Segments {
						backing := lo.FromPtr(segment.SourceBackingTransactionGroupID)
						s.Equal(creditrealization.LineageSegmentStateEarningsRecognized, segment.State)
						s.Equal(creditrealization.LineageSegmentStateAdvanceBackfilled, lo.FromPtr(segment.SourceState))
						s.Equal(recognitionByBacking[backing], lo.FromPtr(segment.BackingTransactionGroupID))
						seen[backing] = segment.Amount.InexactFloat64()
					}
					s.Equal(amountByBacking, seen)
				default:
					t.Fatalf("unexpected lineage origin %q", root.OriginKind)
				}
			}

			// when: the spend is refunded after recognition.
			// then: each paid backing and the promotional source unwind independently.
			err = s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
				CustomerID: customer.GetID(),
				PatchesByChargeID: map[string]charges.Patch{
					usage.ID: lo.Must(meta.NewPatchDelete(meta.NewPatchDeleteInput{ChangeSource: billing.ChangeSourceSystem, Policy: meta.RefundAsCreditsDeletePolicy})),
				},
			})
			s.Require().NoError(err)
			s.requireAccountBalance(business.EarningsAccount, ledger.RouteFilter{Currency: tokens.Reference()}, 0, "refunded earnings")
			s.requireAccountBalance(accounts.AccruedAccount, ledger.RouteFilter{Currency: tokens.Reference()}, 0, "refunded accrued")
			s.requireAccountBalance(accounts.FBOAccount, ledger.RouteFilter{Currency: tokens.Reference()}, 10, "refunded credit")
			roots, err = s.LineageService.LoadLineagesByCustomer(ctx, lineage.LoadLineagesByCustomerInput{Namespace: ns, CustomerID: customer.ID, Currency: tokens.Reference()})
			s.Require().NoError(err)
			for _, root := range roots {
				s.Empty(root.Segments)
			}

			// A retry after correction cannot recognize refunded value.
			retry, err := s.RevenueRecognizer.RecognizeEarnings(ctx, recognizer.RecognizeEarningsInput{
				CustomerID: customer.GetID(), Currency: tokens, At: clock.Now(),
			})
			s.Require().NoError(err)
			s.Zero(retry.RecognizedAmount.InexactFloat64())
			s.Empty(retry.LedgerGroupID)
		})
	}
}
