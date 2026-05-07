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

// BillingMeteredQuantity maps a cumulative charge run quantity to the quantity
// semantics expected by billing.StandardLine. RealizationRun.MeteredQuantity is
// cumulative from the charge service-period start to the run's ServicePeriodTo,
// while standard invoice lines need the current line-period quantity plus the
// quantity already represented by earlier billed lines.
type BillingMeteredQuantity struct {
	// PreLinePeriod is the cumulative quantity already represented by earlier
	// billed runs.
	PreLinePeriod alpacadecimal.Decimal
	// LinePeriod is the quantity represented by the current standard invoice
	// line.
	LinePeriod alpacadecimal.Decimal
}

type CreateRealizationRunInput struct {
	FeatureID                 string                `json:"featureId"`
	Type                      RealizationRunType    `json:"type"`
	StoredAtLT                time.Time             `json:"storedAtLT"`
	ServicePeriodTo           time.Time             `json:"servicePeriodTo"`
	LineID                    *string               `json:"lineId,omitempty"`
	InvoiceID                 *string               `json:"invoiceId,omitempty"`
	MeteredQuantity           alpacadecimal.Decimal `json:"meteredQuantity"`
	Totals                    totals.Totals         `json:"totals"`
	NoFiatTransactionRequired bool                  `json:"noFiatTransactionRequired"`
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

	StoredAtLT                mo.Option[time.Time]             `json:"storedAtLT"`
	DeletedAt                 mo.Option[*time.Time]            `json:"deletedAt,omitempty"`
	LineID                    mo.Option[*string]               `json:"lineId,omitempty"`
	MeteredQuantity           mo.Option[alpacadecimal.Decimal] `json:"meteredQuantity"`
	Totals                    mo.Option[totals.Totals]         `json:"totals"`
	NoFiatTransactionRequired mo.Option[bool]                  `json:"noFiatTransactionRequired"`
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

	if r.DeletedAt.IsPresent() {
		deletedAt := r.DeletedAt.OrEmpty()
		if deletedAt != nil && deletedAt.IsZero() {
			errs = append(errs, fmt.Errorf("deleted at must be non-zero when set"))
		}
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
	MeteredQuantity           alpacadecimal.Decimal `json:"meteredQuantity"`
	Totals                    totals.Totals         `json:"totals"`
	NoFiatTransactionRequired bool                  `json:"noFiatTransactionRequired"`
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

func (r RealizationRuns) MapToBillingMeteredQuantity(currentRun RealizationRun) (BillingMeteredQuantity, error) {
	preLinePeriod := alpacadecimal.Zero
	var latestPriorRun *RealizationRun

	for idx := range r {
		if r[idx].Type != RealizationRunTypeFinalRealization && r[idx].Type != RealizationRunTypePartialInvoice {
			continue
		}

		if !r[idx].ServicePeriodTo.Before(currentRun.ServicePeriodTo) {
			continue
		}

		if latestPriorRun == nil || r[idx].ServicePeriodTo.After(latestPriorRun.ServicePeriodTo) {
			latestPriorRun = &r[idx]
		}
	}

	if latestPriorRun != nil {
		// Standard invoice line quantities intentionally use the prior run's
		// persisted cumulative quantity. That value may have been captured with
		// an older StoredAtLT than the current run. Period-preserving rating may
		// still freshly snapshot prior event-time periods with the current
		// StoredAtLT for correction calculation, but invoice line quantities
		// should reflect what was previously billed.
		preLinePeriod = latestPriorRun.MeteredQuantity
	}

	linePeriod := currentRun.MeteredQuantity.Sub(preLinePeriod)
	if linePeriod.IsNegative() {
		return BillingMeteredQuantity{}, fmt.Errorf(
			"line period metered quantity is negative: current=%s pre_line=%s",
			currentRun.MeteredQuantity.String(),
			preLinePeriod.String(),
		)
	}

	return BillingMeteredQuantity{
		PreLinePeriod: preLinePeriod,
		LinePeriod:    linePeriod,
	}, nil
}

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

func (r RealizationRuns) GetByLineID(lineID string) (RealizationRun, error) {
	run, found := lo.Find(r, func(run RealizationRun) bool {
		return run.LineID != nil && *run.LineID == lineID
	})
	if found {
		return run, nil
	}

	return RealizationRun{}, fmt.Errorf("realization run not found [line_id=%s]", lineID)
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
