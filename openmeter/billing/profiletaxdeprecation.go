package billing

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

const profileTaxConfigDeprecationMessage = "setting a tax code (stripe.code / taxCodeId) on a billing profile's defaultTaxConfig is deprecated and can no longer be added or changed; the organization default tax code is used instead. You may still remove it. (behavior is unaffected.)"

// CheckProfileTaxConfigDeprecation returns a ValidationError when incoming adds or changes a
// deprecated tax-code field (stripe.code or taxCodeId) relative to stored. stored is nil on
// create. Full removal (both fields cleared together) is permitted; partial removal (clearing
// only one when both are stored) is not. Behavior is never restricted. Compare the raw
// API->domain mapped incoming config, BEFORE tax-code resolution.
func CheckProfileTaxConfigDeprecation(stored, incoming *productcatalog.TaxConfig) error {
	if incoming == nil {
		return nil
	}

	taxCodeChanged := incoming.TaxCodeID != nil &&
		(stored == nil || stored.TaxCodeID == nil || *stored.TaxCodeID != *incoming.TaxCodeID)

	stripeCodeChanged := incoming.Stripe != nil && incoming.Stripe.Code != "" &&
		(stored == nil || stored.Stripe == nil || stored.Stripe.Code != incoming.Stripe.Code)

	// When stored has both fields they are treated as a unit: partial removal (clearing one
	// while keeping the other) is not permitted — only full removal of both is allowed.
	incomingLacksTaxCodeID := incoming.TaxCodeID == nil
	incomingLacksStripeCode := incoming.Stripe == nil || incoming.Stripe.Code == ""

	storedHasBoth := stored != nil &&
		stored.TaxCodeID != nil &&
		stored.Stripe != nil && stored.Stripe.Code != ""

	partialRemoval := storedHasBoth && (incomingLacksTaxCodeID != incomingLacksStripeCode)

	if !taxCodeChanged && !stripeCodeChanged && !partialRemoval {
		return nil
	}

	return ValidationError{
		Err: models.NewGenericValidationError(errors.New(profileTaxConfigDeprecationMessage)),
	}
}
