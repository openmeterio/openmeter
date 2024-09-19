package repository

import (
	"context"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/openmeterio/openmeter/openmeter/billing/invoice"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	invoicedb "github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	invoiceitemdb "github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceitem"
	"github.com/openmeterio/openmeter/pkg/currency"
)

type Repository struct {
	db *entdb.Client
}

func (r Repository) GetInvoice(ctx context.Context, id invoice.InvoiceID, inp invoice.RepoGetInvoiceInput) (*invoice.Invoice, error) {
	query := r.db.BillingInvoice.Query().
		Where(invoicedb.ID(id.ID)).
		Where(invoicedb.Namespace(id.Namespace))

	if inp.ExpandItems {
		query = query.WithBillingInvoiceItems()
	}

	invoice, err := query.First(ctx)
	if err != nil {
		return nil, err
	}

	return invoiceFromDBEntity(invoice), nil
}

func (r Repository) CreateInvoiceItems(ctx context.Context, invoiceID *invoice.InvoiceID, items []invoice.InvoiceItem) error {
	for _, item := range items {
		item := r.db.BillingInvoiceItem.Create().
			SetID(item.ID.ID).
			SetNamespace(item.ID.Namespace).
			SetCustomerID(item.Customer.ID).
			SetPeriodStart(item.PeriodStart).
			SetPeriodEnd(item.PeriodEnd).
			SetInvoiceAt(item.InvoiceAt).
			SetQuantity(item.Quantity).
			SetUnitPrice(item.UnitPrice).
			SetCurrency(string(item.Currency)).
			SetTaxCodeOverride(item.TaxCodeOverride).
			SetMetadata(item.Metadata)

		if invoiceID != nil {
			// If invoiceID is set, we are adding the item to an existing invoice, otherwise to the pending list
			item = item.SetInvoiceID(invoiceID.ID)
		}

		if _, err := item.Save(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (r Repository) GetPendingInvoiceItems(ctx context.Context, customerID invoice.CustomerID) ([]invoice.InvoiceItem, error) {
	items, err := r.db.BillingInvoiceItem.Query().
		Where(invoiceitemdb.CustomerID(customerID.ID)).
		Where(invoiceitemdb.Namespace(customerID.Namespace)).
		Where(invoiceitemdb.InvoiceIDIsNil()).
		All(ctx)
	if err != nil {
		return nil, err
	}

	res := make([]invoice.InvoiceItem, 0, len(items))
	for _, item := range items {
		res = append(res, invoiceItemFromDBEntity(item))
	}

	return res, nil
}

func invoiceFromDBEntity(dbInvoice *entdb.BillingInvoice) *invoice.Invoice {
	inv := &invoice.Invoice{
		Invoice: bill.Invoice{
			Type: cbc.Key(dbInvoice.Type),
		},
		ID: invoice.InvoiceID{
			Namespace: dbInvoice.Namespace,
			ID:        dbInvoice.ID,
		},
		Key: dbInvoice.Key,

		Customer: invoice.CustomerID{
			Namespace: dbInvoice.Namespace,
			ID:        dbInvoice.CustomerID,
		},

		BillingProfileID: dbInvoice.BillingProfileID,
		WorkflowConfigID: dbInvoice.WorkflowConfigID,

		ProviderConfig:    dbInvoice.ProviderConfig,
		ProviderReference: dbInvoice.ProviderReference,

		Metadata: dbInvoice.Metadata,

		Currency: currency.Currency(dbInvoice.Currency),

		Status: invoice.InvoiceStatus(dbInvoice.Status),

		PeriodStart: dbInvoice.PeriodStart,
		PeriodEnd:   dbInvoice.PeriodEnd,

		DueDate: dbInvoice.DueDate,

		CreatedAt: dbInvoice.CreatedAt,
		UpdatedAt: dbInvoice.UpdatedAt,
		VoidedAt:  dbInvoice.VoidedAt,

		TotalAmount: dbInvoice.TotalAmount,
	}

	inv.Items = make([]invoice.InvoiceItem, 0, len(dbInvoice.Edges.BillingInvoiceItems))

	for _, dbItem := range dbInvoice.Edges.BillingInvoiceItems {
		inv.Items = append(inv.Items, invoiceItemFromDBEntity(dbItem))
	}

	return inv
}

func invoiceItemFromDBEntity(dbItem *entdb.BillingInvoiceItem) invoice.InvoiceItem {
	return invoice.InvoiceItem{
		ID: invoice.InvoiceItemID{
			Namespace: dbItem.Namespace,
			ID:        dbItem.ID,
		},

		CreatedAt: dbItem.CreatedAt,
		UpdatedAt: dbItem.UpdatedAt,
		DeletedAt: dbItem.DeletedAt,

		Metadata: dbItem.Metadata,
		Invoice: invoice.InvoiceID{
			Namespace: dbItem.Namespace,
			ID:        dbItem.InvoiceID,
		},

		Customer: invoice.CustomerID{
			Namespace: dbItem.Namespace,
			ID:        dbItem.CustomerID,
		},

		PeriodStart: dbItem.PeriodStart,
		PeriodEnd:   dbItem.PeriodEnd,

		InvoiceAt: dbItem.InvoiceAt,

		Quantity:  dbItem.Quantity,
		UnitPrice: dbItem.UnitPrice,
		Currency:  currency.Currency(dbItem.Currency),

		TaxCodeOverride: dbItem.TaxCodeOverride,
	}
}
