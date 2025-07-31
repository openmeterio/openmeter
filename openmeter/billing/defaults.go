package billing

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/datetime"
)

var DefaultWorkflowConfig = WorkflowConfig{
	Collection: CollectionConfig{
		Alignment: AlignmentKindSubscription,
		Interval:  lo.Must(datetime.ISODurationString("PT1H").Parse()),
	},
	Invoicing: InvoicingConfig{
		AutoAdvance:        true,
		DraftPeriod:        lo.Must(datetime.ISODurationString("P0D").Parse()),
		DueAfter:           lo.Must(datetime.ISODurationString("P30D").Parse()),
		ProgressiveBilling: true,
		DefaultTaxConfig:   nil,
	},
	Payment: PaymentConfig{
		CollectionMethod: CollectionMethodChargeAutomatically,
	},
	Tax: WorkflowTaxConfig{
		// By default tax calculation is enabled when tax is supported by the app.
		Enabled: true,

		// By default tax is not enforced. Subscriptions can be created without tax location and
		// invoices can be finalized with missing tax location.
		Enforced: false,
	},
}
