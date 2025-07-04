package billing

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func minimalCreateProfileInputTemplate(appID app.AppID) billing.CreateProfileInput {
	return billing.CreateProfileInput{
		Name:    "Awesome Profile",
		Default: true,

		WorkflowConfig: billing.WorkflowConfig{
			Collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindSubscription,
				// We set the interval to 0 so that the invoice is collected immediately, testcases
				// validating the collection logic can set a different interval
				Interval: lo.Must(datetime.ISODurationString("PT0S").Parse()),
			},
			Invoicing: billing.InvoicingConfig{
				AutoAdvance: true,
				DraftPeriod: lo.Must(datetime.ISODurationString("P1D").Parse()),
				DueAfter:    lo.Must(datetime.ISODurationString("P1W").Parse()),
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
			Invoicing: appID,
			Payment:   appID,
			Tax:       appID,
		},
	}
}
