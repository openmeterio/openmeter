package httpdriver

import (
	"github.com/samber/lo"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/pkg/datex"
)

const (
	DefaultPageSize        = 100
	DefaultPageNumber      = 1
	DefaultIncludeArchived = false
	DefaultInvoiceTimezone = "UTC"
)

var defaultWorkflowConfig = billingentity.WorkflowConfig{
	Collection: billingentity.CollectionConfig{
		Alignment: billingentity.AlignmentKindSubscription,
		Interval:  lo.Must(datex.ISOString("PT2H").Parse()),
	},
	Invoicing: billingentity.InvoicingConfig{
		AutoAdvance: true,
		DraftPeriod: lo.Must(datex.ISOString("P1D").Parse()),
		DueAfter:    lo.Must(datex.ISOString("P1W").Parse()),
	},
	Payment: billingentity.PaymentConfig{
		CollectionMethod: billingentity.CollectionMethodChargeAutomatically,
	},
}
