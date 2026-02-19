package charges

import (
	"errors"
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ChargeStatus string

const (
	// ChargeStatusActive is the status of a charge that is active and is not yet fully settled for the service period.
	ChargeStatusActive ChargeStatus = "active"
	// ChargeStatusSettled is the status of a charge that is settled and is fully settled for the service period. The charge might receive additional
	// late events in the future.
	ChargeStatusSettled ChargeStatus = "settled"
	// ChargeStatusFinal is the status of a charge that is final and is fully settled for the service period. The charge will not receive any additional
	// late events in the future.
	ChargeStatusFinal ChargeStatus = "final"
)

func (s ChargeStatus) Values() []string {
	return []string{
		string(ChargeStatusActive),
		string(ChargeStatusSettled),
		string(ChargeStatusFinal),
	}
}

func (s ChargeStatus) Validate() error {
	if !slices.Contains(s.Values(), string(s)) {
		return fmt.Errorf("invalid charge status: %s", s)
	}

	return nil
}

type ChargeID models.NamespacedID

func (i ChargeID) Validate() error {
	return models.NamespacedID(i).Validate()
}

type Charge struct {
	models.ManagedResource

	Intent       Intent       `json:"intent"`
	Status       ChargeStatus `json:"status"`
	Realizations Realizations `json:"realizations"`

	// TODO: Should this be a realization?
	Expanded ChargeExpanded `json:"expanded"`
}

func (c Charge) Validate() error {
	var errs []error

	if err := c.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent base: %w", err))
	}

	if c.Name == "" {
		errs = append(errs, fmt.Errorf("name is required"))
	}

	if c.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace is required"))
	}

	if err := c.Realizations.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("realizations: %w", err))
	}

	if err := c.Status.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("status: %w", err))
	}

	return errors.Join(errs...)
}

func (c Charge) GetChargeID() ChargeID {
	return ChargeID{
		Namespace: c.Namespace,
		ID:        c.ID,
	}
}

type Charges []Charge

type Realizations struct {
	StandardInvoice StandardInvoiceRealizations `json:"standardInvoice"`
}

func (r Realizations) Validate() error {
	var errs []error

	for idx, realization := range r.StandardInvoice {
		if err := realization.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("standard invoice realization[%d]: %w", idx, err))
		}
	}

	return errors.Join(errs...)
}

type ChargeExpanded struct {
	GatheringLines []billing.GatheringLineWithInvoiceHeader `json:"gatheringLines"`
}
