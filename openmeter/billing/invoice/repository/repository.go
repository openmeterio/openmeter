package repository

import (
	"context"
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/invoice"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	invoicedb "github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	invoiceitemdb "github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceitem"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
)

type Config struct {
	Client *entdb.Client
}

func (c Config) Validate() error {
	if c.Client == nil {
		return errors.New("client is required")
	}

	return nil
}

func New(c Config) (invoice.Repository, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	return repository{db: c.Client}, nil
}

type repository struct {
	db *entdb.Client
}

func (r repository) GetInvoice(ctx context.Context, id invoice.InvoiceID, inp invoice.RepoGetInvoiceInput) (*invoice.Invoice, error) {
	query := r.db.BillingInvoice.Query().
		Where(invoicedb.ID(id.ID)).
		Where(invoicedb.Namespace(id.Namespace)).
		WithCustomer() // Customer is always expanded as it's required for filling customer data for non-snapshoted invoices

	if inp.ExpandItems {
		query = query.WithBillingInvoiceItems()
	}

	if inp.ExpandWorkflowConfig {
		query = query.WithBillingWorkflowConfig()
	}

	invoice, err := query.First(ctx)
	if err != nil {
		return nil, err
	}

	return invoiceFromDBEntity(invoice), nil
}

func (r repository) CreateInvoiceItems(ctx context.Context, invoiceID *invoice.InvoiceID, items []invoice.InvoiceItem) ([]invoice.InvoiceItem, error) {
	result := make([]invoice.InvoiceItem, 0, len(items))

	for _, item := range items {
		item := r.db.BillingInvoiceItem.Create().
			SetNamespace(item.ID.Namespace).
			SetCustomerID(item.CustomerID).
			SetPeriodStart(item.PeriodStart).
			SetPeriodEnd(item.PeriodEnd).
			SetInvoiceAt(item.InvoiceAt).
			SetQuantity(item.Quantity).
			SetUnitPrice(item.UnitPrice).
			SetCurrency(item.Currency).
			SetTaxCodeOverride(item.TaxCodeOverride).
			SetMetadata(item.Metadata)

		if invoiceID != nil {
			// If invoiceID is set, we are adding the item to an existing invoice, otherwise to the pending list
			item = item.SetInvoiceID(invoiceID.ID)
		}

		savedItem, err := item.Save(ctx)
		if err != nil {
			return nil, err
		}

		result = append(result, invoiceItemFromDBEntity(savedItem))
	}

	return result, nil
}

func (r repository) GetPendingInvoiceItems(ctx context.Context, customerID customer.CustomerID) ([]invoice.InvoiceItem, error) {
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
		ID: invoice.InvoiceID{
			Namespace: dbInvoice.Namespace,
			ID:        dbInvoice.ID,
		},
		Key: dbInvoice.Key,

		BillingProfileID: dbInvoice.BillingProfileID,

		ProviderConfig:    dbInvoice.ProviderConfig,
		ProviderReference: dbInvoice.ProviderReference,

		Metadata: dbInvoice.Metadata,

		Currency: currencyx.Code(dbInvoice.Currency),
		Timezone: dbInvoice.Timezone,

		Status: invoice.InvoiceStatus(dbInvoice.Status),

		PeriodStart: dbInvoice.PeriodStart,
		PeriodEnd:   dbInvoice.PeriodEnd,

		DueDate: dbInvoice.DueDate,

		CreatedAt: dbInvoice.CreatedAt,
		UpdatedAt: dbInvoice.UpdatedAt,
		VoidedAt:  dbInvoice.VoidedAt,
		IssuedAt:  dbInvoice.IssuedAt,
	}

	if dbInvoice.Edges.BillingWorkflowConfig != nil {
		inv.WorkflowConfig = workflowConfigFromDBEntity(dbInvoice.Edges.BillingWorkflowConfig)
	}

	if dbInvoice.CustomerSnapshotTaken {
		inv.Customer = invoice.InvoiceCustomer{
			CustomerID: dbInvoice.CustomerID,
			Name:       lo.FromPtrOr(dbInvoice.CustomerName, ""),
			BillingAddress: &models.Address{
				Country:     dbInvoice.BillingAddressCountry,
				PostalCode:  dbInvoice.BillingAddressPostalCode,
				State:       dbInvoice.BillingAddressState,
				City:        dbInvoice.BillingAddressCity,
				Line1:       dbInvoice.BillingAddressLine1,
				Line2:       dbInvoice.BillingAddressLine2,
				PhoneNumber: dbInvoice.BillingAddressPhoneNumber,
			},
		}
	} else {
		inv.Customer = invoice.InvoiceCustomer{
			CustomerID: dbInvoice.CustomerID,
			Name:       lo.FromPtrOr(dbInvoice.Edges.Customer.Name, ""),
			BillingAddress: &models.Address{
				Country:     dbInvoice.Edges.Customer.BillingAddressCountry,
				PostalCode:  dbInvoice.Edges.Customer.BillingAddressPostalCode,
				State:       dbInvoice.Edges.Customer.BillingAddressState,
				City:        dbInvoice.Edges.Customer.BillingAddressCity,
				Line1:       dbInvoice.Edges.Customer.BillingAddressLine1,
				Line2:       dbInvoice.Edges.Customer.BillingAddressLine2,
				PhoneNumber: dbInvoice.Edges.Customer.BillingAddressPhoneNumber,
			},
		}
	}

	inv.Items = make([]invoice.InvoiceItem, 0, len(dbInvoice.Edges.BillingInvoiceItems))

	for _, dbItem := range dbInvoice.Edges.BillingInvoiceItems {
		inv.Items = append(inv.Items, invoiceItemFromDBEntity(dbItem))
	}

	return inv
}

func invoiceItemFromDBEntity(dbItem *entdb.BillingInvoiceItem) invoice.InvoiceItem {
	invoiceItem := invoice.InvoiceItem{
		ID: invoice.InvoiceItemID{
			Namespace: dbItem.Namespace,
			ID:        dbItem.ID,
		},

		CreatedAt: dbItem.CreatedAt,
		UpdatedAt: dbItem.UpdatedAt,
		DeletedAt: dbItem.DeletedAt,

		Metadata:   dbItem.Metadata,
		CustomerID: dbItem.CustomerID,

		PeriodStart: dbItem.PeriodStart,
		PeriodEnd:   dbItem.PeriodEnd,

		InvoiceAt: dbItem.InvoiceAt,

		Quantity:  dbItem.Quantity,
		UnitPrice: dbItem.UnitPrice,
		Currency:  dbItem.Currency,

		TaxCodeOverride: dbItem.TaxCodeOverride,
	}

	if dbItem.InvoiceID != "" {
		invoiceItem.Invoice = &invoice.InvoiceID{
			Namespace: dbItem.Namespace,
			ID:        dbItem.InvoiceID,
		}
	}

	return invoiceItem
}

func workflowConfigFromDBEntity(dbConfig *entdb.BillingWorkflowConfig) *invoice.WorkflowConfig {
	return &invoice.WorkflowConfig{
		ID: models.NamespacedID{
			Namespace: dbConfig.Namespace,
			ID:        dbConfig.ID,
		},
		Collection: invoice.WorkflowCollectionConfig{
			AlignmentKind:    billing.AlignmentKind(dbConfig.CollectionAlignment),
			CollectionPeriod: time.Second * time.Duration(dbConfig.CollectionPeriodSeconds),
		},
		Invoicing: invoice.WorkflowInvoicingConfig{
			AutoAdvance:      dbConfig.InvoiceAutoAdvance,
			DraftPeriod:      time.Second * time.Duration(dbConfig.InvoiceDraftPeriodSeconds),
			DueAfterDays:     int(dbConfig.InvoiceDueAfterDays),
			CollectionMethod: billing.CollectionMethod(dbConfig.InvoiceCollectionMethod),
			Items: invoice.WorkflowItemsConfig{
				Resolution: dbConfig.InvoiceItemResolution,
				PerSubject: dbConfig.InvoiceItemPerSubject,
			},
		},
	}
}
