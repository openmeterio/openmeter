package billingservice

import (
	"github.com/openmeterio/openmeter/openmeter/billing"
)

var _ billing.GatheringInvoiceService = (*Service)(nil)
