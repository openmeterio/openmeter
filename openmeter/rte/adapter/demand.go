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
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	demanddb "github.com/openmeterio/openmeter/openmeter/ent/db/rtedemand"
	lockdb "github.com/openmeterio/openmeter/openmeter/ent/db/rtedemandlock"
	receiptdb "github.com/openmeterio/openmeter/openmeter/ent/db/rteinvalidation"
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

// A competing delivery may insert the same receipt after our UPDATE took its
// snapshot. Roll back our dirty-version increments if it wins that race.
var errConcurrentInvalidation = errors.New("concurrent invalidation already recorded")

func (r *demandRepository) Invalidate(ctx context.Context, input rte.InvalidateInput) (rte.InvalidationResult, error) {
	if err := input.Validate(); err != nil {
		return rte.InvalidationResult{}, err
	}

	if len(input.Selectors) == 0 {
		return rte.InvalidationResult{}, nil
	}

	result, err := transaction.Run(ctx, r, func(ctx context.Context) (rte.InvalidationResult, error) {
		return entutils.TransactingRepo(ctx, r.adapter, func(ctx context.Context, a *adapter) (rte.InvalidationResult, error) {
			query, args, err := (invalidateDemandQuery{input: input, at: clock.Now()}).SQL()
			if err != nil {
				return rte.InvalidationResult{}, err
			}

			updated, err := a.db.ExecContext(ctx, query, args...)
			if err != nil {
				return rte.InvalidationResult{}, err
			}

			matched, err := updated.RowsAffected()
			if err != nil {
				return rte.InvalidationResult{}, err
			}

			if matched == 0 {
				duplicate, err := a.db.RTEInvalidation.Query().Where(
					receiptdb.SourceEQ(input.Source), receiptdb.EventIDEQ(input.EventID),
				).Exist(ctx)

				return rte.InvalidationResult{Duplicate: duplicate}, err
			}

			err = a.db.RTEInvalidation.Create().SetSource(input.Source).SetEventID(input.EventID).
				OnConflict(sql.ConflictColumns(receiptdb.FieldSource, receiptdb.FieldEventID)).DoNothing().Exec(ctx)
			if errors.Is(err, stdsql.ErrNoRows) {
				return rte.InvalidationResult{}, errConcurrentInvalidation
			}

			if err != nil {
				return rte.InvalidationResult{}, fmt.Errorf("remember invalidation: %w", err)
			}

			return rte.InvalidationResult{Matched: int(matched)}, nil
		})
	})
	// A rollback failure is joined to the sentinel and must reach the caller.
	if err == errConcurrentInvalidation {
		return rte.InvalidationResult{Duplicate: true}, nil
	}

	return result, err
}

// A deletion notification can arrive before its effective time. Read the source
// again at processing time in case the writer moved or canceled the deadline.
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
