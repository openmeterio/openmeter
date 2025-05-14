package billing

import (
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/isodate"
)

const (
	DefaultMeterResolution = time.Minute
)

var DefaultWorkflowConfig = WorkflowConfig{
	Collection: CollectionConfig{
		Alignment: AlignmentKindSubscription,
		Interval:  lo.Must(isodate.String("PT1H").Parse()),
	},
	Invoicing: InvoicingConfig{
		AutoAdvance:        true,
		DraftPeriod:        lo.Must(isodate.String("P0D").Parse()),
		DueAfter:           lo.Must(isodate.String("P30D").Parse()),
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
