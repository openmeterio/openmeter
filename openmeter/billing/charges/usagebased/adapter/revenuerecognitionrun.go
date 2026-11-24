package adapter

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeusagebasedrevenuerecognitions "github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedrevenuerecognitions"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

var _ usagebased.RevenueRecognitionRunAdapter = (*adapter)(nil)

func (a *adapter) CreateRevenueRecognitionRun(ctx context.Context, input usagebased.CreateRevenueRecognitionRunInput) (usagebased.RevenueRecognitionRun, error) {
	if err := input.Validate(); err != nil {
		return usagebased.RevenueRecognitionRun{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (usagebased.RevenueRecognitionRun, error) {
		create := tx.db.ChargeUsageBasedRevenueRecognitions.Create().
			SetNamespace(input.ChargeID.Namespace).
			SetChargeID(input.ChargeID.ID).
			SetFeatureID(input.FeatureID).
			SetStoredAtLt(meta.NormalizeTimestamp(input.StoredAtLT)).
			SetServicePeriodTo(meta.NormalizeTimestamp(input.ServicePeriodTo)).
			SetMeteredQuantity(input.MeteredQuantity).
			SetDeltaMeteredQuantity(input.DeltaMeteredQuantity).
			SetDetailedLinesIncludeCreditAllocations(input.DetailedLinesIncludeCreditAllocations).
			SetNillableInvoiceUsageTotal(input.InvoiceUsageTotal)

		create = totals.Set(create, input.Totals.Totals)

		dbRun, err := create.Save(ctx)
		if err != nil {
			return usagebased.RevenueRecognitionRun{}, fmt.Errorf("creating revenue recognition run [charge_id=%s]: %w", input.ChargeID.ID, err)
		}

		runID := usagebased.RevenueRecognitionRunID{
			Namespace: dbRun.Namespace,
			ID:        dbRun.ID,
		}

		if len(input.CreditsAllocated) > 0 {
			creates := lo.Map(input.CreditsAllocated, func(realization creditrealization.CreateInput, _ int) *entdb.ChargeUsageBasedRevenueRecognitionCreditsCreate {
				return setRevenueRecognitionCredit(tx.db.ChargeUsageBasedRevenueRecognitionCredits.Create(), runID, realization)
			})

			if err := tx.db.ChargeUsageBasedRevenueRecognitionCredits.CreateBulk(creates...).Exec(ctx); err != nil {
				return usagebased.RevenueRecognitionRun{}, fmt.Errorf("creating credit allocations for revenue recognition run [id=%s]: %w", runID.ID, err)
			}
		}

		if len(input.FiatOverageCreditRealizations) > 0 {
			creates := lo.Map(input.FiatOverageCreditRealizations, func(realization creditrealization.CreateInput, _ int) *entdb.ChargeUsageBasedRevenueRecognitionOverageCreditsCreate {
				return setRevenueRecognitionCredit(tx.db.ChargeUsageBasedRevenueRecognitionOverageCredits.Create(), runID, realization)
			})

			if err := tx.db.ChargeUsageBasedRevenueRecognitionOverageCredits.CreateBulk(creates...).Exec(ctx); err != nil {
				return usagebased.RevenueRecognitionRun{}, fmt.Errorf("creating fiat overage credit realizations for revenue recognition run [id=%s]: %w", runID.ID, err)
			}
		}

		if len(input.DetailedLines) > 0 {
			creates := lo.Map(input.DetailedLines, func(line usagebased.RevenueRecognitionDetailedLine, _ int) *entdb.ChargeUsageBasedRevenueRecognitionDetailedLineCreate {
				lineCreate := tx.db.ChargeUsageBasedRevenueRecognitionDetailedLine.Create().
					SetNamespace(runID.Namespace).
					SetRunID(runID.ID).
					SetName(line.Name).
					SetNillableDescription(line.Description).
					SetAnnotations(line.Annotations).
					SetMetadata(line.Metadata).
					SetCurrency(line.Currency).
					SetServicePeriodStart(line.ServicePeriod.From.In(time.UTC)).
					SetServicePeriodEnd(line.ServicePeriod.To.In(time.UTC)).
					SetChildUniqueReferenceID(line.ChildUniqueReferenceID).
					SetNillableIndex(line.Index).
					SetQuantity(line.Quantity).
					SetPerUnitAmount(line.PerUnitAmount).
					SetAmount(line.Amount).
					SetAmountDiscounts(line.AmountDiscounts)

				if line.ID != "" {
					lineCreate = lineCreate.SetID(line.ID)
				}

				if len(line.CreditsApplied) > 0 {
					lineCreate = lineCreate.SetCreditsApplied(&line.CreditsApplied)
				}

				return lineCreate
			})

			if err := tx.db.ChargeUsageBasedRevenueRecognitionDetailedLine.CreateBulk(creates...).Exec(ctx); err != nil {
				return usagebased.RevenueRecognitionRun{}, fmt.Errorf("creating detailed lines for revenue recognition run [id=%s]: %w", runID.ID, err)
			}
		}

		return tx.GetRevenueRecognitionRunByID(ctx, runID)
	})
}

func (a *adapter) GetRevenueRecognitionRunByID(ctx context.Context, id usagebased.RevenueRecognitionRunID) (usagebased.RevenueRecognitionRun, error) {
	if err := id.Validate(); err != nil {
		return usagebased.RevenueRecognitionRun{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (usagebased.RevenueRecognitionRun, error) {
		dbRun, err := tx.db.ChargeUsageBasedRevenueRecognitions.Query().
			Where(dbchargeusagebasedrevenuerecognitions.Namespace(id.Namespace)).
			Where(dbchargeusagebasedrevenuerecognitions.ID(id.ID)).
			WithCreditAllocations().
			WithFiatOverageCreditAllocations().
			WithDetailedLines().
			First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return usagebased.RevenueRecognitionRun{}, models.NewGenericNotFoundError(fmt.Errorf("revenue recognition run [id=%s] not found", id.ID))
			}

			return usagebased.RevenueRecognitionRun{}, fmt.Errorf("querying revenue recognition run [id=%s]: %w", id.ID, err)
		}

		return fromDBRevenueRecognitionRun(dbRun)
	})
}

type revenueRecognitionCreditCreator[T any] interface {
	SetID(id string) T
	SetNamespace(namespace string) T
	SetRunID(runID string) T
	SetAnnotations(annotations models.Annotations) T
	SetAmount(amount alpacadecimal.Decimal) T
	SetServicePeriodFrom(servicePeriodFrom time.Time) T
	SetServicePeriodTo(servicePeriodTo time.Time) T
	SetType(t creditrealization.Type) T
	SetNillableLedgerTransactionGroupID(ledgerTransactionGroupID *string) T
}

func setRevenueRecognitionCredit[T revenueRecognitionCreditCreator[T]](create T, runID usagebased.RevenueRecognitionRunID, realization creditrealization.CreateInput) T {
	create = create.
		SetNamespace(runID.Namespace).
		SetRunID(runID.ID).
		SetAnnotations(realization.Annotations).
		SetAmount(realization.Amount).
		SetServicePeriodFrom(realization.ServicePeriod.From.In(time.UTC)).
		SetServicePeriodTo(realization.ServicePeriod.To.In(time.UTC)).
		SetType(realization.Type).
		SetNillableLedgerTransactionGroupID(realization.LedgerTransaction.GetIDOrNull())

	if realization.ID != "" {
		create = create.SetID(realization.ID)
	}

	return create
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

// fromDBRevenueRecognitionCredits returns the realizations in ID order, since
// the table has no sort hint.
func fromDBRevenueRecognitionCredits[T revenueRecognitionCreditGetter](dbEntities []T) creditrealization.Realizations {
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

func fromDBRevenueRecognitionRun(dbRun *entdb.ChargeUsageBasedRevenueRecognitions) (usagebased.RevenueRecognitionRun, error) {
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
		CreditsAllocated:                      fromDBRevenueRecognitionCredits(dbCredits),
		FiatOverageCreditRealizations:         fromDBRevenueRecognitionCredits(dbOverageCredits),
		DetailedLines:                         detailedLines,
	}, nil
}
