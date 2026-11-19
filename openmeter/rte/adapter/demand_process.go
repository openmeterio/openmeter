package adapter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	entitlementdb "github.com/openmeterio/openmeter/openmeter/ent/db/entitlement"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	demanddb "github.com/openmeterio/openmeter/openmeter/ent/db/rtedemand"
	lockdb "github.com/openmeterio/openmeter/openmeter/ent/db/rtedemandlock"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (r *demandRepository) Process(ctx context.Context, input rte.ProcessDemandInput, calculate func(context.Context, rte.CalculationTarget) error) (rte.DemandWorkResult, error) {
	if err := input.Validate(); err != nil {
		return rte.DemandWorkResult{}, err
	}

	if calculate == nil {
		return rte.DemandWorkResult{}, errors.New("calculate callback is required")
	}

	_, transactionErr := transaction.GetDriverFromContext(ctx)
	ownsTransaction := transactionErr != nil

	var retry *demandRetry
	var timing *rte.DemandAttemptTiming
	result, processErr := transaction.Run(ctx, r, func(ctx context.Context) (rte.DemandWorkResult, error) {
		return entutils.TransactingRepo(ctx, r.adapter, func(ctx context.Context, a *adapter) (rte.DemandWorkResult, error) {
			row, err := a.claimDemand(ctx, input.Target)
			if err != nil || row == nil {
				return rte.DemandWorkResult{}, err
			}

			now := clock.Now()
			if input.Target == nil {
				timing = &rte.DemandAttemptTiming{
					Kind:      rte.CalculationKind(row.Kind),
					QueueWait: max(now.Sub(row.NextAttemptAt), 0),
					StartedAt: time.Now(),
				}
			}

			pending, result, err := a.pendingCalculation(ctx, row, input.Target == nil)
			if err != nil || pending == nil {
				return result, err
			}

			// A savepoint lets retry bookkeeping commit after calculator/store/outbox
			// failure, without retaining any writes from the failed attempt.
			calculationErr := transaction.RunWithNoValue(ctx, r, func(ctx context.Context) error {
				attemptCtx, cancel := context.WithTimeout(ctx, input.CalculationTimeout)
				defer cancel()

				target := rte.CalculationTarget{Kind: rte.CalculationKind(pending.row.Kind), Key: pending.row.Key}
				if err := calculate(attemptCtx, target); err != nil {
					return err
				}

				return attemptCtx.Err()
			})
			if calculationErr != nil {
				retry = pending.retry(input, calculationErr)
				err := a.db.RTEDemand.UpdateOneID(pending.row.ID).
					SetAttempts(pending.row.Attempts + 1).SetNextAttemptAt(retry.at).Exec(ctx)
				if err != nil {
					return rte.DemandWorkResult{}, err
				}

				return rte.DemandWorkResult{Processed: true, CalculationError: calculationErr}, nil
			}

			if err := a.completeDemand(ctx, pending, now); err != nil {
				return rte.DemandWorkResult{}, err
			}

			return rte.DemandWorkResult{Processed: true}, nil
		})
	})

	result.Timing = timing
	if processErr != nil && ownsTransaction && retry != nil && ctx.Err() == nil {
		if err := r.recoverRetry(ctx, retry); err != nil {
			return result, errors.Join(processErr, fmt.Errorf("recover demand retry: %w", err))
		}

		return rte.DemandWorkResult{Processed: true, Timing: timing, CalculationError: errors.Join(retry.cause, processErr)}, nil
	}

	return result, processErr
}

func (a *adapter) demandSourceDeletionAt(ctx context.Context, row *db.RTEDemand) (*time.Time, error) {
	if row.Kind != string(rte.CalculationKindMeteredEntitlement) {
		return nil, nil
	}

	source, err := a.db.Entitlement.Query().Where(entitlementdb.NamespaceEQ(row.Namespace), entitlementdb.IDEQ(row.OwnerID)).
		Select(entitlementdb.FieldDeletedAt).Only(ctx)
	if db.IsNotFound(err) {
		// Only a confirmed deletion retires demand. Let the calculator handle
		// an unknown entitlement with its usual lookup and retry semantics.
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read metered entitlement deletion time: %w", err)
	}

	return source.DeletedAt, nil
}

func demandDue(now time.Time) predicate.RTEDemand {
	return demanddb.And(demanddb.DeletedAtIsNil(), demanddb.Or(func(s *sql.Selector) {
		s.Where(sql.ColumnsGT(s.C(demanddb.FieldDirtyVersion), s.C(demanddb.FieldCompletedVersion)))
	}, demanddb.SourceDeleteAtLTE(now)))
}

// Background workers skip busy calculation locks. A pull for a particular key
// waits for its owner, then checks whether that calculation covered its demand.
func (a *adapter) claimDemand(ctx context.Context, target *rte.CalculationTarget) (*db.RTEDemand, error) {
	now := clock.Now()
	query := a.db.RTEDemand.Query().Where(demandDue(now))
	query.Where(func(s *sql.Selector) {
		lock := sql.Dialect(dialect.Postgres).Table(lockdb.Table).As("calc_lock")
		s.Join(lock).On(s.C(demanddb.FieldID), lock.C(lockdb.FieldID))
	})

	opts := []sql.LockOption{sql.WithLockTables("calc_lock")}
	if target == nil {
		// Match the partial work index even when PostgreSQL uses a generic plan.
		query.Where(func(s *sql.Selector) { s.Where(sql.IsTrue(s.C(demanddb.FieldPushEnabled))) }, demanddb.NextAttemptAtLTE(now)).
			Order(demanddb.ByNextAttemptAt(), demanddb.ByID())
		opts = append(opts, sql.WithLockAction(sql.SkipLocked))
	} else {
		_, condition, err := demandIdentity(*target)
		if err != nil {
			return nil, err
		}

		query.Where(condition)
	}

	id, err := query.ForUpdate(opts...).FirstID(ctx)
	if db.IsNotFound(err) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	// READ COMMITTED gives this statement a fresh snapshot after waiting
	// for the calculation lock. Invalidation can still update the demand row.
	return a.dueDemand(ctx, id, target == nil)
}

func (a *adapter) dueDemand(ctx context.Context, id int, background bool) (*db.RTEDemand, error) {
	now := clock.Now()
	fresh := a.db.RTEDemand.Query().Where(demanddb.IDEQ(id), demandDue(now))
	if background {
		fresh.Where(demanddb.PushEnabledEQ(true), demanddb.NextAttemptAtLTE(now))
	}

	row, err := fresh.Only(ctx)
	if db.IsNotFound(err) {
		return nil, nil
	}

	return row, err
}

type pendingDemand struct {
	row             *db.RTEDemand
	sourceDeletesAt *time.Time
}

// A wake-up can mean a dirty balance or a scheduled source deletion. Only a
// dirty balance with a source that still exists needs a calculation.
func (a *adapter) pendingCalculation(ctx context.Context, row *db.RTEDemand, background bool) (*pendingDemand, rte.DemandWorkResult, error) {
	for row != nil {
		deletedAt, err := a.demandSourceDeletionAt(ctx, row)
		if err != nil {
			return nil, rte.DemandWorkResult{}, err
		}

		if deletedAt == nil || deletedAt.After(clock.Now()) {
			if row.DirtyVersion != row.CompletedVersion {
				return &pendingDemand{row: row, sourceDeletesAt: deletedAt}, rte.DemandWorkResult{}, nil
			}

			// The deletion moved or was canceled, or an older worker set a periodic deadline.
			update := a.db.RTEDemand.Update().Where(demanddb.IDEQ(row.ID), demanddb.DirtyVersionEQ(row.DirtyVersion)).
				SetOrClearSourceDeleteAt(deletedAt)
			if deletedAt != nil {
				update.SetNextAttemptAt(*deletedAt)
			}

			return nil, rte.DemandWorkResult{}, update.Exec(ctx)
		}

		retired, err := a.retireDemand(ctx, row)
		if err != nil {
			return nil, rte.DemandWorkResult{}, err
		}

		if retired {
			result := rte.DemandWorkResult{Processed: true, Retired: true}
			if !background {
				result.CalculationError = &entitlement.NotFoundError{EntitlementID: models.NamespacedID{Namespace: row.Namespace, ID: row.OwnerID}}
			}

			return nil, result, nil
		}

		// An invalidation won the race. It may have canceled the deletion.
		row, err = a.dueDemand(ctx, row.ID, background)
		if err != nil {
			return nil, rte.DemandWorkResult{}, err
		}
	}

	return nil, rte.DemandWorkResult{}, nil
}

func (a *adapter) retireDemand(ctx context.Context, row *db.RTEDemand) (bool, error) {
	// Keep the last value readable and the deleted source dirty for explicit pulls.
	update := a.db.RTEDemand.Update().Where(demanddb.IDEQ(row.ID), demanddb.DirtyVersionEQ(row.DirtyVersion)).
		SetPushEnabled(false).ClearPullUntil().ClearSourceDeleteAt().SetAttempts(0)
	if row.DirtyVersion == row.CompletedVersion {
		update.AddDirtyVersion(1)
	}

	count, err := update.Save(ctx)

	return count != 0, err
}

func (a *adapter) completeDemand(ctx context.Context, pending *pendingDemand, calculatedAt time.Time) error {
	row := pending.row
	nextAttempt := clock.Now()
	if pending.sourceDeletesAt != nil {
		current, err := a.db.RTEDemand.Query().Where(demanddb.IDEQ(row.ID)).ForUpdate().Only(ctx)
		if err != nil {
			return err
		}

		// Waiting for a future deletion is safe only when there is no newer work.
		if current.DirtyVersion == row.DirtyVersion {
			nextAttempt = *pending.sourceDeletesAt
		}
	}

	// Acknowledge only the claimed version. A concurrent invalidation gets another turn.
	return a.db.RTEDemand.UpdateOneID(row.ID).SetCompletedVersion(row.DirtyVersion).SetAttempts(0).
		SetLastCalculatedAt(calculatedAt).SetNextAttemptAt(nextAttempt).SetOrClearSourceDeleteAt(pending.sourceDeletesAt).Exec(ctx)
}

type demandRetry struct {
	row   *db.RTEDemand
	at    time.Time
	cause error
}

func (p pendingDemand) retry(input rte.ProcessDemandInput, cause error) *demandRetry {
	delay := input.RetryMinDelay
	for attempt := 0; attempt < p.row.Attempts && delay < input.RetryMaxDelay; attempt++ {
		if delay > input.RetryMaxDelay/2 {
			delay = input.RetryMaxDelay
			break
		}

		delay *= 2
	}

	at := clock.Now().Add(delay)
	if p.sourceDeletesAt != nil && p.sourceDeletesAt.Before(at) {
		at = *p.sourceDeletesAt
	}

	return &demandRetry{row: p.row, at: at, cause: cause}
}

// A timed-out database read can close the connection before the retry is saved.
// Once our transaction has ended, another connection may defer the same attempt
// if it is still active and nobody else has advanced it.
func (r *demandRepository) recoverRetry(ctx context.Context, retry *demandRetry) error {
	row := retry.row

	return transaction.RunWithNoValue(ctx, r, func(ctx context.Context) error {
		return entutils.TransactingRepoWithNoValue(ctx, r.adapter, func(ctx context.Context, a *adapter) error {
			return a.db.RTEDemand.Update().Where(
				demanddb.IDEQ(row.ID), activeDemand(clock.Now()),
				demanddb.DirtyVersionEQ(row.DirtyVersion),
				demanddb.CompletedVersionEQ(row.CompletedVersion),
				demanddb.AttemptsEQ(row.Attempts),
			).SetAttempts(row.Attempts + 1).SetNextAttemptAt(retry.at).Exec(ctx)
		})
	})
}
