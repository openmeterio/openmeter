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
)

func (t RealizationRunType) Values() []string {
	return []string{
		string(RealizationRunTypeFinalRealization),
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
	FeatureID     string                `json:"featureId"`
	Type          RealizationRunType    `json:"type"`
	AsOf          time.Time             `json:"asOf"`
	CollectionEnd time.Time             `json:"collectionEnd,omitempty"`
	LineID        *string               `json:"lineId,omitempty"`
	MeterValue    alpacadecimal.Decimal `json:"meterValue"`
	Totals        totals.Totals         `json:"totals"`
}

func (r CreateRealizationRunInput) Normalized() CreateRealizationRunInput {
	r.AsOf = meta.NormalizeTimestamp(r.AsOf)
	r.CollectionEnd = meta.NormalizeTimestamp(r.CollectionEnd)

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

	if r.AsOf.IsZero() {
		errs = append(errs, fmt.Errorf("as of must be set"))
	}

	if r.MeterValue.IsNegative() {
		errs = append(errs, fmt.Errorf("meter value must be zero or positive"))
	}

	if err := r.Totals.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("totals: %w", err))
	}

	if r.CollectionEnd.IsZero() {
		errs = append(errs, fmt.Errorf("collection end must be set"))
	}

	if r.LineID != nil && *r.LineID == "" {
		errs = append(errs, fmt.Errorf("line id must be non-empty"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type UpdateRealizationRunInput struct {
	ID RealizationRunID

	AsOf       mo.Option[time.Time]             `json:"asOf"`
	LineID     mo.Option[*string]               `json:"lineId,omitempty"`
	MeterValue mo.Option[alpacadecimal.Decimal] `json:"meterValue"`
	Totals     mo.Option[totals.Totals]         `json:"totals"`
}

func (r UpdateRealizationRunInput) Normalized() UpdateRealizationRunInput {
	if r.AsOf.IsPresent() {
		asOf := r.AsOf.OrEmpty()
		r.AsOf = mo.Some(meta.NormalizeTimestamp(asOf))
	}

	return r
}

func (r UpdateRealizationRunInput) Validate() error {
	var errs []error

	if err := r.ID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("namespaced id: %w", err))
	}

	if r.AsOf.IsPresent() && r.AsOf.OrEmpty().IsZero() {
		errs = append(errs, fmt.Errorf("as of must be non-zero when set"))
	}

	if r.LineID.IsPresent() {
		lineID := r.LineID.OrEmpty()
		if lineID != nil && *lineID == "" {
			errs = append(errs, fmt.Errorf("line id must be non-empty"))
		}
	}

	if r.MeterValue.IsPresent() && r.MeterValue.OrEmpty().IsNegative() {
		errs = append(errs, fmt.Errorf("meter value must be zero or positive"))
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

	Type          RealizationRunType    `json:"type"`
	AsOf          time.Time             `json:"asOf"`
	CollectionEnd time.Time             `json:"collectionEnd,omitempty"`
	MeterValue    alpacadecimal.Decimal `json:"meterValue"`
	Totals        totals.Totals         `json:"totals"`
}

func (r RealizationRunBase) Normalized() RealizationRunBase {
	r.AsOf = meta.NormalizeTimestamp(r.AsOf)
	r.CollectionEnd = meta.NormalizeTimestamp(r.CollectionEnd)

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

	if err := r.Type.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("type: %w", err))
	}

	if r.AsOf.IsZero() {
		errs = append(errs, fmt.Errorf("as of must be set"))
	}

	if r.MeterValue.IsNegative() {
		errs = append(errs, fmt.Errorf("meter value must be zero or positive"))
	}

	if err := r.Totals.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("totals: %w", err))
	}

	if r.CollectionEnd.IsZero() {
		errs = append(errs, fmt.Errorf("collection end must be set"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type RealizationRun struct {
	RealizationRunBase

	// Realizations
	CreditsAllocated creditrealization.Realizations `json:"creditsAllocated"`
	InvoiceUsage     *invoicedusage.AccruedUsage    `json:"invoicedUsage"`
	Payment          *payment.Invoiced              `json:"payment"`
	DetailedLines    mo.Option[DetailedLines]       `json:"detailedLines,omitempty"`
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
