package subscriptiontestutils

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/subscription/price"
)

func NewPriceConnector(t *testing.T, dbDeps *DBDeps) price.Connector {
	t.Helper()
	repo := price.NewRepository(dbDeps.dbClient)
	return price.NewConnector(repo)
}

func GetFlatPrice(amount int64) plan.Price {
	return plan.NewPriceFrom(plan.FlatPrice{
		Amount:      alpacadecimal.NewFromInt(amount),
		PriceMeta:   plan.PriceMeta{Type: plan.FlatPriceType},
		PaymentTerm: lo.ToPtr(plan.InAdvancePaymentTerm),
	})
}
