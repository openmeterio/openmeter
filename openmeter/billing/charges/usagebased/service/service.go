package service

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	usagebasedrun "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/run"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

type Config struct {
	Adapter                 usagebased.Adapter
	Handler                 usagebased.Handler
	Lineage                 lineage.Service
	Locker                  *lockr.Locker
	MetaAdapter             meta.Adapter
	InvoiceUpdater          invoiceupdater.Updater
	CustomerOverrideService billing.CustomerOverrideService
	FeatureService          feature.FeatureConnector
	RatingService           rating.Service
	Currencies              currencies.Service
	StreamingConnector      streaming.Connector

	CustomCurrenciesEnabled bool
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter cannot be null"))
	}

	if c.Handler == nil {
		errs = append(errs, errors.New("handler cannot be null"))
	}

	if c.Lineage == nil {
		errs = append(errs, errors.New("lineage service cannot be null"))
	}

	if c.Locker == nil {
		errs = append(errs, errors.New("locker cannot be null"))
	}

	if c.MetaAdapter == nil {
		errs = append(errs, errors.New("meta adapter cannot be null"))
	}

	if c.InvoiceUpdater == nil {
		errs = append(errs, errors.New("invoice updater cannot be null"))
	}

	if c.CustomerOverrideService == nil {
		errs = append(errs, errors.New("customer override service cannot be null"))
	}

	if c.FeatureService == nil {
		errs = append(errs, errors.New("feature service cannot be null"))
	}

	if c.RatingService == nil {
		errs = append(errs, errors.New("rating service cannot be null"))
	}

	if c.Currencies == nil {
		errs = append(errs, errors.New("currencies service cannot be null"))
	}

	if c.StreamingConnector == nil {
		errs = append(errs, errors.New("streaming connector cannot be null"))
	}

	return errors.Join(errs...)
}

func New(config Config) (usagebased.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	rater, err := usagebasedrating.New(usagebasedrating.Config{
		StreamingConnector:   config.StreamingConnector,
		RatingService:        config.RatingService,
		DetailedLinesFetcher: config.Adapter,
	})
	if err != nil {
		return nil, err
	}

	runs, err := usagebasedrun.New(usagebasedrun.Config{
		Adapter: config.Adapter,
		Rater:   rater,
		Handler: config.Handler,
		Lineage: config.Lineage,
	})
	if err != nil {
		return nil, err
	}

	costbasisResolver, err := costbasis.NewResolver(costbasis.ResolverConfig{
		Currencies: config.Currencies,
	})
	if err != nil {
		return nil, err
	}

	svc := &service{
		adapter:                 config.Adapter,
		locker:                  config.Locker,
		metaAdapter:             config.MetaAdapter,
		invoiceUpdater:          config.InvoiceUpdater,
		customerOverrideService: config.CustomerOverrideService,
		featureService:          config.FeatureService,
		ratingService:           config.RatingService,
		rater:                   rater,
		runs:                    runs,
		costbasisResolver:       costbasisResolver,
	}
	svc.enableCustomCurrency.Store(config.CustomCurrenciesEnabled)

	return svc, nil
}

type service struct {
	adapter                 usagebased.Adapter
	locker                  *lockr.Locker
	metaAdapter             meta.Adapter
	invoiceUpdater          invoiceupdater.Updater
	customerOverrideService billing.CustomerOverrideService
	featureService          feature.FeatureConnector
	ratingService           rating.Service

	rater usagebasedrating.Service
	runs  *usagebasedrun.Service

	enableCustomCurrency atomic.Bool
	costbasisResolver    costbasis.Resolver
}

func (s *service) GetLineEngine() billing.LineEngine {
	return &LineEngine{
		service: s,
	}
}

func (s *service) SetEnableCustomCurrency(t *testing.T, enabled bool) error {
	if t == nil {
		return errors.New("testing is nil")
	}

	t.Helper()
	s.enableCustomCurrency.Store(enabled)
	return nil
}
