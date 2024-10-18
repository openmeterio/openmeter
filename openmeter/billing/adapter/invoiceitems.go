package billingadapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceitem"
)

func (r adapter) CreateInvoiceItems(ctx context.Context, input billing.CreateInvoiceItemsInput) ([]billingentity.InvoiceItem, error) {
	result := make([]billingentity.InvoiceItem, 0, len(input.Items))

	for _, item := range input.Items {
		item := r.db.BillingInvoiceItem.Create().
			SetNamespace(input.Namespace).
			SetCustomerID(item.CustomerID).
			SetPeriodStart(item.PeriodStart).
			SetPeriodEnd(item.PeriodEnd).
			SetInvoiceAt(item.InvoiceAt).
			SetType(item.Type).
			SetName(item.Name).
			SetNillableQuantity(item.Quantity).
			SetUnitPrice(item.UnitPrice).
			SetCurrency(item.Currency).
			SetTaxCodeOverride(item.TaxCodeOverride).
			SetMetadata(item.Metadata)

		if input.InvoiceID != nil {
			// If invoiceID is set, we are adding the item to an existing invoice, otherwise to the pending list
			item = item.SetInvoiceID(*input.InvoiceID)
		}

		savedItem, err := item.Save(ctx)
		if err != nil {
			return nil, err
		}

		result = append(result, mapInvoiceItemFromDB(savedItem))
	}

	return result, nil
}

func (r adapter) GetPendingInvoiceItems(ctx context.Context, customerID customerentity.CustomerID) ([]billingentity.InvoiceItem, error) {
	items, err := r.db.BillingInvoiceItem.Query().
		Where(billinginvoiceitem.CustomerID(customerID.ID)).
		Where(billinginvoiceitem.Namespace(customerID.Namespace)).
		Where(billinginvoiceitem.InvoiceIDIsNil()).
		All(ctx)
	if err != nil {
		return nil, err
	}

	res := make([]billingentity.InvoiceItem, 0, len(items))
	for _, item := range items {
		res = append(res, mapInvoiceItemFromDB(item))
	}

	return res, nil
}

func mapInvoiceItemFromDB(dbItem *db.BillingInvoiceItem) billingentity.InvoiceItem {
	invoiceItem := billingentity.InvoiceItem{
		Namespace: dbItem.Namespace,
		ID:        dbItem.ID,

		CreatedAt: dbItem.CreatedAt,
		UpdatedAt: dbItem.UpdatedAt,
		DeletedAt: dbItem.DeletedAt,

		Metadata:   dbItem.Metadata,
		CustomerID: dbItem.CustomerID,
		InvoiceID:  dbItem.InvoiceID,

		PeriodStart: dbItem.PeriodStart,
		PeriodEnd:   dbItem.PeriodEnd,

		InvoiceAt: dbItem.InvoiceAt,

		Name: dbItem.Name,

		Type:      dbItem.Type,
		Quantity:  dbItem.Quantity,
		UnitPrice: dbItem.UnitPrice,
		Currency:  dbItem.Currency,

		TaxCodeOverride: dbItem.TaxCodeOverride,
	}

	return invoiceItem
}
