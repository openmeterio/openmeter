package common

import (
	"log/slog"

	"github.com/google/wire"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	taxcodeadapter "github.com/openmeterio/openmeter/openmeter/taxcode/adapter"
	taxcodeservice "github.com/openmeterio/openmeter/openmeter/taxcode/service"
)

var TaxCode = wire.NewSet(
	NewTaxCodeAdapter,
	NewTaxCodeService,
)

func NewTaxCodeAdapter(logger *slog.Logger, db *entdb.Client) (taxcode.Repository, error) {
	return taxcodeadapter.New(taxcodeadapter.Config{
		Client: db,
		Logger: logger.With("subsystem", "taxcode"),
	})
}

func NewTaxCodeService(logger *slog.Logger, adapter taxcode.Repository) taxcode.Service {
	return taxcodeservice.New(adapter, logger.With("subsystem", "taxcode"))
}
