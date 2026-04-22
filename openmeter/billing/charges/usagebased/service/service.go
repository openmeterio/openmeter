package service

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	usagebasedrun "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/run"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
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
	CustomerOverrideService billing.CustomerOverrideService
	FeatureService          feature.FeatureConnector
	RatingService           rating.Service

	StreamingConnector streaming.Connector
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

	if c.CustomerOverrideService == nil {
		errs = append(errs, errors.New("customer override service cannot be null"))
	}

	if c.FeatureService == nil {
		errs = append(errs, errors.New("feature service cannot be null"))
	}

	if c.RatingService == nil {
		errs = append(errs, errors.New("rating service cannot be null"))
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
		StreamingConnector: config.StreamingConnector,
		RatingService:      config.RatingService,
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

	return &service{
		adapter:                 config.Adapter,
		locker:                  config.Locker,
		metaAdapter:             config.MetaAdapter,
		customerOverrideService: config.CustomerOverrideService,
		featureService:          config.FeatureService,
		ratingService:           config.RatingService,
		rater:                   rater,
		runs:                    runs,
	}, nil
}

type service struct {
	adapter                 usagebased.Adapter
	locker                  *lockr.Locker
	metaAdapter             meta.Adapter
	customerOverrideService billing.CustomerOverrideService
	featureService          feature.FeatureConnector
	ratingService           rating.Service

	rater usagebasedrating.Service
	runs  *usagebasedrun.Service
}

func (s *service) GetLineEngine() billing.LineEngine {
	return &LineEngine{
		service: s,
	}
}
