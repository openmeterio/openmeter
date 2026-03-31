package reconciler

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
)

type invoicePatchCollectionBase struct {
	patches []InvoicePatch
}

func (c invoicePatchCollectionBase) GetBackendType() BackendType {
	return BackendTypeInvoicing
}

func newInvoicePatchCollectionBase(preallocatedCapacity int) invoicePatchCollectionBase {
	if preallocatedCapacity <= 0 {
		preallocatedCapacity = 16
	}

	return invoicePatchCollectionBase{
		patches: make([]InvoicePatch, 0, preallocatedCapacity),
	}
}

func (c *invoicePatchCollectionBase) addPatches(uniqueID string, operation PatchOperation, invoiceUpdates ...invoiceupdater.Patch) error {
	if uniqueID == "" {
		return fmt.Errorf("unique id is required [operation=%s]", operation)
	}

	if len(invoiceUpdates) == 0 {
		return fmt.Errorf("at least one invoice update is required for [operation=%s, uniqueID=%s]", operation, uniqueID)
	}

	c.patches = append(c.patches, newGenericInvoicePatch(uniqueID, operation, invoiceUpdates...))

	return nil
}

func (c invoicePatchCollectionBase) IsEmpty() bool {
	return len(c.patches) == 0
}

func (c invoicePatchCollectionBase) Patches() []InvoicePatch {
	return c.patches
}

type genericInvoicePatch struct {
	uniqueID       string
	operation      PatchOperation
	invoiceUpdates []invoiceupdater.Patch
}

func newGenericInvoicePatch(uniqueID string, operation PatchOperation, invoiceUpdates ...invoiceupdater.Patch) genericInvoicePatch {
	return genericInvoicePatch{
		uniqueID:       uniqueID,
		operation:      operation,
		invoiceUpdates: invoiceUpdates,
	}
}

func (p genericInvoicePatch) Operation() PatchOperation {
	return p.operation
}

func (p genericInvoicePatch) UniqueReferenceID() string {
	return p.uniqueID
}

func (p genericInvoicePatch) GetInvoicePatches() ([]invoiceupdater.Patch, error) {
	return p.invoiceUpdates, nil
}
