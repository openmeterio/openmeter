package common

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/wire"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	taxcodeadapter "github.com/openmeterio/openmeter/openmeter/taxcode/adapter"
	taxcodeservice "github.com/openmeterio/openmeter/openmeter/taxcode/service"
	taxcodehooks "github.com/openmeterio/openmeter/openmeter/taxcode/service/hooks"
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

// TaxCodePlanHook prevents deleting tax codes that are still referenced by plans.
type TaxCodePlanHook taxcodehooks.PlanHook

// NewTaxCodePlanServiceHook builds the plan-reference hook and registers it
// on the tax code service. It depends on both the plan and tax code services so wire constructs
// it only after both exist, avoiding a construction cycle (plan already depends on tax code).
func NewTaxCodePlanServiceHook(
	planService plan.Service,
	taxCodeService taxcode.Service,
) (TaxCodePlanHook, error) {
	h, err := taxcodehooks.NewPlanHook(taxcodehooks.PlanHookConfig{
		PlanService: planService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tax code plan hook: %w", err)
	}

	taxCodeService.RegisterHooks(h)

	return h, nil
}

func NewTaxCodeNamespaceHandler(
	logger *slog.Logger,
	service taxcode.Service,
	repository taxcode.Repository,
	cfg config.TaxCodeConfiguration,
) (*taxcode.NamespaceHandler, error) {
	seeds := lo.Map(cfg.Seeds, func(s config.TaxCodeSeed, _ int) taxcode.SeedEntry {
		return taxcode.SeedEntry{
			Key:         strings.TrimSpace(s.Key),
			Name:        strings.TrimSpace(s.Name),
			Description: s.Description,
			AppMappings: lo.Map(s.AppMappings, func(m config.TaxCodeAppMapping, _ int) taxcode.TaxCodeAppMapping {
				return taxcode.TaxCodeAppMapping{
					AppType: app.AppType(strings.TrimSpace(m.AppType)),
					TaxCode: strings.TrimSpace(m.TaxCode),
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
