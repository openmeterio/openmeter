package service

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	creditpurchaserealizations "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/service/realizations"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
)

type Config struct {
	Adapter     creditpurchase.Adapter
	Handler     creditpurchase.Handler
	Lineage     lineage.Service
	MetaAdapter meta.Adapter

	CustomCurrenciesEnabled bool
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter cannot be null"))
	}

	if c.Handler == nil {
		errs = append(errs, errors.New("credit purchase handler cannot be null"))
	}

	if c.Lineage == nil {
		errs = append(errs, errors.New("lineage service cannot be null"))
	}

	if c.MetaAdapter == nil {
		errs = append(errs, errors.New("meta adapter cannot be null"))
	}

	return errors.Join(errs...)
}

func New(config Config) (creditpurchase.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	realizations, err := creditpurchaserealizations.New(creditpurchaserealizations.Config{
		Adapter: config.Adapter,
		Handler: config.Handler,
		Lineage: config.Lineage,
	})
	if err != nil {
		return nil, fmt.Errorf("realizations: %w", err)
	}

	svc := &service{
		adapter:      config.Adapter,
		handler:      config.Handler,
		lineage:      config.Lineage,
		metaAdapter:  config.MetaAdapter,
		realizations: realizations,
	}
	svc.enableCustomCurrency.Store(config.CustomCurrenciesEnabled)

	return svc, nil
}

type service struct {
	adapter              creditpurchase.Adapter
	metaAdapter          meta.Adapter
	handler              creditpurchase.Handler
	lineage              lineage.Service
	realizations         *creditpurchaserealizations.Service
	enableCustomCurrency atomic.Bool
}

func (s *service) SetEnableCustomCurrency(t *testing.T, enabled bool) error {
	if t == nil {
		return errors.New("testing is nil")
	}

	t.Helper()
	s.enableCustomCurrency.Store(enabled)
	return nil
}
