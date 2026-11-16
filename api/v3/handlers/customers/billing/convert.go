//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./
package customersbilling

import (
	"context"
	"fmt"

	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/openmeter/billing"
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
	FromAPIBillingAppStripeCreateCheckoutSessionRequestOptions func(source apiv3models.BillingAppStripeCreateCheckoutSessionRequestOptions) (apiv1models.CreateStripeCheckoutSessionRequestOptions, error)

	// goverter:enum:map BillingAppStripeCreateCheckoutSessionBillingAddressCollectionAuto CreateStripeCheckoutSessionBillingAddressCollectionAuto
	// goverter:enum:map BillingAppStripeCreateCheckoutSessionBillingAddressCollectionRequired CreateStripeCheckoutSessionBillingAddressCollectionRequired
	FromAPIBillingAppStripeCreateCheckoutSessionBillingAddressCollection func(source apiv3models.BillingAppStripeCreateCheckoutSessionBillingAddressCollection) (apiv1models.CreateStripeCheckoutSessionBillingAddressCollection, error)

	// goverter:enum:map BillingAppStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPositionAuto CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPositionAuto
	// goverter:enum:map BillingAppStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPositionHidden CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPositionHidden
	FromAPIBillingAppStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition func(source apiv3models.BillingAppStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition) (apiv1models.CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition, error)

	// goverter:enum:map BillingAppStripeCreateCheckoutSessionConsentCollectionPromotionsAuto CreateStripeCheckoutSessionConsentCollectionPromotionsAuto
	// goverter:enum:map BillingAppStripeCreateCheckoutSessionConsentCollectionPromotionsNone CreateStripeCheckoutSessionConsentCollectionPromotionsNone
	FromAPIBillingAppStripeCreateCheckoutSessionConsentCollectionPromotions func(source apiv3models.BillingAppStripeCreateCheckoutSessionConsentCollectionPromotions) (apiv1models.CreateStripeCheckoutSessionConsentCollectionPromotions, error)

	// goverter:enum:map BillingAppStripeCreateCheckoutSessionConsentCollectionTermsOfServiceNone CreateStripeCheckoutSessionConsentCollectionTermsOfServiceNone
	// goverter:enum:map BillingAppStripeCreateCheckoutSessionConsentCollectionTermsOfServiceRequired CreateStripeCheckoutSessionConsentCollectionTermsOfServiceRequired
	FromAPIBillingAppStripeCreateCheckoutSessionConsentCollectionTermsOfService func(source apiv3models.BillingAppStripeCreateCheckoutSessionConsentCollectionTermsOfService) (apiv1models.CreateStripeCheckoutSessionConsentCollectionTermsOfService, error)

	// goverter:enum:map BillingAppStripeCreateCheckoutSessionCustomerUpdateBehaviorAuto CreateStripeCheckoutSessionCustomerUpdateBehaviorAuto
	// goverter:enum:map BillingAppStripeCreateCheckoutSessionCustomerUpdateBehaviorNever CreateStripeCheckoutSessionCustomerUpdateBehaviorNever
	FromAPIBillingAppStripeCreateCheckoutSessionCustomerUpdateBehavior func(source apiv3models.BillingAppStripeCreateCheckoutSessionCustomerUpdateBehavior) (apiv1models.CreateStripeCheckoutSessionCustomerUpdateBehavior, error)

	// goverter:enum:map BillingAppStripeCreateCheckoutSessionRedirectOnCompletionAlways CreateStripeCheckoutSessionRedirectOnCompletionAlways
	// goverter:enum:map BillingAppStripeCreateCheckoutSessionRedirectOnCompletionIfRequired CreateStripeCheckoutSessionRedirectOnCompletionIfRequired
	// goverter:enum:map BillingAppStripeCreateCheckoutSessionRedirectOnCompletionNever CreateStripeCheckoutSessionRedirectOnCompletionNever
	FromAPIBillingAppStripeCreateCheckoutSessionRedirectOnCompletion func(source apiv3models.BillingAppStripeCreateCheckoutSessionRedirectOnCompletion) (apiv1models.CreateStripeCheckoutSessionRedirectOnCompletion, error)

	// goverter:enum:map BillingAppStripeCreateCheckoutSessionTaxIdCollectionRequiredIfSupported CreateCheckoutSessionTaxIdCollectionRequiredIfSupported
	// goverter:enum:map BillingAppStripeCreateCheckoutSessionTaxIdCollectionRequiredNever CreateCheckoutSessionTaxIdCollectionRequiredNever
	FromAPIBillingAppStripeCreateCheckoutSessionTaxIdCollectionRequired func(source apiv3models.BillingAppStripeCreateCheckoutSessionTaxIdCollectionRequired) (apiv1models.CreateCheckoutSessionTaxIdCollectionRequired, error)

	// goverter:enum:map BillingAppStripeCheckoutSessionUIModeEmbedded CheckoutSessionUIModeEmbedded
	// goverter:enum:map BillingAppStripeCheckoutSessionUIModeHosted CheckoutSessionUIModeHosted
	FromAPIBillingAppStripeCheckoutSessionUIMode func(source apiv3models.BillingAppStripeCheckoutSessionUIMode) (apiv1models.CheckoutSessionUIMode, error)

	// goverter:map Configuration.ID ConfigurationId
	ToAPIBillingAppStripeCreateCustomerPortalSessionResult func(portalSession appstripe.StripePortalSession) apiv3models.BillingAppStripeCreateCustomerPortalSessionResult

	// goverter:autoMap StripeCheckoutSession
	ToAPIBillingAppStripeCreateCheckoutSessionResult func(source appstripe.CreateCheckoutSessionOutput) apiv3models.BillingAppStripeCreateCheckoutSessionResult
)

func ResolveIDFromCustomerId(namespacedID customer.CustomerID) string {
	return namespacedID.ID
}

func fromAPIBillingAppCustomerData(ctx context.Context, application app.App, data app.CustomerData) (*apiv3models.BillingAppCustomerData, error) {
	appData := &apiv3models.BillingAppCustomerData{}

	switch application.GetType() {
	case app.AppTypeStripe:
		// data is nil when the customer has no stripe data for the configured
		// payment app
		if data == nil {
			appData.Stripe = nullable.NewNullNullable[apiv3models.BillingAppCustomerDataStripe]()
			return appData, nil
		}

		// The handler selected the app by type, so the type must match. A
		// mismatch is a programming error, not a "no data" state.
		stripeData, ok := data.(appstripe.CustomerData)
		if !ok {
			return nil, apierrors.NewInternalError(ctx, fmt.Errorf("stripe app returned %T, want appstripe.CustomerData", data))
		}

		// TODO: we don't have metadata on the stripe customer data yet
		appData.Stripe = nullable.NewNullableWithValue(apiv3models.BillingAppCustomerDataStripe{
			CustomerId:             &stripeData.StripeCustomerID,
			DefaultPaymentMethodId: stripeData.StripeDefaultPaymentMethodID,
		})

		return appData, nil

	case app.AppTypeCustomInvoicing:
		if data == nil {
			appData.ExternalInvoicing = nullable.NewNullNullable[apiv3models.BillingAppCustomerDataExternalInvoicing]()
			return appData, nil
		}

		invoicingData, ok := data.(appcustominvoicing.CustomerData)
		if !ok {
			return nil, apierrors.NewInternalError(ctx, fmt.Errorf("custom invoicing app returned %T, want appcustominvoicing.CustomerData", data))
		}

		appData.ExternalInvoicing = nullable.NewNullableWithValue(apiv3models.BillingAppCustomerDataExternalInvoicing{
			Labels: (*apiv3models.Labels)(lo.ToPtr(invoicingData.Metadata.ToMap())),
		})

		return appData, nil

	case app.AppTypeSandbox:
		// No app data.
		return nil, nil

	default:
		return nil, apierrors.NewInternalError(ctx, fmt.Errorf("unsupported app type: %s", application.GetType()))
	}
}

func mapCustomerBillingProfileToAPI(customerProfile billing.CustomerBillingProfile) apiv3models.BillingCustomerProfile {
	profile := apiv3models.BillingCustomerProfile{
		EffectiveBillingProfile: apiv3models.BillingProfileReference{Id: customerProfile.EffectiveProfile.ID},
	}
	if customerProfile.AssignedProfile != nil {
		profile.BillingProfile = nullable.NewNullableWithValue(apiv3models.BillingProfileReference{
			Id: customerProfile.AssignedProfile.ProfileID,
		})
	}

	return profile
}
