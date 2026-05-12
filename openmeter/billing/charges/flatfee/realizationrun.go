package flatfee

import (
	"errors"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
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

type RealizationRunID models.NamespacedID

func (i RealizationRunID) Validate() error {
	return models.NamespacedID(i).Validate()
}

type RealizationRunBase struct {
	ID RealizationRunID `json:"id"`
	models.ManagedModel

	LineID    *string `json:"lineId,omitempty"`
	InvoiceID *string `json:"invoiceId,omitempty"`

	Type        RealizationRunType `json:"type"`
	InitialType RealizationRunType `json:"initialType"`

	ServicePeriod             timeutil.ClosedPeriod `json:"servicePeriod"`
	AmountAfterProration      alpacadecimal.Decimal `json:"amountAfterProration"`
	Totals                    totals.Totals         `json:"totals"`
	NoFiatTransactionRequired bool                  `json:"noFiatTransactionRequired"`
}

func (r RealizationRunBase) Normalized() RealizationRunBase {
	r.ServicePeriod = meta.NormalizeClosedPeriod(r.ServicePeriod)

	return r
}

func (r RealizationRunBase) Validate() error {
	var errs []error

	if err := r.ID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("namespaced id: %w", err))
	}

	if err := r.ManagedModel.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("managed model: %w", err))
	}

	if r.LineID != nil && *r.LineID == "" {
		errs = append(errs, fmt.Errorf("line id must be non-empty"))
	}

	if r.InvoiceID != nil && *r.InvoiceID == "" {
		errs = append(errs, fmt.Errorf("invoice id must be non-empty"))
	}

	if err := r.Type.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("type: %w", err))
	}

	if err := r.InitialType.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("initial type: %w", err))
	}

	if r.InitialType == RealizationRunTypeInvalidDueToUnsupportedCreditNote {
		errs = append(errs, fmt.Errorf("initial type cannot be %s", RealizationRunTypeInvalidDueToUnsupportedCreditNote))
	}

	if r.ServicePeriod.From.IsZero() {
		errs = append(errs, fmt.Errorf("service period from must be set"))
	}

	if r.ServicePeriod.To.IsZero() {
		errs = append(errs, fmt.Errorf("service period to must be set"))
	}

	if r.ServicePeriod.To.Before(r.ServicePeriod.From) {
		errs = append(errs, fmt.Errorf("service period to must be after service period from"))
	}

	if r.AmountAfterProration.IsNegative() {
		errs = append(errs, fmt.Errorf("amount after proration must be zero or positive"))
	}

	if err := r.Totals.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("totals: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type RealizationRun struct {
	RealizationRunBase

	CreditRealizations creditrealization.Realizations `json:"creditRealizations"`
	AccruedUsage       *invoicedusage.AccruedUsage    `json:"accruedUsage"`
	Payment            *payment.Invoiced              `json:"payment"`
	DetailedLines      mo.Option[DetailedLines]       `json:"detailedLines,omitzero"`
}

func (r RealizationRun) Validate() error {
	var errs []error

	if err := r.RealizationRunBase.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("realization run: %w", err))
	}

	if err := r.CreditRealizations.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("credit realizations: %w", err))
	}

	if r.AccruedUsage != nil {
		if err := r.AccruedUsage.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("accrued usage: %w", err))
		}
	}

	if r.Payment != nil {
		if err := r.Payment.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("payment: %w", err))
		}
	}

	if r.DetailedLines.IsPresent() {
		if err := r.DetailedLines.OrEmpty().Validate(); err != nil {
			errs = append(errs, fmt.Errorf("detailed lines: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type RealizationRuns []RealizationRun

func (r RealizationRuns) Validate() error {
	var errs []error
	for idx, run := range r {
		if err := run.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("realization run[%d]: %w", idx, err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
