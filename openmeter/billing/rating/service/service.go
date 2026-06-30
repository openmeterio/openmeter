package service

import "github.com/openmeterio/openmeter/openmeter/billing/rating"

// Config carries the deploy-wide rating configuration.
type Config struct {
	// UnitConfigEnabled gates the unit_config pre-calculation mutator. When false,
	// rating output is byte-identical to having no unit_config on the rate card.
	UnitConfigEnabled bool
}

type service struct {
	unitConfigEnabled bool
}

func New(cfg Config) rating.Service {
	return &service{
		unitConfigEnabled: cfg.UnitConfigEnabled,
	}
}
