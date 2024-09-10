package subscription

import (
	"fmt"
	"reflect"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/datex"
)

// RateCard is a local implementation of plan.RateCard until productcatalog models are available
// TODO: extract ProductCatalog models and use them, doing it like this is a mess....
type RateCard struct {
	// Name of the RateCard
	Name string `json:"name"`

	// Description for the RateCard
	Description *string `json:"description,omitempty"`

	// Feature defines optional Feature assigned to RateCard
	FeatureKey *string `json:"featureKey,omitempty"`

	// EntitlementTemplate defines the template used for instantiating entitlement.Entitlement.
	// If Feature is set then template must be provided as well.
	EntitlementTemplate *productcatalog.EntitlementTemplate `json:"entitlementTemplate,omitempty"`

	// TaxConfig defines provider specific tax information.
	TaxConfig *productcatalog.TaxConfig `json:"taxConfig,omitempty"`

	// Price defines the price for the RateCard
	Price *productcatalog.Price `json:"price"`

	// BillingCadence defines the billing cadence of the RateCard in ISO8601 format.
	// Example: "P1D12H"
	BillingCadence *datex.Period `json:"billingCadence"`
}

func (r RateCard) Equals(other RateCard) bool {
	return reflect.DeepEqual(r, other)
}

// TODO: these should live on actual RateCard model once it exists
func (r RateCard) Validate() error {
	// Lets validate all nested models
	if r.EntitlementTemplate != nil {
		if err := r.EntitlementTemplate.Validate(); err != nil {
			return fmt.Errorf("invalid EntitlementTemplate: %w", err)
		}
	}

	if r.TaxConfig != nil {
		if err := r.TaxConfig.Validate(); err != nil {
			return fmt.Errorf("invalid TaxConfig: %w", err)
		}
	}

	if r.Price != nil {
		if err := r.Price.Validate(); err != nil {
			return fmt.Errorf("invalid Price: %w", err)
		}
	}

	// Let's validate that everything around the Price is configured correctly
	if r.Price != nil {
		// If a Price is configured, so must be Billing Cadence and Tax Config
		if r.BillingCadence == nil {
			return fmt.Errorf("missing billing cadence")
		}
		if r.TaxConfig == nil {
			return fmt.Errorf("missing tax config")
		}

		// If the price is usage based, feature must also be configured
		switch r.Price.Type() {
		case productcatalog.TieredPriceType, productcatalog.UnitPriceType:
			if r.FeatureKey == nil {
				return fmt.Errorf("feature must be defined for usage based price")
			}
		}
	}

	return nil
}

// TODO: PaymentTerm makes sense for all prices, but it's not available from the price type, should be moved there
func GetPricePaymentTerm(price productcatalog.Price) productcatalog.PaymentTermType {
	switch price.Type() {
	case productcatalog.FlatPriceType:
		p, err := price.AsFlat()
		if err != nil {
			break
		}
		return p.PaymentTerm
	case productcatalog.TieredPriceType:
		return productcatalog.InArrearsPaymentTerm
	case productcatalog.UnitPriceType:
		return productcatalog.InArrearsPaymentTerm
	}

	// It makes no sense for this method signature to error, so we'll use the default payment term
	// if it cannot be determined
	return productcatalog.DefaultPaymentTerm
}
