package common

import (
	"log/slog"

	"github.com/google/wire"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	taxcodeadapter "github.com/openmeterio/openmeter/openmeter/taxcode/adapter"
	taxcodeservice "github.com/openmeterio/openmeter/openmeter/taxcode/service"
)

var TaxCode = wire.NewSet(
	NewTaxCodeAdapter,
	NewTaxCodeService,
)

var TaxCodeNamespaceHandler = wire.NewSet(
	wire.FieldsOf(new(config.Configuration), "TaxCode"),
	NewTaxCodeNamespaceHandler,
)

func NewTaxCodeAdapter(logger *slog.Logger, db *entdb.Client) (taxcode.Repository, error) {
	return taxcodeadapter.New(taxcodeadapter.Config{
		Client: db,
		Logger: logger.With("subsystem", "taxcode"),
	})
}

func NewTaxCodeService(
	logger *slog.Logger,
	adapter taxcode.Repository,
) (taxcode.Service, error) {
	return taxcodeservice.New(taxcodeservice.Config{
		Adapter: adapter,
		Logger:  logger.With("subsystem", "taxcode"),
	})
}

func NewTaxCodeNamespaceHandler(
	logger *slog.Logger,
	service taxcode.Service,
	repository taxcode.Repository,
	cfg config.TaxCodeConfiguration,
) (*taxcode.NamespaceHandler, error) {
	seeds := lo.Map(cfg.Seeds, func(s config.TaxCodeSeed, _ int) taxcode.SeedEntry {
		return taxcode.SeedEntry{
			Key:         s.Key,
			Name:        s.Name,
			Description: s.Description,
			AppMappings: lo.Map(s.AppMappings, func(m config.TaxCodeAppMapping, _ int) taxcode.TaxCodeAppMapping {
				return taxcode.TaxCodeAppMapping{
					AppType: app.AppType(m.AppType),
					TaxCode: m.TaxCode,
				}
			}),
			DefaultInvoicing:   s.DefaultInvoicing,
			DefaultCreditGrant: s.DefaultCreditGrant,
		}
	})

	return taxcode.NewNamespaceHandler(taxcode.NamespaceHandlerConfig{
		Logger:             logger.With("subsystem", "taxcode", "component", "namespace-handler"),
		Service:            service,
		Seeds:              seeds,
		TransactionManager: repository,
	})
}
