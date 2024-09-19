package invoice

import (
	"context"
)

type RepoGetInvoiceInput struct {
	ExpandItems bool
}

type Repository interface {
	GetInvoice(ctx context.Context, id InvoiceID, params RepoGetInvoiceInput) (*Invoice, error)

	GetPendingInvoiceItems(ctx context.Context, customerID CustomerID) ([]InvoiceItem, error)
	CreateInvoiceItems(ctx context.Context, invoice *InvoiceID, items []InvoiceItem) error
}
