package adapter

import (
	"context"
	stdsql "database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	entitlementdb "github.com/openmeterio/openmeter/openmeter/ent/db/entitlement"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	demanddb "github.com/openmeterio/openmeter/openmeter/ent/db/rtedemand"
	lockdb "github.com/openmeterio/openmeter/openmeter/ent/db/rtedemandlock"
	receiptdb "github.com/openmeterio/openmeter/openmeter/ent/db/rteinvalidation"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type demandRepository struct{ *adapter }

func NewDemandRepository(config Config) (rte.DemandRepository, error) {
	a, err := newAdapter(config)
	if err != nil {
		return nil, err
	}

	return &demandRepository{adapter: a}, nil
}

func demandIdentity(target rte.CalculationTarget) (rte.TargetIdentity, predicate.RTEDemand, error) {
	identity, err := target.Identity()
	if err != nil {
		return rte.TargetIdentity{}, nil, err
	}

	return identity, demanddb.And(
		demanddb.NamespaceEQ(identity.Namespace),
		demanddb.KindEQ(string(target.Kind)),
		demanddb.KeyHashEQ(identity.KeyHash),
		demanddb.DeletedAtIsNil(),
	), nil
}

func activeDemand(at time.Time) predicate.RTEDemand {
	return demanddb.And(demanddb.DeletedAtIsNil(), demanddb.Or(demanddb.PushEnabledEQ(true), demanddb.PullUntilGT(at)))
}

func (r *demandRepository) Activate(ctx context.Context, target rte.CalculationTarget) error {
	_, err := r.ensure(ctx, ensureDemandInput{Target: target, PushEnabled: true})

	return err
}

func (r *demandRepository) EnsurePull(ctx context.Context, input rte.EnsureFreshInput) (bool, error) {
	if err := input.Validate(); err != nil {
		return false, err
	}

	return r.ensure(ctx, ensureDemandInput{Target: input.Target, PullTTL: input.PullTTL, MaxAge: input.MaxAge})
}

type ensureDemandInput struct {
	Target      rte.CalculationTarget
	PushEnabled bool
	PullTTL     time.Duration
	MaxAge      time.Duration
}

// The driver encodes query arguments after acquiring a connection. A busy
// pool can hold a renewal until the previously committed lease has expired.
type pullRenewalTime struct {
	now    func() time.Time
	offset time.Duration
}

func (t pullRenewalTime) Value() (driver.Value, error) {
	return t.now().Add(t.offset), nil
}

func (i ensureDemandInput) Validate() error {
	var errs []error
	errs = append(errs, i.Target.Validate())
	if i.PullTTL < 0 || i.MaxAge < 0 {
		errs = append(errs, errors.New("demand durations must not be negative"))
	}

	if !i.PushEnabled && i.PullTTL == 0 {
		errs = append(errs, errors.New("push or pull demand is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (r *demandRepository) ensure(ctx context.Context, input ensureDemandInput) (bool, error) {
	if err := input.Validate(); err != nil {
		return false, err
	}

	identity, condition, err := demandIdentity(input.Target)
	if err != nil {
		return false, err
	}

	key, err := input.Target.DecodeKey()
	if err != nil {
		return false, err
	}

	encodedKey, err := json.Marshal(key)
	if err != nil {
		return false, fmt.Errorf("encode demand key: %w", err)
	}

	_, txErr := transaction.GetDriverFromContext(ctx)
	callerOwnsTransaction := txErr == nil
	if input.PullTTL > 0 && input.MaxAge == 0 && !callerOwnsTransaction {
		clean, err := entutils.TransactingRepo(ctx, r.adapter, func(ctx context.Context, a *adapter) (bool, error) {
			// If another transaction holds the row, the tracking period might expire before
			// it commits. Let registration recheck the lease after the lock is ours.
			now := sync.OnceValue(clock.Now)
			query, args := (renewPullQuery{
				identity: identity,
				kind:     input.Target.Kind,
				key:      string(encodedKey),
				at:       pullRenewalTime{now: now},
				until:    pullRenewalTime{now: now, offset: input.PullTTL},
			}).SQL()
			rows, err := a.db.QueryContext(ctx, query, args...)
			if err != nil {
				return false, err
			}
			defer rows.Close()

			return sql.ScanBool(rows)
		})
		switch {
		case err == nil:
			return clean, nil
		case !errors.Is(err, stdsql.ErrNoRows):
			return false, fmt.Errorf("renew pull demand: %w", err)
		}
	}

	return transaction.Run(ctx, r, func(ctx context.Context) (bool, error) {
		return entutils.TransactingRepo(ctx, r.adapter, func(ctx context.Context, a *adapter) (bool, error) {
			now := clock.Now()
			pg := sql.Dialect(dialect.Postgres)
			create := func() *db.RTEDemandCreate {
				return a.db.RTEDemand.Create().SetNamespace(identity.Namespace).SetKind(string(input.Target.Kind)).
					SetOwnerID(identity.OwnerID).SetKeyHash(identity.KeyHash).SetKey(encodedKey).
					SetDirtyVersion(0).SetNextAttemptAt(now)
			}

			if callerOwnsTransaction {
				// Registration and targeted calculation can share an outer transaction.
				// Take locks in the worker's order before renewal writes the demand row.
				// DO NOTHING also handles two callers registering a new key together.
				err := create().OnConflict(sql.DoNothing()).Exec(ctx)
				if err != nil && !errors.Is(err, stdsql.ErrNoRows) {
					return false, err
				}

				if _, err := a.db.RTEDemandLock.Query().Where(lockdb.HasDemandWith(condition)).ForUpdate().OnlyID(ctx); err != nil {
					return false, err
				}
			}

			err := create().
				OnConflict(
					sql.ConflictColumns(demanddb.FieldNamespace, demanddb.FieldKind, demanddb.FieldKeyHash),
					sql.ConflictWhere(sql.IsNull(demanddb.FieldDeletedAt)),
					sql.UpdateWhere(sql.ColumnsNEQ(pg.Table(demanddb.Table).C(demanddb.FieldKey), pg.Table("excluded").C(demanddb.FieldKey))),
				).
				Update(func(u *db.RTEDemandUpsert) {
					// Earlier registrations stored the caller's raw JSON. Repair its field
					// names and recalculate once for invalidations routing may have missed.
					u.SetExcluded(demanddb.FieldKey)
					markDemandDirty(u.UpdateBuilder, now)
				}).
				Exec(ctx)
			if err != nil && !errors.Is(err, stdsql.ErrNoRows) {
				return false, fmt.Errorf("create demand: %w", err)
			}

			row, err := a.db.RTEDemand.Query().Where(condition).ForUpdate().Only(ctx)
			if err != nil {
				return false, err
			}

			now = clock.Now()
			wasActive := row.PushEnabled || (row.PullUntil != nil && row.PullUntil.After(now))
			update := a.db.RTEDemand.UpdateOneID(row.ID)
			if input.PushEnabled {
				update.SetPushEnabled(true)
				if !row.PushEnabled && row.Attempts == 0 && row.DirtyVersion > row.CompletedVersion {
					update.SetNextAttemptAt(now)
				}
			}

			if input.PullTTL > 0 {
				until := now.Add(input.PullTTL)
				if row.PullUntil == nil || until.After(*row.PullUntil) {
					update.SetPullUntil(until)
				}
			}

			stale := input.MaxAge > 0 && (row.LastCalculatedAt == nil || !row.LastCalculatedAt.After(now.Add(-input.MaxAge)))
			clean := row.CompletedVersion == row.DirtyVersion
			// Reopening tracking must advance even an already dirty key. Its
			// in-flight calculation may predate changes missed during the gap.
			if !wasActive || (stale && clean) {
				update.AddDirtyVersion(1)
				if row.Attempts == 0 {
					update.SetNextAttemptAt(now)
				}
				clean = false
			}

			if err := update.Exec(ctx); err != nil {
				return false, err
			}

			return clean && (row.SourceDeleteAt == nil || row.SourceDeleteAt.After(now)), nil
		})
	})
}

func (r *demandRepository) Deactivate(ctx context.Context, target rte.CalculationTarget) error {
	_, condition, err := demandIdentity(target)
	if err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, r.adapter, func(ctx context.Context, a *adapter) error {
		return a.db.RTEDemand.Update().Where(condition).SetPushEnabled(false).ClearPullUntil().ClearSourceDeleteAt().Exec(ctx)
	})
}

func (r *demandRepository) Invalidate(ctx context.Context, input rte.InvalidateInput) (rte.InvalidationResult, error) {
	if err := input.Validate(); err != nil {
		return rte.InvalidationResult{}, err
	}

	return transaction.Run(ctx, r, func(ctx context.Context) (rte.InvalidationResult, error) {
		return entutils.TransactingRepo(ctx, r.adapter, func(ctx context.Context, a *adapter) (rte.InvalidationResult, error) {
			err := a.db.RTEInvalidation.Create().SetSource(input.Source).SetEventID(input.EventID).
				OnConflict(sql.ConflictColumns(receiptdb.FieldSource, receiptdb.FieldEventID)).DoNothing().Exec(ctx)
			if errors.Is(err, stdsql.ErrNoRows) {
				return rte.InvalidationResult{Duplicate: true}, nil
			}

			if err != nil {
				return rte.InvalidationResult{}, fmt.Errorf("remember invalidation: %w", err)
			}

			if len(input.Selectors) == 0 {
				return rte.InvalidationResult{}, nil
			}

			query, args, err := (invalidateDemandQuery{selectors: input.Selectors, at: clock.Now()}).SQL()
			if err != nil {
				return rte.InvalidationResult{}, err
			}

			result, err := a.db.ExecContext(ctx, query, args...)
			if err != nil {
				return rte.InvalidationResult{}, err
			}

			matched, err := result.RowsAffected()

			return rte.InvalidationResult{Matched: int(matched)}, err
		})
	})
}

// A deletion notification can arrive before its effective time. Read the source
// again at processing time in case the writer moved or canceled the deadline.
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

func (r *demandRepository) Process(ctx context.Context, input rte.ProcessDemandInput, calculate func(context.Context, rte.CalculationTarget) error) (rte.DemandWorkResult, error) {
	if err := input.Validate(); err != nil {
		return rte.DemandWorkResult{}, err
	}

	if calculate == nil {
		return rte.DemandWorkResult{}, errors.New("calculate callback is required")
	}

	_, transactionErr := transaction.GetDriverFromContext(ctx)
	ownsTransaction := transactionErr != nil
	var failedAttempt *db.RTEDemand
	var retryAt time.Time
	var calculationFailure error
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

			deletedAt, err := a.demandSourceDeletionAt(ctx, row)
			if err != nil {
				return rte.DemandWorkResult{}, err
			}

			for deletedAt != nil && !deletedAt.After(clock.Now()) {
				// Keep the last successful value readable. Leave this version
				// outstanding so an explicit pull still sees a deleted source.
				update := a.db.RTEDemand.Update().Where(demanddb.IDEQ(row.ID), demanddb.DirtyVersionEQ(row.DirtyVersion)).
					SetPushEnabled(false).ClearPullUntil().ClearSourceDeleteAt().SetAttempts(0)
				if row.DirtyVersion == row.CompletedVersion {
					update.AddDirtyVersion(1)
				}

				retired, err := update.Save(ctx)
				if err != nil {
					return rte.DemandWorkResult{}, err
				}

				if retired == 0 {
					// A source change won the race. It may have canceled the deletion.
					row, err = a.dueDemand(ctx, row.ID, input.Target == nil)
					if err != nil || row == nil {
						return rte.DemandWorkResult{}, err
					}
					deletedAt, err = a.demandSourceDeletionAt(ctx, row)
					if err != nil {
						return rte.DemandWorkResult{}, err
					}
					continue
				}

				result := rte.DemandWorkResult{Processed: true, Retired: true}
				if input.Target != nil {
					result.CalculationError = &entitlement.NotFoundError{EntitlementID: models.NamespacedID{Namespace: row.Namespace, ID: row.OwnerID}}
				}

				return result, nil
			}

			if row.DirtyVersion == row.CompletedVersion {
				// The deletion was moved or canceled, or an older worker set a periodic
				// deadline. Update the wake-up time without running a clean calculation.
				update := a.db.RTEDemand.Update().Where(demanddb.IDEQ(row.ID), demanddb.DirtyVersionEQ(row.DirtyVersion)).
					SetOrClearSourceDeleteAt(deletedAt)
				if deletedAt != nil {
					update.SetNextAttemptAt(*deletedAt)
				}

				return rte.DemandWorkResult{}, update.Exec(ctx)
			}

			target := rte.CalculationTarget{Kind: rte.CalculationKind(row.Kind), Key: row.Key}
			// A savepoint lets retry bookkeeping commit after calculator/store/outbox
			// failure, without retaining any writes from the failed attempt.
			calculationErr := transaction.RunWithNoValue(ctx, r, func(ctx context.Context) error {
				attemptCtx, cancel := context.WithTimeout(ctx, input.CalculationTimeout)
				defer cancel()
				if err := calculate(attemptCtx, target); err != nil {
					return err
				}

				return attemptCtx.Err()
			})
			update := a.db.RTEDemand.UpdateOneID(row.ID)
			if calculationErr != nil {
				delay := input.RetryMinDelay
				for attempt := 0; attempt < row.Attempts && delay < input.RetryMaxDelay; attempt++ {
					if delay > input.RetryMaxDelay/2 {
						delay = input.RetryMaxDelay
						break
					}

					delay *= 2
				}

				failedAttempt, calculationFailure, retryAt = row, calculationErr, clock.Now().Add(delay)
				if deletedAt != nil && deletedAt.Before(retryAt) {
					retryAt = *deletedAt
				}

				update.SetAttempts(row.Attempts + 1).SetNextAttemptAt(retryAt)
			} else {
				// Acknowledge only the claimed version. Concurrent invalidations
				// leave this key dirty, eligible for another turn after completion.
				nextAttempt := clock.Now()
				if deletedAt != nil {
					current, err := a.db.RTEDemand.Query().Where(demanddb.IDEQ(row.ID)).ForUpdate().Only(ctx)
					if err != nil {
						return rte.DemandWorkResult{}, err
					}

					// Waiting for a future deletion is safe only when there is no newer work.
					if current.DirtyVersion == row.DirtyVersion {
						nextAttempt = *deletedAt
					}
				}

				update.SetCompletedVersion(row.DirtyVersion).SetAttempts(0).
					SetLastCalculatedAt(now).SetNextAttemptAt(nextAttempt).SetOrClearSourceDeleteAt(deletedAt)
			}

			if err := update.Exec(ctx); err != nil {
				return rte.DemandWorkResult{}, err
			}

			return rte.DemandWorkResult{Processed: true, CalculationError: calculationErr}, nil
		})
	})
	result.Timing = timing
	if processErr != nil && ownsTransaction && failedAttempt != nil && ctx.Err() == nil {
		// pgx can close the connection when a database read times out, making
		// rollback-to-savepoint and the retry update impossible. The attempt
		// transaction is finished now; defer its unchanged version using a new
		// connection. Never commit this independently of a caller-owned txn.
		recoveryErr := transaction.RunWithNoValue(ctx, r, func(ctx context.Context) error {
			return entutils.TransactingRepoWithNoValue(ctx, r.adapter, func(ctx context.Context, a *adapter) error {
				return a.db.RTEDemand.Update().Where(
					demanddb.IDEQ(failedAttempt.ID), activeDemand(clock.Now()),
					demanddb.DirtyVersionEQ(failedAttempt.DirtyVersion),
					demanddb.CompletedVersionEQ(failedAttempt.CompletedVersion),
					demanddb.AttemptsEQ(failedAttempt.Attempts),
				).SetAttempts(failedAttempt.Attempts + 1).SetNextAttemptAt(retryAt).Exec(ctx)
			})
		})
		if recoveryErr != nil {
			return result, errors.Join(processErr, fmt.Errorf("recover demand retry: %w", recoveryErr))
		}

		return rte.DemandWorkResult{Processed: true, Timing: timing, CalculationError: errors.Join(calculationFailure, processErr)}, nil
	}

	return result, processErr
}

func (r *demandRepository) PruneInvalidations(ctx context.Context, input rte.PruneInvalidationsInput) (int, error) {
	if err := input.Validate(); err != nil {
		return 0, err
	}

	return transaction.Run(ctx, r, func(ctx context.Context) (int, error) {
		return entutils.TransactingRepo(ctx, r.adapter, func(ctx context.Context, a *adapter) (int, error) {
			ids, err := a.db.RTEInvalidation.Query().Where(receiptdb.CreatedAtLT(input.Before)).
				Order(receiptdb.ByCreatedAt(), receiptdb.ByID()).Limit(input.Limit).ForUpdate(sql.WithLockAction(sql.SkipLocked)).IDs(ctx)
			if err != nil {
				return 0, err
			}

			if len(ids) == 0 {
				return 0, nil
			}

			return a.db.RTEInvalidation.Delete().Where(receiptdb.IDIn(ids...)).Exec(ctx)
		})
	})
}
