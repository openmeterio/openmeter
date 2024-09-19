package invoice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/customer"
)

type RepoGetInvoiceInput struct {
	ExpandItems          bool
	ExpandWorkflowConfig bool
}

type Repository interface {
	GetInvoice(ctx context.Context, id InvoiceID, params RepoGetInvoiceInput) (*Invoice, error)

	GetPendingInvoiceItems(ctx context.Context, customerID customer.CustomerID) ([]InvoiceItem, error)
	CreateInvoiceItems(ctx context.Context, invoice *InvoiceID, items []InvoiceItem) ([]InvoiceItem, error)
}
