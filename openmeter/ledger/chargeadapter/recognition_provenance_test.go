package chargeadapter_test

import (
	"slices"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	chargeflatfee "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCustomCurrencyRecognizedCorrectionPreservesSpend(t *testing.T) {
	for _, tc := range []struct {
		name           string
		differentBasis bool
		sources        int
	}{
		{name: "same cost basis", sources: 1},
		{name: "different cost bases", differentBasis: true, sources: 1},
		{name: "multiple sources per allocation", sources: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := newFlatFeeHandlerTestEnv(t)
			custom := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
			env.currency = custom
			env.CustomCurrency = &custom
			env.Currency = custom.GetCode()
			amount := alpacadecimal.NewFromInt(30)
			basis := alpacadecimal.NewFromFloat(0.5)
			fiat := currencyx.Code("USD")
			deps := transactions.ResolverDependencies{AccountService: env.Deps.ResolversService, AccountCatalog: env.Deps.AccountService, BalanceQuerier: env.Deps.HistoricalLedger}
			charges := map[string]chargeflatfee.Charge{}
			// given: two real credit-backed spends recognized together for one managed currency.
			for i := range 2 {
				if i == 1 && tc.differentBasis {
					basis = alpacadecimal.NewFromFloat(0.8)
				}
				for range tc.sources {
					sourceID := ulid.Make().String()
					inputs, err := transactions.ResolveTransactions(t.Context(), deps, transactions.ResolutionScope{CustomerID: env.CustomerID, Namespace: env.Namespace}, transactions.IssueCustomerReceivableTemplate{
						At: env.Now(), Amount: amount.Div(alpacadecimal.NewFromInt(int64(tc.sources))), Currency: custom.Reference(), CostBasis: &basis, CostBasisCurrency: &fiat, SourceChargeID: &sourceID,
					})
					require.NoError(t, err)
					_, err = env.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(env.Namespace, nil, inputs...))
					require.NoError(t, err)
				}
				input := env.newAssignmentInputWithMode(amount, productcatalog.CreditOnlySettlementMode)
				input.Charge.ID = ulid.Make().String()
				allocations, err := env.handler.OnAllocateCredits(t.Context(), input)
				require.NoError(t, err)
				require.Len(t, allocations, 1)
				charge := env.newChargeWithCreditRealizationsAndAccruedUsage(allocations, alpacadecimal.Zero)
				charge.ID = input.Charge.ID
				intent := charge.Intent.GetBaseIntent()
				intent.SettlementMode = productcatalog.CreditOnlySettlementMode
				charge.Intent = intent.AsOverridableIntent()
				env.createInitialLineages(t, charge.ID, charge.Realizations.CurrentRun.CreditRealizations)
				charges[charge.ID] = charge
			}
			groupID := env.recognizeCreditAccrued(t, amount.Mul(alpacadecimal.NewFromInt(2)))
			recognitionGroup, err := env.Deps.HistoricalLedger.GetTransactionGroup(t.Context(), models.NamespacedID{Namespace: env.Namespace, ID: groupID})
			require.NoError(t, err)
			var entries []ledger.Entry
			for _, tx := range recognitionGroup.Transactions() {
				entries = append(entries, tx.Entries()...)
			}
			var spends []string
			for _, entry := range entries {
				if entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerAccrued && entry.Amount().IsNegative() {
					require.NotNil(t, entry.SpendChargeID())
					if !slices.Contains(spends, *entry.SpendChargeID()) {
						spends = append(spends, *entry.SpendChargeID())
					}
				}
			}
			require.Len(t, spends, 2)
			// Correct the first spend in the shared recognition transaction.
			charge := charges[spends[0]]
			segments := env.assertRecognizedSegments(t, charge.Realizations.CurrentRun.CreditRealizations, groupID)
			// when: repeated partial corrections consume only this spend's recognized slices.
			for _, corrected := range []int64{10, 20} {
				corrections, err := env.handler.OnCorrectCreditAllocations(t.Context(), chargeflatfee.CorrectCreditAllocationsInput{
					Charge: charge, BookedAt: env.Now(),
					Corrections:                  creditrealization.CorrectionRequest{{Allocation: charge.Realizations.CurrentRun.CreditRealizations[0], Amount: alpacadecimal.NewFromInt(-corrected)}},
					LineageSegmentsByRealization: segments,
				})
				require.NoError(t, err)
				require.Len(t, corrections, 1)
				correctionGroup, err := env.Deps.HistoricalLedger.GetTransactionGroup(t.Context(), models.NamespacedID{Namespace: env.Namespace, ID: corrections[0].LedgerTransaction.TransactionGroupID})
				require.NoError(t, err)
				var correctionEntries []ledger.Entry
				for _, tx := range correctionGroup.Transactions() {
					correctionEntries = append(correctionEntries, tx.Entries()...)
				}
				for _, entry := range correctionEntries {
					if entry.PostingAddress().AccountType() == ledger.AccountTypeEarnings && entry.Amount().IsNegative() {
						require.NotNil(t, entry.SpendChargeID())
						require.Equal(t, charge.ID, *entry.SpendChargeID(), "correction reversed another spend's recognized earnings")
					}
				}
			}
			// then: the corrected spend has no accrued/earned value and the other retains all 30.
			for _, accountType := range []ledger.AccountType{ledger.AccountTypeCustomerAccrued, ledger.AccountTypeEarnings} {
				balances := map[string]alpacadecimal.Decimal{}
				page, err := env.Deps.HistoricalLedger.ListTransactions(t.Context(), ledger.ListTransactionsInput{Namespace: env.Namespace, Limit: 100})
				require.NoError(t, err)
				require.Nil(t, page.NextCursor)
				for _, tx := range page.Items {
					for _, entry := range tx.Entries() {
						if entry.PostingAddress().AccountType() == accountType && entry.SpendChargeID() != nil {
							balances[*entry.SpendChargeID()] = balances[*entry.SpendChargeID()].Add(entry.Amount())
						}
					}
				}
				require.Equal(t, float64(0), balances[charge.ID].InexactFloat64())
				expected := float64(0)
				if accountType == ledger.AccountTypeEarnings {
					expected = 30
				}
				require.Equal(t, expected, balances[spends[1]].InexactFloat64())
			}
		})
	}
}
