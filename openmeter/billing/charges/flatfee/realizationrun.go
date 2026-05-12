package flatfee

import (
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/pkg/models"
)

type RealizationRunType string

const (
	RealizationRunTypeFinalRealization                  RealizationRunType = "final_realization"
	RealizationRunTypeInvalidDueToUnsupportedCreditNote RealizationRunType = "invalid_due_to_unsupported_credit_note"
)

func (t RealizationRunType) Values() []string {
	return []string{
		string(RealizationRunTypeFinalRealization),
		string(RealizationRunTypeInvalidDueToUnsupportedCreditNote),
	}
}

func (t RealizationRunType) Validate() error {
	if !slices.Contains(t.Values(), string(t)) {
		return models.NewGenericValidationError(fmt.Errorf("invalid realization run type: %s", t))
	}

	return nil
}

func (t RealizationRunType) IsVoidedBillingHistory() bool {
	return t == RealizationRunTypeInvalidDueToUnsupportedCreditNote
}
