package transactions

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

func TestAccruedCollectionSeparatesOriginAndLegacyBalances(t *testing.T) {
	// given legacy and origin-tracked collections with the same route and charge provenance.
	env := newTransactionsTestEnv(t)
	costBasis := alpacadecimal.NewFromInt(1)
	source, spend, origin := testChargeID(1), testChargeID(2), ulid.Make().String()
	fbo := env.fundPriorityWithCostBasis(t, 1, 50, &costBasis, &source)

	for _, tc := range []struct {
		name   string
		origin *string
		amount int64
	}{
		{
			name:   "legacy",
			amount: 20,
		},
		{
			name:   "origin-tracked",
			origin: &origin,
			amount: 30,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inputs := env.resolve(t, TransferCustomerFBOToAccruedTemplate{
				At:       env.Now(),
				Currency: env.CurrencyReference(),
				Sources: []PostingAmount{{
					Address: fbo.Address(),
					Amount:  alpacadecimal.NewFromInt(tc.amount),
					Identity: ledger.EntryIdentityParts{Provenance: ledger.Provenance{
						SourceChargeID:     &source,
						SpendChargeID:      &spend,
						CollectionOriginID: tc.origin,
					}},
				}},
			})
			group, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), GroupInputs(env.Namespace, nil, inputs...))
			require.NoError(t, err)

			// Legacy template correction accepts originless entries; tracked entries need the collector.
			_, err = CorrectTransaction(
				t.Context(),
				env.resolverDeps(),
				CorrectionInput{
					At:                  env.Now(),
					Amount:              alpacadecimal.NewFromInt(1),
					OriginalTransaction: group.Transactions()[0],
				},
			)
			if tc.origin == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "origin-tracked transactions require provenance-aware correction")
			}
		})
	}

	// when selecting each recognition pool against the combined accrued balance.
	for _, tracked := range []bool{false, true} {
		selected, err := collectFromAttributableCustomerAccrued(
			t.Context(),
			env.resolverDeps(),
			collectFromAttributableCustomerAccruedInput{
				CustomerID:    env.CustomerID,
				Currency:      env.CurrencyReference(),
				Target:        alpacadecimal.NewFromInt(50),
				OriginTracked: tracked,
				AsOf:          env.Now(),
			},
		)
		require.NoError(t, err)

		// then only that pool contributes, even though both pools share the same subaccount.
		require.Len(t, selected, 1)

		if tracked {
			require.Equal(t, float64(30), selected[0].amount.InexactFloat64())
			require.Equal(t, &origin, selected[0].identity.CollectionOriginID)
		} else {
			require.Equal(t, float64(20), selected[0].amount.InexactFloat64())
			require.Nil(t, selected[0].identity.CollectionOriginID)
		}
	}
}
