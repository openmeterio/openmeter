package fbo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SourceQuery struct {
	CustomerID customer.CustomerID
	Currency   currencies.CurrencyReference
	Filters    ledger.CreditFilters
	AsOf       time.Time
}

var _ models.Validator = SourceQuery{}

func (q SourceQuery) Validate() error {
	var errs []error
	if err := q.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	if err := q.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	} else if q.Currency.IsCustom() && !q.Currency.IsResolved() {
		errs = append(errs, errors.New("custom currency must be resolved"))
	}

	if err := q.Filters.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("filters: %w", err))
	}

	if q.AsOf.IsZero() {
		errs = append(errs, errors.New("as of is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type AvailableSource struct {
	Address        ledger.PostingAddress
	SourceChargeID *string
	Amount         alpacadecimal.Decimal
	ExpiresAt      *time.Time
	BreakagePlanID *string
}

// ListSources returns ordered eligible source slices at the historical boundary.
// It uses the current collection policy; it is not the future spendable-balance API.
func (s *service) ListSources(ctx context.Context, query SourceQuery) ([]AvailableSource, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}

	sources, err := s.listCustomerFBOSources(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	out := make([]AvailableSource, 0, len(sources))
	for _, source := range sources {
		available := AvailableSource{
			Address:        source.address,
			SourceChargeID: source.sourceChargeID,
			Amount:         source.available,
			ExpiresAt:      source.expiresAt,
		}
		if source.breakagePlan != nil {
			available.BreakagePlanID = &source.breakagePlan.ID.ID
		}

		out = append(out, available)
	}

	return out, nil
}
