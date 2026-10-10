package testutils

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger/advance"
	advanceservice "github.com/openmeterio/openmeter/openmeter/ledger/advance/service"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	omtestutils "github.com/openmeterio/openmeter/openmeter/testutils"
)

func NewService(t testing.TB, deps ledgertestutils.Deps, breakageService breakage.Service) advance.Service {
	t.Helper()

	service, err := advanceservice.New(advanceservice.Config{
		Logger:          omtestutils.NewDiscardLogger(t),
		Ledger:          deps.HistoricalLedger,
		BalanceQuerier:  deps.HistoricalLedger,
		AccountResolver: deps.ResolversService,
		AccountCatalog:  deps.AccountService,
		Breakage:        breakageService,
	})
	require.NoError(t, err)

	return service
}
