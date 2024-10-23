package subscriptiontestutils

import (
	"testing"

	"github.com/openmeterio/openmeter/openmeter/subscription/price"
)

func NewPriceConnector(t *testing.T, dbDeps *DBDeps) price.Connector {
	t.Helper()
	repo := price.NewRepository(dbDeps.dbClient)
	return price.NewConnector(repo)
}
