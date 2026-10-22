package testutils

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"

	advancetestutils "github.com/openmeterio/openmeter/openmeter/ledger/advance/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/fbo"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	omtestutils "github.com/openmeterio/openmeter/openmeter/testutils"
)

func NewService(t testing.TB, deps ledgertestutils.Deps, breakageService breakage.Service) fbo.Service {
	t.Helper()
	service, err := fbo.NewService(fbo.Config{
		Logger:        omtestutils.NewDiscardLogger(t),
		Meter:         noop.NewMeterProvider().Meter("test"),
		Advance:       advancetestutils.NewService(t, deps, breakageService),
		Breakage:      breakageService,
		AccountLocker: deps.AccountService,
		Dependencies: transactions.ResolverDependencies{
			AccountService: deps.ResolversService,
			AccountCatalog: deps.AccountService,
			BalanceQuerier: deps.HistoricalLedger,
		},
	})
	require.NoError(t, err)

	return service
}
