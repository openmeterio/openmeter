package persistedstate

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type State struct {
	ByUniqueID        map[string]Item
	ChargesByUniqueID map[string]Item
	Invoices          Invoices
}

func (s State) Validate() error {
	if s.ByUniqueID == nil {
		return errors.New("by unique id is required")
	}

	if s.ChargesByUniqueID == nil {
		return errors.New("charges by unique id is required")
	}

	if s.Invoices == nil {
		return errors.New("invoices are required")
	}

	return nil
}

type Invoices map[string]billing.Invoice

func (i Invoices) IsGatheringInvoice(invoiceID string) (bool, error) {
	invoice, ok := i[invoiceID]
	if !ok {
		return false, fmt.Errorf("invoice not found in state: %s", invoiceID)
	}

	return invoice.Type() == billing.InvoiceTypeGathering, nil
}
