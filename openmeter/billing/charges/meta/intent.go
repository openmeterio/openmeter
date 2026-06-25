package meta

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type Intent struct {
	ManagedBy  billing.InvoiceLineManagedBy `json:"managedBy"`
	CustomerID string                       `json:"customerID"`

	Annotations models.Annotations `json:"annotations"`

	Currency currencyx.Code `json:"currency"`

	UniqueReferenceID *string                `json:"childUniqueReferenceID"`
	Subscription      *SubscriptionReference `json:"subscription"`
}

func (i Intent) Clone() Intent {
	out := i

	// Keep intent cloning infallible for developer ergonomics; annotations are
	// only shallow-cloned here so GetEffectiveIntent does not need an error return.
	out.Annotations = maps.Clone(i.Annotations)

	if i.UniqueReferenceID != nil {
		out.UniqueReferenceID = lo.ToPtr(*i.UniqueReferenceID)
	}

	if i.Subscription != nil {
		out.Subscription = lo.ToPtr(*i.Subscription)
	}

	return out
}

func (i Intent) Validate() error {
	var errs []error

	if !slices.Contains(billing.InvoiceLineManagedBy("").Values(), string(i.ManagedBy)) {
		errs = append(errs, fmt.Errorf("invalid managed by %s", i.ManagedBy))
	}

	if i.CustomerID == "" {
		errs = append(errs, fmt.Errorf("customer ID is required"))
	}

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if i.Subscription != nil {
		if err := i.Subscription.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("subscription: %w", err))
		}
	}

	if i.UniqueReferenceID != nil && *i.UniqueReferenceID == "" {
		errs = append(errs, fmt.Errorf("unique reference ID cannot be empty"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type IntentMutableFields struct {
	Name        string          `json:"name"`
	Description *string         `json:"description"`
	Metadata    models.Metadata `json:"metadata"`

	ServicePeriod     timeutil.ClosedPeriod `json:"servicePeriod"`
	FullServicePeriod timeutil.ClosedPeriod `json:"fullServicePeriod"`
	BillingPeriod     timeutil.ClosedPeriod `json:"billingPeriod"`

	TaxConfig productcatalog.TaxCodeConfig `json:"taxConfig"`
}

func (i IntentMutableFields) Clone() IntentMutableFields {
	out := i

	if i.Description != nil {
		out.Description = lo.ToPtr(*i.Description)
	}

	out.Metadata = i.Metadata.Clone()

	if i.TaxConfig.Behavior != nil {
		out.TaxConfig.Behavior = lo.ToPtr(*i.TaxConfig.Behavior)
	}

	return out
}

func (i IntentMutableFields) Validate() error {
	var errs []error

	if i.Name == "" {
		errs = append(errs, fmt.Errorf("name is required"))
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

	if err := i.TaxConfig.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("tax config: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
