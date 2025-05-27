package billing

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

var minimalCreateProfileInputTemplate = billing.CreateProfileInput{
	Name:    "Awesome Profile",
	Default: true,

	WorkflowConfig: billing.WorkflowConfig{
		Collection: billing.CollectionConfig{
			Alignment: billing.AlignmentKindSubscription,
			// We set the interval to 0 so that the invoice is collected immediately, testcases
			// validating the collection logic can set a different interval
			Interval: lo.Must(isodate.String("PT0S").Parse()),
		},
		Invoicing: billing.InvoicingConfig{
			AutoAdvance: true,
			DraftPeriod: lo.Must(isodate.String("P1D").Parse()),
			DueAfter:    lo.Must(isodate.String("P1W").Parse()),
		},
		Payment: billing.PaymentConfig{
			CollectionMethod: billing.CollectionMethodChargeAutomatically,
		},
		Tax: billing.WorkflowTaxConfig{
			Enabled:  true,
			Enforced: false,
		},
	},

	Supplier: billing.SupplierContact{
		Name: "Awesome Supplier",
		Address: models.Address{
			Country: lo.ToPtr(models.CountryCode("US")),
		},
	},

	Apps: billing.CreateProfileAppsInput{
		Invoicing: billing.AppReference{
			Type: app.AppTypeSandbox,
		},
		Payment: billing.AppReference{
			Type: app.AppTypeSandbox,
		},
		Tax: billing.AppReference{
			Type: app.AppTypeSandbox,
		},
	},
}
