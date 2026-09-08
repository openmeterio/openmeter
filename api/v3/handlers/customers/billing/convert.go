//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./
package customersbilling

import (
	"context"
	"fmt"

	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"

	apilegacy "github.com/openmeterio/openmeter/api"
	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

// goverter:variables
// goverter:skipCopySameType
// goverter:output:file ./convert.gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:useUnderlyingTypeMethods
// goverter:matchIgnoreCase
// goverter:enum:unknown @error
// goverter:extend ResolveIDFromCustomerId
var (
	FromAPIBillingAppStripeCreateCheckoutSessionRequestOptions func(source api.BillingAppStripeCreateCheckoutSessionRequestOptions) (apilegacy.CreateStripeCheckoutSessionRequestOptions, error)

	// goverter:enum:map BillingAppStripeCreateCheckoutSessionBillingAddressCollectionAuto CreateStripeCheckoutSessionBillingAddressCollectionAuto
	// goverter:enum:map BillingAppStripeCreateCheckoutSessionBillingAddressCollectionRequired CreateStripeCheckoutSessionBillingAddressCollectionRequired
	FromAPIBillingAppStripeCreateCheckoutSessionBillingAddressCollection func(source api.BillingAppStripeCreateCheckoutSessionBillingAddressCollection) (apilegacy.CreateStripeCheckoutSessionBillingAddressCollection, error)

	// goverter:enum:map BillingAppStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPositionAuto CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPositionAuto
	// goverter:enum:map BillingAppStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPositionHidden CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPositionHidden
	FromAPIBillingAppStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition func(source api.BillingAppStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition) (apilegacy.CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition, error)

	// goverter:enum:map BillingAppStripeCreateCheckoutSessionConsentCollectionPromotionsAuto CreateStripeCheckoutSessionConsentCollectionPromotionsAuto
	// goverter:enum:map BillingAppStripeCreateCheckoutSessionConsentCollectionPromotionsNone CreateStripeCheckoutSessionConsentCollectionPromotionsNone
	FromAPIBillingAppStripeCreateCheckoutSessionConsentCollectionPromotions func(source api.BillingAppStripeCreateCheckoutSessionConsentCollectionPromotions) (apilegacy.CreateStripeCheckoutSessionConsentCollectionPromotions, error)

	// goverter:enum:map BillingAppStripeCreateCheckoutSessionConsentCollectionTermsOfServiceNone CreateStripeCheckoutSessionConsentCollectionTermsOfServiceNone
	// goverter:enum:map BillingAppStripeCreateCheckoutSessionConsentCollectionTermsOfServiceRequired CreateStripeCheckoutSessionConsentCollectionTermsOfServiceRequired
	FromAPIBillingAppStripeCreateCheckoutSessionConsentCollectionTermsOfService func(source api.BillingAppStripeCreateCheckoutSessionConsentCollectionTermsOfService) (apilegacy.CreateStripeCheckoutSessionConsentCollectionTermsOfService, error)

	// goverter:enum:map BillingAppStripeCreateCheckoutSessionCustomerUpdateBehaviorAuto CreateStripeCheckoutSessionCustomerUpdateBehaviorAuto
	// goverter:enum:map BillingAppStripeCreateCheckoutSessionCustomerUpdateBehaviorNever CreateStripeCheckoutSessionCustomerUpdateBehaviorNever
	FromAPIBillingAppStripeCreateCheckoutSessionCustomerUpdateBehavior func(source api.BillingAppStripeCreateCheckoutSessionCustomerUpdateBehavior) (apilegacy.CreateStripeCheckoutSessionCustomerUpdateBehavior, error)

	// goverter:enum:map BillingAppStripeCreateCheckoutSessionRedirectOnCompletionAlways CreateStripeCheckoutSessionRedirectOnCompletionAlways
	// goverter:enum:map BillingAppStripeCreateCheckoutSessionRedirectOnCompletionIfRequired CreateStripeCheckoutSessionRedirectOnCompletionIfRequired
	// goverter:enum:map BillingAppStripeCreateCheckoutSessionRedirectOnCompletionNever CreateStripeCheckoutSessionRedirectOnCompletionNever
	FromAPIBillingAppStripeCreateCheckoutSessionRedirectOnCompletion func(source api.BillingAppStripeCreateCheckoutSessionRedirectOnCompletion) (apilegacy.CreateStripeCheckoutSessionRedirectOnCompletion, error)

	// goverter:enum:map BillingAppStripeCreateCheckoutSessionTaxIdCollectionRequiredIfSupported CreateCheckoutSessionTaxIdCollectionRequiredIfSupported
	// goverter:enum:map BillingAppStripeCreateCheckoutSessionTaxIdCollectionRequiredNever CreateCheckoutSessionTaxIdCollectionRequiredNever
	FromAPIBillingAppStripeCreateCheckoutSessionTaxIdCollectionRequired func(source api.BillingAppStripeCreateCheckoutSessionTaxIdCollectionRequired) (apilegacy.CreateCheckoutSessionTaxIdCollectionRequired, error)

	// goverter:enum:map BillingAppStripeCheckoutSessionUIModeEmbedded CheckoutSessionUIModeEmbedded
	// goverter:enum:map BillingAppStripeCheckoutSessionUIModeHosted CheckoutSessionUIModeHosted
	FromAPIBillingAppStripeCheckoutSessionUIMode func(source api.BillingAppStripeCheckoutSessionUIMode) (apilegacy.CheckoutSessionUIMode, error)

	// goverter:map Configuration.ID ConfigurationId
	ToAPIBillingAppStripeCreateCustomerPortalSessionResult func(portalSession appstripe.StripePortalSession) api.BillingAppStripeCreateCustomerPortalSessionResult

	// goverter:autoMap StripeCheckoutSession
	ToAPIBillingAppStripeCreateCheckoutSessionResult func(source appstripe.CreateCheckoutSessionOutput) api.BillingAppStripeCreateCheckoutSessionResult
)

func ResolveIDFromCustomerId(namespacedID customer.CustomerID) string {
	return namespacedID.ID
}

func fromAPIBillingAppCustomerData(ctx context.Context, application app.App, data app.CustomerData) (*api.BillingAppCustomerData, error) {
	appData := &api.BillingAppCustomerData{}

	switch application.GetType() {
	case app.AppTypeStripe:
		// data is nil when the customer has no stripe data for the configured
		// payment app
		if data == nil {
			appData.Stripe = nullable.NewNullNullable[api.BillingAppCustomerDataStripe]()
			return appData, nil
		}

		// The handler selected the app by type, so the type must match. A
		// mismatch is a programming error, not a "no data" state.
		stripeData, ok := data.(appstripe.CustomerData)
		if !ok {
			return nil, apierrors.NewInternalError(ctx, fmt.Errorf("stripe app returned %T, want appstripe.CustomerData", data))
		}

		// TODO: we don't have metadata on the stripe customer data yet
		appData.Stripe = nullable.NewNullableWithValue(api.BillingAppCustomerDataStripe{
			CustomerId:             &stripeData.StripeCustomerID,
			DefaultPaymentMethodId: stripeData.StripeDefaultPaymentMethodID,
		})
		return appData, nil

	case app.AppTypeCustomInvoicing:
		if data == nil {
			appData.ExternalInvoicing = nullable.NewNullNullable[api.BillingAppCustomerDataExternalInvoicing]()
			return appData, nil
		}

		invoicingData, ok := data.(appcustominvoicing.CustomerData)
		if !ok {
			return nil, apierrors.NewInternalError(ctx, fmt.Errorf("custom invoicing app returned %T, want appcustominvoicing.CustomerData", data))
		}

		appData.ExternalInvoicing = nullable.NewNullableWithValue(api.BillingAppCustomerDataExternalInvoicing{
			Labels: (*api.Labels)(lo.ToPtr(invoicingData.Metadata.ToMap())),
		})
		return appData, nil

	case app.AppTypeSandbox:
		// No app data.
		return nil, nil

	default:
		return nil, apierrors.NewInternalError(ctx, fmt.Errorf("unsupported app type: %s", application.GetType()))
	}
}
