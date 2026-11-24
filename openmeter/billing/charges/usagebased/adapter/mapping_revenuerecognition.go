package adapter

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
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

	dbOverageCredits, err := dbRun.Edges.FiatOverageCreditAllocationsOrErr()
	if err != nil {
		return usagebased.RevenueRecognitionRun{}, fmt.Errorf("mapping fiat overage credit realizations for revenue recognition run [id=%s]: %w", dbRun.ID, err)
	}

	dbLines, err := dbRun.Edges.DetailedLinesOrErr()
	if err != nil {
		return usagebased.RevenueRecognitionRun{}, fmt.Errorf("mapping detailed lines for revenue recognition run [id=%s]: %w", dbRun.ID, err)
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
		MeteredQuantity:                       dbRun.MeteredQuantity,
		DeltaMeteredQuantity:                  dbRun.DeltaMeteredQuantity,
		Totals:                                totals.DeltaTotals{Totals: totals.FromDB(dbRun)},
		DetailedLinesIncludeCreditAllocations: dbRun.DetailedLinesIncludeCreditAllocations,
		InvoiceUsageTotal:                     dbRun.InvoiceUsageTotal,
		CreditsAllocated:                      fromDBChargeUsageBasedRevenueRecognitionCredits(dbCredits),
		FiatOverageCreditRealizations:         fromDBChargeUsageBasedRevenueRecognitionCredits(dbOverageCredits),
		DetailedLines:                         fromDBChargeUsageBasedRevenueRecognitionDetailedLines(dbLines),
	}, nil
}

type revenueRecognitionCreditGetter interface {
	entutils.TimeMixinGetter
	entutils.NamespaceMixinGetter
	entutils.IDMixinGetter
	entutils.AnnotationsMixinGetter

	GetAmount() alpacadecimal.Decimal
	GetServicePeriodFrom() time.Time
	GetServicePeriodTo() time.Time
	GetType() creditrealization.Type
	GetLedgerTransactionGroupID() *string
}

// fromDBChargeUsageBasedRevenueRecognitionCredits also maps the overage
// credits table, which has the same columns. The realizations come back in ID
// order, since the tables have no sort hint.
func fromDBChargeUsageBasedRevenueRecognitionCredits[T revenueRecognitionCreditGetter](dbEntities []T) creditrealization.Realizations {
	realizations := lo.Map(dbEntities, func(dbEntity T, _ int) creditrealization.Realization {
		return creditrealization.Realization{
			NamespacedModel: models.NamespacedModel{
				Namespace: dbEntity.GetNamespace(),
			},
			ManagedModel: entutils.MapTimeMixinFromDB(dbEntity),
			CreateInput: creditrealization.CreateInput{
				ID:          dbEntity.GetID(),
				Annotations: dbEntity.GetAnnotations(),
				ServicePeriod: timeutil.ClosedPeriod{
					From: dbEntity.GetServicePeriodFrom().In(time.UTC),
					To:   dbEntity.GetServicePeriodTo().In(time.UTC),
				},
				Amount: dbEntity.GetAmount(),
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: lo.FromPtr(dbEntity.GetLedgerTransactionGroupID()),
				},
				Type: dbEntity.GetType(),
			},
		}
	})

	slices.SortFunc(realizations, func(a, b creditrealization.Realization) int {
		return strings.Compare(a.ID, b.ID)
	})

	return realizations
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
