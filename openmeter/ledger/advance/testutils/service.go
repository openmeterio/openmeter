package testutils

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger/advance"
	advanceservice "github.com/openmeterio/openmeter/openmeter/ledger/advance/service"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	omtestutils "github.com/openmeterio/openmeter/openmeter/testutils"
)

func NewService(t testing.TB, deps ledgertestutils.Deps) advance.Service {
	t.Helper()

	service, err := advanceservice.New(advanceservice.Config{
		Logger:          omtestutils.NewDiscardLogger(t),
		Ledger:          deps.HistoricalLedger,
		BalanceQuerier:  deps.HistoricalLedger,
		AccountResolver: deps.ResolversService,
	})
	require.NoError(t, err)

	return service
}
