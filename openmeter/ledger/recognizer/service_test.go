package recognizer_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	lineageadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/adapter"
	lineageservice "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger/recognizer"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type recognizerTestEnv struct {
	*ledgertestutils.IntegrationEnv
	recognizer recognizer.Service
	lineage    lineage.Service
}

func newRecognizerTestEnv(t *testing.T) *recognizerTestEnv {
	t.Helper()

	base := ledgertestutils.NewIntegrationEnv(t, "recognizer")
	deps := transactions.ResolverDependencies{
		AccountService:    base.Deps.ResolversService,
		SubAccountService: base.Deps.AccountService,
	}

	lngeAdapter, err := lineageadapter.New(lineageadapter.Config{
		Client: base.DB,
	})
	require.NoError(t, err)

	lngeSvc, err := lineageservice.New(lineageservice.Config{
		Adapter: lngeAdapter,
	})
	require.NoError(t, err)

	recSvc, err := recognizer.NewService(recognizer.Config{
		Ledger:             base.Deps.HistoricalLedger,
		Dependencies:       deps,
		Lineage:            lngeSvc,
		TransactionManager: enttx.NewCreator(base.DB),
	})
	require.NoError(t, err)

	return &recognizerTestEnv{
		IntegrationEnv: base,
		recognizer:     recSvc,
		lineage:        lngeSvc,
	}
}

func testID() string {
	return ulid.Make().String()
}

func (e *recognizerTestEnv) resolverDeps() transactions.ResolverDependencies {
	return transactions.ResolverDependencies{
		AccountService:    e.Deps.ResolversService,
		SubAccountService: e.Deps.AccountService,
	}
}

func (e *recognizerTestEnv) resolveAndCommit(t *testing.T, templates ...transactions.TransactionTemplate) {
	t.Helper()

	inputs, err := transactions.ResolveTransactions(
		t.Context(),
		e.resolverDeps(),
		transactions.ResolutionScope{
			CustomerID: e.CustomerID,
			Namespace:  e.Namespace,
		},
		templates...,
	)
	require.NoError(t, err)

	_, err = e.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(e.Namespace, nil, inputs...))
	require.NoError(t, err)
}

// ensureCharge creates a minimal charge record in the DB if it doesn't exist.
func (e *recognizerTestEnv) ensureCharge(t *testing.T, chargeID string) {
	t.Helper()

	exists, err := e.DB.Charge.Get(t.Context(), chargeID)
	if err == nil && exists != nil {
		return
	}

	_, err = e.DB.Charge.Create().
		SetID(chargeID).
		SetNamespace(e.Namespace).
		SetType(meta.ChargeTypeFlatFee).
		Save(t.Context())
	require.NoError(t, err)
}

// createLineageForRealization creates a lineage record for a realization, mimicking
// what the charges system does after credit allocation.
func (e *recognizerTestEnv) createLineageForRealization(t *testing.T, chargeID, realizationID string, amount alpacadecimal.Decimal, originKind creditrealization.LineageOriginKind) {
	t.Helper()

	e.ensureCharge(t, chargeID)

	state := creditrealization.InitialLineageSegmentState(originKind)

	err := e.lineage.CreateInitialLineages(t.Context(), lineage.CreateInitialLineagesInput{
		Namespace:  e.Namespace,
		ChargeID:   chargeID,
		CustomerID: e.CustomerID.ID,
		Currency:   e.Currency,
		Realizations: creditrealization.Realizations{
			{
				CreateInput: creditrealization.CreateInput{
					ID:     realizationID,
					Amount: amount,
					Type:   creditrealization.TypeAllocation,
					ServicePeriod: timeutil.ClosedPeriod{
						From: clock.Now().Add(-24 * time.Hour),
						To:   clock.Now(),
					},
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: "test-group-" + realizationID,
					},
					Annotations: creditrealization.LineageAnnotations(originKind),
				},
			},
		},
	})
	require.NoError(t, err)

	_ = state
}

func TestRecognizeEarnings_IdempotencyOnUnchangedState(t *testing.T) {
	env := newRecognizerTestEnv(t)
	costBasis := alpacadecimal.NewFromInt(1)

	chargeID := testID()
	realID := testID()

	// Set up accrued balance and lineage.
	env.resolveAndCommit(t, transactions.TransferCustomerReceivableToAccruedTemplate{
		At: env.Now(), Amount: alpacadecimal.NewFromInt(50), Currency: env.Currency, CostBasis: &costBasis,
	})
	env.createLineageForRealization(t, chargeID, realID, alpacadecimal.NewFromInt(50), creditrealization.LineageOriginKindRealCredit)

	// First recognition.
	result1, err := env.recognizer.RecognizeEarnings(t.Context(), recognizer.RecognizeEarningsInput{
		CustomerID: env.CustomerID,
		At:         clock.Now(),
		Currency:   env.Currency,
	})
	require.NoError(t, err)
	require.True(t, result1.RecognizedAmount.Equal(alpacadecimal.NewFromInt(50)))
	require.NotEmpty(t, result1.LedgerGroupID)

	// Second recognition with unchanged state should be a no-op.
	result2, err := env.recognizer.RecognizeEarnings(t.Context(), recognizer.RecognizeEarningsInput{
		CustomerID: env.CustomerID,
		At:         clock.Now(),
		Currency:   env.Currency,
	})
	require.NoError(t, err)
	require.True(t, result2.RecognizedAmount.IsZero())
	require.Empty(t, result2.LedgerGroupID)

	// Balances should be stable.
	require.True(t, env.SumBalance(t, env.AccruedSubAccountWithCostBasis(t, &costBasis)).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, env.EarningsSubAccountWithCostBasis(t, &costBasis)).Equal(alpacadecimal.NewFromInt(50)))
}

func TestRecognizeEarnings_DeterministicAllocationAndSegmentTransition(t *testing.T) {
	env := newRecognizerTestEnv(t)
	costBasis := alpacadecimal.NewFromInt(1)
	chargeID := testID()
	realA := testID()
	realB := testID()

	// Set up accrued balance and two lineages.
	env.resolveAndCommit(t, transactions.TransferCustomerReceivableToAccruedTemplate{
		At: env.Now(), Amount: alpacadecimal.NewFromInt(70), Currency: env.Currency, CostBasis: &costBasis,
	})
	env.createLineageForRealization(t, chargeID, realA, alpacadecimal.NewFromInt(30), creditrealization.LineageOriginKindRealCredit)
	env.createLineageForRealization(t, chargeID, realB, alpacadecimal.NewFromInt(40), creditrealization.LineageOriginKindRealCredit)

	result, err := env.recognizer.RecognizeEarnings(t.Context(), recognizer.RecognizeEarningsInput{
		CustomerID: env.CustomerID,
		At:         clock.Now(),
		Currency:   env.Currency,
	})
	require.NoError(t, err)
	require.True(t, result.RecognizedAmount.Equal(alpacadecimal.NewFromInt(70)))

	// Verify segments transitioned to earnings_recognized.
	lineages, err := env.lineage.LoadLineagesByCustomer(t.Context(), lineage.LoadLineagesByCustomerInput{
		Namespace:  env.Namespace,
		CustomerID: env.CustomerID.ID,
		Currency:   env.Currency,
	})
	require.NoError(t, err)

	for _, l := range lineages {
		for _, seg := range l.Segments {
			require.Equal(t, creditrealization.LineageSegmentStateEarningsRecognized, seg.State,
				"segment %s should be earnings_recognized", seg.ID)
			require.NotNil(t, seg.BackingTransactionGroupID)
			require.NotNil(t, seg.SourceState)
			require.Equal(t, creditrealization.LineageSegmentStateRealCredit, *seg.SourceState)
		}
	}
}
