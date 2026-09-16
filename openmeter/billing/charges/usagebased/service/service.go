package service

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	usagebasedrun "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/run"
	billingfeaturemeterservice "github.com/openmeterio/openmeter/openmeter/billing/featuremeter/service"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/subscription/validators/itemreference"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

type Config struct {
	Adapter                usagebased.Adapter
	Handler                usagebased.Handler
	Lineage                lineage.Service
	Locker                 *lockr.Locker
	MetaAdapter            meta.Adapter
	InvoiceUpdater         invoiceupdater.Updater
	BillingService         billing.Service
	FeatureMeterResolver   *billingfeaturemeterservice.Resolver
	RatingService          rating.Service
	Currencies             currencies.Service
	ItemReferenceValidator itemreference.Validator

	StreamingConnector streaming.Connector
}

type LineSubscriptionReferenceService interface {
	SetGatheringLineSubscriptionReferenceByChargeID(ctx context.Context, input billing.SetLineSubscriptionReferenceByChargeIDInput) error
	SetStandardLineSubscriptionReferenceByChargeID(ctx context.Context, input billing.SetLineSubscriptionReferenceByChargeIDInput) error
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

	if c.BillingService == nil {
		errs = append(errs, errors.New("billing service cannot be null"))
	}

	if c.FeatureMeterResolver == nil {
		errs = append(errs, errors.New("feature meter resolver cannot be null"))
	}

	if c.RatingService == nil {
		errs = append(errs, errors.New("rating service cannot be null"))
	}

	if c.Currencies == nil {
		errs = append(errs, errors.New("currencies service cannot be null"))
	}

	if c.ItemReferenceValidator == nil {
		errs = append(errs, errors.New("subscription item reference validator cannot be null"))
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

	return &service{
		adapter:                          config.Adapter,
		locker:                           config.Locker,
		metaAdapter:                      config.MetaAdapter,
		invoiceUpdater:                   config.InvoiceUpdater,
		customerOverrideService:          config.BillingService,
		featureMeterResolver:             config.FeatureMeterResolver,
		ratingService:                    config.RatingService,
		rater:                            rater,
		runs:                             runs,
		costbasisResolver:                costbasisResolver,
		itemReferenceValidator:           config.ItemReferenceValidator,
		lineSubscriptionReferenceService: config.BillingService,
	}, nil
}

type service struct {
	adapter                 usagebased.Adapter
	locker                  *lockr.Locker
	metaAdapter             meta.Adapter
	invoiceUpdater          invoiceupdater.Updater
	customerOverrideService billing.CustomerOverrideService
	featureMeterResolver    *billingfeaturemeterservice.Resolver
	ratingService           rating.Service

	rater usagebasedrating.Service
	runs  *usagebasedrun.Service

	costbasisResolver                costbasis.Resolver
	itemReferenceValidator           itemreference.Validator
	lineSubscriptionReferenceService LineSubscriptionReferenceService
}

func (s *service) GetLineEngine() billing.LineEngine {
	return &LineEngine{
		service: s,
	}
}
