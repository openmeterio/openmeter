package creditrealization

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
)

type ActiveLineageSegment struct {
	ID                        string                `json:"id"`
	Amount                    alpacadecimal.Decimal `json:"amount"`
	State                     LineageSegmentState   `json:"state"`
	BackingTransactionGroupID *string               `json:"backingTransactionGroupId,omitempty"`
}

func (s ActiveLineageSegment) Validate() error {
	var errs []error

	if !s.Amount.IsPositive() {
		errs = append(errs, errors.New("amount must be positive"))
	}

	if err := s.State.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("state: %w", err))
	}

	if s.State == LineageSegmentStateAdvanceBackfilled && (s.BackingTransactionGroupID == nil || *s.BackingTransactionGroupID == "") {
		errs = append(errs, errors.New("backing transaction group id is required for advance_backfilled"))
	}

	return errors.Join(errs...)
}
