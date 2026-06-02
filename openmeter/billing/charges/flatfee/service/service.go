package service

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeerealizations "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service/realizations"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

type Config struct {
	Adapter       flatfee.Adapter
	Handler       flatfee.Handler
	Lineage       lineage.Service
	MetaAdapter   meta.Adapter
	Locker        *lockr.Locker
	RatingService rating.Service
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

	if c.MetaAdapter == nil {
		errs = append(errs, errors.New("meta adapter cannot be null"))
	}

	if c.Locker == nil {
		errs = append(errs, errors.New("locker cannot be null"))
	}

	if c.RatingService == nil {
		errs = append(errs, errors.New("rating service cannot be null"))
	}

	return errors.Join(errs...)
}

func New(config Config) (flatfee.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	realizations, err := flatfeerealizations.New(flatfeerealizations.Config{
		Adapter:       config.Adapter,
		Handler:       config.Handler,
		Lineage:       config.Lineage,
		RatingService: config.RatingService,
	})
	if err != nil {
		return nil, err
	}

	svc := &service{
		adapter:      config.Adapter,
		handler:      config.Handler,
		metaAdapter:  config.MetaAdapter,
		locker:       config.Locker,
		realizations: realizations,
	}
	svc.creditNotesSupported.Store(charges.CreditNotesSupportedByLineUpdater)

	return svc, nil
}

type service struct {
	adapter              flatfee.Adapter
	handler              flatfee.Handler
	metaAdapter          meta.Adapter
	locker               *lockr.Locker
	realizations         *flatfeerealizations.Service
	creditNotesSupported atomic.Bool
}

func (s *service) GetLineEngine() billing.LineEngine {
	return &LineEngine{
		service: s,
	}
}

// SetCreditNotesSupportedByLineUpdater sets the credit notes supported by the line updater.
// This is used to test the credit notes supported by the line updater, but must not be used
// in production code.
func (s *service) SetCreditNotesSupportedByLineUpdater(t *testing.T, supported bool) error {
	if t == nil {
		return errors.New("testing is nil")
	}

	t.Helper()
	s.creditNotesSupported.Store(supported)
	return nil
}
