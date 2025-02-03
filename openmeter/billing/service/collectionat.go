package billingservice

import (
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

// UpdateInvoiceCollectionAt updates the collectionAt attribute of the invoice with gathering type
// using the customers collection configuration. It returns true if the attribute has been updated.
// The collectionAt is calculated by adding the collection interval (from CollectionConfig) to the earliest invoicedAt
// timestamp of the invoice lines on the gathering invoice.
func UpdateInvoiceCollectionAt(invoice *billing.Invoice, collection billing.CollectionConfig) bool {
	if invoice == nil || invoice.Status != billing.InvoiceStatusGathering {
		return false
	}

	invoiceAt := GetEarliestValidInvoiceAt(invoice.Lines)

	if invoiceAt.IsZero() {
		return false
	}

	interval, ok := collection.Interval.Duration()
	if !ok {
		return false
	}

	collectionAt := invoiceAt.Add(interval)

	if lo.FromPtr(invoice.CollectionAt).Equal(collectionAt) {
		return false
	}

	invoice.CollectionAt = &collectionAt

	return true
}

func GetEarliestValidInvoiceAt(lines billing.LineChildren) time.Time {
	var invoiceAt time.Time

	// Find the invoice line with the earliest invoiceAt attribute
	lines.ForEach(func(v []*billing.Line) {
		for _, line := range v {
			if line == nil || line.Status != billing.InvoiceLineStatusValid {
				continue
			}

			if line.DeletedAt != nil {
				continue
			}

			if invoiceAt.IsZero() {
				invoiceAt = line.InvoiceAt
				continue
			}

			if line.InvoiceAt.Before(invoiceAt) {
				invoiceAt = line.InvoiceAt
			}
		}
	})

	return invoiceAt
}

func GetLatestValidInvoiceAtAsOf(lines billing.LineChildren, asOf time.Time) time.Time {
	var invoiceAt time.Time

	// Find the invoice line with the latest invoiceAt attribute before asOf time
	lines.ForEach(func(v []*billing.Line) {
		for _, line := range v {
			if line == nil || line.Status != billing.InvoiceLineStatusValid {
				continue
			}

			if line.DeletedAt != nil {
				continue
			}

			if line.InvoiceAt.After(invoiceAt) && line.InvoiceAt.Before(asOf) {
				invoiceAt = line.InvoiceAt
			}

			if line.InvoiceAt.Equal(asOf) {
				invoiceAt = line.InvoiceAt

				break
			}
		}
	})

	return invoiceAt
}

func GetInvoiceWithEarliestCollectionAt(invoices []billing.Invoice) billing.Invoice {
	var idx int

	collectAt := time.Now()

	for i, invoice := range invoices {
		if invoice.CollectionAt.Before(collectAt) {
			idx = i
		}
	}

	return invoices[idx]
}
