package usagebased

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ meta.ChargeAccessor = (*Charge)(nil)

type Charge struct {
	meta.ManagedResource

	Intent Intent `json:"intent"`
	Status Status `json:"status"`

	State State `json:"state"`
}

func (c Charge) Validate() error {
	var errs []error

	if err := c.ManagedResource.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("managed resource: %w", err))
	}

	if err := c.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	if err := c.Status.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("status: %w", err))
	}

	if err := c.State.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("state: %w", err))
	}

	return errors.Join(errs...)
}

func (c Charge) GetChargeID() meta.ChargeID {
	return meta.ChargeID{
		Namespace: c.Namespace,
		ID:        c.ID,
	}
}

func (c Charge) ErrorAttributes() models.Attributes {
	return models.Attributes{
		"charge_id":   c.ID,
		"namespace":   c.Namespace,
		"charge_type": string(meta.ChargeTypeUsageBased),
	}
}

type Intent struct {
	meta.Intent

	InvoiceAt      time.Time                     `json:"invoiceAt"`
	SettlementMode productcatalog.SettlementMode `json:"settlementMode"`

	FeatureKey string `json:"featureKey"`

	Price productcatalog.Price `json:"price"`

	Discounts productcatalog.Discounts `json:"discounts"`
}

func (i Intent) Validate() error {
	var errs []error

	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	if err := i.SettlementMode.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("settlement mode: %w", err))
	}

	if err := i.Discounts.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("discounts: %w", err))
	}

	if i.InvoiceAt.IsZero() {
		errs = append(errs, fmt.Errorf("invoice at is required"))
	}

	if err := i.Price.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("price: %w", err))
	}

	if i.FeatureKey == "" {
		errs = append(errs, fmt.Errorf("feature key is required"))
	}

	return errors.Join(errs...)
}

type State struct {
	RealizationRuns         RealizationRuns `json:"realizationRuns"`
	CurrentRealizationRunID *string         `json:"currentRealizationRunID"`
}

func (s State) Validate() error {
	var errs []error

	if s.CurrentRealizationRunID != nil && !slices.ContainsFunc(s.RealizationRuns, func(run RealizationRun) bool {
		return run.ID == *s.CurrentRealizationRunID
	}) {
		errs = append(errs, fmt.Errorf("current realization run id must reference one of the realization runs"))
	}

	if err := s.RealizationRuns.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("realization runs: %w", err))
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

type RealizationRunType string

const (
	RealizationRunTypeInvoice RealizationRunType = "invoice"
)

func (t RealizationRunType) Values() []string {
	return []string{
		string(RealizationRunTypeInvoice),
	}
}

func (t RealizationRunType) Validate() error {
	if !slices.Contains(t.Values(), string(t)) {
		return fmt.Errorf("invalid realization run type: %s", t)
	}
	return nil
}

type RealizationRun struct {
	models.NamespacedID
	models.ManagedModel

	Type       RealizationRunType    `json:"type"`
	AsOf       time.Time             `json:"asOf"`
	MeterValue alpacadecimal.Decimal `json:"meterValue"`

	// Realizations
	CreditsAllocated creditrealization.Realizations `json:"creditsAllocated"`
	InvoiceUsage     *invoicedusage.AccruedUsage    `json:"invoicedUsage"`
	Payment          *payment.Invoiced              `json:"payment"`
}

func (r RealizationRun) Validate() error {
	var errs []error

	if err := r.NamespacedID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("namespaced id: %w", err))
	}

	if err := r.ManagedModel.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("managed model: %w", err))
	}

	if err := r.Type.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("type: %w", err))
	}

	if r.MeterValue.IsNegative() {
		errs = append(errs, fmt.Errorf("meter value must be zero or positive"))
	}

	if r.AsOf.IsZero() {
		errs = append(errs, fmt.Errorf("as of must be set"))
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
