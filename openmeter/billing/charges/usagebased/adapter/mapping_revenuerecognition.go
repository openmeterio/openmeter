package adapter

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func fromDBChargeUsageBasedRevenueRecognitions(dbRun *entdb.ChargeUsageBasedRevenueRecognitions) (usagebased.RevenueRecognitionRun, error) {
	dbCredits, err := dbRun.Edges.CreditAllocationsOrErr()
	if err != nil {
		return usagebased.RevenueRecognitionRun{}, fmt.Errorf("mapping credit allocations for revenue recognition run [id=%s]: %w", dbRun.ID, err)
	}

	dbInvoicedUsages, err := dbRun.Edges.InvoicedUsagesOrErr()
	if err != nil {
		return usagebased.RevenueRecognitionRun{}, fmt.Errorf("mapping invoiced usages for revenue recognition run [id=%s]: %w", dbRun.ID, err)
	}

	dbLines, err := dbRun.Edges.DetailedLinesOrErr()
	if err != nil {
		return usagebased.RevenueRecognitionRun{}, fmt.Errorf("mapping detailed lines for revenue recognition run [id=%s]: %w", dbRun.ID, err)
	}

	var priorRevRecogRunID *usagebased.RevenueRecognitionRunID
	if dbRun.PriorRevRecogRunID != nil {
		if *dbRun.PriorRevRecogRunID == dbRun.ID {
			return usagebased.RevenueRecognitionRun{}, fmt.Errorf("revenue recognition run cannot reference itself as prior run [id=%s]", dbRun.ID)
		}

		priorRevRecogRunID = &usagebased.RevenueRecognitionRunID{
			Namespace: dbRun.Namespace,
			ID:        *dbRun.PriorRevRecogRunID,
		}
	}

	return usagebased.RevenueRecognitionRun{
		ID: usagebased.RevenueRecognitionRunID{
			Namespace: dbRun.Namespace,
			ID:        dbRun.ID,
		},
		ManagedModel: entutils.MapTimeMixinFromDB(dbRun),

		ChargeID:                              dbRun.ChargeID,
		FeatureID:                             dbRun.FeatureID,
		StoredAtLT:                            dbRun.StoredAtLt.UTC(),
		ServicePeriodTo:                       dbRun.ServicePeriodTo.UTC(),
		PriorRevRecogRunID:                    priorRevRecogRunID,
		MeteredQuantity:                       dbRun.MeteredQuantity,
		Totals:                                totals.DeltaTotals{Totals: totals.FromDB(dbRun)},
		DetailedLinesIncludeCreditAllocations: dbRun.DetailedLinesIncludeCreditAllocations,
		CreditsAllocated:                      creditrealization.FromDBMixinRealizations(dbCredits),
		InvoicedUsages:                        creditrealization.FromDBMixinRealizations(dbInvoicedUsages),
		DetailedLines:                         fromDBChargeUsageBasedRevenueRecognitionDetailedLines(dbLines),
	}, nil
}

func fromDBChargeUsageBasedRevenueRecognitionDetailedLines(dbLines []*entdb.ChargeUsageBasedRevenueRecognitionDetailedLine) []usagebased.RevenueRecognitionDetailedLine {
	detailedLines := lo.Map(dbLines, func(dbLine *entdb.ChargeUsageBasedRevenueRecognitionDetailedLine, _ int) usagebased.RevenueRecognitionDetailedLine {
		return usagebased.RevenueRecognitionDetailedLine{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{
					Namespace: dbLine.Namespace,
				},
				ManagedModel: entutils.MapTimeMixinFromDB(dbLine),
				ID:           dbLine.ID,
				Name:         dbLine.Name,
				Description:  dbLine.Description,
			},
			Annotations: dbLine.Annotations,
			Metadata:    dbLine.Metadata,
			Currency:    dbLine.Currency,
			ServicePeriod: timeutil.ClosedPeriod{
				From: dbLine.ServicePeriodStart.In(time.UTC),
				To:   dbLine.ServicePeriodEnd.In(time.UTC),
			},
			ChildUniqueReferenceID: dbLine.ChildUniqueReferenceID,
			Index:                  dbLine.Index,
			Quantity:               dbLine.Quantity,
			PerUnitAmount:          dbLine.PerUnitAmount,
			Amount:                 dbLine.Amount,
			AmountDiscounts:        dbLine.AmountDiscounts,
			CreditsApplied:         lo.FromPtr(dbLine.CreditsApplied),
		}
	})

	// Lines without an index come last; the ID keeps the order stable.
	slices.SortFunc(detailedLines, func(a, b usagebased.RevenueRecognitionDetailedLine) int {
		if c := cmp.Compare(lo.FromPtrOr(a.Index, math.MaxInt), lo.FromPtrOr(b.Index, math.MaxInt)); c != 0 {
			return c
		}

		return cmp.Compare(a.ID, b.ID)
	})

	return detailedLines
}
