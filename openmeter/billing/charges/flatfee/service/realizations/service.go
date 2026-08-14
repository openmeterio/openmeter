package realizations

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/currencies"
)

// Service owns flat-fee realization mechanics: credit allocation/correction and
// realization lineage persistence. It must not make state-machine decisions.
type Service struct {
	adapter       flatfee.Adapter
	handler       flatfee.Handler
	lineage       lineage.Service
	ratingService rating.Service
}

type Config struct {
	Adapter       flatfee.Adapter
	Handler       flatfee.Handler
	Lineage       lineage.Service
	RatingService rating.Service
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter is required"))
	}

	if c.Handler == nil {
		errs = append(errs, errors.New("handler is required"))
	}

	if c.Lineage == nil {
		errs = append(errs, errors.New("lineage service is required"))
	}

	if c.RatingService == nil {
		errs = append(errs, errors.New("rating service is required"))
	}

	return errors.Join(errs...)
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		adapter:       config.Adapter,
		handler:       config.Handler,
		lineage:       config.Lineage,
		ratingService: config.RatingService,
	}, nil
}

func (s *Service) createCreditRealizationLineages(
	ctx context.Context,
	charge flatfee.Charge,
	currency currencies.Currency,
	realizations creditrealization.Realizations,
) error {
	featureKey := charge.Intent.GetFeatureKey()
	if err := s.lineage.CreateInitialLineages(ctx, lineage.CreateInitialLineagesInput{
		Namespace:    charge.Namespace,
		ChargeID:     charge.ID,
		CustomerID:   charge.Intent.GetCustomerID(),
		Currency:     currency,
		Features:     lo.Ternary(featureKey == "", nil, []string{featureKey}),
		Realizations: realizations,
	}); err != nil {
		return fmt.Errorf("create initial credit realization lineages: %w", err)
	}

	if err := s.lineage.PersistCorrectionLineageSegments(ctx, lineage.PersistCorrectionLineageSegmentsInput{
		Namespace:    charge.Namespace,
		Realizations: realizations,
	}); err != nil {
		return fmt.Errorf("persist correction lineage segments: %w", err)
	}

	return nil
}
