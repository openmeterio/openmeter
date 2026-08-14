package service

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/invoicing/legacy/splitlinegroup"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Config struct {
	Adapter splitlinegroup.Adapter
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, fmt.Errorf("adapter is required"))
	}

	return models.NewNillableGenericValidationError(errs...)
}

type Service struct {
	adapter splitlinegroup.Adapter
}

func NewService(config Config) *Service {
	return &Service{
		adapter: config.Adapter,
	}
}
