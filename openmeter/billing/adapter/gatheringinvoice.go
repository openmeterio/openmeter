package billingadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	"github.com/samber/lo"
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

		workflowConfig := mapWorkflowConfigToDB(input.MergedProfile.WorkflowConfig, clonedWorkflowConfig.ID)

		// Force cloning of the workflow
		// TODO: Is this needed?
		workflowConfig.ID = ""

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
			// TODO: Remove all below this line once we have seperate tables for gathering invoices
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

		return tx.mapGatheringInvoiceFromDB(ctx, newInvoice, billing.InvoiceExpandAll)
	})
}

func (a *adapter) mapGatheringInvoiceFromDB(ctx context.Context, invoice *db.BillingInvoice, expand billing.InvoiceExpand) (billing.GatheringInvoice, error) {
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
					CreatedAt: invoice.CreatedAt,
					UpdatedAt: invoice.UpdatedAt,
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

	if expand.Lines {
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

func (a *adapter) mapGatheringInvoiceLinesFromDB(schemaLevel int, dbLines []*db.BillingInvoiceLine) ([]billing.GatheringLine, error) {
	return slicesx.MapWithErr(dbLines, func(dbLine *db.BillingInvoiceLine) (billing.GatheringLine, error) {
		return a.mapGatheringInvoiceLineFromDB(schemaLevel, dbLine)
	})
}

func (a *adapter) mapGatheringInvoiceLineFromDB(schemaLevel int, dbLine *db.BillingInvoiceLine) (billing.GatheringLine, error) {
	if dbLine.Type != billing.InvoiceLineTypeUsageBased {
		return billing.GatheringLine{}, fmt.Errorf("only usage based lines can be gathering invoice lines [line_id=%s]", dbLine.ID)
	}

	ubpLine := dbLine.Edges.UsageBasedLine
	if ubpLine == nil {
		return billing.GatheringLine{}, fmt.Errorf("usage based line data is missing [line_id=%s]", dbLine.ID)
	}

	line := billing.GatheringLine{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
			Namespace:   dbLine.Namespace,
			ID:          dbLine.ID,
			CreatedAt:   dbLine.CreatedAt.In(time.UTC),
			UpdatedAt:   dbLine.UpdatedAt.In(time.UTC),
			DeletedAt:   convert.TimePtrIn(dbLine.DeletedAt, time.UTC),
			Name:        dbLine.Name,
			Description: dbLine.Description,
		}),

		Metadata:    dbLine.Metadata,
		Annotations: dbLine.Annotations,
		InvoiceID:   dbLine.InvoiceID,
		ManagedBy:   dbLine.ManagedBy,

		ServicePeriod: timeutil.ClosedPeriod{
			From: dbLine.PeriodStart.In(time.UTC),
			To:   dbLine.PeriodEnd.In(time.UTC),
		},

		SplitLineGroupID:       dbLine.SplitLineGroupID,
		ChildUniqueReferenceID: dbLine.ChildUniqueReferenceID,

		InvoiceAt: dbLine.InvoiceAt.In(time.UTC),

		Currency: dbLine.Currency,

		TaxConfig:         lo.EmptyableToPtr(dbLine.TaxConfig),
		RateCardDiscounts: lo.FromPtr(dbLine.RatecardDiscounts),

		UBPConfigID: ubpLine.ID,
		FeatureKey:  lo.FromPtr(ubpLine.FeatureKey),
		Price:       lo.FromPtr(ubpLine.Price),
	}

	if dbLine.SubscriptionID != nil && dbLine.SubscriptionPhaseID != nil && dbLine.SubscriptionItemID != nil {
		line.Subscription = &billing.SubscriptionReference{
			SubscriptionID: *dbLine.SubscriptionID,
			PhaseID:        *dbLine.SubscriptionPhaseID,
			ItemID:         *dbLine.SubscriptionItemID,
			BillingPeriod: timeutil.ClosedPeriod{
				From: lo.FromPtr(dbLine.SubscriptionBillingPeriodFrom).In(time.UTC),
				To:   lo.FromPtr(dbLine.SubscriptionBillingPeriodTo).In(time.UTC),
			},
		}
	}

	return line, nil
}
