package adapter

import (
	"context"
	stdsql "database/sql"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/lib/pq"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	demanddb "github.com/openmeterio/openmeter/openmeter/ent/db/rtedemand"
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

func demandIdentity(target rte.CalculationTarget) (predicate.RTEDemand, error) {
	identity, err := target.Identity()
	if err != nil {
		return nil, err
	}

	return demanddb.And(demanddb.NamespaceEQ(identity.Namespace), demanddb.KindEQ(string(target.Kind)), demanddb.KeyHashEQ(identity.KeyHash), demanddb.DeletedAtIsNil()), nil
}

func activeDemand(at time.Time) predicate.RTEDemand {
	return demanddb.And(demanddb.DeletedAtIsNil(), demanddb.Or(demanddb.PushEnabledEQ(true), demanddb.PullUntilGT(at)))
}

func (r *demandRepository) Activate(ctx context.Context, input rte.ActivateDemandInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return r.ensure(ctx, ensureDemandInput{Target: input.Target, PushEnabled: true, RefreshInterval: input.RefreshInterval})
}

func (r *demandRepository) EnsurePull(ctx context.Context, input rte.EnsureFreshInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	until := clock.Now().Add(input.PullTTL)

	return r.ensure(ctx, ensureDemandInput{Target: input.Target, PullUntil: &until, MaxAge: input.MaxAge})
}

type ensureDemandInput struct {
	Target          rte.CalculationTarget
	PushEnabled     bool
	RefreshInterval time.Duration
	PullUntil       *time.Time
	MaxAge          time.Duration
}

func (i ensureDemandInput) Validate() error {
	var errs []error
	errs = append(errs, i.Target.Validate())
	if i.RefreshInterval < 0 || i.MaxAge < 0 {
		errs = append(errs, errors.New("demand durations must not be negative"))
	}

	if !i.PushEnabled && i.PullUntil == nil {
		errs = append(errs, errors.New("push or pull demand is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (r *demandRepository) ensure(ctx context.Context, input ensureDemandInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	identity, err := input.Target.Identity()
	if err != nil {
		return err
	}

	condition, err := demandIdentity(input.Target)
	if err != nil {
		return err
	}

	return transaction.RunWithNoValue(ctx, r, func(ctx context.Context) error {
		return entutils.TransactingRepoWithNoValue(ctx, r.adapter, func(ctx context.Context, a *adapter) error {
			now := clock.Now()
			err := a.db.RTEDemand.Create().SetNamespace(identity.Namespace).SetKind(string(input.Target.Kind)).SetOwnerID(identity.OwnerID).
				SetKeyHash(identity.KeyHash).SetKey(input.Target.Key).SetNextAttemptAt(now).
				OnConflict(sql.ConflictColumns(demanddb.FieldNamespace, demanddb.FieldKind, demanddb.FieldKeyHash), sql.ConflictWhere(sql.IsNull(demanddb.FieldDeletedAt))).DoNothing().Exec(ctx)
			if err != nil && !errors.Is(err, stdsql.ErrNoRows) {
				return fmt.Errorf("create demand: %w", err)
			}

			row, err := a.db.RTEDemand.Query().Where(condition).ForUpdate().Only(ctx)
			if err != nil {
				return err
			}

			wasActive := row.PushEnabled || (row.PullUntil != nil && row.PullUntil.After(now))
			update := a.db.RTEDemand.UpdateOneID(row.ID)
			if input.PushEnabled {
				update.SetPushEnabled(true).SetRefreshInterval(int64(input.RefreshInterval))
				if input.RefreshInterval > 0 {
					if row.NextRefreshAt == nil || row.RefreshInterval != int64(input.RefreshInterval) {
						nextRefresh := now.Add(input.RefreshInterval)
						update.SetNextRefreshAt(nextRefresh)
						if row.CompletedVersion == row.DirtyVersion {
							update.SetNextAttemptAt(nextRefresh)
						}
					}
				} else {
					update.ClearNextRefreshAt()
				}
			}

			if input.PullUntil != nil && (row.PullUntil == nil || input.PullUntil.After(*row.PullUntil)) {
				update.SetPullUntil(*input.PullUntil)
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
	condition, err := demandIdentity(target)
	if err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, r.adapter, func(ctx context.Context, a *adapter) error {
		return a.db.RTEDemand.Update().Where(condition).SetPushEnabled(false).ClearPullUntil().ClearNextRefreshAt().SetRefreshInterval(0).Exec(ctx)
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
			count, err := a.db.RTEDemand.Update().Where(activeDemand(now), demanddb.Or(selectors...)).AddDirtyVersion(1).SetNextAttemptAt(now).Save(ctx)

			return rte.InvalidationResult{Matched: count}, err
		})
	})
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
	var failedVersion int64
	var retryAt time.Time
	var calculationFailure error
	result, processErr := transaction.Run(ctx, r, func(ctx context.Context) (rte.DemandWorkResult, error) {
		return entutils.TransactingRepo(ctx, r.adapter, func(ctx context.Context, a *adapter) (rte.DemandWorkResult, error) {
			now := clock.Now()
			due := demanddb.Or(func(s *sql.Selector) {
				s.Where(sql.ColumnsGT(s.C(demanddb.FieldDirtyVersion), s.C(demanddb.FieldCompletedVersion)))
			}, demanddb.NextRefreshAtLTE(now))
			query := a.db.RTEDemand.Query().Where(demanddb.DeletedAtIsNil(), due).Order(demanddb.ByNextAttemptAt(), demanddb.ByID()).Limit(1)
			if input.Target == nil {
				// Keep the push predicate literal so prepared generic plans can
				// use the partial ready-work index too.
				query.Where(func(s *sql.Selector) { s.Where(sql.IsTrue(s.C(demanddb.FieldPushEnabled))) }, demanddb.NextAttemptAtLTE(now)).ForUpdate(sql.WithLockAction(sql.SkipLocked))
			} else {
				condition, err := demandIdentity(*input.Target)
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
			workVersion := row.DirtyVersion
			if row.DirtyVersion == row.CompletedVersion {
				workVersion++
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
			update := a.db.RTEDemand.UpdateOneID(row.ID).SetDirtyVersion(workVersion)
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
				failedVersion = workVersion
				update.SetAttempts(row.Attempts + 1).SetNextAttemptAt(retryAt)
			} else {
				update.SetCompletedVersion(workVersion).SetAttempts(0).SetLastCalculatedAt(now).SetNextAttemptAt(clock.Now())
				if row.RefreshInterval > 0 {
					nextRefresh := clock.Now().Add(time.Duration(row.RefreshInterval))
					update.SetNextRefreshAt(nextRefresh).SetNextAttemptAt(nextRefresh)
				} else {
					update.ClearNextRefreshAt()
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
					demanddb.IDEQ(failedAttempt.ID), demanddb.DeletedAtIsNil(),
					demanddb.DirtyVersionEQ(failedAttempt.DirtyVersion),
					demanddb.CompletedVersionEQ(failedAttempt.CompletedVersion),
					demanddb.AttemptsEQ(failedAttempt.Attempts),
				).SetDirtyVersion(failedVersion).SetAttempts(failedAttempt.Attempts + 1).SetNextAttemptAt(retryAt).Exec(ctx)
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
