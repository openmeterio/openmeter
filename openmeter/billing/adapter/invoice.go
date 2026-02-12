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
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicevalidationissue"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

var _ billing.InvoiceAdapter = (*adapter)(nil)

func (a *adapter) GetStandardInvoiceById(ctx context.Context, in billing.GetStandardInvoiceByIdInput) (billing.StandardInvoice, error) {
	if err := in.Validate(); err != nil {
		return billing.StandardInvoice{}, billing.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.StandardInvoice, error) {
		query := tx.db.BillingInvoice.Query().
			Where(billinginvoice.ID(in.Invoice.ID)).
			Where(billinginvoice.Namespace(in.Invoice.Namespace)).
			Where(billinginvoice.StatusNEQ(billing.StandardInvoiceStatusGathering)).
			WithBillingInvoiceValidationIssues(func(q *db.BillingInvoiceValidationIssueQuery) {
				q.Where(billinginvoicevalidationissue.DeletedAtIsNil())
			}).
			WithBillingWorkflowConfig()

		if in.Expand.Has(billing.StandardInvoiceExpandLines) {
			query = tx.expandInvoiceLineItems(query, in.Expand)
		}

		invoice, err := query.Only(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return billing.StandardInvoice{}, billing.NotFoundError{
					Err: fmt.Errorf("%w [id=%s]", billing.ErrInvoiceNotFound, in.Invoice.ID),
				}
			}

			return billing.StandardInvoice{}, err
		}

		return tx.mapStandardInvoiceFromDB(ctx, invoice, in.Expand)
	})
}

func (a *adapter) expandInvoiceLineItems(query *db.BillingInvoiceQuery, expand billing.StandardInvoiceExpands) *db.BillingInvoiceQuery {
	return query.WithBillingInvoiceLines(func(q *db.BillingInvoiceLineQuery) {
		if !expand.Has(billing.StandardInvoiceExpandDeletedLines) {
			q = q.Where(billinginvoiceline.DeletedAtIsNil())
		}

		requestedStatuses := []billing.InvoiceLineStatus{billing.InvoiceLineStatusValid}

		q = q.Where(
			// Detailed lines are sub-lines of a line and should not be included in the top-level invoice
			billinginvoiceline.StatusIn(requestedStatuses...),
		)

		a.expandLineItemsWithDetailedLines(q)
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
			Where(billinginvoice.StatusEQ(billing.StandardInvoiceStatusGathering)).
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

func (a *adapter) ListInvoices(ctx context.Context, input billing.ListInvoicesAdapterInput) (billing.ListInvoicesResponse, error) {
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

		if len(input.IDs) > 0 {
			query = query.Where(billinginvoice.IDIn(input.IDs...))
		}

		if !input.IncludeDeleted {
			query = query.Where(billinginvoice.DeletedAtIsNil())
		}

		if input.OnlyGathering {
			query = query.Where(billinginvoice.StatusEQ(billing.StandardInvoiceStatusGathering))
		}

		if input.OnlyStandard {
			query = query.Where(billinginvoice.StatusNEQ(billing.StandardInvoiceStatusGathering))
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

		if len(input.ExtendedStatuses) > 0 {
			query = query.Where(billinginvoice.StatusIn(input.ExtendedStatuses...))
		}

		if input.DraftUntilLTE != nil {
			query = query.Where(billinginvoice.DraftUntilLTE(*input.DraftUntilLTE))
		}

		if input.CollectionAtLTE != nil {
			query = query.Where(billinginvoice.Or(
				billinginvoice.CollectionAtLTE(*input.CollectionAtLTE),
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

		if input.Expand.Has(billing.InvoiceExpandLines) {
			query = tx.expandInvoiceLineItems(query, billing.
				StandardInvoiceExpands{billing.StandardInvoiceExpandLines}.
				If(input.Expand.Has(billing.InvoiceExpandDeletedLines), billing.StandardInvoiceExpandDeletedLines))
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
			switch invoice.Status {
			case billing.StandardInvoiceStatusGathering:
				mapped, err := tx.mapGatheringInvoiceFromDB(ctx, invoice, billing.GatheringInvoiceExpands{}.
					If(input.Expand.Has(billing.InvoiceExpandLines), billing.GatheringInvoiceExpandLines).
					If(input.Expand.Has(billing.InvoiceExpandDeletedLines), billing.GatheringInvoiceExpandDeletedLines),
				)
				if err != nil {
					return response, err
				}
				result = append(result, billing.NewInvoice(mapped))
			default:
				mapped, err := tx.mapStandardInvoiceFromDB(ctx, invoice, billing.StandardInvoiceExpands{}.
					If(input.Expand.Has(billing.InvoiceExpandLines), billing.StandardInvoiceExpandLines).
					If(input.Expand.Has(billing.InvoiceExpandDeletedLines), billing.StandardInvoiceExpandDeletedLines),
				)
				if err != nil {
					return response, err
				}

				result = append(result, billing.NewInvoice(mapped))
			}
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

		currentSchemaLevel, err := tx.GetInvoiceDefaultSchemaLevel(ctx)
		if err != nil {
			return billing.CreateInvoiceAdapterRespone{}, fmt.Errorf("get invoice write schema level: %w", err)
		}

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
			SetNillableSupplierTaxCode(supplier.TaxCode).
			SetNillableCollectionAt(input.CollectionAt).
			SetSchemaLevel(currentSchemaLevel)

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
		if usageAttr := mapCustomerUsageAttributionToDB(input.Customer); usageAttr != nil {
			createMut = createMut.SetCustomerUsageAttribution(usageAttr)
		}
		createMut = createMut.
			SetCustomerName(customer.Name)

		newInvoice, err := createMut.Save(ctx)
		if err != nil {
			return billing.CreateInvoiceAdapterRespone{}, err
		}

		// Let's add required edges for mapping
		newInvoice.Edges.BillingWorkflowConfig = clonedWorkflowConfig

		return tx.mapStandardInvoiceFromDB(ctx, newInvoice, billing.StandardInvoiceExpandAll)
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

func (a *adapter) validateUpdateRequest(req billing.UpdateStandardInvoiceAdapterInput, existing *db.BillingInvoice) error {
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
func (a *adapter) UpdateStandardInvoice(ctx context.Context, in billing.UpdateStandardInvoiceAdapterInput) (billing.StandardInvoice, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.StandardInvoice, error) {
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
			SetOrClearPaymentProcessingEnteredAt(convert.SafeToUTC(in.PaymentProcessingEnteredAt)).
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

		updatedLines := billing.StandardInvoiceLines{}
		if in.Lines.IsPresent() {
			// Note: this only supports adding new lines or setting the DeletedAt field
			// we don't support moving lines between invoices here, as the cross invoice
			// coordination is not something the adapter should deal with. The service
			// is needed to lock and recalculate both invoices or do the necessary splits.

			lines, err := tx.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
				Namespace:   in.Namespace,
				Lines:       in.Lines.OrEmpty(),
				SchemaLevel: in.SchemaLevel,
				InvoiceID:   in.ID,
			})
			if err != nil {
				return in, err
			}

			updatedLines = billing.NewStandardInvoiceLines(lines)
		}

		// If we had just updated the lines, let's reuse that result, as it's quite an expensive operation
		// to look up the lines again.
		if in.ExpandedFields.Has(billing.StandardInvoiceExpandLines) && updatedLines.IsPresent() {
			updatedInvoice, err := tx.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
				Invoice: billing.InvoiceID{
					Namespace: in.Namespace,
					ID:        in.ID,
				},
				Expand: in.ExpandedFields.Without(billing.StandardInvoiceExpandLines),
			})
			if err != nil {
				return in, err
			}

			updatedInvoice.Lines = updatedLines
			// Let's make sure that subsequent calls preserve the same expansion settings
			updatedInvoice.ExpandedFields = in.ExpandedFields

			return updatedInvoice, nil
		}

		return tx.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
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

func (a *adapter) mapStandardInvoiceBaseFromDB(invoice *db.BillingInvoice) billing.StandardInvoiceBase {
	return billing.StandardInvoiceBase{
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
			UsageAttribution: mapCustomerUsageAttributionFromDB(invoice.CustomerID, invoice.CustomerKey, invoice.CustomerUsageAttribution),
		},
		Period:    mapPeriodFromDB(invoice.PeriodStart, invoice.PeriodEnd),
		IssuedAt:  convert.TimePtrIn(invoice.IssuedAt, time.UTC),
		CreatedAt: invoice.CreatedAt.In(time.UTC),
		UpdatedAt: invoice.UpdatedAt.In(time.UTC),
		DeletedAt: convert.TimePtrIn(invoice.DeletedAt, time.UTC),

		CollectionAt:               lo.ToPtr(invoice.CollectionAt.In(time.UTC)),
		PaymentProcessingEnteredAt: convert.TimePtrIn(invoice.PaymentProcessingEnteredAt, time.UTC),

		ExternalIDs: billing.InvoiceExternalIDs{
			Invoicing: lo.FromPtr(invoice.InvoicingAppExternalID),
			Payment:   lo.FromPtr(invoice.PaymentAppExternalID),
		},

		SchemaLevel: invoice.SchemaLevel,
	}
}

func (a *adapter) mapStandardInvoiceFromDB(ctx context.Context, invoice *db.BillingInvoice, expand billing.StandardInvoiceExpands) (billing.StandardInvoice, error) {
	base := a.mapStandardInvoiceBaseFromDB(invoice)

	res := billing.StandardInvoice{
		StandardInvoiceBase: base,

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
		return billing.StandardInvoice{}, err
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

	if expand.Has(billing.StandardInvoiceExpandLines) {
		mappedLines, err := a.mapStandardInvoiceLinesFromDB(map[string]int{invoice.ID: invoice.SchemaLevel}, invoice.Edges.BillingInvoiceLines)
		if err != nil {
			return billing.StandardInvoice{}, err
		}

		hierarchyByLineID, err := a.expandSplitLineHierarchy(ctx, invoice.Namespace, mappedLines.AsGenericLines())
		if err != nil {
			return billing.StandardInvoice{}, err
		}

		err = setSplitLineHierarchForLines[*billing.StandardLine](mappedLines, hierarchyByLineID)
		if err != nil {
			return billing.StandardInvoice{}, err
		}

		res.Lines = billing.NewStandardInvoiceLines(mappedLines)
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

func mapCustomerUsageAttributionFromDB(customerID string, customerKey *string, vua *billing.VersionedCustomerUsageAttribution) *streaming.CustomerUsageAttribution {
	if vua == nil {
		return nil
	}

	switch vua.Type {
	case billing.CustomerUsageAttributionTypeVersionV1:
		// For version 1, we backfill the usage attribution from the explicit fields
		return lo.ToPtr(streaming.NewCustomerUsageAttribution(customerID, customerKey, vua.CustomerUsageAttribution.SubjectKeys))
	case billing.CustomerUsageAttributionTypeVersionV2:
		return &vua.CustomerUsageAttribution
	default:
		return nil
	}
}

func mapCustomerUsageAttributionToDB(customer customer.Customer) *billing.VersionedCustomerUsageAttribution {
	// We allow invoices without usage attribution, but we don't store them in the database.
	// We only allow them when lines are not usage based.
	if err := customer.GetUsageAttribution().Validate(); err != nil {
		return nil
	}

	return &billing.VersionedCustomerUsageAttribution{
		Type:                     billing.CustomerUsageAttributionTypeVersionV2,
		CustomerUsageAttribution: customer.GetUsageAttribution(),
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
				billing.StandardInvoiceStatusGathering,
				billing.StandardInvoiceStatusIssuingSyncing,
				billing.StandardInvoiceStatusIssuingSyncFailed,
				billing.StandardInvoiceStatusIssued,
				billing.StandardInvoiceStatusPaymentProcessingPending,
				billing.StandardInvoiceStatusPaymentProcessingFailed,
				billing.StandardInvoiceStatusPaymentProcessingActionRequired,
				billing.StandardInvoiceStatusOverdue,
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

func (a *adapter) GetInvoiceType(ctx context.Context, input billing.GetInvoiceTypeAdapterInput) (billing.InvoiceType, error) {
	if err := input.Validate(); err != nil {
		return "", err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.InvoiceType, error) {
		invoice, err := tx.db.BillingInvoice.Query().
			Where(billinginvoice.ID(input.ID)).
			Where(billinginvoice.Namespace(input.Namespace)).
			Only(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return "", billing.NotFoundError{
					Err: fmt.Errorf("invoice not found: %w", err),
				}
			}

			return "", err
		}

		if invoice.Status == billing.StandardInvoiceStatusGathering {
			return billing.InvoiceTypeGathering, nil
		}

		return billing.InvoiceTypeStandard, nil
	})
}
