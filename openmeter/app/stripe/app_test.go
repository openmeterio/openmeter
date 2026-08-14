package appstripe

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestDeletedAppPreservesIdentityAndRejectsOperations(t *testing.T) {
	appBase := app.AppBase{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
			ID:        "app-id",
			Namespace: "namespace",
			Name:      "Stripe",
		}),
		Type: app.AppTypeStripe,
	}

	deleted := NewDeleted(Meta{AppBase: appBase})
	require.Equal(t, appBase.GetID(), deleted.GetID())
	require.Equal(t, app.AppTypeStripe, deleted.GetType())
	require.ErrorIs(t, deleted.ValidateCapabilities(app.CapabilityTypeInvoiceCustomers), app.ErrAppDeleted)
	require.ErrorIs(t, deleted.DeleteStandardInvoice(t.Context(), billing.StandardInvoice{}), billing.WarnInvoiceWorkflowAppDeleteSkipped)
}
