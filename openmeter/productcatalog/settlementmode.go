package productcatalog

import (
	"fmt"
	"slices"
)

type SettlementMode string

const (
	InvoiceOnlySettlementMode       SettlementMode = "invoice_only"
	CreditThenInvoiceSettlementMode SettlementMode = "credit_then_invoice"
	CreditOnlySettlementMode        SettlementMode = "credit_only"
)

func (s SettlementMode) Values() []string {
	return []string{
		string(InvoiceOnlySettlementMode),
		string(CreditThenInvoiceSettlementMode),
		string(CreditOnlySettlementMode),
	}
}

func (s SettlementMode) Validate() error {
	if !slices.Contains(s.Values(), string(s)) {
		return fmt.Errorf("invalid settlement mode: %s", s)
	}

	return nil
}
