package adapter

import (
	dbchargecreditpurchase "github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchase"
	dbcustomcurrency "github.com/openmeterio/openmeter/openmeter/ent/db/customcurrency"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func hasCustomCurrencyCode(namespace string, codes ...currencyx.Code) predicate.ChargeCreditPurchase {
	// Retiring a catalog currency must not hide its grants from currency filters.
	return dbchargecreditpurchase.HasCustomCurrencyWith(
		dbcustomcurrency.CodeIn(codes...),
		dbcustomcurrency.Namespace(namespace),
	)
}
