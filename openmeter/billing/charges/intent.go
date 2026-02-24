package charges

import (
	"errors"
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type IntentMeta struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`

	Metadata    models.Metadata              `json:"metadata"`
	Annotations models.Annotations           `json:"annotations"`
	ManagedBy   billing.InvoiceLineManagedBy `json:"managedBy"`
	CustomerID  string                       `json:"customerID"`

	Currency currencyx.Code `json:"currency"`

	ServicePeriod     timeutil.ClosedPeriod `json:"servicePeriod"`
	FullServicePeriod timeutil.ClosedPeriod `json:"fullServicePeriod"`
	BillingPeriod     timeutil.ClosedPeriod `json:"billingPeriod"`

	TaxConfig         *productcatalog.TaxConfig `json:"taxConfig"`
	UniqueReferenceID *string                   `json:"childUniqueReferenceID"`

	Subscription *SubscriptionReference `json:"subscription"`
}

func (i IntentMeta) Validate() error {
	var errs []error

	if !slices.Contains(billing.InvoiceLineManagedBy("").Values(), string(i.ManagedBy)) {
		errs = append(errs, fmt.Errorf("invalid managed by %s", i.ManagedBy))
	}

	if i.Name == "" {
		errs = append(errs, fmt.Errorf("name is required"))
	}

	if i.CustomerID == "" {
		errs = append(errs, fmt.Errorf("customer ID is required"))
	}

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if err := i.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if err := i.FullServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("full service period: %w", err))
	}

	if err := i.BillingPeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("billing period: %w", err))
	}

	if i.TaxConfig != nil {
		if err := i.TaxConfig.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("tax config: %w", err))
		}
	}

	if i.Subscription != nil {
		if err := i.Subscription.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("subscription: %w", err))
		}
	}

	if i.UniqueReferenceID != nil && *i.UniqueReferenceID == "" {
		errs = append(errs, fmt.Errorf("unique reference ID cannot be empty"))
	}

	return errors.Join(errs...)
}
