package reconciler

import (
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type PatchOperation string

const (
	PatchOperationCreate  PatchOperation = "create"
	PatchOperationDelete  PatchOperation = "delete"
	PatchOperationShrink  PatchOperation = "shrink"
	PatchOperationExtend  PatchOperation = "extend"
	PatchOperationProrate PatchOperation = "prorate"
)

type GetInvoicePatchesInput struct {
	Subscription subscription.Subscription
	Currency     currencyx.Calculator
	Invoices     persistedstate.Invoices
}

type Patch interface {
	Operation() PatchOperation
	UniqueReferenceID() string
	GetInvoicePatches(input GetInvoicePatchesInput) ([]invoiceupdater.Patch, error)
}
