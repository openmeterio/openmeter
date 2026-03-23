package usagebased

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"

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
		return fmt.Errorf("invalid realization run type: %s", t)
	}
	return nil
}

type RealizationRunID models.NamespacedID

func (i RealizationRunID) Validate() error {
	return models.NamespacedID(i).Validate()
}

type CreateRealizationRunInput struct {
	Type          RealizationRunType    `json:"type"`
	AsOf          time.Time             `json:"asOf"`
	CollectionEnd time.Time             `json:"collectionEnd,omitempty"`
	MeterValue    alpacadecimal.Decimal `json:"meterValue"`
	Totals        totals.Totals         `json:"totals"`
}

func (r CreateRealizationRunInput) Validate() error {
	var errs []error

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

	return errors.Join(errs...)
}

type UpdateRealizationRunInput struct {
	ID RealizationRunID

	AsOf       time.Time             `json:"asOf"`
	MeterValue alpacadecimal.Decimal `json:"meterValue"`
	Totals     totals.Totals         `json:"totals"`
}

func (r UpdateRealizationRunInput) Validate() error {
	var errs []error

	if err := r.ID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("namespaced id: %w", err))
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

	return errors.Join(errs...)
}

type RealizationRunBase struct {
	ID RealizationRunID
	models.ManagedModel

	Type          RealizationRunType    `json:"type"`
	AsOf          time.Time             `json:"asOf"`
	CollectionEnd time.Time             `json:"collectionEnd,omitempty"`
	MeterValue    alpacadecimal.Decimal `json:"meterValue"`
	Totals        totals.Totals         `json:"totals"`
}

func (r RealizationRunBase) Validate() error {
	var errs []error

	if err := r.ID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("namespaced id: %w", err))
	}

	if err := r.ManagedModel.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("managed model: %w", err))
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

	return errors.Join(errs...)
}

type RealizationRun struct {
	RealizationRunBase

	// Realizations
	CreditsAllocated creditrealization.Realizations `json:"creditsAllocated"`
	InvoiceUsage     *invoicedusage.AccruedUsage    `json:"invoicedUsage"`
	Payment          *payment.Invoiced              `json:"payment"`
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

	return errors.Join(errs...)
}

type RealizationRuns []RealizationRun

func (r RealizationRuns) Validate() error {
	var errs []error
	for idx, realizationRun := range r {
		if err := realizationRun.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("realization run[%d]: %w", idx, err))
		}
	}
	return errors.Join(errs...)
}

func (r RealizationRuns) GetByID(id string) (RealizationRun, error) {
	for _, run := range r {
		if run.ID.ID == id {
			return run, nil
		}
	}
	return RealizationRun{}, fmt.Errorf("realization run not found [id=%s]", id)
}
