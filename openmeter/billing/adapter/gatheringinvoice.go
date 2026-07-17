package billingadapter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinggatheringinvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinggatheringinvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicesearchv1"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

var _ billing.GatheringInvoiceAdapter = (*adapter)(nil)

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
			SetCreditsTotal(alpacadecimal.Zero).
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
			SetCreditsTotal(alpacadecimal.Zero).
			SetDiscountsTotal(alpacadecimal.Zero).
			SetTaxesTotal(alpacadecimal.Zero).
			SetTaxesExclusiveTotal(alpacadecimal.Zero).
			SetTaxesInclusiveTotal(alpacadecimal.Zero).
			SetTotal(alpacadecimal.Zero).
			SetOrClearCollectionAt(convert.SafeToUTC(in.NextCollectionAt))

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
		query := tx.db.BillingInvoiceSearchV1.Query().
			Where(billinginvoicesearchv1.InvoiceTypeEQ(billing.InvoiceTypeGathering))

		if len(input.Namespaces) > 0 {
			query = query.Where(billinginvoicesearchv1.NamespaceIn(input.Namespaces...))
		}

		if len(input.ExcludedNamespaces) > 0 {
			query = query.Where(billinginvoicesearchv1.NamespaceNotIn(input.ExcludedNamespaces...))
		}

		if len(input.Customers) > 0 {
			query = query.Where(billinginvoicesearchv1.CustomerIDIn(input.Customers...))
		}

		if len(input.Currencies) > 0 {
			query = query.Where(billinginvoicesearchv1.CurrencyIn(input.Currencies...))
		}

		order := entutils.GetOrdering(sortx.OrderDefault)
		if !input.Order.IsDefaultValue() {
			order = entutils.GetOrdering(input.Order)
		}

		query = filter.ApplyToQuery(query, &input.CollectionAt, billinginvoicesearchv1.FieldCollectionAt)
		if len(input.IDs) > 0 {
			query = query.Where(billinginvoicesearchv1.IDIn(input.IDs...))
		}

		switch input.OrderBy {
		case api.InvoiceOrderByCustomerName:
			query = query.Order(billinginvoicesearchv1.ByCustomerName(order...))
		case api.InvoiceOrderByIssuedAt:
			query = query.Order(billinginvoicesearchv1.ByIssuedAt(order...))
		case api.InvoiceOrderByPeriodStart:
			query = query.Order(billinginvoicesearchv1.ByServicePeriodStart(order...))
		case api.InvoiceOrderByStatus:
			query = query.Order(billinginvoicesearchv1.ByStatus(order...))
		case api.InvoiceOrderByUpdatedAt:
			query = query.Order(billinginvoicesearchv1.ByUpdatedAt(order...))
		case api.InvoiceOrderByCreatedAt:
			fallthrough
		default:
			query = query.Order(billinginvoicesearchv1.ByCreatedAt(order...))
		}

		if !input.IncludeDeleted {
			query = query.Where(billinginvoicesearchv1.DeletedAtIsNil())
		}

		response := pagination.Result[billing.GatheringInvoice]{
			Page: input.Page,
		}

		paged, err := query.Paginate(ctx, input.Page)
		if err != nil {
			return response, err
		}

		if len(paged.Items) == 0 {
			response.TotalCount = paged.TotalCount
			return response, nil
		}

		invoicesToBeLoaded := lo.GroupByMap(paged.Items, func(hit *db.BillingInvoiceSearchV1) (billinginvoicesearchv1.StorageTable, string) {
			return hit.StorageTable, hit.ID
		})

		// Downstream list operations can be permissive about namespace filtering because
		// response assembly only accepts invoices matching a search hit by namespace and ID.
		hydrated := make(map[models.NamespacedID]billing.GatheringInvoice, len(paged.Items))
		for storageTable, ids := range invoicesToBeLoaded {
			var invoices []billing.GatheringInvoice
			var err error

			switch storageTable {
			case billinginvoicesearchv1.StorageTableBillingInvoice:
				invoices, err = tx.listGatheringInvoices(ctx, listGatheringInvoicesInput{
					Namespaces:     input.Namespaces,
					IDs:            ids,
					Expand:         input.Expand,
					IncludeDeleted: true,
				})
			case billinginvoicesearchv1.StorageTableBillingGatheringInvoice:
				invoices, err = tx.listDedicatedGatheringInvoices(ctx, listGatheringInvoicesInput{
					Namespaces:     input.Namespaces,
					IDs:            ids,
					Expand:         input.Expand,
					IncludeDeleted: true,
				})
			default:
				return response, fmt.Errorf("unsupported gathering invoice storage table [storage_table=%s]", storageTable)
			}
			if err != nil {
				return response, fmt.Errorf("listing gathering invoices for search hydration [storage_table=%s]: %w", storageTable, err)
			}

			for _, invoice := range invoices {
				hydrated[models.NamespacedID{Namespace: invoice.Namespace, ID: invoice.ID}] = invoice
			}
		}

		result, err := lo.MapErr(paged.Items, func(hit *db.BillingInvoiceSearchV1, _ int) (billing.GatheringInvoice, error) {
			invoice, ok := hydrated[models.NamespacedID{Namespace: hit.Namespace, ID: hit.ID}]
			if !ok {
				return billing.GatheringInvoice{}, fmt.Errorf("gathering invoice search result could not be hydrated [namespace=%s, id=%s, storage_table=%s]", hit.Namespace, hit.ID, hit.StorageTable)
			}

			return invoice, nil
		})
		if err != nil {
			return response, err
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

func (a *adapter) expandGatheringInvoiceLines(q *db.BillingInvoiceQuery, expand billing.GatheringInvoiceExpands) *db.BillingInvoiceQuery {
	return q.WithBillingInvoiceLines(func(q *db.BillingInvoiceLineQuery) {
		if !expand.Has(billing.GatheringInvoiceExpandDeletedLines) {
			q = q.Where(billinginvoiceline.DeletedAtIsNil())
		}

		q.
			Where(billinginvoiceline.TypeEQ(billing.InvoiceLineAdapterTypeUsageBased)). // Only include usage based lines (there are some detailed lines existing for gathering invoices)
			Where(billinginvoiceline.ParentLineIDIsNil()).                              // Only include top-level lines (there are some detailed lines existing for gathering invoices)
			WithUsageBasedLine().
			WithTaxCode()
	})
}

func (a *adapter) expandDedicatedGatheringInvoiceLines(q *db.BillingGatheringInvoiceQuery, expand billing.GatheringInvoiceExpands) *db.BillingGatheringInvoiceQuery {
	return q.WithBillingGatheringInvoiceLines(func(q *db.BillingGatheringInvoiceLineQuery) {
		if !expand.Has(billing.GatheringInvoiceExpandDeletedLines) {
			q = q.Where(billinggatheringinvoiceline.DeletedAtIsNil())
		}

		q.WithTaxCode()
	})
}

func (a *adapter) GetGatheringInvoiceById(ctx context.Context, input billing.GetGatheringInvoiceByIdInput) (billing.GatheringInvoice, error) {
	if err := input.Validate(); err != nil {
		return billing.GatheringInvoice{}, fmt.Errorf("validating get gathering invoice by id input: %w", err)
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.GatheringInvoice, error) {
		searchResults, err := tx.db.BillingInvoiceSearchV1.Query().
			Where(billinginvoicesearchv1.ID(input.Invoice.ID)).
			Where(billinginvoicesearchv1.Namespace(input.Invoice.Namespace)).
			All(ctx)
		if err != nil {
			return billing.GatheringInvoice{}, err
		}

		if len(searchResults) == 0 {
			return billing.GatheringInvoice{}, billing.NotFoundError{
				Err: fmt.Errorf("%w [id=%s]", billing.ErrInvoiceNotFound, input.Invoice.ID),
			}
		}

		if len(searchResults) > 1 {
			return billing.GatheringInvoice{}, models.NewGenericConflictError(fmt.Errorf("invoice exists in multiple storage tables [namespace=%s, id=%s]", input.Invoice.Namespace, input.Invoice.ID))
		}

		searchResult := searchResults[0]
		if searchResult.InvoiceType != billing.InvoiceTypeGathering {
			return billing.GatheringInvoice{}, billing.ValidationError{
				Err: fmt.Errorf("invoice is not a gathering invoice [id=%s]", input.Invoice.ID),
			}
		}

		switch searchResult.StorageTable {
		case billinginvoicesearchv1.StorageTableBillingInvoice:
			return tx.getGatheringInvoiceById(ctx, input)
		case billinginvoicesearchv1.StorageTableBillingGatheringInvoice:
			return tx.getDedicatedGatheringInvoiceById(ctx, input)
		default:
			return billing.GatheringInvoice{}, fmt.Errorf("unsupported gathering invoice storage table [namespace=%s, id=%s, storage_table=%s]", input.Invoice.Namespace, input.Invoice.ID, searchResult.StorageTable)
		}
	})
}

func (a *adapter) getGatheringInvoiceById(ctx context.Context, input billing.GetGatheringInvoiceByIdInput) (billing.GatheringInvoice, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.GatheringInvoice, error) {
		query := tx.db.BillingInvoice.Query().
			Where(billinginvoice.ID(input.Invoice.ID)).
			Where(billinginvoice.Namespace(input.Invoice.Namespace))

		if input.Expand.Has(billing.GatheringInvoiceExpandLines) {
			query = tx.expandGatheringInvoiceLines(query, input.Expand)
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

func (a *adapter) getDedicatedGatheringInvoiceById(ctx context.Context, input billing.GetGatheringInvoiceByIdInput) (billing.GatheringInvoice, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.GatheringInvoice, error) {
		query := tx.db.BillingGatheringInvoice.Query().
			Where(billinggatheringinvoice.ID(input.Invoice.ID)).
			Where(billinggatheringinvoice.Namespace(input.Invoice.Namespace))

		if input.Expand.Has(billing.GatheringInvoiceExpandLines) {
			query = tx.expandDedicatedGatheringInvoiceLines(query, input.Expand)
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

		return tx.fromDBBillingGatheringInvoice(invoice, input.Expand)
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
			NextCollectionAt: convert.TimePtrIn(invoice.CollectionAt, time.UTC),
			SchemaLevel:      invoice.SchemaLevel,
		},

		Expands: expand,
	}

	if expand.Has(billing.GatheringInvoiceExpandLines) {
		mappedLines, err := a.mapGatheringInvoiceLinesFromDB(invoice.SchemaLevel, invoice.Edges.BillingInvoiceLines)
		if err != nil {
			return billing.GatheringInvoice{}, err
		}

		if expand.Has(billing.GatheringInvoiceExpandSplitLineHierarchy) {
			hierarchyByLineID, err := a.expandSplitLineHierarchy(ctx, invoice.Namespace, mappedLines.AsGenericLines())
			if err != nil {
				return billing.GatheringInvoice{}, err
			}

			mappedLinePtrs, err := withSplitLineHierarchyForLines(lo.Map(mappedLines, func(_ billing.GatheringLine, idx int) *billing.GatheringLine {
				return &mappedLines[idx]
			}), hierarchyByLineID)
			if err != nil {
				return billing.GatheringInvoice{}, err
			}

			mappedLines = lo.Map(mappedLinePtrs, func(line *billing.GatheringLine, _ int) billing.GatheringLine {
				return *line
			})
		}

		res.Lines = billing.NewGatheringInvoiceLines(mappedLines)
	}

	return res, nil
}

func (a *adapter) fromDBBillingGatheringInvoice(invoice *db.BillingGatheringInvoice, expand billing.GatheringInvoiceExpands) (billing.GatheringInvoice, error) {
	period := timeutil.ClosedPeriod{}
	if invoice.ServicePeriodStart != nil && invoice.ServicePeriodEnd != nil {
		period = timeutil.ClosedPeriod{
			From: invoice.ServicePeriodStart.In(time.UTC),
			To:   invoice.ServicePeriodEnd.In(time.UTC),
		}
	}

	result := billing.GatheringInvoice{
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
				Name:        invoice.Name,
				Description: invoice.Description,
			},
			Metadata:         invoice.Metadata,
			Number:           invoice.Number,
			CustomerID:       invoice.CustomerID,
			Currency:         invoice.Currency,
			ServicePeriod:    period,
			NextCollectionAt: convert.TimePtrIn(invoice.NextCollectionAt, time.UTC),
			SchemaLevel:      invoice.SchemaLevel,
		},
		Expands: expand,
	}

	if expand.Has(billing.GatheringInvoiceExpandLines) {
		lines := make(billing.GatheringLines, 0, len(invoice.Edges.BillingGatheringInvoiceLines))
		for _, dbLine := range invoice.Edges.BillingGatheringInvoiceLines {
			line, err := a.fromDBBillingGatheringInvoiceLine(dbLine)
			if err != nil {
				return billing.GatheringInvoice{}, fmt.Errorf("mapping gathering invoice line [id=%s]: %w", dbLine.ID, err)
			}
			lines = append(lines, line)
		}

		result.Lines = billing.NewGatheringInvoiceLines(lines)
	}

	return result, nil
}

type listGatheringInvoicesInput struct {
	Namespaces     []string
	IDs            []string
	Expand         billing.GatheringInvoiceExpands
	IncludeDeleted bool
}

func (a *adapter) listGatheringInvoices(ctx context.Context, input listGatheringInvoicesInput) ([]billing.GatheringInvoice, error) {
	query := a.db.BillingInvoice.Query().
		Where(billinginvoice.StatusEQ(billing.StandardInvoiceStatusGathering)).
		Where(billinginvoice.IDIn(input.IDs...))
	if len(input.Namespaces) > 0 {
		query = query.Where(billinginvoice.NamespaceIn(input.Namespaces...))
	}
	if !input.IncludeDeleted {
		query = query.Where(billinginvoice.DeletedAtIsNil())
	}
	if input.Expand.Has(billing.GatheringInvoiceExpandLines) {
		query = a.expandGatheringInvoiceLines(query, input.Expand)
	}

	invoices, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]billing.GatheringInvoice, 0, len(invoices))
	for _, invoice := range invoices {
		mapped, err := a.mapGatheringInvoiceFromDB(ctx, invoice, input.Expand)
		if err != nil {
			return nil, fmt.Errorf("mapping gathering invoice [namespace=%s, id=%s]: %w", invoice.Namespace, invoice.ID, err)
		}
		result = append(result, mapped)
	}

	return result, nil
}

func (a *adapter) listDedicatedGatheringInvoices(ctx context.Context, input listGatheringInvoicesInput) ([]billing.GatheringInvoice, error) {
	query := a.db.BillingGatheringInvoice.Query().
		Where(billinggatheringinvoice.IDIn(input.IDs...))
	if len(input.Namespaces) > 0 {
		query = query.Where(billinggatheringinvoice.NamespaceIn(input.Namespaces...))
	}
	if !input.IncludeDeleted {
		query = query.Where(billinggatheringinvoice.DeletedAtIsNil())
	}
	if input.Expand.Has(billing.GatheringInvoiceExpandLines) {
		query = a.expandDedicatedGatheringInvoiceLines(query, input.Expand)
	}

	invoices, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]billing.GatheringInvoice, 0, len(invoices))
	for _, invoice := range invoices {
		mapped, err := a.fromDBBillingGatheringInvoice(invoice, input.Expand)
		if err != nil {
			return nil, fmt.Errorf("mapping gathering invoice [namespace=%s, id=%s]: %w", invoice.Namespace, invoice.ID, err)
		}
		result = append(result, mapped)
	}

	return result, nil
}
