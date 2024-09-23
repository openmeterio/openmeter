package invoice

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
)

type InvoiceType cbc.Key

const (
	InvoiceTypeStandard   InvoiceType = InvoiceType(bill.InvoiceTypeStandard)
	InvoiceTypeCreditNote InvoiceType = InvoiceType(bill.InvoiceTypeCreditNote)
)

func (t InvoiceType) Values() []string {
	return []string{
		string(InvoiceTypeStandard),
		string(InvoiceTypeCreditNote),
	}
}

func (t InvoiceType) CBCKey() cbc.Key {
	return cbc.Key(t)
}
