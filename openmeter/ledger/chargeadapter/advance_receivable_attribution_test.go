package chargeadapter_test

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
)

func TestCreditPurchaseAttributesReceivableWithoutLineage(t *testing.T) {
	for _, scenario := range []struct {
		name  string
		roots []lineage.Lineage
	}{
		{name: "nil"},
		{name: "empty", roots: []lineage.Lineage{}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// Given 40 of journal-only receivable, with no accrued or lineage.
			env := newCreditPurchaseHandlerTestEnv(t)
			env.createReceivableOnlyExposure(t, advanceExposureInput{Currency: env.currency, Amount: alpacadecimal.NewFromInt(40)})
			costBasis := alpacadecimal.NewFromFloat(0.5)
			purchase := env.newExternalCharge(alpacadecimal.NewFromInt(25), costBasis)

			// When 25 of credit is purchased, nil and empty roots behave identically.
			result, err := env.handler.OnCreditPurchaseInitiated(t.Context(), creditpurchase.CreditGrantInput{
				Charge: purchase, AdvanceLineages: scenario.roots,
			})
			require.NoError(t, err)

			// Then 25 is attributed and 15 remains; no accrued backing is invented.
			require.Empty(t, result.BackfillAllocations)
			require.Equal(t, float64(-15), env.sumBalance(t, env.unknownReceivableSubAccount(t)).InexactFloat64())
			require.Equal(t, float64(-25), env.sumBalance(t, env.receivableSubAccount(t, costBasis)).InexactFloat64())
			require.Equal(t, float64(0), env.sumBalance(t, env.accruedSubAccount(t, costBasis)).InexactFloat64())
			require.Equal(t, float64(0), env.sumBalance(t, env.fboSubAccount(t, costBasis)).InexactFloat64())
			require.Equal(t, []string{transactions.TemplateCode(transactions.AttributeCustomerAdvanceReceivableCostBasisTemplate{})}, env.transactionTemplateCodes(t, result.TransactionGroupID))
		})
	}
}

func TestCreditPurchaseAttributesRemainingReceivableAfterAccruedBackfill(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)
	spendChargeID := ulid.Make().String()
	taxCode := ulid.Make().String()
	// Given 20 of receivable but only 5 of accrued with uncovered lineage.
	// The additional receivable is a standalone journal posting, not a normal collection.
	env.createAdvance(t, advanceExposureInput{
		Currency: env.currency, Amount: alpacadecimal.NewFromInt(5), SpendChargeID: &spendChargeID, TaxCode: &taxCode,
	})
	env.createReceivableOnlyExposure(t, advanceExposureInput{
		Currency: env.currency, Amount: alpacadecimal.NewFromInt(15), SpendChargeID: &spendChargeID,
	})

	// When a purchase of 15 arrives, 5 backs accrued and 10 only attributes receivable.
	firstCostBasis := alpacadecimal.NewFromFloat(0.5)
	firstPurchase := env.newExternalCharge(alpacadecimal.NewFromInt(15), firstCostBasis)
	firstPurchase.ID = ulid.Make().String()
	first, err := env.grantCredits(t, firstPurchase)
	require.NoError(t, err)
	require.Len(t, first.BackfillAllocations, 1)
	require.Equal(t, float64(5), first.BackfillAllocations[0].Amount.InexactFloat64())
	require.Equal(t, float64(-5), env.sumBalance(t, env.unknownReceivableSubAccount(t)).InexactFloat64())
	require.Equal(t, float64(-15), env.sumBalance(t, env.receivableSubAccount(t, firstCostBasis)).InexactFloat64())
	require.Equal(t, float64(5), env.sumBalance(t, env.AccruedSubAccountWithCostBasisAndTaxCode(t, &firstCostBasis, &taxCode)).InexactFloat64())
	require.Equal(t, float64(0), env.sumBalance(t, env.accruedSubAccount(t, firstCostBasis)).InexactFloat64())
	require.Equal(t, float64(0), env.sumBalance(t, env.fboSubAccount(t, firstCostBasis)).InexactFloat64())
	// Tax stays on accrued; both receivable portions share one attribution leg.
	require.ElementsMatch(t, []string{
		transactions.TemplateCode(transactions.AttributeCustomerAdvanceReceivableCostBasisTemplate{}),
		transactions.TemplateCode(transactions.TranslateCustomerAccruedCostBasisTemplate{}),
	}, env.transactionTemplateCodes(t, first.TransactionGroupID))

	// Then a second purchase of 10 attributes only the remaining 5 and issues 5 into FBO.
	secondCostBasis := alpacadecimal.NewFromFloat(0.8)
	secondPurchase := env.newExternalCharge(alpacadecimal.NewFromInt(10), secondCostBasis)
	secondPurchase.ID = ulid.Make().String()
	second, err := env.grantCredits(t, secondPurchase)
	require.NoError(t, err)
	require.Empty(t, second.BackfillAllocations)
	require.Equal(t, float64(0), env.sumBalance(t, env.unknownReceivableSubAccount(t)).InexactFloat64())
	require.Equal(t, float64(5), env.sumBalance(t, env.fboSubAccount(t, secondCostBasis)).InexactFloat64())
	require.Equal(t, float64(0), env.sumBalance(t, env.accruedSubAccount(t, secondCostBasis)).InexactFloat64())
	env.requireAccountSourceSpendBucketAmounts(t, env.receivableSubAccount(t, secondCostBasis).AccountID().ID, map[string]float64{
		sourceSpendChargeKey(&firstPurchase.ID, &spendChargeID):  -15,
		sourceSpendChargeKey(&secondPurchase.ID, &spendChargeID): -5,
		sourceSpendChargeKey(&secondPurchase.ID, nil):            -5,
	})
	roots, err := env.lineage.LoadLineagesByCustomer(t.Context(), lineage.LoadLineagesByCustomerInput{
		Namespace: env.Namespace, CustomerID: env.CustomerID.ID, Currency: env.currency.Reference(),
	})
	require.NoError(t, err)
	require.Len(t, roots, 1)
	require.Len(t, roots[0].Segments, 1)
	segment := roots[0].Segments[0]
	require.Equal(t, creditrealization.LineageSegmentStateAdvanceBackfilled, segment.State)
	require.Equal(t, float64(5), segment.Amount.InexactFloat64())
	require.NotNil(t, segment.BackingTransactionGroupID)
	require.Equal(t, first.TransactionGroupID, *segment.BackingTransactionGroupID)
}

func TestCreditPurchaseReceivableOnlyAttributionPreservesLegacyFeatureRoutes(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)
	// Given nil-spend receivable of 20 for API, 30 for storage, and 10 unrestricted.
	for _, exposure := range []struct {
		amount   int64
		features []string
	}{
		{20, []string{"api-calls"}},
		{30, []string{"storage"}},
		{10, nil},
	} {
		env.createReceivableOnlyExposure(t, advanceExposureInput{
			Currency: env.currency, Amount: alpacadecimal.NewFromInt(exposure.amount), Features: exposure.features,
		})
	}

	// When 25 of API-restricted credit arrives without any accrued or lineage.
	costBasis := alpacadecimal.NewFromFloat(0.5)
	purchase := env.newExternalCharge(alpacadecimal.NewFromInt(25), costBasis)
	purchase.Intent.FeatureFilters = creditpurchase.FeatureFilters{"api-calls"}
	result, err := env.grantCredits(t, purchase)
	require.NoError(t, err)

	// Then only the API receivable is attributed; the excess 5 becomes restricted FBO.
	require.Empty(t, result.BackfillAllocations)
	require.Equal(t, float64(0), env.sumBalance(t, env.unknownReceivableSubAccountWithFeatures(t, []string{"api-calls"})).InexactFloat64())
	require.Equal(t, float64(-30), env.sumBalance(t, env.unknownReceivableSubAccountWithFeatures(t, []string{"storage"})).InexactFloat64())
	require.Equal(t, float64(-10), env.sumBalance(t, env.unknownReceivableSubAccount(t)).InexactFloat64())
	require.Equal(t, float64(-25), env.sumBalance(t, env.receivableSubAccountWithFeatures(t, costBasis, []string{"api-calls"})).InexactFloat64())
	require.Equal(t, float64(5), env.sumBalance(t, env.fboSubAccountWithFeatures(t, costBasis, []string{"api-calls"})).InexactFloat64())
}

// Standalone receivable postings exercise the historical receivable-only path
// without inventing an accrued collection or lineage for the unmatched amount.
func (e *creditPurchaseHandlerTestEnv) createReceivableOnlyExposure(t *testing.T, input advanceExposureInput) {
	t.Helper()
	inputs, err := transactions.ResolveTransactions(t.Context(), transactions.ResolverDependencies{
		AccountService: e.Deps.ResolversService, AccountCatalog: e.Deps.AccountService, BalanceQuerier: e.Deps.HistoricalLedger,
	}, transactions.ResolutionScope{CustomerID: e.CustomerID, Namespace: e.Namespace}, transactions.IssueCustomerReceivableTemplate{
		At: e.Now(), Amount: input.Amount, Currency: input.Currency.Reference(), Features: input.Features, SpendChargeID: input.SpendChargeID,
	})
	require.NoError(t, err)
	_, err = e.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(e.Namespace, nil, inputs...))
	require.NoError(t, err)
}
