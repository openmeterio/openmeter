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
	ServicePeriod     timeutil.ClosedPeriod             `json:"servicePeriod"`
	LedgerTransaction *ledgertransaction.GroupReference `json:"ledgerTransaction"`

	Totals totals.Totals `json:"totals"`
}

func (r AccruedUsage) Validate() error {
	var errs []error

	if err := r.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if err := r.Totals.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("totals: %w", err))
	}

	if r.LedgerTransaction != nil {
		if err := r.LedgerTransaction.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("ledger transaction: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
