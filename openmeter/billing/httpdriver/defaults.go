package httpdriver

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/datex"
)

const (
	DefaultPageSize        = 100
	DefaultPageNumber      = 1
	DefaultIncludeArchived = false
	DefaultInvoiceTimezone = "UTC"
)

var defaultWorkflowConfig = billing.WorkflowConfig{
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
}
