package correction

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Input struct {
	Namespace                    string
	ChargeID                     string
	CustomerID                   string
	Annotations                  models.Annotations
	AllocateAt                   time.Time
	Corrections                  creditrealization.CorrectionRequest
	LineageSegmentsByRealization legacylineage.ActiveSegmentsByRealizationID
}

func (i Input) Validate() error {
	var errs []error

	if err := (models.NamespacedID{
		Namespace: i.Namespace,
		ID:        i.ChargeID,
	}).Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := (customer.CustomerID{
		Namespace: i.Namespace,
		ID:        i.CustomerID,
	}).Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	if i.AllocateAt.IsZero() {
		errs = append(errs, errors.New("allocate at is required"))
	}

	seen := make(map[string]bool)

	type allocationSource struct {
		groupID  string
		sortHint int
	}

	seenSources := make(map[allocationSource]bool)

	for idx, correction := range i.Corrections {
		if err := correction.Allocation.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("corrections[%d].allocation: %w", idx, err))
		}

		if correction.Amount.IsPositive() {
			errs = append(errs, fmt.Errorf("corrections[%d]: amount must be non-positive", idx))
		}

		if seen[correction.Allocation.ID] {
			errs = append(errs, errors.New("a correction batch cannot repeat an allocation"))
		}

		seen[correction.Allocation.ID] = true

		source := allocationSource{
			groupID:  correction.Allocation.LedgerTransaction.TransactionGroupID,
			sortHint: correction.Allocation.SortHint,
		}
		if seenSources[source] {
			errs = append(errs, errors.New("a correction batch cannot repeat an original collection source"))
		}

		seenSources[source] = true

		if correction.Allocation.Namespace != i.Namespace || correction.Allocation.Type != creditrealization.TypeAllocation {
			errs = append(errs, fmt.Errorf("corrections[%d]: allocation must belong to the correction namespace", idx))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
