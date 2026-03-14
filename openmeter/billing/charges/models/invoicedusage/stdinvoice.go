package invoicedusage

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type AccruedUsage struct {
	models.NamespacedID
	models.ManagedModel

	Annotations       models.Annotations                `json:"annotations"`
	LineID            *string                           `json:"lineID"`
	ServicePeriod     timeutil.ClosedPeriod             `json:"servicePeriod"`
	Mutable           bool                              `json:"mutable"`
	LedgerTransaction *ledgertransaction.GroupReference `json:"ledgerTransaction"`

	Totals totals.Totals `json:"totals"`
}

func (r AccruedUsage) Validate() error {
	var errs []error

	if !r.Mutable {
		if r.LineID == nil {
			errs = append(errs, fmt.Errorf("line ID is required when mutable is false"))
		}
	}

	if err := r.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if err := r.Totals.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("totals: %w", err))
	}

	if r.LineID != nil {
		if *r.LineID == "" {
			errs = append(errs, fmt.Errorf("line ID is required"))
		}
	}

	if r.LedgerTransaction != nil {
		if err := r.LedgerTransaction.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("ledger transaction: %w", err))
		}
	}

	return errors.Join(errs...)
}
