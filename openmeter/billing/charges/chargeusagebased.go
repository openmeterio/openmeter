package charges

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type UsageBasedCharge struct {
	ManagedResource

	Intent UsageBasedIntent `json:"intent"`

	Status ChargeStatus `json:"status"`

	State UsageBasedState `json:"state"`
}

func (c UsageBasedCharge) Validate() error {
	var errs []error

	if err := c.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	if err := c.State.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("state: %w", err))
	}

	if err := c.Status.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("status: %w", err))
	}

	return errors.Join(errs...)
}

func (c UsageBasedCharge) AsCharge() Charge {
	return Charge{
		t:          ChargeTypeUsageBased,
		usageBased: &c,
	}
}

type UsageBasedIntent struct {
	IntentMeta

	Price          productcatalog.Price          `json:"price"`
	FeatureKey     string                        `json:"featureKey,omitempty"`
	InvoiceAt      time.Time                     `json:"invoiceAt"`
	SettlementMode productcatalog.SettlementMode `json:"settlementMode"`

	Discounts *productcatalog.Discounts `json:"rateCardDiscounts"`
}

func (i UsageBasedIntent) Validate() error {
	var errs []error

	if err := i.IntentMeta.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.InvoiceAt.IsZero() || i.InvoiceAt.Before(i.ServicePeriod.From) {
		errs = append(errs, fmt.Errorf("invoice at must be after service period from"))
	}

	if err := i.Price.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("price: %w", err))
	}

	if i.Discounts != nil {
		if err := i.Discounts.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("discounts: %w", err))
		}
	}

	if i.FeatureKey == "" {
		errs = append(errs, fmt.Errorf("feature key is required"))
	}

	if i.TaxConfig != nil {
		if err := i.TaxConfig.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("tax config: %w", err))
		}
	}

	if err := i.SettlementMode.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("settlement mode: %w", err))
	}

	return errors.Join(errs...)
}

type UsageBasedState struct {
	StandardInvoiceSettlements []StandardInvoiceSettlement `json:"standardInvoiceSettlements"`
}

func (s UsageBasedState) Validate() error {
	var errs []error

	for idx, si := range s.StandardInvoiceSettlements {
		if err := si.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("standard invoice settlement [%s/%d]: %w", si.ID, idx, err))
		}
	}

	return errors.Join(errs...)
}
