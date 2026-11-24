package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeusagebasedrevenuerecognitions "github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedrevenuerecognitions"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ usagebased.RevenueRecognitionRunAdapter = (*adapter)(nil)

func (a *adapter) CreateRevenueRecognitionRun(ctx context.Context, input usagebased.CreateRevenueRecognitionRunInput) (usagebased.RevenueRecognitionRun, error) {
	if err := input.Validate(); err != nil {
		return usagebased.RevenueRecognitionRun{}, err
	}

	// A run and its child rows are written together or not at all, so start a
	// transaction when the caller has none.
	return transaction.Run(ctx, a, func(ctx context.Context) (usagebased.RevenueRecognitionRun, error) {
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

			// An empty period falls back to the column default, which is daily.
			if !input.Period.IsZero() {
				create = create.SetPeriod(input.Period.ISOString())
			}

			if input.PriorRunID != nil {
				create = create.SetPriorRunID(input.PriorRunID.ID)
			}

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

		return fromDBChargeUsageBasedRevenueRecognitions(dbRun)
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
