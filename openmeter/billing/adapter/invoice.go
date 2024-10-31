package billingadapter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

var _ billing.InvoiceAdapter = (*adapter)(nil)

func (r *adapter) GetInvoiceById(ctx context.Context, in billing.GetInvoiceByIdInput) (billingentity.Invoice, error) {
	if err := in.Validate(); err != nil {
		return billingentity.Invoice{}, billingentity.ValidationError{
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
			return billingentity.Invoice{}, billingentity.NotFoundError{
				Entity: billingentity.EntityInvoice,
				ID:     in.Invoice.ID,
				Err:    err,
			}
		}

		return billingentity.Invoice{}, err
	}

	return mapInvoiceFromDB(*invoice, in.Expand)
}

func (r *adapter) LockInvoicesForUpdate(ctx context.Context, input billing.LockInvoicesForUpdateInput) error {
	if err := input.Validate(); err != nil {
		return billingentity.ValidationError{
			Err: err,
		}
	}

	ids, err := r.db.BillingInvoice.Query().
		Where(billinginvoice.IDIn(input.InvoiceIDs...)).
		Where(billinginvoice.Namespace(input.Namespace)).
		ForUpdate().
		Select(billinginvoice.FieldID).
		Strings(ctx)
	if err != nil {
		return err
	}

	missingIds := lo.Without(input.InvoiceIDs, ids...)
	if len(missingIds) > 0 {
		return billingentity.NotFoundError{
			Entity: billingentity.EntityInvoice,
			ID:     strings.Join(missingIds, ","),
			Err:    fmt.Errorf("cannot select invoices for update"),
		}
	}

	return nil
}

func (r *adapter) DeleteInvoices(ctx context.Context, input billing.DeleteInvoicesAdapterInput) error {
	if err := input.Validate(); err != nil {
		return billingentity.ValidationError{
			Err: err,
		}
	}

	nAffected, err := r.db.BillingInvoice.Update().
		Where(billinginvoice.IDIn(input.InvoiceIDs...)).
		Where(billinginvoice.Namespace(input.Namespace)).
		SetDeletedAt(clock.Now()).
		Save(ctx)

	if nAffected != len(input.InvoiceIDs) {
		return billingentity.ValidationError{
			Err: errors.New("invoices failed to delete"),
		}
	}

	return err
}

// expandLineItems adds the required edges to the query so that line items can be properly mapped
func (r *adapter) expandLineItems(query *db.BillingInvoiceQuery) *db.BillingInvoiceQuery {
	return query.WithBillingInvoiceLines(func(bilq *db.BillingInvoiceLineQuery) {
		bilq.WithBillingInvoiceManualLines()
	})
}

func (r *adapter) ListInvoices(ctx context.Context, input billing.ListInvoicesInput) (billing.ListInvoicesResponse, error) {
	if err := input.Validate(); err != nil {
		return billing.ListInvoicesResponse{}, billingentity.ValidationError{
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

	if len(input.ExtendedStatuses) > 0 {
		query = query.Where(billinginvoice.StatusIn(input.ExtendedStatuses...))
	}

	if len(input.Statuses) > 0 {
		query = query.Where(func(s *sql.Selector) {
			s.Where(sql.Or(
				lo.Map(input.Statuses, func(status string, _ int) *sql.Predicate {
					return sql.Like(billinginvoice.FieldStatus, status+"%")
				})...,
			))
		})
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

	return mapInvoiceFromDB(*newInvoice, billingentity.InvoiceExpandAll)
}

type lineCountQueryOut struct {
	InvoiceID string `json:"invoice_id"`
	Count     int64  `json:"count"`
}

func (r *adapter) AssociatedLineCounts(ctx context.Context, input billing.AssociatedLineCountsAdapterInput) (billing.AssociatedLineCountsAdapterResponse, error) {
	queryOut := []lineCountQueryOut{}

	err := r.db.BillingInvoiceLine.Query().
		Where(billinginvoiceline.DeletedAtIsNil()).
		Where(billinginvoiceline.Namespace(input.Namespace)).
		Where(billinginvoiceline.InvoiceIDIn(input.InvoiceIDs...)).
		Where(billinginvoiceline.StatusIn(billingentity.InvoiceLineStatusValid)).
		GroupBy(billinginvoiceline.FieldInvoiceID).
		Aggregate(
			db.Count(),
		).
		Scan(ctx, &queryOut)
	if err != nil {
		return billing.AssociatedLineCountsAdapterResponse{}, err
	}

	res := lo.Associate(queryOut, func(q lineCountQueryOut) (billingentity.InvoiceID, int64) {
		return billingentity.InvoiceID{
			Namespace: input.Namespace,
			ID:        q.InvoiceID,
		}, q.Count
	})

	for _, invoiceID := range input.InvoiceIDs {
		id := billingentity.InvoiceID{
			Namespace: input.Namespace,
			ID:        invoiceID,
		}
		if _, found := res[id]; !found {
			res[id] = 0
		}
	}

	return billing.AssociatedLineCountsAdapterResponse{
		Counts: res,
	}, nil
}

func (r *adapter) validateUpdateRequest(req billing.UpdateInvoiceAdapterInput, existing *db.BillingInvoice) error {
	// The user is expected to submit the updatedAt of the source invoice version it based the update on
	// if this doesn't match the current updatedAt, we can't allow the update as it might overwrite some already
	// changed values.
	if !existing.UpdatedAt.Equal(req.UpdatedAt) {
		return billingentity.ConflictError{
			Entity: billingentity.EntityInvoice,
			ID:     req.ID,
			Err:    fmt.Errorf("invoice has been updated since last read"),
		}
	}

	if req.Currency != existing.Currency {
		return billingentity.ValidationError{
			Err: fmt.Errorf("currency cannot be changed"),
		}
	}

	if req.Type != existing.Type {
		return billingentity.ValidationError{
			Err: fmt.Errorf("type cannot be changed"),
		}
	}

	if req.Customer.CustomerID != existing.CustomerID {
		return billingentity.ValidationError{
			Err: fmt.Errorf("customer cannot be changed"),
		}
	}

	return nil
}

// UpdateInvoice updates the specified invoice. It does not return the new invoice, as we would either
// ways need to re-fetch the invoice to get the updated edges.
func (r *adapter) UpdateInvoice(ctx context.Context, in billing.UpdateInvoiceAdapterInput) error {
	existingInvoice, err := r.db.BillingInvoice.Query().
		Where(billinginvoice.ID(in.ID)).
		Where(billinginvoice.Namespace(in.Namespace)).
		Only(ctx)
	if err != nil {
		return err
	}

	if err := r.validateUpdateRequest(in, existingInvoice); err != nil {
		return err
	}

	updateQuery := r.db.BillingInvoice.UpdateOneID(in.ID).
		Where(billinginvoice.Namespace(in.Namespace)).
		SetMetadata(in.Metadata).
		// Currency is immutable
		SetStatus(in.Status).
		// Type is immutable
		SetOrClearNumber(in.Number).
		SetOrClearDescription(in.Description).
		SetOrClearDueAt(in.DueAt).
		SetOrClearDraftUntil(in.DraftUntil).
		SetOrClearIssuedAt(in.IssuedAt)

	if in.Period != nil {
		updateQuery = updateQuery.
			SetPeriodStart(in.Period.Start).
			SetPeriodEnd(in.Period.End)
	} else {
		updateQuery = updateQuery.
			ClearPeriodStart().
			ClearPeriodEnd()
	}

	// Supplier
	updateQuery = updateQuery.
		SetSupplierName(in.Supplier.Name).
		SetOrClearSupplierAddressCountry(in.Supplier.Address.Country).
		SetOrClearSupplierAddressPostalCode(in.Supplier.Address.PostalCode).
		SetOrClearSupplierAddressCity(in.Supplier.Address.City).
		SetOrClearSupplierAddressState(in.Supplier.Address.State).
		SetOrClearSupplierAddressLine1(in.Supplier.Address.Line1).
		SetOrClearSupplierAddressLine2(in.Supplier.Address.Line2).
		SetOrClearSupplierAddressPhoneNumber(in.Supplier.Address.PhoneNumber)

	// Customer
	updateQuery = updateQuery.
		// CustomerID is immutable
		SetCustomerName(in.Customer.Name).
		SetOrClearCustomerAddressCountry(in.Customer.BillingAddress.Country).
		SetOrClearCustomerAddressPostalCode(in.Customer.BillingAddress.PostalCode).
		SetOrClearCustomerAddressCity(in.Customer.BillingAddress.City).
		SetOrClearCustomerAddressState(in.Customer.BillingAddress.State).
		SetOrClearCustomerAddressLine1(in.Customer.BillingAddress.Line1).
		SetOrClearCustomerAddressLine2(in.Customer.BillingAddress.Line2).
		SetOrClearCustomerAddressPhoneNumber(in.Customer.BillingAddress.PhoneNumber).
		SetOrClearCustomerTimezone(in.Customer.Timezone)

	_, err = updateQuery.Save(ctx)
	if err != nil {
		return err
	}

	err = r.persistValidationIssues(ctx,
		billingentity.InvoiceID{
			Namespace: in.Namespace,
			ID:        in.ID,
		}, in.ValidationIssues)
	if err != nil {
		return err
	}

	if in.ExpandedFields.Workflow {
		// Update the workflow config
		_, err := r.updateWorkflowConfig(ctx, in.Namespace, in.Workflow.Config.ID, in.Workflow.Config)
		if err != nil {
			return err
		}
	}

	if in.ExpandedFields.Lines {
		// TODO[later]: line updates (with changed flag)
		r.logger.Warn("line updates are not yet implemented")
	}

	return nil
}

func mapInvoiceFromDB(invoice db.BillingInvoice, expand billingentity.InvoiceExpand) (billingentity.Invoice, error) {
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
		DraftUntil:  invoice.DraftUntil,
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

		ExpandedFields: expand,
	}

	if expand.Workflow {
		workflowConfig, err := mapWorkflowConfigFromDB(invoice.Edges.BillingWorkflowConfig)
		if err != nil {
			return billingentity.Invoice{}, err
		}

		res.Workflow = &billingentity.InvoiceWorkflow{
			Config:                 workflowConfig,
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
