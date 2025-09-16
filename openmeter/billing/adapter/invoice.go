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
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicevalidationissue"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

var _ billing.InvoiceAdapter = (*adapter)(nil)

func (a *adapter) GetInvoiceById(ctx context.Context, in billing.GetInvoiceByIdInput) (billing.Invoice, error) {
	if err := in.Validate(); err != nil {
		return billing.Invoice{}, billing.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.Invoice, error) {
		query := tx.db.BillingInvoice.Query().
			Where(billinginvoice.ID(in.Invoice.ID)).
			Where(billinginvoice.Namespace(in.Invoice.Namespace)).
			WithBillingInvoiceValidationIssues(func(q *db.BillingInvoiceValidationIssueQuery) {
				q.Where(billinginvoicevalidationissue.DeletedAtIsNil())
			}).
			WithBillingWorkflowConfig()

		if in.Expand.Lines {
			query = tx.expandInvoiceLineItems(query, in.Expand)
		}

		invoice, err := query.Only(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return billing.Invoice{}, billing.NotFoundError{
					Err: fmt.Errorf("%w [id=%s]", billing.ErrInvoiceNotFound, in.Invoice.ID),
				}
			}

			return billing.Invoice{}, err
		}

		return tx.mapInvoiceFromDB(ctx, invoice, in.Expand)
	})
}

func (a *adapter) expandInvoiceLineItems(query *db.BillingInvoiceQuery, expand billing.InvoiceExpand) *db.BillingInvoiceQuery {
	return query.WithBillingInvoiceLines(func(q *db.BillingInvoiceLineQuery) {
		if !expand.DeletedLines {
			q = q.Where(billinginvoiceline.DeletedAtIsNil())
		}

		requestedStatuses := []billing.InvoiceLineStatus{billing.InvoiceLineStatusValid}

		q = q.Where(
			// Detailed lines are sub-lines of a line and should not be included in the top-level invoice
			billinginvoiceline.StatusIn(requestedStatuses...),
		)

		a.expandLineItems(q)
	})
}

func (a *adapter) DeleteGatheringInvoices(ctx context.Context, input billing.DeleteGatheringInvoicesInput) error {
	if err := input.Validate(); err != nil {
		return billing.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		nAffected, err := tx.db.BillingInvoice.Update().
			Where(billinginvoice.IDIn(input.InvoiceIDs...)).
			Where(billinginvoice.Namespace(input.Namespace)).
			Where(billinginvoice.StatusEQ(billing.InvoiceStatusGathering)).
			ClearPeriodStart().
			ClearPeriodEnd().
			SetDeletedAt(clock.Now()).
			Save(ctx)
		if err != nil {
			return err
		}

		if nAffected != len(input.InvoiceIDs) {
			return billing.ValidationError{
				Err: errors.New("invoices failed to delete"),
			}
		}

		return nil
	})
}

func (a *adapter) ListInvoices(ctx context.Context, input billing.ListInvoicesInput) (billing.ListInvoicesResponse, error) {
	if err := input.Validate(); err != nil {
		return billing.ListInvoicesResponse{}, billing.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.ListInvoicesResponse, error) {
		// Note: we are not filtering for deleted invoices here (as in deleted_at is not nil), as we have the deleted
		// status that we can use to filter for.

		query := tx.db.BillingInvoice.Query().
			WithBillingInvoiceValidationIssues(func(q *db.BillingInvoiceValidationIssueQuery) {
				q.Where(billinginvoicevalidationissue.DeletedAtIsNil())
			}).
			WithBillingWorkflowConfig()

		if len(input.Namespaces) > 0 {
			query = query.Where(billinginvoice.NamespaceIn(input.Namespaces...))
		}

		if len(input.Customers) > 0 {
			query = query.Where(billinginvoice.CustomerIDIn(input.Customers...))
		}

		if input.IssuedAfter != nil {
			query = query.Where(billinginvoice.IssuedAtGTE(*input.IssuedAfter))
		}

		if input.IssuedBefore != nil {
			query = query.Where(billinginvoice.IssuedAtLTE(*input.IssuedBefore))
		}

		if input.PeriodStartAfter != nil {
			query = query.Where(billinginvoice.PeriodStartGTE(*input.PeriodStartAfter))
		}

		if input.PeriodStartBefore != nil {
			query = query.Where(billinginvoice.PeriodStartLTE(*input.PeriodStartBefore))
		}

		if input.CreatedAfter != nil {
			query = query.Where(billinginvoice.CreatedAtGTE(*input.CreatedAfter))
		}

		if input.CreatedBefore != nil {
			query = query.Where(billinginvoice.CreatedAtLTE(*input.CreatedBefore))
		}

		if len(input.ExtendedStatuses) > 0 {
			query = query.Where(billinginvoice.StatusIn(input.ExtendedStatuses...))
		}

		if len(input.IDs) > 0 {
			query = query.Where(billinginvoice.IDIn(input.IDs...))
		}

		if !input.IncludeDeleted {
			query = query.Where(billinginvoice.DeletedAtIsNil())
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

		if input.DraftUntil != nil {
			query = query.Where(billinginvoice.DraftUntilLTE(*input.DraftUntil))
		}

		if input.CollectionAt != nil {
			query = query.Where(billinginvoice.Or(
				billinginvoice.CollectionAtLTE(*input.CollectionAt),
				billinginvoice.CollectionAtIsNil(),
			))
		}

		if len(input.HasAvailableAction) > 0 {
			query = query.Where(
				billinginvoice.Or(
					lo.Map(
						input.HasAvailableAction,
						func(action billing.InvoiceAvailableActionsFilter, _ int) predicate.BillingInvoice {
							return entutils.JSONBKeyExistsInObject(billinginvoice.FieldStatusDetailsCache, "availableActions", string(action))
						},
					)...,
				),
			)
		}

		if input.ExternalIDs != nil {
			switch input.ExternalIDs.Type {
			case billing.InvoicingExternalIDType:
				query = query.Where(billinginvoice.InvoicingAppExternalIDIn(input.ExternalIDs.IDs...))
			case billing.PaymentExternalIDType:
				query = query.Where(billinginvoice.PaymentAppExternalIDIn(input.ExternalIDs.IDs...))
			case billing.TaxExternalIDType:
				query = query.Where(billinginvoice.TaxAppExternalIDIn(input.ExternalIDs.IDs...))
			}
		}

		order := entutils.GetOrdering(sortx.OrderDefault)
		if !input.Order.IsDefaultValue() {
			order = entutils.GetOrdering(input.Order)
		}

		if input.Expand.Lines {
			query = tx.expandInvoiceLineItems(query, input.Expand)
		}

		switch input.OrderBy {
		case api.InvoiceOrderByCustomerName:
			query = query.Order(billinginvoice.ByCustomerName(order...))
		case api.InvoiceOrderByIssuedAt:
			query = query.Order(billinginvoice.ByIssuedAt(order...))
		case api.InvoiceOrderByPeriodStart:
			query = query.Order(billinginvoice.ByPeriodStart(order...))
		case api.InvoiceOrderByStatus:
			query = query.Order(billinginvoice.ByStatus(order...))
		case api.InvoiceOrderByUpdatedAt:
			query = query.Order(billinginvoice.ByUpdatedAt(order...))
		case api.InvoiceOrderByCreatedAt:
			fallthrough
		default:
			query = query.Order(billinginvoice.ByCreatedAt(order...))
		}

		response := pagination.Result[billing.Invoice]{
			Page: input.Page,
		}

		paged, err := query.Paginate(ctx, input.Page)
		if err != nil {
			return response, err
		}

		result := make([]billing.Invoice, 0, len(paged.Items))
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

		workflowConfig := mapWorkflowConfigToDB(input.Profile.WorkflowConfig, clonedWorkflowConfig.ID)

		// Force cloning of the workflow
		workflowConfig.ID = ""

		createMut := tx.db.BillingInvoice.Create().
			SetNamespace(input.Namespace).
			SetMetadata(input.Metadata).
			SetCurrency(input.Currency).
			SetStatus(input.Status).
			SetSourceBillingProfileID(input.Profile.ID).
			SetType(input.Type).
			SetNumber(input.Number).
			SetNillableDescription(input.Description).
			SetNillableDueAt(input.DueAt).
			SetNillableIssuedAt(lo.EmptyableToPtr(input.IssuedAt)).
			// Customer snapshot about usage attribution fields
			SetCustomerID(input.Customer.ID).
			SetNillableCustomerKey(input.Customer.Key).
			SetCustomerUsageAttribution(&billing.VersionedCustomerUsageAttribution{
				Type:                     billing.CustomerUsageAttributionTypeVersion,
				CustomerUsageAttribution: input.Customer.UsageAttribution,
			}).
			// Workflow (cloned)
			SetBillingWorkflowConfigID(clonedWorkflowConfig.ID).
			// TODO[later]: By cloning the AppIDs here we could support changing the apps in the billing profile if needed
			SetTaxAppID(input.Profile.Apps.Tax.GetID().ID).
			SetInvoicingAppID(input.Profile.Apps.Invoicing.GetID().ID).
			SetPaymentAppID(input.Profile.Apps.Payment.GetID().ID).
			// Totals
			SetAmount(input.Totals.Amount).
			SetChargesTotal(input.Totals.ChargesTotal).
			SetDiscountsTotal(input.Totals.DiscountsTotal).
			SetTaxesTotal(input.Totals.TaxesTotal).
			SetTaxesExclusiveTotal(input.Totals.TaxesExclusiveTotal).
			SetTaxesInclusiveTotal(input.Totals.TaxesInclusiveTotal).
			SetTotal(input.Totals.Total).
			// Supplier contacts
			SetNillableSupplierAddressCountry(supplier.Address.Country).
			SetNillableSupplierAddressPostalCode(supplier.Address.PostalCode).
			SetNillableSupplierAddressState(supplier.Address.State).
			SetNillableSupplierAddressCity(supplier.Address.City).
			SetNillableSupplierAddressLine1(supplier.Address.Line1).
			SetNillableSupplierAddressLine2(supplier.Address.Line2).
			SetNillableSupplierAddressPhoneNumber(supplier.Address.PhoneNumber).
			SetSupplierName(supplier.Name).
			SetNillableSupplierTaxCode(supplier.TaxCode)

		// Set collection_at only for new gathering invoices
		if input.Status == billing.InvoiceStatusGathering {
			createMut = createMut.SetCollectionAt(clock.Now())
		}

		if customer.BillingAddress != nil {
			createMut = createMut.
				// Customer contacts
				SetNillableCustomerAddressCountry(customer.BillingAddress.Country).
				SetNillableCustomerAddressPostalCode(customer.BillingAddress.PostalCode).
				SetNillableCustomerAddressState(customer.BillingAddress.State).
				SetNillableCustomerAddressCity(customer.BillingAddress.City).
				SetNillableCustomerAddressLine1(customer.BillingAddress.Line1).
				SetNillableCustomerAddressLine2(customer.BillingAddress.Line2).
				SetNillableCustomerAddressPhoneNumber(customer.BillingAddress.PhoneNumber)
		}
		createMut = createMut.
			SetCustomerName(customer.Name)

		newInvoice, err := createMut.Save(ctx)
		if err != nil {
			return billing.CreateInvoiceAdapterRespone{}, err
		}

		// Let's add required edges for mapping
		newInvoice.Edges.BillingWorkflowConfig = clonedWorkflowConfig

		return tx.mapInvoiceFromDB(ctx, newInvoice, billing.InvoiceExpandAll)
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
			Where(billinginvoiceline.StatusIn(billing.InvoiceLineStatusValid)).
			GroupBy(billinginvoiceline.FieldInvoiceID).
			Aggregate(
				db.Count(),
			).
			Scan(ctx, &queryOut)
		if err != nil {
			return billing.AssociatedLineCountsAdapterResponse{}, err
		}

		res := lo.Associate(queryOut, func(q lineCountQueryOut) (billing.InvoiceID, int64) {
			return billing.InvoiceID{
				Namespace: input.Namespace,
				ID:        q.InvoiceID,
			}, q.Count
		})

		for _, invoiceID := range input.InvoiceIDs {
			id := billing.InvoiceID{
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
		return billing.ValidationError{
			Err: fmt.Errorf("currency cannot be changed"),
		}
	}

	if req.Type != existing.Type {
		return billing.ValidationError{
			Err: fmt.Errorf("type cannot be changed"),
		}
	}

	if req.Customer.CustomerID != existing.CustomerID {
		return billing.ValidationError{
			Err: fmt.Errorf("customer cannot be changed"),
		}
	}

	return nil
}

// UpdateInvoice updates the specified invoice.
func (a *adapter) UpdateInvoice(ctx context.Context, in billing.UpdateInvoiceAdapterInput) (billing.Invoice, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.Invoice, error) {
		existingInvoice, err := tx.db.BillingInvoice.Query().
			Where(billinginvoice.ID(in.ID)).
			Where(billinginvoice.Namespace(in.Namespace)).
			WithBillingWorkflowConfig().
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
			SetOrClearStatusDetailsCache(lo.EmptyableToPtr(in.StatusDetails)).
			// Type is immutable
			SetNumber(in.Number).
			SetOrClearDescription(in.Description).
			SetOrClearDueAt(convert.SafeToUTC(in.DueAt)).
			SetOrClearCollectionAt(convert.SafeToUTC(in.CollectionAt)).
			SetOrClearDraftUntil(convert.SafeToUTC(in.DraftUntil)).
			SetOrClearIssuedAt(convert.SafeToUTC(in.IssuedAt)).
			SetOrClearDeletedAt(convert.SafeToUTC(in.DeletedAt)).
			SetOrClearSentToCustomerAt(convert.SafeToUTC(in.SentToCustomerAt)).
			SetOrClearQuantitySnapshotedAt(convert.SafeToUTC(in.QuantitySnapshotedAt)).
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
				SetPeriodStart(in.Period.Start.In(time.UTC)).
				SetPeriodEnd(in.Period.End.In(time.UTC))
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
			SetCustomerName(in.Customer.Name)

		if in.Customer.Key != nil {
			updateQuery = updateQuery.SetCustomerKey(*in.Customer.Key)
		} else {
			updateQuery = updateQuery.ClearCustomerKey()
		}

		if in.Customer.BillingAddress != nil {
			updateQuery = updateQuery.
				SetOrClearCustomerAddressCountry(in.Customer.BillingAddress.Country).
				SetOrClearCustomerAddressPostalCode(in.Customer.BillingAddress.PostalCode).
				SetOrClearCustomerAddressCity(in.Customer.BillingAddress.City).
				SetOrClearCustomerAddressState(in.Customer.BillingAddress.State).
				SetOrClearCustomerAddressLine1(in.Customer.BillingAddress.Line1).
				SetOrClearCustomerAddressLine2(in.Customer.BillingAddress.Line2).
				SetOrClearCustomerAddressPhoneNumber(in.Customer.BillingAddress.PhoneNumber)
		} else {
			updateQuery = updateQuery.
				ClearCustomerAddressCountry().
				ClearCustomerAddressPostalCode().
				ClearCustomerAddressCity().
				ClearCustomerAddressState().
				ClearCustomerAddressLine1().
				ClearCustomerAddressLine2().
				ClearCustomerAddressPhoneNumber()
		}

		// ExternalIDs
		updateQuery = updateQuery.
			SetOrClearInvoicingAppExternalID(lo.EmptyableToPtr(in.ExternalIDs.Invoicing)).
			SetOrClearPaymentAppExternalID(lo.EmptyableToPtr(in.ExternalIDs.Payment))

		_, err = updateQuery.Save(ctx)
		if err != nil {
			return in, err
		}

		err = tx.persistValidationIssues(ctx,
			billing.InvoiceID{
				Namespace: in.Namespace,
				ID:        in.ID,
			}, in.ValidationIssues)
		if err != nil {
			return in, err
		}

		// Update the workflow config
		_, err = tx.updateWorkflowConfig(ctx, in.Namespace, existingInvoice.Edges.BillingWorkflowConfig.ID, in.Workflow.Config)
		if err != nil {
			return in, err
		}

		updatedLines := billing.InvoiceLines{}
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

			updatedLines = billing.NewInvoiceLines(lines)
		}

		// Let's return the updated invoice
		if !in.ExpandedFields.DeletedLines && updatedLines.IsPresent() {
			// If we haven't requested deleted lines, let's filter them out, as if there were lines marked deleted
			// the adapter update would return them as well.
			updatedLines = billing.NewInvoiceLines(
				lo.Filter(updatedLines.OrEmpty(), func(line *billing.Line, _ int) bool {
					return line.DeletedAt == nil
				}),
			)
		}

		// If we had just updated the lines, let's reuse that result, as it's quite an expensive operation
		// to look up the lines again.
		if in.ExpandedFields.Lines && updatedLines.IsPresent() {
			updatedInvoice, err := tx.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
				Invoice: billing.InvoiceID{
					Namespace: in.Namespace,
					ID:        in.ID,
				},
				Expand: in.ExpandedFields.SetLines(false),
			})
			if err != nil {
				return in, err
			}

			updatedInvoice.Lines = updatedLines
			// Let's make sure that subsequent calls preserve the same expansion settings
			updatedInvoice.ExpandedFields = in.ExpandedFields

			return updatedInvoice, nil
		}

		return tx.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
			Invoice: billing.InvoiceID{
				Namespace: in.Namespace,
				ID:        in.ID,
			},
			Expand: in.ExpandedFields,
		})
	})
}

func (a *adapter) GetInvoiceOwnership(ctx context.Context, in billing.GetInvoiceOwnershipAdapterInput) (billing.GetOwnershipAdapterResponse, error) {
	if err := in.Validate(); err != nil {
		return billing.GetOwnershipAdapterResponse{}, billing.ValidationError{
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
				return billing.GetOwnershipAdapterResponse{}, billing.NotFoundError{
					Entity: billing.EntityInvoice,
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

func (a *adapter) mapInvoiceBaseFromDB(ctx context.Context, invoice *db.BillingInvoice) billing.InvoiceBase {
	return billing.InvoiceBase{
		ID:                   invoice.ID,
		Namespace:            invoice.Namespace,
		Metadata:             invoice.Metadata,
		Currency:             invoice.Currency,
		Status:               invoice.Status,
		StatusDetails:        invoice.StatusDetailsCache,
		Type:                 invoice.Type,
		Number:               invoice.Number,
		Description:          invoice.Description,
		DueAt:                convert.TimePtrIn(invoice.DueAt, time.UTC),
		DraftUntil:           convert.TimePtrIn(invoice.DraftUntil, time.UTC),
		SentToCustomerAt:     convert.TimePtrIn(invoice.SentToCustomerAt, time.UTC),
		QuantitySnapshotedAt: convert.TimePtrIn(invoice.QuantitySnapshotedAt, time.UTC),
		Supplier: billing.SupplierContact{
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

		Customer: billing.InvoiceCustomer{
			Key:        invoice.CustomerKey,
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
			UsageAttribution: invoice.CustomerUsageAttribution.CustomerUsageAttribution,
		},
		Period:    mapPeriodFromDB(invoice.PeriodStart, invoice.PeriodEnd),
		IssuedAt:  convert.TimePtrIn(invoice.IssuedAt, time.UTC),
		CreatedAt: invoice.CreatedAt.In(time.UTC),
		UpdatedAt: invoice.UpdatedAt.In(time.UTC),
		DeletedAt: convert.TimePtrIn(invoice.DeletedAt, time.UTC),

		CollectionAt: lo.ToPtr(invoice.CollectionAt.In(time.UTC)),

		ExternalIDs: billing.InvoiceExternalIDs{
			Invoicing: lo.FromPtr(invoice.InvoicingAppExternalID),
			Payment:   lo.FromPtr(invoice.PaymentAppExternalID),
		},
	}
}

func (a *adapter) mapInvoiceFromDB(ctx context.Context, invoice *db.BillingInvoice, expand billing.InvoiceExpand) (billing.Invoice, error) {
	base := a.mapInvoiceBaseFromDB(ctx, invoice)

	res := billing.Invoice{
		InvoiceBase: base,

		Totals: billing.Totals{
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
		return billing.Invoice{}, err
	}

	res.Workflow = billing.InvoiceWorkflow{
		Config:                 workflowConfig,
		SourceBillingProfileID: invoice.SourceBillingProfileID,

		AppReferences: billing.ProfileAppReferences{
			Tax: app.AppID{
				Namespace: invoice.Namespace,
				ID:        invoice.TaxAppID,
			},
			Invoicing: app.AppID{
				Namespace: invoice.Namespace,
				ID:        invoice.InvoicingAppID,
			},
			Payment: app.AppID{
				Namespace: invoice.Namespace,
				ID:        invoice.PaymentAppID,
			},
		},
	}

	if expand.Lines {
		mappedLines, err := a.mapInvoiceLineFromDB(ctx, mapInvoiceLineFromDBInput{
			lines:          invoice.Edges.BillingInvoiceLines,
			includeDeleted: expand.DeletedLines,
		})
		if err != nil {
			return billing.Invoice{}, err
		}

		mappedLines, err = a.expandSplitLineHierarchy(ctx, invoice.Namespace, mappedLines)
		if err != nil {
			return billing.Invoice{}, err
		}

		res.Lines = billing.NewInvoiceLines(mappedLines)
	}

	if len(invoice.Edges.BillingInvoiceValidationIssues) > 0 {
		res.ValidationIssues = lo.Map(invoice.Edges.BillingInvoiceValidationIssues, func(issue *db.BillingInvoiceValidationIssue, _ int) billing.ValidationIssue {
			return billing.ValidationIssue{
				ID:        issue.ID,
				CreatedAt: issue.CreatedAt.In(time.UTC),
				UpdatedAt: issue.UpdatedAt.In(time.UTC),
				DeletedAt: convert.TimePtrIn(issue.DeletedAt, time.UTC),

				Severity:  issue.Severity,
				Message:   issue.Message,
				Code:      lo.FromPtr(issue.Code),
				Component: billing.ComponentName(issue.Component),
				Path:      lo.FromPtr(issue.Path),
			}
		})
	}

	return res, nil
}

func mapPeriodFromDB(start, end *time.Time) *billing.Period {
	if start == nil || end == nil {
		return nil
	}
	return &billing.Period{
		Start: start.In(time.UTC),
		End:   end.In(time.UTC),
	}
}

// IsAppUsed checks if the app is used in any invoice.
func (a *adapter) IsAppUsed(ctx context.Context, appID app.AppID) error {
	if err := appID.Validate(); err != nil {
		return billing.ValidationError{
			Err: fmt.Errorf("invalid app ID: %w", err),
		}
	}

	// Check if the app is used in any billing profile
	err := a.isBillingProfileUsed(ctx, appID)
	if err != nil {
		return err
	}

	// Check if the app is used in any invoice in gathering or issued states
	usedInInvoices, err := a.db.BillingInvoice.
		Query().
		Where(billinginvoice.Namespace(appID.Namespace)).
		Where(
			// The non-final states are listed here, so that we can make sure that all
			// invoices can reach a final state before the app is removed.
			billinginvoice.StatusIn(
				billing.InvoiceStatusGathering,
				billing.InvoiceStatusIssuingSyncing,
				billing.InvoiceStatusIssuingSyncFailed,
				billing.InvoiceStatusIssued,
				billing.InvoiceStatusPaymentProcessingPending,
				billing.InvoiceStatusPaymentProcessingFailed,
				billing.InvoiceStatusPaymentProcessingActionRequired,
				billing.InvoiceStatusOverdue,
			),
			billinginvoice.DeletedAtIsNil(),
		).
		Where(
			billinginvoice.Or(
				billinginvoice.InvoicingAppID(appID.ID),
				billinginvoice.PaymentAppID(appID.ID),
				billinginvoice.TaxAppID(appID.ID),
			),
		).
		All(ctx)
	if err != nil {
		return err
	}

	if len(usedInInvoices) > 0 {
		return models.NewGenericConflictError(fmt.Errorf("app is used in %d non-finalized invoices: %s", len(usedInInvoices), strings.Join(lo.Map(usedInInvoices, func(invoice *db.BillingInvoice, _ int) string {
			return fmt.Sprintf("%s[%s]", invoice.Number, invoice.ID)
		}), ",")))
	}

	return nil
}
