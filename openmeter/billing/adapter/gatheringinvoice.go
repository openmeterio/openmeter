package billingadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

var _ billing.GatheringInvoiceAdapter = (*adapter)(nil)

func (a *adapter) CreateGatheringInvoice(ctx context.Context, input billing.CreateGatheringInvoiceAdapterInput) (billing.GatheringInvoice, error) {
	if err := input.Validate(); err != nil {
		return billing.GatheringInvoice{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.GatheringInvoice, error) {
		customer := input.Customer
		supplier := input.MergedProfile.Supplier

		// Clone the workflow config
		clonedWorkflowConfig, err := tx.createWorkflowConfig(ctx, input.Namespace, input.MergedProfile.WorkflowConfig)
		if err != nil {
			return billing.GatheringInvoice{}, fmt.Errorf("clone workflow config: %w", err)
		}

		currentSchemaLevel, err := tx.GetInvoiceDefaultSchemaLevel(ctx)
		if err != nil {
			return billing.GatheringInvoice{}, fmt.Errorf("get invoice write schema level: %w", err)
		}

		createMut := tx.db.BillingInvoice.Create().
			SetNamespace(input.Namespace).
			SetMetadata(input.Metadata).
			SetCurrency(input.Currency).
			SetStatus(billing.StandardInvoiceStatusGathering).
			SetSourceBillingProfileID(input.MergedProfile.ID).
			SetType(billing.InvoiceTypeStandard). // TODO: Migrate to GatheringInvoiceType once we have the type in the database
			SetNumber(input.Number).
			SetNillableDescription(input.Description).
			SetNillableCollectionAt(input.NextCollectionAt).
			SetSchemaLevel(currentSchemaLevel).
			// Customer snapshot about usage attribution fields
			SetCustomerID(input.Customer.ID).
			// TODO: Remove all below this line once we have separate tables for gathering invoices
			SetBillingWorkflowConfigID(clonedWorkflowConfig.ID).
			SetTaxAppID(input.MergedProfile.Apps.Tax.GetID().ID).
			SetInvoicingAppID(input.MergedProfile.Apps.Invoicing.GetID().ID).
			SetPaymentAppID(input.MergedProfile.Apps.Payment.GetID().ID).
			// Totals
			SetAmount(alpacadecimal.Zero).
			SetChargesTotal(alpacadecimal.Zero).
			SetDiscountsTotal(alpacadecimal.Zero).
			SetTaxesTotal(alpacadecimal.Zero).
			SetTaxesExclusiveTotal(alpacadecimal.Zero).
			SetTaxesInclusiveTotal(alpacadecimal.Zero).
			SetTotal(alpacadecimal.Zero).
			// Supplier contacts
			SetSupplierName(supplier.Name)

		// Customer usage attribution
		if usageAttr := mapCustomerUsageAttributionToDB(input.Customer); usageAttr != nil {
			createMut = createMut.SetCustomerUsageAttribution(usageAttr)
		}
		createMut = createMut.
			SetCustomerName(customer.Name)

		newInvoice, err := createMut.Save(ctx)
		if err != nil {
			return billing.GatheringInvoice{}, err
		}

		// Let's add required edges for mapping
		newInvoice.Edges.BillingWorkflowConfig = clonedWorkflowConfig

		return tx.mapGatheringInvoiceFromDB(ctx, newInvoice, billing.GatheringInvoiceExpands{})
	})
}

func (a *adapter) UpdateGatheringInvoice(ctx context.Context, in billing.GatheringInvoice) error {
	if err := in.Validate(); err != nil {
		return fmt.Errorf("validating gathering invoice: %w", err)
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		existingInvoice, err := tx.db.BillingInvoice.Query().
			Where(billinginvoice.ID(in.ID)).
			Where(billinginvoice.Namespace(in.Namespace)).
			Only(ctx)
		if err != nil {
			return err
		}

		if err := tx.validateUpdateGatheringInvoiceRequest(in, existingInvoice); err != nil {
			return err
		}

		updateQuery := tx.db.BillingInvoice.UpdateOneID(in.ID).
			Where(billinginvoice.Namespace(in.Namespace)).
			SetMetadata(in.Metadata).
			// Currency is immutable
			SetStatus(billing.StandardInvoiceStatusGathering).
			ClearStatusDetailsCache().
			// Type is immutable
			SetNumber(in.Number).
			SetOrClearDescription(in.Description).
			ClearDueAt().
			ClearPaymentProcessingEnteredAt().
			ClearDraftUntil().
			ClearIssuedAt().
			SetOrClearDeletedAt(convert.SafeToUTC(in.DeletedAt)).
			ClearSentToCustomerAt().
			ClearQuantitySnapshotedAt().
			// Totals
			SetAmount(alpacadecimal.Zero).
			SetChargesTotal(alpacadecimal.Zero).
			SetDiscountsTotal(alpacadecimal.Zero).
			SetTaxesTotal(alpacadecimal.Zero).
			SetTaxesExclusiveTotal(alpacadecimal.Zero).
			SetTaxesInclusiveTotal(alpacadecimal.Zero).
			SetTotal(alpacadecimal.Zero)

		if !in.NextCollectionAt.IsZero() {
			updateQuery = updateQuery.SetCollectionAt(in.NextCollectionAt.In(time.UTC))
		} else {
			updateQuery = updateQuery.ClearCollectionAt()
		}

		// Clear period when the invoice is soft-deleted
		if in.DeletedAt != nil {
			updateQuery = updateQuery.
				ClearPeriodStart().
				ClearPeriodEnd()
		} else {
			updateQuery = updateQuery.
				SetPeriodStart(in.ServicePeriod.From.In(time.UTC)).
				SetPeriodEnd(in.ServicePeriod.To.In(time.UTC))
		}

		// Supplier
		updateQuery = updateQuery.
			SetSupplierName("UNSET").        // Hack until we split the invoices table
			SetSupplierAddressCountry("XX"). // Hack until we split the invoices table
			ClearSupplierAddressPostalCode().
			ClearSupplierAddressCity().
			ClearSupplierAddressState().
			ClearSupplierAddressLine1().
			ClearSupplierAddressLine2().
			ClearSupplierAddressPhoneNumber()

		// Customer
		updateQuery = updateQuery.
			// CustomerID is immutable
			SetCustomerName("UNSET"). // hack until we split the invoices table
			ClearCustomerKey()

		updateQuery = updateQuery.
			ClearCustomerAddressCountry().
			ClearCustomerAddressPostalCode().
			ClearCustomerAddressCity().
			ClearCustomerAddressState().
			ClearCustomerAddressLine1().
			ClearCustomerAddressLine2().
			ClearCustomerAddressPhoneNumber()

		// ExternalIDs
		updateQuery = updateQuery.
			ClearInvoicingAppExternalID().
			ClearPaymentAppExternalID()

		_, err = updateQuery.Save(ctx)
		if err != nil {
			return err
		}

		if in.Lines.IsPresent() {
			err := tx.updateGatheringLines(ctx, in.Lines.OrEmpty())
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (a *adapter) ListGatheringInvoices(ctx context.Context, input billing.ListGatheringInvoicesInput) (pagination.Result[billing.GatheringInvoice], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[billing.GatheringInvoice]{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (pagination.Result[billing.GatheringInvoice], error) {
		query := tx.db.BillingInvoice.Query().
			Where(billinginvoice.NamespaceIn(input.Namespaces...)).
			Where(billinginvoice.StatusEQ(billing.StandardInvoiceStatusGathering))

		if len(input.Customers) > 0 {
			query = query.Where(billinginvoice.CustomerIDIn(input.Customers...))
		}

		if len(input.Currencies) > 0 {
			query = query.Where(billinginvoice.CurrencyIn(input.Currencies...))
		}

		order := entutils.GetOrdering(sortx.OrderDefault)
		if !input.Order.IsDefaultValue() {
			order = entutils.GetOrdering(input.Order)
		}

		if input.Expand.Has(billing.GatheringInvoiceExpandLines) {
			query = query.WithBillingInvoiceLines(func(q *db.BillingInvoiceLineQuery) {
				if !input.Expand.Has(billing.GatheringInvoiceExpandDeletedLines) {
					q = q.Where(billinginvoiceline.DeletedAtIsNil())
				}
				q.WithUsageBasedLine()
			})
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

		if !input.IncludeDeleted {
			query = query.Where(billinginvoice.DeletedAtIsNil())
		}

		response := pagination.Result[billing.GatheringInvoice]{
			Page: input.Page,
		}

		paged, err := query.Paginate(ctx, input.Page)
		if err != nil {
			return response, err
		}

		result := make([]billing.GatheringInvoice, 0, len(paged.Items))
		for _, invoice := range paged.Items {
			mapped, err := tx.mapGatheringInvoiceFromDB(ctx, invoice, input.Expand)
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

func (a *adapter) validateUpdateGatheringInvoiceRequest(req billing.GatheringInvoice, existing *db.BillingInvoice) error {
	if req.Currency != existing.Currency {
		return billing.ValidationError{
			Err: fmt.Errorf("currency cannot be changed"),
		}
	}

	if billing.InvoiceTypeStandard != existing.Type {
		return billing.ValidationError{
			Err: fmt.Errorf("type cannot be changed"),
		}
	}

	if req.CustomerID != existing.CustomerID {
		return billing.ValidationError{
			Err: fmt.Errorf("customer cannot be changed"),
		}
	}

	return nil
}

func (a *adapter) DeleteGatheringInvoice(ctx context.Context, input billing.DeleteGatheringInvoiceAdapterInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating delete gathering invoice input: %w", err)
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		invoice, err := tx.db.BillingInvoice.Query().
			Where(billinginvoice.ID(input.ID)).
			Where(billinginvoice.Namespace(input.Namespace)).
			Only(ctx)
		if err != nil {
			return err
		}

		if invoice.Status != billing.StandardInvoiceStatusGathering {
			return billing.ValidationError{
				Err: fmt.Errorf("invoice is not a gathering invoice [id=%s]", invoice.ID),
			}
		}

		if invoice.DeletedAt != nil {
			return nil
		}

		_, err = tx.db.BillingInvoice.Update().
			Where(billinginvoice.ID(input.ID)).
			Where(billinginvoice.Namespace(input.Namespace)).
			SetDeletedAt(clock.Now()).
			Save(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}

func (a *adapter) GetGatheringInvoiceById(ctx context.Context, input billing.GetGatheringInvoiceByIdInput) (billing.GatheringInvoice, error) {
	if err := input.Validate(); err != nil {
		return billing.GatheringInvoice{}, fmt.Errorf("validating get gathering invoice by id input: %w", err)
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.GatheringInvoice, error) {
		query := tx.db.BillingInvoice.Query().
			Where(billinginvoice.ID(input.Invoice.ID)).
			Where(billinginvoice.Namespace(input.Invoice.Namespace))

		if input.Expand.Has(billing.GatheringInvoiceExpandLines) {
			query = query.WithBillingInvoiceLines(func(q *db.BillingInvoiceLineQuery) {
				if !input.Expand.Has(billing.GatheringInvoiceExpandDeletedLines) {
					q = q.Where(billinginvoiceline.DeletedAtIsNil())
				}
				q.WithUsageBasedLine()
			})
		}

		invoice, err := query.Only(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return billing.GatheringInvoice{}, billing.NotFoundError{
					Err: fmt.Errorf("%w [id=%s]", billing.ErrInvoiceNotFound, input.Invoice.ID),
				}
			}

			return billing.GatheringInvoice{}, err
		}

		return tx.mapGatheringInvoiceFromDB(ctx, invoice, input.Expand)
	})
}

func (a *adapter) mapGatheringInvoiceFromDB(ctx context.Context, invoice *db.BillingInvoice, expand billing.GatheringInvoiceExpands) (billing.GatheringInvoice, error) {
	if invoice.Status != billing.StandardInvoiceStatusGathering {
		return billing.GatheringInvoice{}, fmt.Errorf("invoice is not a gathering invoice [id=%s]", invoice.ID)
	}

	period := timeutil.ClosedPeriod{}

	if invoice.PeriodStart != nil && invoice.PeriodEnd != nil {
		period = timeutil.ClosedPeriod{
			From: invoice.PeriodStart.In(time.UTC),
			To:   invoice.PeriodEnd.In(time.UTC),
		}
	}

	res := billing.GatheringInvoice{
		GatheringInvoiceBase: billing.GatheringInvoiceBase{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{
					Namespace: invoice.Namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: invoice.CreatedAt.In(time.UTC),
					UpdatedAt: invoice.UpdatedAt.In(time.UTC),
					DeletedAt: convert.TimePtrIn(invoice.DeletedAt, time.UTC),
				},
				ID:          invoice.ID,
				Name:        invoice.Number,
				Description: invoice.Description,
			},

			Metadata:         invoice.Metadata,
			Number:           invoice.Number,
			CustomerID:       invoice.CustomerID,
			Currency:         invoice.Currency,
			ServicePeriod:    period,
			NextCollectionAt: invoice.CollectionAt.In(time.UTC),
			SchemaLevel:      invoice.SchemaLevel,
		},
	}

	if expand.Has(billing.GatheringInvoiceExpandLines) {
		mappedLines, err := a.mapGatheringInvoiceLinesFromDB(invoice.SchemaLevel, invoice.Edges.BillingInvoiceLines)
		if err != nil {
			return billing.GatheringInvoice{}, err
		}

		// TODO[later]: Implement this once we have proper union type for invoices
		// mappedLines, err = a.expandSplitLineHierarchy(ctx, invoice.Namespace, mappedLines)
		// if err != nil {
		// 	return billing.StandardInvoice{}, err
		// }

		res.Lines = billing.NewGatheringInvoiceLines(mappedLines)
	}

	return res, nil
}
