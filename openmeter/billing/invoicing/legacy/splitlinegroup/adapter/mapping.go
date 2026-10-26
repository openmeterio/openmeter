package adapter

import (
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/invoicing/legacy/splitlinegroup"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (a *adapter) mapSplitLineGroupFromDB(dbSplitLineGroup *db.BillingInvoiceSplitLineGroup) (splitlinegroup.SplitLineGroup, error) {
	if dbSplitLineGroup.Price == nil {
		return splitlinegroup.SplitLineGroup{}, fmt.Errorf("price is required")
	}

	var subscriptionRef *billing.SubscriptionReference
	if dbSplitLineGroup.SubscriptionID != nil || dbSplitLineGroup.SubscriptionPhaseID != nil || dbSplitLineGroup.SubscriptionItemID != nil {
		subscriptionRef = &billing.SubscriptionReference{
			SubscriptionID: lo.FromPtr(dbSplitLineGroup.SubscriptionID),
			PhaseID:        lo.FromPtr(dbSplitLineGroup.SubscriptionPhaseID),
			ItemID:         lo.FromPtr(dbSplitLineGroup.SubscriptionItemID),
			BillingPeriod: timeutil.ClosedPeriod{
				From: lo.FromPtr(dbSplitLineGroup.SubscriptionBillingPeriodFrom).In(time.UTC),
				To:   lo.FromPtr(dbSplitLineGroup.SubscriptionBillingPeriodTo).In(time.UTC),
			},
		}

		if err := subscriptionRef.Validate(); err != nil {
			return splitlinegroup.SplitLineGroup{}, err
		}
	}

	return splitlinegroup.SplitLineGroup{
		NamespacedID: models.NamespacedID{
			Namespace: dbSplitLineGroup.Namespace,
			ID:        dbSplitLineGroup.ID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: dbSplitLineGroup.CreatedAt,
			UpdatedAt: dbSplitLineGroup.UpdatedAt,
			DeletedAt: dbSplitLineGroup.DeletedAt,
		},
		SplitLineGroupMutableFields: splitlinegroup.SplitLineGroupMutableFields{
			Name:        dbSplitLineGroup.Name,
			Description: dbSplitLineGroup.Description,
			Metadata:    dbSplitLineGroup.Metadata,

			ServicePeriod: timeutil.ClosedPeriod{
				From: dbSplitLineGroup.ServicePeriodStart.UTC(),
				To:   dbSplitLineGroup.ServicePeriodEnd.UTC(),
			},

			RatecardDiscounts: lo.FromPtr(dbSplitLineGroup.RatecardDiscounts),
		},
		UniqueReferenceID: dbSplitLineGroup.UniqueReferenceID,

		Currency:     dbSplitLineGroup.Currency,
		Price:        dbSplitLineGroup.Price,
		FeatureKey:   dbSplitLineGroup.FeatureKey,
		Subscription: subscriptionRef,
	}, nil
}

func (a *adapter) mapSplitLineHierarchyFromDB(dbSplitLineGroup *db.BillingInvoiceSplitLineGroup) (splitlinegroup.SplitLineHierarchy, error) {
	empty := splitlinegroup.SplitLineHierarchy{}

	group, err := a.mapSplitLineGroupFromDB(dbSplitLineGroup)
	if err != nil {
		return empty, err
	}

	mappedLines, err := a.mapSplitLineHierarchyLinesFromDB(dbSplitLineGroup.Edges.BillingInvoiceLines)
	if err != nil {
		return empty, err
	}

	return splitlinegroup.SplitLineHierarchy{
		Group: group,
		Lines: mappedLines,
	}, nil
}

func (a *adapter) mapSplitLineHierarchyLinesFromDB(dbLines []*db.BillingInvoiceLine) ([]splitlinegroup.SplitLineHierarchyLine, error) {
	return slicesx.MapWithErr(dbLines, func(dbLine *db.BillingInvoiceLine) (splitlinegroup.SplitLineHierarchyLine, error) {
		dbInvoice := dbLine.Edges.BillingInvoice
		if dbInvoice == nil {
			return splitlinegroup.SplitLineHierarchyLine{}, fmt.Errorf("billing invoice must be expanded when mapping split line hierarchy lines [id=%s]", dbLine.ID)
		}

		invoiceType := billing.InvoiceTypeStandard
		if dbInvoice.Status == billing.StandardInvoiceStatusGathering {
			invoiceType = billing.InvoiceTypeGathering
		}

		if dbLine.Type != billing.InvoiceLineAdapterTypeUsageBased {
			if invoiceType == billing.InvoiceTypeGathering {
				return splitlinegroup.SplitLineHierarchyLine{}, fmt.Errorf("only usage based lines can be gathering invoice lines [line_id=%s]", dbLine.ID)
			}

			return splitlinegroup.SplitLineHierarchyLine{}, fmt.Errorf("only usage based lines can be top level lines [line_id=%s]", dbLine.ID)
		}

		var subscription *billing.SubscriptionReference
		if dbLine.SubscriptionID != nil && dbLine.SubscriptionPhaseID != nil && dbLine.SubscriptionItemID != nil {
			subscription = &billing.SubscriptionReference{
				SubscriptionID: *dbLine.SubscriptionID,
				PhaseID:        *dbLine.SubscriptionPhaseID,
				ItemID:         *dbLine.SubscriptionItemID,
				BillingPeriod: timeutil.ClosedPeriod{
					From: lo.FromPtr(dbLine.SubscriptionBillingPeriodFrom).UTC(),
					To:   lo.FromPtr(dbLine.SubscriptionBillingPeriodTo).UTC(),
				},
			}
		}

		return splitlinegroup.SplitLineHierarchyLine{
			LineID: billing.LineID{
				Namespace: dbLine.Namespace,
				ID:        dbLine.ID,
			},
			Invoice: splitlinegroup.SplitLineHierarchyInvoice{
				InvoiceID: billing.InvoiceID{
					Namespace: dbInvoice.Namespace,
					ID:        dbInvoice.ID,
				},
				Type:      invoiceType,
				DeletedAt: convert.TimePtrIn(dbInvoice.DeletedAt, time.UTC),
				Status:    dbInvoice.Status,
			},
			DeletedAt: convert.TimePtrIn(dbLine.DeletedAt, time.UTC),
			ServicePeriod: timeutil.ClosedPeriod{
				From: dbLine.PeriodStart.UTC(),
				To:   dbLine.PeriodEnd.UTC(),
			},
			ManagedBy:    dbLine.ManagedBy,
			Annotations:  dbLine.Annotations,
			Subscription: subscription,
			InvoiceAt:    dbLine.InvoiceAt.UTC(),
			Amount:       dbLine.Amount,
			ChargesTotal: dbLine.ChargesTotal,
		}, nil
	})
}
