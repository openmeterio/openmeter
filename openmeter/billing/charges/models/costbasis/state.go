package costbasis

import (
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/models"
)

type State struct {
	CostBasis   alpacadecimal.Decimal
	CostBasisID *string
	ResolvedAt  time.Time
}

func (s State) String() string {
	return fmt.Sprintf("cost_basis=%s cost_basis_id=%v resolved_at=%s", s.CostBasis, s.CostBasisID, s.ResolvedAt.UTC().Format(time.RFC3339))
}

func (s State) Validate() error {
	var errs []error

	if !s.CostBasis.IsPositive() {
		errs = append(errs, fmt.Errorf("cost basis must be positive"))
	}

	if s.ResolvedAt.IsZero() {
		errs = append(errs, fmt.Errorf("resolved at is required"))
	}

	if s.CostBasisID != nil && *s.CostBasisID == "" {
		errs = append(errs, fmt.Errorf("cost basis id is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
