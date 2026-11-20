package testutils

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/invoicing/legacy/splitlinegroup"
	splitlinegroupadapter "github.com/openmeterio/openmeter/openmeter/billing/invoicing/legacy/splitlinegroup/adapter"
	splitlinegroupservice "github.com/openmeterio/openmeter/openmeter/billing/invoicing/legacy/splitlinegroup/service"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
)

func NewTestEnv(t testing.TB, client *entdb.Client, logger *slog.Logger) splitlinegroup.Service {
	t.Helper()

	adapter, err := splitlinegroupadapter.New(splitlinegroupadapter.Config{
		Client: client,
		Logger: logger,
	})
	require.NoError(t, err)

	service, err := splitlinegroupservice.New(splitlinegroupservice.Config{
		Adapter: adapter,
		Logger:  logger,
	})
	require.NoError(t, err)

	return service
}
