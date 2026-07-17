package billingadapter

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicevalidationissue"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

var _ billing.StandardInvoiceAdapter = (*adapter)(nil)

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
			WithBillingWorkflowConfig(workflowConfigWithTaxCode)

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

func (a *adapter) ListStandardInvoices(ctx context.Context, input billing.ListStandardInvoicesInput) (billing.ListStandardInvoicesResponse, error) {
	if err := input.Validate(); err != nil {
		return billing.ListStandardInvoicesResponse{}, billing.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.ListStandardInvoicesResponse, error) {
		// Note: we are not filtering for deleted invoices here (as in deleted_at is not nil), as we have the deleted
		// status that we can use to filter for.

		query := tx.db.BillingInvoice.Query().
			Where(billinginvoice.StatusNEQ(billing.StandardInvoiceStatusGathering)).
			WithBillingInvoiceValidationIssues(func(q *db.BillingInvoiceValidationIssueQuery) {
				q.Where(billinginvoicevalidationissue.DeletedAtIsNil())
			}).
			WithBillingWorkflowConfig(workflowConfigWithTaxCode)

		if len(input.Namespaces) > 0 {
			query = query.Where(billinginvoice.NamespaceIn(input.Namespaces...))
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

		if input.Expand.Has(billing.StandardInvoiceExpandLines) {
			query = tx.expandInvoiceLineItems(query, input.Expand)
		}

		query = query.Order(billinginvoice.ByCreatedAt(entutils.GetOrdering(sortx.OrderDefault)...))

		response := billing.ListStandardInvoicesResponse{
			Page: input.Page,
		}

		paged, err := query.Paginate(ctx, input.Page)
		if err != nil {
			return response, err
		}

		result := make([]billing.StandardInvoice, 0, len(paged.Items))
		for _, invoice := range paged.Items {
			mapped, err := tx.mapStandardInvoiceFromDB(ctx, invoice, input.Expand)
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
