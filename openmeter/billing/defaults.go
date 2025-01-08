package billing

import (
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/datex"
)

const (
	DefaultMeterResolution = time.Minute
)

var DefaultWorkflowConfig = WorkflowConfig{
	Collection: CollectionConfig{
		Alignment: AlignmentKindSubscription,
		Interval:  lo.Must(datex.ISOString("PT2H").Parse()),
	},
	Invoicing: InvoicingConfig{
		AutoAdvance:        true,
		DraftPeriod:        lo.Must(datex.ISOString("P1D").Parse()),
		DueAfter:           lo.Must(datex.ISOString("P1W").Parse()),
		ProgressiveBilling: false,
	},
	Payment: PaymentConfig{
		CollectionMethod: CollectionMethodChargeAutomatically,
	},
}
