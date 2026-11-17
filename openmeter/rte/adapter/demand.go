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
	"github.com/lib/pq"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	entitlementdb "github.com/openmeterio/openmeter/openmeter/ent/db/entitlement"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	demanddb "github.com/openmeterio/openmeter/openmeter/ent/db/rtedemand"
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

	if input.PullTTL > 0 && input.MaxAge == 0 {
		clean, err := entutils.TransactingRepo(ctx, r.adapter, func(ctx context.Context, a *adapter) (bool, error) {
			// If another transaction holds the row, the tracking period might expire before
			// it commits. Let registration recheck the lease after the lock is ours.
			const query = `UPDATE rte_demands
				SET pull_until = GREATEST(pull_until, $1), updated_at = $2
				WHERE id = (SELECT id FROM rte_demands
				  WHERE namespace = $3 AND kind = $4 AND key_hash = $5 AND key = $6::jsonb
				    AND deleted_at IS NULL AND NOT push_enabled AND pull_until > $2
				  FOR UPDATE SKIP LOCKED)
				RETURNING completed_version = dirty_version AND (next_refresh_at IS NULL OR next_refresh_at > $2)`

			now := sync.OnceValue(clock.Now)
			rows, err := a.db.QueryContext(ctx, query, pullRenewalTime{now: now, offset: input.PullTTL}, pullRenewalTime{now: now}, identity.Namespace, string(input.Target.Kind), identity.KeyHash, string(encodedKey))
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

	return false, transaction.RunWithNoValue(ctx, r, func(ctx context.Context) error {
		return entutils.TransactingRepoWithNoValue(ctx, r.adapter, func(ctx context.Context, a *adapter) error {
			now := clock.Now()
			err := a.db.RTEDemand.Create().SetNamespace(identity.Namespace).SetKind(string(input.Target.Kind)).SetOwnerID(identity.OwnerID).
				SetKeyHash(identity.KeyHash).SetKey(encodedKey).SetNextAttemptAt(now).
				OnConflict(
					sql.ConflictColumns(demanddb.FieldNamespace, demanddb.FieldKind, demanddb.FieldKeyHash),
					sql.ConflictWhere(sql.IsNull(demanddb.FieldDeletedAt)),
					sql.UpdateWhere(sql.ExprP("rte_demands.key <> excluded.key")),
				).
				Update(func(u *db.RTEDemandUpsert) {
					// Earlier registrations stored the caller's raw JSON. Repair its field
					// names and recalculate once for invalidations routing may have missed.
					u.SetExcluded(demanddb.FieldKey)
					markDemandDirty(u.UpdateBuilder, now)
				}).
				Exec(ctx)
			if err != nil && !errors.Is(err, stdsql.ErrNoRows) {
				return fmt.Errorf("create demand: %w", err)
			}

			row, err := a.db.RTEDemand.Query().Where(condition).ForUpdate().Only(ctx)
			if err != nil {
				return err
			}

			now = clock.Now()
			wasActive := row.PushEnabled || (row.PullUntil != nil && row.PullUntil.After(now))
			update := a.db.RTEDemand.UpdateOneID(row.ID)
			if input.PushEnabled {
				update.SetPushEnabled(true)
			}

			if input.PullTTL > 0 {
				until := now.Add(input.PullTTL)
				if row.PullUntil == nil || until.After(*row.PullUntil) {
					update.SetPullUntil(until)
				}
			}

			stale := input.MaxAge > 0 && (row.LastCalculatedAt == nil || !row.LastCalculatedAt.After(now.Add(-input.MaxAge)))
			if !wasActive || stale {
				if row.CompletedVersion == row.DirtyVersion {
					update.AddDirtyVersion(1)
				}

				update.SetNextAttemptAt(now)
			}

			return update.Exec(ctx)
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

			selectors := make([]predicate.RTEDemand, 0, len(input.Selectors))
			for _, selector := range input.Selectors {
				parts := []predicate.RTEDemand{demanddb.NamespaceEQ(selector.Namespace), demanddb.KindEQ(string(selector.Kind))}
				if len(selector.OwnerIDs) > 0 {
					parts = append(parts, func(s *sql.Selector) {
						s.Where(sql.P(func(b *sql.Builder) {
							b.Ident(s.C(demanddb.FieldOwnerID)).WriteString(" = ANY(").Arg(pq.Array(selector.OwnerIDs)).WriteString("::text[])")
						}))
					})
				}

				if len(selector.Targets) > 0 {
					hashes := make([]string, 0, len(selector.Targets))
					for _, target := range selector.Targets {
						identity, err := target.Identity()
						if err != nil {
							return rte.InvalidationResult{}, err
						}

						hashes = append(hashes, identity.KeyHash)
					}

					// A usage batch can touch more keys than PostgreSQL's parameter
					// limit. Bind their identities as one array instead of one per key.
					parts = append(parts, func(s *sql.Selector) {
						s.Where(sql.P(func(b *sql.Builder) {
							b.Ident(s.C(demanddb.FieldKeyHash)).WriteString(" = ANY(").Arg(pq.Array(hashes)).WriteString("::text[])")
						}))
					})
				}

				selectors = append(selectors, demanddb.And(parts...))
			}

			if len(selectors) == 0 {
				return rte.InvalidationResult{}, nil
			}

			now := clock.Now()
			builder := sql.Dialect(dialect.Postgres)
			matching := builder.Select().From(builder.Table(demanddb.Table))
			demanddb.And(activeDemand(now), demanddb.Or(selectors...))(matching)

			update := builder.Update(demanddb.Table).FromSelect(matching)
			markDemandDirty(update, now)
			query, args := update.Query()
			result, err := a.db.ExecContext(ctx, query, args...)
			if err != nil {
				return rte.InvalidationResult{}, err
			}

			matched, err := result.RowsAffected()

			return rte.InvalidationResult{Matched: int(matched)}, err
		})
	})
}

// A new fact makes a healthy key immediately eligible for calculation. Failed
// keys accumulate the same work while keeping their existing retry deadline.
func markDemandDirty(update *sql.UpdateBuilder, now time.Time) {
	update.Add(demanddb.FieldDirtyVersion, 1).
		Set(demanddb.FieldUpdatedAt, now).
		Set(demanddb.FieldNextAttemptAt, sql.ExprFunc(func(b *sql.Builder) {
			b.WriteString("CASE WHEN rte_demands.attempts = 0 THEN ").Arg(now).
				WriteString(" ELSE rte_demands.next_attempt_at END")
		}))
}

// A deletion notification can arrive before its effective time. Read the source
// row under the demand lock so redelivery and changes to a scheduled deletion
// always use the latest committed deadline.
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
	result, processErr := transaction.Run(ctx, r, func(ctx context.Context) (rte.DemandWorkResult, error) {
		return entutils.TransactingRepo(ctx, r.adapter, func(ctx context.Context, a *adapter) (rte.DemandWorkResult, error) {
			now := clock.Now()
			due := demanddb.Or(func(s *sql.Selector) {
				s.Where(sql.ColumnsGT(s.C(demanddb.FieldDirtyVersion), s.C(demanddb.FieldCompletedVersion)))
			}, demanddb.SourceDeleteAtLTE(now))
			query := a.db.RTEDemand.Query().Where(demanddb.DeletedAtIsNil(), due).Order(demanddb.ByNextAttemptAt(), demanddb.ByID()).Limit(1)
			if input.Target == nil {
				// Keep the push predicate literal so prepared generic plans can
				// use the partial ready-work index too.
				query.Where(func(s *sql.Selector) { s.Where(sql.IsTrue(s.C(demanddb.FieldPushEnabled))) }, demanddb.NextAttemptAtLTE(now)).ForUpdate(sql.WithLockAction(sql.SkipLocked))
			} else {
				_, condition, err := demandIdentity(*input.Target)
				if err != nil {
					return rte.DemandWorkResult{}, err
				}

				query.Where(condition).ForUpdate()
			}

			row, err := query.First(ctx)
			if db.IsNotFound(err) {
				return rte.DemandWorkResult{}, nil
			}

			if err != nil {
				return rte.DemandWorkResult{}, err
			}

			now = clock.Now()

			deletedAt, err := a.demandSourceDeletionAt(ctx, row)
			if err != nil {
				return rte.DemandWorkResult{}, err
			}

			if deletedAt != nil && !deletedAt.After(clock.Now()) {
				// Keep the last successful value readable. Leave this version
				// outstanding so an explicit pull still sees a deleted source.
				err := a.db.RTEDemand.UpdateOneID(row.ID).SetDirtyVersion(max(row.DirtyVersion, row.CompletedVersion+1)).
					SetPushEnabled(false).ClearPullUntil().ClearSourceDeleteAt().SetAttempts(0).Exec(ctx)
				if err != nil {
					return rte.DemandWorkResult{}, err
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
				update := a.db.RTEDemand.UpdateOneID(row.ID).SetOrClearSourceDeleteAt(deletedAt)
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
				update.SetCompletedVersion(row.DirtyVersion).SetAttempts(0).SetLastCalculatedAt(now).SetNextAttemptAt(clock.Now())
				if deletedAt != nil {
					update.SetSourceDeleteAt(*deletedAt).SetNextAttemptAt(*deletedAt)
				} else {
					update.ClearSourceDeleteAt()
				}
			}

			if err := update.Exec(ctx); err != nil {
				return rte.DemandWorkResult{}, err
			}

			return rte.DemandWorkResult{Processed: true, CalculationError: calculationErr}, nil
		})
	})
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
			return rte.DemandWorkResult{}, errors.Join(processErr, fmt.Errorf("recover demand retry: %w", recoveryErr))
		}

		return rte.DemandWorkResult{Processed: true, CalculationError: errors.Join(calculationFailure, processErr)}, nil
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
