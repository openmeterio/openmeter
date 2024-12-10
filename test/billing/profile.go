package billing

import (
	"github.com/samber/lo"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
)

var MinimalCreateProfileInputTemplate = billing.CreateProfileInput{
	Name:    "Awesome Profile",
	Default: true,

	WorkflowConfig: billing.WorkflowConfig{
		Collection: billing.CollectionConfig{
			Alignment: billing.AlignmentKindSubscription,
			Interval:  lo.Must(datex.ISOString("PT2H").Parse()),
		},
		Invoicing: billing.InvoicingConfig{
			AutoAdvance: true,
			DraftPeriod: lo.Must(datex.ISOString("P1D").Parse()),
			DueAfter:    lo.Must(datex.ISOString("P1W").Parse()),
		},
		Payment: billing.PaymentConfig{
			CollectionMethod: billing.CollectionMethodChargeAutomatically,
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
			Type: appentitybase.AppTypeSandbox,
		},
		Payment: billing.AppReference{
			Type: appentitybase.AppTypeSandbox,
		},
		Tax: billing.AppReference{
			Type: appentitybase.AppTypeSandbox,
		},
	},
}
