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
		Interval:  lo.Must(isodate.String("PT2H").Parse()),
	},
	Invoicing: InvoicingConfig{
		AutoAdvance:        true,
		DraftPeriod:        lo.Must(isodate.String("P0D").Parse()),
		DueAfter:           lo.Must(isodate.String("P1W").Parse()),
		ProgressiveBilling: false,
		DefaultTaxConfig:   nil,
	},
	Payment: PaymentConfig{
		CollectionMethod: CollectionMethodChargeAutomatically,
	},
}
