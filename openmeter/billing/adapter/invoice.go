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
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicevalidationissue"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

var _ billing.InvoiceAdapter = (*adapter)(nil)

func (a *adapter) GetInvoiceById(ctx context.Context, in billing.GetInvoiceByIdInput) (billingentity.Invoice, error) {
	if err := in.Validate(); err != nil {
		return billingentity.Invoice{}, billingentity.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billingentity.Invoice, error) {
		query := tx.db.BillingInvoice.Query().
			Where(billinginvoice.ID(in.Invoice.ID)).
			Where(billinginvoice.Namespace(in.Invoice.Namespace)).
			WithBillingInvoiceValidationIssues(func(q *db.BillingInvoiceValidationIssueQuery) {
				q.Where(billinginvoicevalidationissue.DeletedAtIsNil())
			}).
			WithBillingWorkflowConfig()

		if in.Expand.Lines {
			query = tx.expandInvoiceLineItems(query, in.Expand.DeletedLines)
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

		return tx.mapInvoiceFromDB(ctx, invoice, in.Expand)
	})
}

func (a *adapter) expandInvoiceLineItems(query *db.BillingInvoiceQuery, includeDeleted bool) *db.BillingInvoiceQuery {
	return query.WithBillingInvoiceLines(func(q *db.BillingInvoiceLineQuery) {
		if !includeDeleted {
			q = q.Where(billinginvoiceline.DeletedAtIsNil())
		}

		q = q.Where(
			// Detailed lines are sub-lines of a line and should not be included in the top-level invoice
			billinginvoiceline.StatusIn(billingentity.InvoiceLineStatusValid),
		)

		a.expandLineItems(q)
	})
}

func (a *adapter) LockInvoicesForUpdate(ctx context.Context, input billing.LockInvoicesForUpdateInput) error {
	if err := input.Validate(); err != nil {
		return billingentity.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		ids, err := tx.db.BillingInvoice.Query().
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
	})
}

func (a *adapter) DeleteInvoices(ctx context.Context, input billing.DeleteInvoicesAdapterInput) error {
	if err := input.Validate(); err != nil {
		return billingentity.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		nAffected, err := tx.db.BillingInvoice.Update().
			Where(billinginvoice.IDIn(input.InvoiceIDs...)).
			Where(billinginvoice.Namespace(input.Namespace)).
			SetDeletedAt(clock.Now()).
			Save(ctx)
		if err != nil {
			return err
		}

		if nAffected != len(input.InvoiceIDs) {
			return billingentity.ValidationError{
				Err: errors.New("invoices failed to delete"),
			}
		}

		return nil
	})
}

func (a *adapter) ListInvoices(ctx context.Context, input billing.ListInvoicesInput) (billing.ListInvoicesResponse, error) {
	if err := input.Validate(); err != nil {
		return billing.ListInvoicesResponse{}, billingentity.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.ListInvoicesResponse, error) {
		// Note: we are not filtering for deleted invoices here (as in deleted_at is not nil), as we have the deleted
		// status that we can use to filter for.

		query := tx.db.BillingInvoice.Query().
			Where(billinginvoice.Namespace(input.Namespace)).
			WithBillingInvoiceValidationIssues(func(q *db.BillingInvoiceValidationIssueQuery) {
				q.Where(billinginvoicevalidationissue.DeletedAtIsNil())
			}).
			WithBillingWorkflowConfig()

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
			query = tx.expandInvoiceLineItems(query, input.Expand.DeletedLines)
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
			mapped, err := tx.mapInvoiceFromDB(ctx, invoice, input.Expand)
			if err != nil {
				return response, err
			}

			result = append(result, mapped)
		}

		response.TotalCount = paged.TotalCount
		response.Items = result

		return response, nil
	})
}

func (a *adapter) CreateInvoice(ctx context.Context, input billing.CreateInvoiceAdapterInput) (billing.CreateInvoiceAdapterRespone, error) {
	if err := input.Validate(); err != nil {
		return billing.CreateInvoiceAdapterRespone{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.CreateInvoiceAdapterRespone, error) {
		customer := input.Customer
		supplier := input.Profile.Supplier

		// Clone the workflow config
		clonedWorkflowConfig, err := tx.createWorkflowConfig(ctx, input.Namespace, input.Profile.WorkflowConfig)
		if err != nil {
			return billing.CreateInvoiceAdapterRespone{}, fmt.Errorf("clone workflow config: %w", err)
		}

		workflowConfig := mapWorkflowConfigToDB(input.Profile.WorkflowConfig)

		// Force cloning of the workflow
		workflowConfig.ID = ""
		workflowConfig.CreatedAt = time.Time{}
		workflowConfig.UpdatedAt = time.Time{}
		workflowConfig.DeletedAt = nil

		newInvoice, err := tx.db.BillingInvoice.Create().
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
			SetCustomerUsageAttribution(&billingentity.VersionedCustomerUsageAttribution{
				Type:                     billingentity.CustomerUsageAttributionTypeVersion,
				CustomerUsageAttribution: input.Customer.UsageAttribution,
			}).
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
			// Totals
			SetAmount(input.Totals.Amount).
			SetChargesTotal(input.Totals.ChargesTotal).
			SetDiscountsTotal(input.Totals.DiscountsTotal).
			SetTaxesTotal(input.Totals.TaxesTotal).
			SetTaxesExclusiveTotal(input.Totals.TaxesExclusiveTotal).
			SetTaxesInclusiveTotal(input.Totals.TaxesInclusiveTotal).
			SetTotal(input.Totals.Total).
			Save(ctx)
		if err != nil {
			return billing.CreateInvoiceAdapterRespone{}, err
		}

		// Let's add required edges for mapping
		newInvoice.Edges.BillingWorkflowConfig = clonedWorkflowConfig

		return tx.mapInvoiceFromDB(ctx, newInvoice, billingentity.InvoiceExpandAll)
	})
}

type lineCountQueryOut struct {
	InvoiceID string `json:"invoice_id"`
	Count     int64  `json:"count"`
}

func (a *adapter) AssociatedLineCounts(ctx context.Context, input billing.AssociatedLineCountsAdapterInput) (billing.AssociatedLineCountsAdapterResponse, error) {
	queryOut := []lineCountQueryOut{}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.AssociatedLineCountsAdapterResponse, error) {
		err := tx.db.BillingInvoiceLine.Query().
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
	})
}

func (a *adapter) validateUpdateRequest(req billing.UpdateInvoiceAdapterInput, existing *db.BillingInvoice) error {
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

// UpdateInvoice updates the specified invoice.
func (a *adapter) UpdateInvoice(ctx context.Context, in billing.UpdateInvoiceAdapterInput) (billingentity.Invoice, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billingentity.Invoice, error) {
		existingInvoice, err := tx.db.BillingInvoice.Query().
			Where(billinginvoice.ID(in.ID)).
			Where(billinginvoice.Namespace(in.Namespace)).
			Only(ctx)
		if err != nil {
			return in, err
		}

		if err := tx.validateUpdateRequest(in, existingInvoice); err != nil {
			return in, err
		}

		updateQuery := tx.db.BillingInvoice.UpdateOneID(in.ID).
			Where(billinginvoice.Namespace(in.Namespace)).
			SetMetadata(in.Metadata).
			// Currency is immutable
			SetStatus(in.Status).
			// Type is immutable
			SetOrClearNumber(in.Number).
			SetOrClearDescription(in.Description).
			SetOrClearDueAt(in.DueAt).
			SetOrClearDraftUntil(in.DraftUntil).
			SetOrClearIssuedAt(in.IssuedAt).
			SetOrClearDeletedAt(in.DeletedAt).
			// Totals
			SetAmount(in.Totals.Amount).
			SetChargesTotal(in.Totals.ChargesTotal).
			SetDiscountsTotal(in.Totals.DiscountsTotal).
			SetTaxesTotal(in.Totals.TaxesTotal).
			SetTaxesExclusiveTotal(in.Totals.TaxesExclusiveTotal).
			SetTaxesInclusiveTotal(in.Totals.TaxesInclusiveTotal).
			SetTotal(in.Totals.Total)

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
			return in, err
		}

		err = tx.persistValidationIssues(ctx,
			billingentity.InvoiceID{
				Namespace: in.Namespace,
				ID:        in.ID,
			}, in.ValidationIssues)
		if err != nil {
			return in, err
		}

		// Update the workflow config
		_, err = tx.updateWorkflowConfig(ctx, in.Namespace, in.Workflow.Config.ID, in.Workflow.Config)
		if err != nil {
			return in, err
		}

		updatedLines := billingentity.LineChildren{}
		if in.Lines.IsPresent() {
			// Note: this only supports adding new lines or setting the DeletedAt field
			// we don't support moving lines between invoices here, as the cross invoice
			// coordination is not something the adapter should deal with. The service
			// is needed to lock and recalculate both invoices or do the necessary splits.

			lines, err := tx.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
				Namespace: in.Namespace,
				Lines:     in.Lines.OrEmpty(),
			})
			if err != nil {
				return in, err
			}

			updatedLines = billingentity.NewLineChildren(lines)
		}

		// Let's return the updated invoice

		// If we had just updated the lines, let's reuse that result, as it's quite an expensive operation
		// to look up the lines again.
		if in.ExpandedFields.Lines && updatedLines.IsPresent() {
			updatedInvoice, err := tx.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
				Invoice: billingentity.InvoiceID{
					Namespace: in.Namespace,
					ID:        in.ID,
				},
				Expand: in.ExpandedFields.SetLines(false),
			})
			if err != nil {
				return in, err
			}

			updatedInvoice.Lines = updatedLines
			return updatedInvoice, nil
		}

		return tx.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
			Invoice: billingentity.InvoiceID{
				Namespace: in.Namespace,
				ID:        in.ID,
			},
			Expand: in.ExpandedFields,
		})
	})
}

func (a *adapter) GetInvoiceOwnership(ctx context.Context, in billing.GetInvoiceOwnershipAdapterInput) (billing.GetOwnershipAdapterResponse, error) {
	if err := in.Validate(); err != nil {
		return billing.GetOwnershipAdapterResponse{}, billingentity.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.GetOwnershipAdapterResponse, error) {
		dbInvoice, err := tx.db.BillingInvoice.Query().
			Where(billinginvoice.ID(in.ID)).
			Where(billinginvoice.Namespace(in.Namespace)).
			First(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return billing.GetOwnershipAdapterResponse{}, billingentity.NotFoundError{
					Entity: billingentity.EntityInvoice,
					ID:     in.ID,
					Err:    err,
				}
			}

			return billing.GetOwnershipAdapterResponse{}, err
		}

		return billing.GetOwnershipAdapterResponse{
			Namespace:  dbInvoice.Namespace,
			InvoiceID:  dbInvoice.ID,
			CustomerID: dbInvoice.CustomerID,
		}, nil
	})
}

func (a *adapter) mapInvoiceFromDB(ctx context.Context, invoice *db.BillingInvoice, expand billingentity.InvoiceExpand) (billingentity.Invoice, error) {
	res := billingentity.Invoice{
		InvoiceBase: billingentity.InvoiceBase{
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
				Timezone:         invoice.CustomerTimezone,
				UsageAttribution: invoice.CustomerUsageAttribution.CustomerUsageAttribution,
			},
			Period:    mapPeriodFromDB(invoice.PeriodStart, invoice.PeriodEnd),
			IssuedAt:  convert.TimePtrIn(invoice.IssuedAt, time.UTC),
			CreatedAt: invoice.CreatedAt.In(time.UTC),
			UpdatedAt: invoice.UpdatedAt.In(time.UTC),
			DeletedAt: convert.TimePtrIn(invoice.DeletedAt, time.UTC),
		},

		Totals: billingentity.Totals{
			Amount:              invoice.Amount,
			ChargesTotal:        invoice.ChargesTotal,
			DiscountsTotal:      invoice.DiscountsTotal,
			TaxesTotal:          invoice.TaxesTotal,
			TaxesExclusiveTotal: invoice.TaxesExclusiveTotal,
			TaxesInclusiveTotal: invoice.TaxesInclusiveTotal,
			Total:               invoice.Total,
		},

		ExpandedFields: expand,
	}

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

	if expand.Lines {
		mappedLines, err := a.mapInvoiceLineFromDB(ctx, invoice.Edges.BillingInvoiceLines)
		if err != nil {
			return billingentity.Invoice{}, err
		}

		res.Lines = billingentity.NewLineChildren(mappedLines)
	}

	if len(invoice.Edges.BillingInvoiceValidationIssues) > 0 {
		res.ValidationIssues = lo.Map(invoice.Edges.BillingInvoiceValidationIssues, func(issue *db.BillingInvoiceValidationIssue, _ int) billingentity.ValidationIssue {
			return billingentity.ValidationIssue{
				ID:        issue.ID,
				CreatedAt: issue.CreatedAt.In(time.UTC),
				UpdatedAt: issue.UpdatedAt.In(time.UTC),
				DeletedAt: convert.TimePtrIn(issue.DeletedAt, time.UTC),

				Severity:  issue.Severity,
				Message:   issue.Message,
				Code:      lo.FromPtrOr(issue.Code, ""),
				Component: billingentity.ComponentName(issue.Component),
				Path:      lo.FromPtrOr(issue.Path, ""),
			}
		})
	}

	return res, nil
}

func mapPeriodFromDB(start, end *time.Time) *billingentity.Period {
	if start == nil || end == nil {
		return nil
	}
	return &billingentity.Period{
		Start: start.In(time.UTC),
		End:   end.In(time.UTC),
	}
}
