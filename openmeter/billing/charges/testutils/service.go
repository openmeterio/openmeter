package testutils

import (
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargesadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	creditpurchaseadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/adapter"
	creditpurchaselineengine "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/lineengine"
	creditpurchaseservice "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeeadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/adapter"
	flatfeeservice "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service"
	lineageadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/adapter"
	lineageservice "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/service"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	chargesservice "github.com/openmeterio/openmeter/openmeter/billing/charges/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/adapter"
	usagebasedservice "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service"
	billingratingservice "github.com/openmeterio/openmeter/openmeter/billing/rating/service"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

type Config struct {
	Client *entdb.Client
	Logger *slog.Logger

	BillingService     billing.Service
	FeatureService     feature.FeatureConnector
	StreamingConnector streaming.Connector

	FlatFeeHandler        flatfee.Handler
	CreditPurchaseHandler creditpurchase.Handler
	UsageBasedHandler     usagebased.Handler
}

func (c Config) Validate() error {
	var errs []error

	if c.Client == nil {
		errs = append(errs, fmt.Errorf("client is required"))
	}

	if c.BillingService == nil {
		errs = append(errs, fmt.Errorf("billing service is required"))
	}

	if c.FeatureService == nil {
		errs = append(errs, fmt.Errorf("feature service is required"))
	}

	if c.StreamingConnector == nil {
		errs = append(errs, fmt.Errorf("streaming connector is required"))
	}

	if c.FlatFeeHandler == nil {
		errs = append(errs, fmt.Errorf("flat fee handler is required"))
	}

	if c.CreditPurchaseHandler == nil {
		errs = append(errs, fmt.Errorf("credit purchase handler is required"))
	}

	if c.UsageBasedHandler == nil {
		errs = append(errs, fmt.Errorf("usage based handler is required"))
	}

	return errors.Join(errs...)
}

type Services struct {
	ChargesService        charges.Service
	UsageBasedService     usagebased.Service
	FlatFeeService        flatfee.Service
	CreditPurchaseService creditpurchase.Service
}

// NewServices constructs the charges stack from external dependencies and handlers.
func NewServices(t testing.TB, config Config) (*Services, error) {
	t.Helper()

	if err := config.Validate(); err != nil {
		return nil, err
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	metaAdapter, err := metaadapter.New(metaadapter.Config{
		Client: config.Client,
		Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("creating meta adapter: %w", err)
	}

	locker, err := lockr.NewLocker(&lockr.LockerConfig{
		Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("creating locker: %w", err)
	}

	lineageAdapter, err := lineageadapter.New(lineageadapter.Config{
		Client: config.Client,
	})
	if err != nil {
		return nil, fmt.Errorf("creating lineage adapter: %w", err)
	}

	lineageService, err := lineageservice.New(lineageservice.Config{
		Adapter: lineageAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("creating lineage service: %w", err)
	}

	flatFeeAdapter, err := flatfeeadapter.New(flatfeeadapter.Config{
		Client:      config.Client,
		Logger:      logger,
		MetaAdapter: metaAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("creating flat fee adapter: %w", err)
	}

	flatFeeService, err := flatfeeservice.New(flatfeeservice.Config{
		Adapter:       flatFeeAdapter,
		Handler:       config.FlatFeeHandler,
		Lineage:       lineageService,
		MetaAdapter:   metaAdapter,
		Locker:        locker,
		RatingService: billingratingservice.New(),
	})
	if err != nil {
		return nil, fmt.Errorf("creating flat fee service: %w", err)
	}

	if err := config.BillingService.RegisterLineEngine(flatFeeService.GetLineEngine()); err != nil {
		return nil, fmt.Errorf("registering flat fee line engine: %w", err)
	}

	usageBasedAdapter, err := usagebasedadapter.New(usagebasedadapter.Config{
		Client:      config.Client,
		Logger:      logger,
		MetaAdapter: metaAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("creating usage based adapter: %w", err)
	}

	usageBasedService, err := usagebasedservice.New(usagebasedservice.Config{
		Adapter:                 usageBasedAdapter,
		Handler:                 config.UsageBasedHandler,
		Lineage:                 lineageService,
		Locker:                  locker,
		MetaAdapter:             metaAdapter,
		CustomerOverrideService: config.BillingService,
		FeatureService:          config.FeatureService,
		RatingService:           billingratingservice.New(),
		StreamingConnector:      config.StreamingConnector,
	})
	if err != nil {
		return nil, fmt.Errorf("creating usage based service: %w", err)
	}

	if err := config.BillingService.RegisterLineEngine(usageBasedService.GetLineEngine()); err != nil {
		return nil, fmt.Errorf("registering usage based line engine: %w", err)
	}

	creditPurchaseAdapter, err := creditpurchaseadapter.New(creditpurchaseadapter.Config{
		Client:      config.Client,
		Logger:      logger,
		MetaAdapter: metaAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("creating credit purchase adapter: %w", err)
	}

	creditPurchaseService, err := creditpurchaseservice.New(creditpurchaseservice.Config{
		Adapter:     creditPurchaseAdapter,
		Handler:     config.CreditPurchaseHandler,
		Lineage:     lineageService,
		MetaAdapter: metaAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("creating credit purchase service: %w", err)
	}

	creditPurchaseLineEngine, err := creditpurchaselineengine.New(creditpurchaselineengine.Config{
		RatingService: billingratingservice.New(),
	})
	if err != nil {
		return nil, fmt.Errorf("creating credit purchase line engine: %w", err)
	}

	if err := config.BillingService.RegisterLineEngine(creditPurchaseLineEngine); err != nil {
		return nil, fmt.Errorf("registering credit purchase line engine: %w", err)
	}

	rootAdapter, err := chargesadapter.New(chargesadapter.Config{
		Client: config.Client,
		Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("creating charges adapter: %w", err)
	}

	chargesService, err := chargesservice.New(chargesservice.Config{
		Adapter:               rootAdapter,
		FeatureService:        config.FeatureService,
		MetaAdapter:           metaAdapter,
		FlatFeeService:        flatFeeService,
		CreditPurchaseService: creditPurchaseService,
		UsageBasedService:     usageBasedService,
		BillingService:        config.BillingService,
	})
	if err != nil {
		return nil, fmt.Errorf("creating charges service: %w", err)
	}

	return &Services{
		ChargesService:        chargesService,
		UsageBasedService:     usageBasedService,
		FlatFeeService:        flatFeeService,
		CreditPurchaseService: creditPurchaseService,
	}, nil
}
