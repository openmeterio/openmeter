package appservice

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestInstallAppRejectsCustomInvoicingAutoProvisioning(t *testing.T) {
	// Given a Custom Invoicing install request with automatic billing profile provisioning.
	svc := &Service{}

	// When the install service validates the request.
	_, err := svc.InstallApp(t.Context(), app.InstallAppV3Input{
		MarketplaceListingID:        app.MarketplaceListingID{Type: app.AppTypeCustomInvoicing},
		Namespace:                   "test-namespace",
		CreateDefaultBillingProfile: true,
	})

	// Then it fails as validation before touching the install adapter.
	require.ErrorIs(t, err, app.ErrCustomInvoicingAutoProvisioningUnsupported)
	require.True(t, models.IsGenericValidationError(err))
}
