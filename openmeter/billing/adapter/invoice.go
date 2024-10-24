package billingadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

var _ billing.InvoiceAdapter = (*adapter)(nil)

func (r *adapter) GetInvoiceById(ctx context.Context, in billing.GetInvoiceByIdInput) (billingentity.Invoice, error) {
	if err := in.Validate(); err != nil {
		return billingentity.Invoice{}, billing.ValidationError{
			Err: err,
		}
	}

	query := r.db.BillingInvoice.Query().
		Where(billinginvoice.ID(in.Invoice.ID)).
		Where(billinginvoice.Namespace(in.Invoice.Namespace))

	if in.Expand.Workflow {
		query = query.WithBillingWorkflowConfig()
	}

	if in.Expand.Lines {
		query = r.expandLineItems(query)
	}

	invoice, err := query.Only(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return billingentity.Invoice{}, billing.NotFoundError{
				Entity: billing.EntityInvoice,
				ID:     in.Invoice.ID,
				Err:    err,
			}
		}

		return billingentity.Invoice{}, err
	}

	return mapInvoiceFromDB(*invoice, in.Expand)
}

// expandLineItems adds the required edges to the query so that line items can be properly mapped
func (r *adapter) expandLineItems(query *db.BillingInvoiceQuery) *db.BillingInvoiceQuery {
	return query.WithBillingInvoiceLines(func(bilq *db.BillingInvoiceLineQuery) {
		bilq.WithBillingInvoiceManualLines()
	})
}

func (r *adapter) ListInvoices(ctx context.Context, input billing.ListInvoicesInput) (billing.ListInvoicesResponse, error) {
	if err := input.Validate(); err != nil {
		return billing.ListInvoicesResponse{}, billing.ValidationError{
			Err: err,
		}
	}

	query := r.db.BillingInvoice.Query().
		Where(billinginvoice.Namespace(input.Namespace))

	if input.Expand.Workflow {
		query = query.WithBillingWorkflowConfig()
	}

	// TODO[OM-944]: We should allow by customer key and name too
	if len(input.Customers) > 0 {
		query = query.Where(billinginvoice.CustomerIDIn(input.Customers...))
	}

	if input.IssuedAfter != nil {
		query = query.Where(billinginvoice.IssuedAtGTE(*input.IssuedAfter))
	}

	if input.IssuedBefore != nil {
		query = query.Where(billinginvoice.IssuedAtLTE(*input.IssuedBefore))
	}

	if len(input.Statuses) > 0 {
		query = query.Where(billinginvoice.StatusIn(input.Statuses...))
	}

	if len(input.Currencies) > 0 {
		query = query.Where(billinginvoice.CurrencyIn(input.Currencies...))
	}

	order := entutils.GetOrdering(sortx.OrderDefault)
	if !input.Order.IsDefaultValue() {
		order = entutils.GetOrdering(input.Order)
	}

	if input.Expand.Lines {
		query = r.expandLineItems(query)
	}

	switch input.OrderBy {
	case api.BillingInvoiceOrderByCustomerName:
		query = query.Order(billinginvoice.ByCustomerName(order...))
	case api.BillingInvoiceOrderByIssuedAt:
		query = query.Order(billinginvoice.ByIssuedAt(order...))
	case api.BillingInvoiceOrderByStatus:
		query = query.Order(billinginvoice.ByStatus(order...))
	case api.BillingInvoiceOrderByUpdatedAt:
		query = query.Order(billinginvoice.ByUpdatedAt(order...))
	case api.BillingInvoiceOrderByCreatedAt:
		fallthrough
	default:
		query = query.Order(billinginvoice.ByCreatedAt(order...))
	}

	response := pagination.PagedResponse[billingentity.Invoice]{
		Page: input.Page,
	}

	paged, err := query.Paginate(ctx, input.Page)
	if err != nil {
		return response, err
	}

	result := make([]billingentity.Invoice, 0, len(paged.Items))
	for _, invoice := range paged.Items {
		mapped, err := mapInvoiceFromDB(*invoice, input.Expand)
		if err != nil {
			return response, err
		}

		result = append(result, mapped)
	}

	response.TotalCount = paged.TotalCount
	response.Items = result

	return response, nil
}

func (r *adapter) CreateInvoice(ctx context.Context, input billing.CreateInvoiceAdapterInput) (billing.CreateInvoiceAdapterRespone, error) {
	if err := input.Validate(); err != nil {
		return billing.CreateInvoiceAdapterRespone{}, err
	}

	customer := input.Customer
	supplier := input.Profile.Supplier

	// Clone the workflow config
	clonedWorkflowConfig, err := r.createWorkflowConfig(ctx, input.Namespace, input.Profile.WorkflowConfig)
	if err != nil {
		return billing.CreateInvoiceAdapterRespone{}, fmt.Errorf("clone workflow config: %w", err)
	}

	workflowConfig := mapWorkflowConfigToDB(input.Profile.WorkflowConfig)

	// Force cloning of the workflow
	workflowConfig.ID = ""
	workflowConfig.CreatedAt = time.Time{}
	workflowConfig.UpdatedAt = time.Time{}
	workflowConfig.DeletedAt = nil

	newInvoice, err := r.db.BillingInvoice.Create().
		SetNamespace(input.Namespace).
		SetMetadata(input.Metadata).
		SetCurrency(input.Currency).
		SetStatus(input.Status).
		SetSourceBillingProfileID(input.Profile.ID).
		SetCustomerID(input.Customer.ID).
		SetType(input.Type).
		SetNillableDescription(input.Description).
		SetNillableDueAt(input.DueAt).
		SetNillableCustomerTimezone(customer.Timezone).
		SetNillableIssuedAt(lo.EmptyableToPtr(input.IssuedAt)).
		// Workflow (cloned)
		SetBillingWorkflowConfigID(clonedWorkflowConfig.ID).
		// TODO[later]: By cloning the AppIDs here we could support changing the apps in the billing profile if needed
		SetTaxAppID(input.Profile.Apps.Tax.GetID().ID).
		SetInvoicingAppID(input.Profile.Apps.Invoicing.GetID().ID).
		SetPaymentAppID(input.Profile.Apps.Payment.GetID().ID).
		// Customer contacts
		SetNillableCustomerAddressCountry(customer.BillingAddress.Country).
		SetNillableCustomerAddressPostalCode(customer.BillingAddress.PostalCode).
		SetNillableCustomerAddressState(customer.BillingAddress.State).
		SetNillableCustomerAddressCity(customer.BillingAddress.City).
		SetNillableCustomerAddressLine1(customer.BillingAddress.Line1).
		SetNillableCustomerAddressLine2(customer.BillingAddress.Line2).
		SetNillableCustomerAddressPhoneNumber(customer.BillingAddress.PhoneNumber).
		SetCustomerName(customer.Name).
		SetNillableCustomerTimezone(customer.Timezone).
		// Supplier contacts
		SetNillableSupplierAddressCountry(supplier.Address.Country).
		SetNillableSupplierAddressPostalCode(supplier.Address.PostalCode).
		SetNillableSupplierAddressState(supplier.Address.State).
		SetNillableSupplierAddressCity(supplier.Address.City).
		SetNillableSupplierAddressLine1(supplier.Address.Line1).
		SetNillableSupplierAddressLine2(supplier.Address.Line2).
		SetNillableSupplierAddressPhoneNumber(supplier.Address.PhoneNumber).
		SetSupplierName(supplier.Name).
		SetNillableSupplierTaxCode(supplier.TaxCode).
		Save(ctx)
	if err != nil {
		return billing.CreateInvoiceAdapterRespone{}, err
	}

	// Let's add required edges for mapping
	newInvoice.Edges.BillingWorkflowConfig = clonedWorkflowConfig

	return mapInvoiceFromDB(*newInvoice, billing.InvoiceExpandAll)
}

func mapInvoiceFromDB(invoice db.BillingInvoice, expand billing.InvoiceExpand) (billingentity.Invoice, error) {
	res := billingentity.Invoice{
		ID:          invoice.ID,
		Namespace:   invoice.Namespace,
		Metadata:    invoice.Metadata,
		Currency:    invoice.Currency,
		Status:      invoice.Status,
		Type:        invoice.Type,
		Number:      invoice.Number,
		Description: invoice.Description,
		DueAt:       invoice.DueAt,
		Supplier: billingentity.SupplierContact{
			Name: invoice.SupplierName,
			Address: models.Address{
				Country:     invoice.SupplierAddressCountry,
				PostalCode:  invoice.SupplierAddressPostalCode,
				City:        invoice.SupplierAddressCity,
				State:       invoice.SupplierAddressState,
				Line1:       invoice.SupplierAddressLine1,
				Line2:       invoice.SupplierAddressLine2,
				PhoneNumber: invoice.SupplierAddressPhoneNumber,
			},
			TaxCode: invoice.SupplierTaxCode,
		},

		Customer: billingentity.InvoiceCustomer{
			CustomerID: invoice.CustomerID,
			Name:       invoice.CustomerName,
			BillingAddress: &models.Address{
				Country:     invoice.CustomerAddressCountry,
				PostalCode:  invoice.CustomerAddressPostalCode,
				City:        invoice.CustomerAddressCity,
				State:       invoice.CustomerAddressState,
				Line1:       invoice.CustomerAddressLine1,
				Line2:       invoice.CustomerAddressLine2,
				PhoneNumber: invoice.CustomerAddressPhoneNumber,
			},
			Timezone: invoice.CustomerTimezone,
		},
		Period:    mapPeriodFromDB(invoice.PeriodStart, invoice.PeriodEnd),
		IssuedAt:  invoice.IssuedAt,
		CreatedAt: invoice.CreatedAt,
		UpdatedAt: invoice.UpdatedAt,
		DeletedAt: invoice.DeletedAt,
	}

	if expand.Workflow {
		workflowConfig, err := mapWorkflowConfigFromDB(invoice.Edges.BillingWorkflowConfig)
		if err != nil {
			return billingentity.Invoice{}, err
		}

		res.Workflow = &billingentity.InvoiceWorkflow{
			WorkflowConfig:         workflowConfig,
			SourceBillingProfileID: invoice.SourceBillingProfileID,

			AppReferences: billingentity.ProfileAppReferences{
				Tax: billingentity.AppReference{
					ID: invoice.TaxAppID,
				},
				Invoicing: billingentity.AppReference{
					ID: invoice.InvoicingAppID,
				},
				Payment: billingentity.AppReference{
					ID: invoice.PaymentAppID,
				},
			},
		}
	}

	if len(invoice.Edges.BillingInvoiceLines) > 0 {
		res.Lines = make([]billingentity.Line, 0, len(invoice.Edges.BillingInvoiceLines))
		for _, line := range invoice.Edges.BillingInvoiceLines {
			res.Lines = append(res.Lines, mapInvoiceLineFromDB(line))
		}
	}

	return res, nil
}

func mapPeriodFromDB(start, end *time.Time) *billingentity.Period {
	if start == nil || end == nil {
		return nil
	}
	return &billingentity.Period{
		Start: *start,
		End:   *end,
	}
}
