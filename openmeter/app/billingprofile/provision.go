// Package billingprofile provisions the default billing profile created when an app
// is installed with CreateDefaultBillingProfile set. It is shared by every HTTP driver
// that installs apps (currently the v1 marketplace endpoints and the v3 apps endpoint)
// so the provisioning rules stay identical across API versions.
package billingprofile

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/openmeter/billing"
)

// CreateDefault creates a default billing profile for the installed app based on its type.
// Assign it to app.InstallAppV3Input.CreateDefaultBillingProfileFn (bound to concrete
// billingService/stripeAppService instances) to enable CreateDefaultBillingProfile.
func CreateDefault(ctx context.Context, billingService billing.Service, stripeAppService appstripe.Service, installedApp app.App) ([]app.CapabilityType, error) {
	switch installedApp.GetType() {
	case app.AppTypeStripe:
		return makeStripeDefaultBillingApp(ctx, billingService, stripeAppService, installedApp)
	case app.AppTypeSandbox:
		namespace := installedApp.GetID().Namespace
		if err := billingService.ProvisionDefaultBillingProfile(ctx, namespace); err != nil {
			return nil, fmt.Errorf("provision default billing profile: %w", err)
		}
		return []app.CapabilityType{
			app.CapabilityTypeCalculateTax,
			app.CapabilityTypeInvoiceCustomers,
			app.CapabilityTypeCollectPayments,
		}, nil
	case app.AppTypeCustomInvoicing:
		// TODO: Implement custom invoicing billing profile creation
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown app type: %s", installedApp.GetType())
	}
}

// Make Stripe app the default billing app if current one is Sandbox app
func makeStripeDefaultBillingApp(ctx context.Context, billingService billing.Service, stripeAppService appstripe.Service, stripeApp app.App) ([]app.CapabilityType, error) {
	defaultForCapabilityTypes := []app.CapabilityType{}

	appID := stripeApp.GetID()

	// Check if it's a Stripe app
	if stripeApp.GetType() != app.AppTypeStripe {
		return defaultForCapabilityTypes, fmt.Errorf("app is not a stripe app: %s", appID.ID)
	}

	// Check if the default billing profile is a sandbox app type
	defaultBillingProfile, err := billingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
		Namespace: appID.Namespace,
	})
	if err != nil {
		return defaultForCapabilityTypes, fmt.Errorf("failed to get default billing profile: %w", err)
	}

	// Set default billing profile if the current default is the sandbox
	setDefaultBillingProfile := defaultBillingProfile != nil && defaultBillingProfile.Apps != nil && defaultBillingProfile.Apps.Invoicing.GetType() == app.AppTypeSandbox

	// Get supplier contract from stripe app
	supplierContract, err := stripeAppService.GetSupplierContact(ctx, appstripe.GetSupplierContactInput{
		AppID: appID,
	})
	if err != nil {
		return defaultForCapabilityTypes, fmt.Errorf("failed to get supplier contract for stripe app %s: %w", appID.ID, err)
	}

	// Create new default billing profile
	_, err = billingService.CreateProfile(ctx, billing.CreateProfileInput{
		Namespace:      appID.Namespace,
		Name:           "Stripe Billing Profile",
		Description:    lo.ToPtr("Stripe Billing Profile, created automatically"),
		Default:        setDefaultBillingProfile,
		Supplier:       supplierContract,
		WorkflowConfig: billing.DefaultWorkflowConfig,
		Apps: billing.ProfileAppReferences{
			Tax:       appID,
			Invoicing: appID,
			Payment:   appID,
		},
	})
	if err != nil {
		return defaultForCapabilityTypes, fmt.Errorf("failed to create billing profile for stripe app %s: %w", appID.ID, err)
	}

	defaultForCapabilityTypes = []app.CapabilityType{
		app.CapabilityTypeCalculateTax,
		app.CapabilityTypeInvoiceCustomers,
		app.CapabilityTypeCollectPayments,
	}

	return defaultForCapabilityTypes, nil
}
