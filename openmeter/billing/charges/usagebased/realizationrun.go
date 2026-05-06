package usagebased

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/pkg/models"
)

type RealizationRunType string

const (
	RealizationRunTypeFinalRealization RealizationRunType = "final_realization"
	RealizationRunTypePartialInvoice   RealizationRunType = "partial_invoice"
)

func (t RealizationRunType) Values() []string {
	return []string{
		string(RealizationRunTypeFinalRealization),
		string(RealizationRunTypePartialInvoice),
	}
}

func (t RealizationRunType) Validate() error {
	if !slices.Contains(t.Values(), string(t)) {
		return models.NewGenericValidationError(fmt.Errorf("invalid realization run type: %s", t))
	}
	return nil
}

type RealizationRunID models.NamespacedID

func (i RealizationRunID) Validate() error {
	return models.NamespacedID(i).Validate()
}

type CreateRealizationRunInput struct {
	FeatureID       string                `json:"featureId"`
	Type            RealizationRunType    `json:"type"`
	StoredAtLT      time.Time             `json:"storedAtLT"`
	ServicePeriodTo time.Time             `json:"servicePeriodTo"`
	LineID          *string               `json:"lineId,omitempty"`
	InvoiceID       *string               `json:"invoiceId,omitempty"`
	MeteredQuantity alpacadecimal.Decimal `json:"meteredQuantity"`
	Totals          totals.Totals         `json:"totals"`
}

func (r CreateRealizationRunInput) Normalized() CreateRealizationRunInput {
	r.StoredAtLT = meta.NormalizeTimestamp(r.StoredAtLT)
	r.ServicePeriodTo = meta.NormalizeTimestamp(r.ServicePeriodTo)

	return r
}

func (r CreateRealizationRunInput) Validate() error {
	var errs []error

	if err := r.Type.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("type: %w", err))
	}

	if r.FeatureID == "" {
		errs = append(errs, fmt.Errorf("feature id must be set"))
	}

	if r.StoredAtLT.IsZero() {
		errs = append(errs, fmt.Errorf("stored at lt must be set"))
	}

	if r.MeteredQuantity.IsNegative() {
		errs = append(errs, fmt.Errorf("metered quantity must be zero or positive"))
	}

	if err := r.Totals.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("totals: %w", err))
	}

	if r.ServicePeriodTo.IsZero() {
		errs = append(errs, fmt.Errorf("service period to must be set"))
	}

	if r.LineID != nil && *r.LineID == "" {
		errs = append(errs, fmt.Errorf("line id must be non-empty"))
	}

	if r.InvoiceID != nil && *r.InvoiceID == "" {
		errs = append(errs, fmt.Errorf("invoice id must be non-empty"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type UpdateRealizationRunInput struct {
	ID RealizationRunID

	StoredAtLT      mo.Option[time.Time]             `json:"storedAtLT"`
	LineID          mo.Option[*string]               `json:"lineId,omitempty"`
	MeteredQuantity mo.Option[alpacadecimal.Decimal] `json:"meteredQuantity"`
	Totals          mo.Option[totals.Totals]         `json:"totals"`
}

func (r UpdateRealizationRunInput) Normalized() UpdateRealizationRunInput {
	if r.StoredAtLT.IsPresent() {
		storedAtLT := r.StoredAtLT.OrEmpty()
		r.StoredAtLT = mo.Some(meta.NormalizeTimestamp(storedAtLT))
	}

	return r
}

func (r UpdateRealizationRunInput) Validate() error {
	var errs []error

	if err := r.ID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("namespaced id: %w", err))
	}

	if r.StoredAtLT.IsPresent() && r.StoredAtLT.OrEmpty().IsZero() {
		errs = append(errs, fmt.Errorf("stored at lt must be non-zero when set"))
	}

	if r.LineID.IsPresent() {
		lineID := r.LineID.OrEmpty()
		if lineID != nil && *lineID == "" {
			errs = append(errs, fmt.Errorf("line id must be non-empty"))
		}
	}

	if r.MeteredQuantity.IsPresent() && r.MeteredQuantity.OrEmpty().IsNegative() {
		errs = append(errs, fmt.Errorf("metered quantity must be zero or positive"))
	}

	if r.Totals.IsPresent() {
		if err := r.Totals.OrEmpty().Validate(); err != nil {
			errs = append(errs, fmt.Errorf("totals: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type RealizationRunBase struct {
	ID RealizationRunID
	models.ManagedModel

	FeatureID string  `json:"featureId"`
	LineID    *string `json:"lineId,omitempty"`
	InvoiceID *string `json:"invoiceId,omitempty"`

	Type       RealizationRunType `json:"type"`
	StoredAtLT time.Time          `json:"storedAtLT"`
	// ServicePeriodTo is the end of the service period for the realization run.
	ServicePeriodTo time.Time `json:"servicePeriodTo"`
	// MeteredQuantity is the metered quantity for time IN [intent.servicePeriod.from, servicePeriodTo) capped by stored_at < StoredAtLT.
	MeteredQuantity alpacadecimal.Decimal `json:"meteredQuantity"`
	Totals          totals.Totals         `json:"totals"`
}

func (r RealizationRunBase) Normalized() RealizationRunBase {
	r.StoredAtLT = meta.NormalizeTimestamp(r.StoredAtLT)
	r.ServicePeriodTo = meta.NormalizeTimestamp(r.ServicePeriodTo)

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

	if r.FeatureID == "" {
		errs = append(errs, fmt.Errorf("feature id must be set"))
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

	if r.StoredAtLT.IsZero() {
		errs = append(errs, fmt.Errorf("stored at lt must be set"))
	}

	if r.MeteredQuantity.IsNegative() {
		errs = append(errs, fmt.Errorf("metered quantity must be zero or positive"))
	}

	if err := r.Totals.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("totals: %w", err))
	}

	if r.ServicePeriodTo.IsZero() {
		errs = append(errs, fmt.Errorf("service period to must be set"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type RealizationRun struct {
	RealizationRunBase

	// Realizations
	CreditsAllocated creditrealization.Realizations `json:"creditsAllocated"`
	InvoiceUsage     *invoicedusage.AccruedUsage    `json:"invoicedUsage"`
	Payment          *payment.Invoiced              `json:"payment"`
	DetailedLines    mo.Option[DetailedLines]       `json:"detailedLines,omitzero"`
}

func (r RealizationRun) Validate() error {
	var errs []error

	if err := r.RealizationRunBase.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("realization run: %w", err))
	}

	if err := r.CreditsAllocated.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("credits allocated: %w", err))
	}

	if r.InvoiceUsage != nil {
		if err := r.InvoiceUsage.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invoice usage: %w", err))
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
	for idx, realizationRun := range r {
		if err := realizationRun.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("realization run[%d]: %w", idx, err))
		}
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Sum returns the aggregate totals across all realization runs.
func (r RealizationRuns) Sum() totals.Totals {
	return totals.Sum(lo.Map(r, func(run RealizationRun, _ int) totals.Totals {
		return run.Totals
	})...)
}

func (r RealizationRuns) GetByID(id string) (RealizationRun, error) {
	for _, run := range r {
		if run.ID.ID == id {
			return run, nil
		}
	}
	return RealizationRun{}, fmt.Errorf("realization run not found [id=%s]", id)
}

func (r RealizationRuns) Without(id RealizationRunID) RealizationRuns {
	return lo.Filter(r, func(run RealizationRun, _ int) bool {
		return run.ID != id
	})
}

func (r *RealizationRuns) SetRealizationRun(updatedRun RealizationRun) error {
	for idx, realizationRun := range *r {
		if realizationRun.ID.ID == updatedRun.ID.ID {
			(*r)[idx] = updatedRun
			return nil
		}
	}
	return fmt.Errorf("realization run not found [id=%s]", updatedRun.ID.ID)
}
