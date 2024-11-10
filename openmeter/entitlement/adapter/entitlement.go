package adapter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	db_entitlement "github.com/openmeterio/openmeter/openmeter/ent/db/entitlement"
	db_feature "github.com/openmeterio/openmeter/openmeter/ent/db/feature"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	db_usagereset "github.com/openmeterio/openmeter/openmeter/ent/db/usagereset"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/entitlement/balanceworker"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/recurrence"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type entitlementDBAdapter struct {
	db *db.Client
}

type repo interface {
	entitlement.EntitlementRepo
	balanceworker.BalanceWorkerRepository
}

var _ repo = (*entitlementDBAdapter)(nil)

var _ interface {
	transaction.Creator
	entutils.TxUser[*entitlementDBAdapter]
} = (*entitlementDBAdapter)(nil)

func NewPostgresEntitlementRepo(db *db.Client) *entitlementDBAdapter {
	return &entitlementDBAdapter{
		db: db,
	}
}

func (a *entitlementDBAdapter) GetEntitlement(ctx context.Context, entitlementID models.NamespacedID) (*entitlement.Entitlement, error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) (*entitlement.Entitlement, error) {
			res, err := withLatestUsageReset(repo.db.Entitlement.Query(), []string{entitlementID.Namespace}).
				Where(
					db_entitlement.ID(entitlementID.ID),
					db_entitlement.Namespace(entitlementID.Namespace),
					db_entitlement.Or(db_entitlement.DeletedAtGT(clock.Now()), db_entitlement.DeletedAtIsNil()),
				).
				First(ctx)
			if err != nil {
				if db.IsNotFound(err) {
					return nil, &entitlement.NotFoundError{EntitlementID: entitlementID}
				}
				return nil, err
			}

			return mapEntitlementEntity(res), nil
		},
	)
}

func (a *entitlementDBAdapter) GetActiveEntitlementOfSubjectAt(ctx context.Context, namespace string, subjectKey string, featureKey string, at time.Time) (*entitlement.Entitlement, error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) (*entitlement.Entitlement, error) {
			res, err := withLatestUsageReset(repo.db.Entitlement.Query(), []string{namespace}).
				Where(entitlementActiveAt(at)...).
				Where(
					db_entitlement.Or(db_entitlement.DeletedAtGT(at), db_entitlement.DeletedAtIsNil()),
					db_entitlement.SubjectKey(subjectKey),
					db_entitlement.Namespace(namespace),
					db_entitlement.FeatureKey(featureKey),
				).
				First(ctx)
			if err != nil {
				if db.IsNotFound(err) {
					return nil, &entitlement.NotFoundError{
						EntitlementID: models.NamespacedID{
							Namespace: namespace,
							ID:        featureKey,
						},
					}
				}
				return nil, err
			}

			return mapEntitlementEntity(res), nil
		},
	)
}

func (a *entitlementDBAdapter) CreateEntitlement(ctx context.Context, ent entitlement.CreateEntitlementRepoInputs) (*entitlement.Entitlement, error) {
	return entutils.TransactingRepo[*entitlement.Entitlement, *entitlementDBAdapter](
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) (*entitlement.Entitlement, error) {
			cmd := repo.db.Entitlement.Create().
				SetEntitlementType(db_entitlement.EntitlementType(ent.EntitlementType)).
				SetNamespace(ent.Namespace).
				SetFeatureID(ent.FeatureID).
				SetMetadata(ent.Metadata).
				SetSubjectKey(ent.SubjectKey).
				SetFeatureKey(ent.FeatureKey).
				SetNillableMeasureUsageFrom(ent.MeasureUsageFrom).
				SetNillableIssueAfterReset(ent.IssueAfterReset).
				SetNillableIssueAfterResetPriority(ent.IssueAfterResetPriority).
				SetNillablePreserveOverageAtReset(ent.PreserveOverageAtReset).
				SetNillableIsSoftLimit(ent.IsSoftLimit).
				SetNillableActiveFrom(ent.ActiveFrom).
				SetNillableActiveTo(ent.ActiveTo)

			if ent.UsagePeriod != nil {
				dbInterval := db_entitlement.UsagePeriodInterval(ent.UsagePeriod.Interval)

				cmd.SetNillableUsagePeriodAnchor(&ent.UsagePeriod.Anchor).
					SetNillableUsagePeriodInterval(&dbInterval)
			}

			if ent.CurrentUsagePeriod != nil {
				cmd.SetNillableCurrentUsagePeriodStart(&ent.CurrentUsagePeriod.From).
					SetNillableCurrentUsagePeriodEnd(&ent.CurrentUsagePeriod.To)
			}

			if ent.Config != nil {
				cmd.SetConfig(ent.Config)
			}

			res, err := cmd.Save(ctx)
			if err != nil {
				return nil, err
			}

			return mapEntitlementEntity(res), nil
		},
	)
}

func (a *entitlementDBAdapter) DeleteEntitlement(ctx context.Context, entitlementID models.NamespacedID) error {
	_, err := entutils.TransactingRepo[*entitlement.Entitlement, *entitlementDBAdapter](
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) (*entitlement.Entitlement, error) {
			affectedCount, err := repo.db.Entitlement.Update().
				Where(db_entitlement.ID(entitlementID.ID), db_entitlement.Namespace(entitlementID.Namespace)).
				SetDeletedAt(clock.Now()).
				Save(ctx)
			if err != nil {
				return nil, err
			}
			if affectedCount == 0 {
				return nil, &entitlement.NotFoundError{EntitlementID: entitlementID}
			}
			return nil, nil
		},
	)
	return err
}

func (a *entitlementDBAdapter) DeactivateEntitlement(ctx context.Context, entitlementID models.NamespacedID, at time.Time) error {
	_, err := entutils.TransactingRepo[interface{}, *entitlementDBAdapter](
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) (interface{}, error) {
			ent, err := repo.GetEntitlement(ctx, entitlementID)
			if err != nil {
				return nil, err
			}

			if ent.ActiveTo != nil {
				return nil, fmt.Errorf("entitlement %s is already deactivated", entitlementID.ID)
			}

			return nil, repo.db.Entitlement.UpdateOneID(ent.ID).SetActiveTo(at).Exec(ctx)
		},
	)

	return err
}

func (a *entitlementDBAdapter) ListAffectedEntitlements(ctx context.Context, eventFilters []balanceworker.IngestEventQueryFilter) ([]balanceworker.IngestEventDataResponse, error) {
	return entutils.TransactingRepo[[]balanceworker.IngestEventDataResponse, *entitlementDBAdapter](
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) ([]balanceworker.IngestEventDataResponse, error) {
			if len(eventFilters) == 0 {
				return nil, fmt.Errorf("no eventFilters provided")
			}

			query := repo.db.Entitlement.Query()

			var ep predicate.Entitlement
			for i, pair := range eventFilters {
				p := db_entitlement.And(
					db_entitlement.Namespace(pair.Namespace),
					db_entitlement.SubjectKey(pair.SubjectKey),
					db_entitlement.HasFeatureWith(db_feature.MeterSlugIn(pair.MeterSlugs...)),
				)
				if i == 0 {
					ep = p
					continue
				}
				ep = db_entitlement.Or(ep, p)
			}

			query = query.Where(ep)

			entities, err := query.WithFeature().All(ctx)
			if err != nil {
				return nil, err
			}

			result := make([]balanceworker.IngestEventDataResponse, 0, len(entities))
			for _, e := range entities {
				result = append(result, balanceworker.IngestEventDataResponse{
					Namespace:     e.Namespace,
					EntitlementID: e.ID,
					SubjectKey:    e.SubjectKey,
					MeterSlug:     e.Edges.Feature.MeterSlug,
				})
			}

			return result, nil
		})
}

func (a *entitlementDBAdapter) GetActiveEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey, at time.Time) ([]entitlement.Entitlement, error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) ([]entitlement.Entitlement, error) {
			res, err := withLatestUsageReset(repo.db.Entitlement.Query(), []string{namespace}).
				Where(entitlementActiveAt(at)...).
				Where(
					db_entitlement.Or(db_entitlement.DeletedAtGT(clock.Now()), db_entitlement.DeletedAtIsNil()),
					db_entitlement.SubjectKey(string(subjectKey)),
					db_entitlement.Namespace(namespace),
				).
				All(ctx)
			if err != nil {
				return nil, err
			}

			result := make([]entitlement.Entitlement, 0, len(res))
			for _, e := range res {
				result = append(result, *mapEntitlementEntity(e))
			}

			return result, nil
		},
	)
}

func (a *entitlementDBAdapter) HasEntitlementForMeter(ctx context.Context, namespace string, meterSlug string) (bool, error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) (bool, error) {
			exists, err := repo.db.Entitlement.Query().
				Where(
					db_entitlement.Or(db_entitlement.DeletedAtGT(clock.Now()), db_entitlement.DeletedAtIsNil()),
					db_entitlement.Namespace(namespace),
					db_entitlement.HasFeatureWith(db_feature.MeterSlugEQ(meterSlug)),
				).
				Exist(ctx)
			if err != nil {
				return false, err
			}

			return exists, nil
		},
	)
}

func (a *entitlementDBAdapter) ListEntitlements(ctx context.Context, params entitlement.ListEntitlementsParams) (pagination.PagedResponse[entitlement.Entitlement], error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) (pagination.PagedResponse[entitlement.Entitlement], error) {
			query := repo.db.Entitlement.Query()

			if len(params.Namespaces) > 0 {
				query = query.Where(db_entitlement.NamespaceIn(params.Namespaces...))
			}

			query = withLatestUsageReset(query, params.Namespaces)

			if len(params.SubjectKeys) > 0 {
				query = query.Where(db_entitlement.SubjectKeyIn(params.SubjectKeys...))
			}

			if len(params.EntitlementTypes) > 0 {
				query = query.Where(db_entitlement.EntitlementTypeIn(slicesx.Map(params.EntitlementTypes, func(t entitlement.EntitlementType) db_entitlement.EntitlementType {
					return db_entitlement.EntitlementType(t)
				})...))
			}

			if len(params.IDs) > 0 {
				query = query.Where(db_entitlement.IDIn(params.IDs...))
			}

			if len(params.FeatureIDsOrKeys) > 0 {
				var ep predicate.Entitlement
				for i, idOrKey := range params.FeatureIDsOrKeys {
					p := db_entitlement.Or(db_entitlement.FeatureID(idOrKey), db_entitlement.FeatureKey(idOrKey))
					if i == 0 {
						ep = p
						continue
					}
					ep = db_entitlement.Or(ep, p)
				}
				query = query.Where(ep)
			}

			if len(params.FeatureIDs) > 0 {
				query = query.Where(db_entitlement.FeatureIDIn(params.FeatureIDs...))
			}

			if len(params.FeatureKeys) > 0 {
				query = query.Where(db_entitlement.FeatureKeyIn(params.FeatureKeys...))
			}

			if !params.IncludeDeleted {
				query = query.Where(db_entitlement.Or(db_entitlement.DeletedAtGT(clock.Now()), db_entitlement.DeletedAtIsNil()))
			}

			if !params.IncludeDeletedAfter.IsZero() {
				query = query.Where(db_entitlement.Or(db_entitlement.DeletedAtGT(params.IncludeDeletedAfter), db_entitlement.DeletedAtIsNil()))
			}

			if params.OrderBy != "" {
				order := []sql.OrderTermOption{}
				if !params.Order.IsDefaultValue() {
					order = entutils.GetOrdering(params.Order)
				}
				switch params.OrderBy {
				case entitlement.ListEntitlementsOrderByCreatedAt:
					query = query.Order(db_entitlement.ByCreatedAt(order...))
				case entitlement.ListEntitlementsOrderByUpdatedAt:
					query = query.Order(db_entitlement.ByUpdatedAt(order...))
				}
			}

			response := pagination.PagedResponse[entitlement.Entitlement]{
				Page: params.Page,
			}

			// we're using limit and offset
			if params.Page.IsZero() {
				if params.Limit > 0 {
					query = query.Limit(params.Limit)
				}
				if params.Offset > 0 {
					query = query.Offset(params.Offset)
				}

				entities, err := query.All(ctx)
				if err != nil {
					return response, err
				}

				mapped := make([]entitlement.Entitlement, 0, len(entities))
				for _, entity := range entities {
					mapped = append(mapped, *mapEntitlementEntity(entity))
				}

				response.Items = mapped
				return response, nil
			}

			paged, err := query.Paginate(ctx, params.Page)
			if err != nil {
				return response, err
			}

			result := make([]entitlement.Entitlement, 0, len(paged.Items))
			for _, e := range paged.Items {
				result = append(result, *mapEntitlementEntity(e))
			}

			response.TotalCount = paged.TotalCount
			response.Items = result

			return response, nil
		},
	)
}

func mapEntitlementEntity(e *db.Entitlement) *entitlement.Entitlement {
	ent := &entitlement.Entitlement{
		GenericProperties: entitlement.GenericProperties{
			NamespacedModel: models.NamespacedModel{
				Namespace: e.Namespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: e.CreatedAt.UTC(),
				UpdatedAt: e.UpdatedAt.UTC(),
				DeletedAt: convert.SafeToUTC(e.DeletedAt),
			},
			ID:              e.ID,
			SubjectKey:      e.SubjectKey,
			FeatureID:       e.FeatureID,
			FeatureKey:      e.FeatureKey,
			EntitlementType: entitlement.EntitlementType(e.EntitlementType),
			Metadata:        e.Metadata,
			ActiveFrom:      convert.SafeToUTC(e.ActiveFrom),
			ActiveTo:        convert.SafeToUTC(e.ActiveTo),
		},
		MeasureUsageFrom:        e.MeasureUsageFrom,
		IssueAfterReset:         e.IssueAfterReset,
		IssueAfterResetPriority: e.IssueAfterResetPriority,
		IsSoftLimit:             e.IsSoftLimit,
		PreserveOverageAtReset:  e.PreserveOverageAtReset,
	}

	switch {
	case len(e.Edges.UsageReset) > 0:
		ent.LastReset = convert.ToPointer(e.Edges.UsageReset[0].ResetTime.In(time.UTC))
	case e.MeasureUsageFrom != nil:
		ent.LastReset = convert.ToPointer(e.MeasureUsageFrom.In(time.UTC))
	}

	if e.Config != nil {
		ent.Config = e.Config
	}

	if e.UsagePeriodAnchor != nil && e.UsagePeriodInterval != nil {
		ent.UsagePeriod = &entitlement.UsagePeriod{
			Anchor:   e.UsagePeriodAnchor.In(time.UTC),
			Interval: recurrence.RecurrenceInterval(*e.UsagePeriodInterval),
		}
	}

	if e.CurrentUsagePeriodEnd != nil && e.CurrentUsagePeriodStart != nil {
		ent.CurrentUsagePeriod = &recurrence.Period{
			From: e.CurrentUsagePeriodStart.In(time.UTC),
			To:   e.CurrentUsagePeriodEnd.In(time.UTC),
		}
	}

	return ent
}

func (a *entitlementDBAdapter) UpdateEntitlementUsagePeriod(ctx context.Context, entitlementID models.NamespacedID, params entitlement.UpdateEntitlementUsagePeriodParams) error {
	_, err := entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) (*entitlement.Entitlement, error) {
			update := repo.db.Entitlement.Update().
				Where(db_entitlement.ID(entitlementID.ID), db_entitlement.Namespace(entitlementID.Namespace)).
				SetCurrentUsagePeriodStart(params.CurrentUsagePeriod.From).
				SetCurrentUsagePeriodEnd(params.CurrentUsagePeriod.To)

			if params.NewAnchor != nil {
				update = update.SetUsagePeriodAnchor(*params.NewAnchor)
			}

			_, err := update.Save(ctx)
			return nil, err
		},
	)
	return err
}

func (a *entitlementDBAdapter) ListActiveEntitlementsWithExpiredUsagePeriod(ctx context.Context, namespaces []string, expiredBefore time.Time) ([]entitlement.Entitlement, error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) ([]entitlement.Entitlement, error) {
			query := withLatestUsageReset(repo.db.Entitlement.Query(), namespaces).
				Where(entitlementActiveAt(expiredBefore)...).
				Where(
					db_entitlement.CurrentUsagePeriodEndNotNil(),
					db_entitlement.CurrentUsagePeriodEndLTE(expiredBefore),
					db_entitlement.Or(db_entitlement.DeletedAtIsNil(), db_entitlement.DeletedAtGT(clock.Now())),
				)

			if len(namespaces) > 0 {
				query = query.Where(db_entitlement.NamespaceIn(namespaces...))
			}

			res, err := query.All(ctx)
			if err != nil {
				return nil, err
			}

			result := make([]entitlement.Entitlement, 0, len(res))
			for _, e := range res {
				result = append(result, *mapEntitlementEntity(e))
			}

			return result, nil
		},
	)
}

func (a *entitlementDBAdapter) LockEntitlementForTx(ctx context.Context, tx *entutils.TxDriver, entitlementID models.NamespacedID) error {
	pgLockNotAvailableErrorCode := "55P03"

	if tx == nil {
		return fmt.Errorf("lock entitlement for tx called from outside a transaction")
	}
	_, err := a.WithTx(ctx, tx).db.Entitlement.
		Query().
		Where(db_entitlement.ID(entitlementID.ID), db_entitlement.Namespace(entitlementID.Namespace)).
		ForUpdate().
		Only(ctx)
	if err != nil {
		if strings.Contains(err.Error(), pgLockNotAvailableErrorCode) {
			// TODO: return a more specific error
			return fmt.Errorf("acquiring lock for entitlement %s failed: %w", entitlementID.ID, err)
		}
	}

	return err
}

type namespacesWithCount struct {
	Namespace string
	Count     int
}

func (a *entitlementDBAdapter) ListNamespacesWithActiveEntitlements(ctx context.Context, includeDeletedAfter time.Time) ([]string, error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) ([]string, error) {
			now := clock.Now()
			namespaces := []namespacesWithCount{}

			query := repo.db.Entitlement.Query().
				Where(
					db_entitlement.Or(db_entitlement.DeletedAtGT(includeDeletedAfter), db_entitlement.DeletedAtIsNil()),
				)

			if includeDeletedAfter.Before(now) || includeDeletedAfter.Equal(now) {
				query = query.Where(entitlementActiveAt(now)...)
			} else {
				query = query.Where(entitlementActiveBetween(includeDeletedAfter, now)...)
			}

			query2 := query.
				GroupBy(db_entitlement.FieldNamespace).
				Aggregate(db.Count())

			err := query2.Scan(ctx, &namespaces)
			if err != nil {
				return nil, err
			}

			return slicesx.Map(namespaces, func(n namespacesWithCount) string {
				return n.Namespace
			}), nil
		},
	)
}

func (a *entitlementDBAdapter) GetScheduledEntitlements(ctx context.Context, namespace string, subjectKey models.SubjectKey, featureKey string, starting time.Time) ([]entitlement.Entitlement, error) {
	res, err := entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) (*[]entitlement.Entitlement, error) {
			res, err := repo.db.Entitlement.Query().
				Where(
					db_entitlement.Or(
						db_entitlement.ActiveToIsNil(),
						db_entitlement.ActiveToGT(starting),
					),
				).
				Where(
					db_entitlement.Or(db_entitlement.DeletedAtIsNil(), db_entitlement.DeletedAtGT(clock.Now())),
					db_entitlement.Namespace(namespace),
					db_entitlement.SubjectKey(string(subjectKey)),
					db_entitlement.FeatureKey(featureKey),
				).Order(
				func(s *sql.Selector) {
					// order by COALESCE(ActiveFrom, CreatedAt) ASC
					orderBy := fmt.Sprintf("COALESCE(%s, %s) ASC", db_entitlement.FieldActiveFrom, db_entitlement.FieldCreatedAt)
					s.OrderBy(orderBy)
				},
			).All(ctx)
			if err != nil {
				return nil, err
			}

			result := make([]entitlement.Entitlement, 0, len(res))
			for _, e := range res {
				result = append(result, *mapEntitlementEntity(e))
			}

			return &result, nil
		},
	)
	return defaultx.WithDefault(res, nil), err
}

func entitlementActiveBetween(from, to time.Time) []predicate.Entitlement {
	return []predicate.Entitlement{
		db_entitlement.Or(
			db_entitlement.And(db_entitlement.ActiveFromIsNil(), db_entitlement.CreatedAtLTE(to)),
			db_entitlement.ActiveFromLTE(to),
		),
		db_entitlement.Or(
			db_entitlement.ActiveToIsNil(),
			db_entitlement.ActiveToGT(from),
		),
	}
}

func entitlementActiveAt(at time.Time) []predicate.Entitlement {
	return []predicate.Entitlement{
		db_entitlement.Or(
			// If activeFrom is nil activity starts at creation time
			db_entitlement.And(db_entitlement.ActiveFromIsNil(), db_entitlement.CreatedAtLTE(at)),
			db_entitlement.ActiveFromLTE(at),
		),
		db_entitlement.Or(
			db_entitlement.ActiveToIsNil(),
			db_entitlement.ActiveToGT(at),
		),
	}
}

func withLatestUsageReset(q *db.EntitlementQuery, namespaces []string) *db.EntitlementQuery {
	return q.WithUsageReset(func(urq *db.UsageResetQuery) {
		urq.
			Order(db_usagereset.ByResetTime(sql.OrderDesc())).
			Limit(1)

		if len(namespaces) > 0 {
			urq.Where(db_usagereset.NamespaceIn(namespaces...))
		}
	})
}
