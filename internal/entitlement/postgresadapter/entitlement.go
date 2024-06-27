package postgresadapter

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/internal/entitlement/postgresadapter/ent/db"
	db_entitlement "github.com/openmeterio/openmeter/internal/entitlement/postgresadapter/ent/db/entitlement"
	"github.com/openmeterio/openmeter/internal/entitlement/postgresadapter/ent/db/usagereset"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

type entitlementDBAdapter struct {
	db *db.Client
}

func NewPostgresEntitlementRepo(db *db.Client) entitlement.EntitlementRepo {
	return &entitlementDBAdapter{
		db: db,
	}
}

func (a *entitlementDBAdapter) GetEntitlement(ctx context.Context, entitlementID models.NamespacedID) (*entitlement.Entitlement, error) {
	res, err := withLatestUsageReset(a.db.Entitlement.Query()).
		Where(
			db_entitlement.ID(entitlementID.ID),
			db_entitlement.Namespace(entitlementID.Namespace),
			db_entitlement.Or(db_entitlement.DeletedAtGT(time.Now()), db_entitlement.DeletedAtIsNil()),
		).
		First(ctx)

	if err != nil {
		if db.IsNotFound(err) {
			return nil, &entitlement.NotFoundError{EntitlementID: entitlementID}
		}
		return nil, err
	}

	return mapEntitlementEntity(res), nil
}

func (a *entitlementDBAdapter) GetEntitlementOfSubject(ctx context.Context, namespace string, subjectKey string, id string) (*entitlement.Entitlement, error) {
	res, err := withLatestUsageReset(a.db.Entitlement.Query()).
		Where(
			db_entitlement.Or(db_entitlement.DeletedAtGT(time.Now()), db_entitlement.DeletedAtIsNil()),
			db_entitlement.SubjectKey(string(subjectKey)),
			db_entitlement.Namespace(namespace),
			db_entitlement.ID(id),
		).
		First(ctx)

	if err != nil {
		if db.IsNotFound(err) {
			return nil, &entitlement.NotFoundError{
				EntitlementID: models.NamespacedID{
					Namespace: namespace,
					ID:        id,
				},
			}
		}
		return nil, err
	}

	return mapEntitlementEntity(res), nil
}

func (a *entitlementDBAdapter) CreateEntitlement(ctx context.Context, entitlement entitlement.CreateEntitlementRepoInputs) (*entitlement.Entitlement, error) {
	cmd := a.db.Entitlement.Create().
		SetEntitlementType(db_entitlement.EntitlementType(entitlement.EntitlementType)).
		SetNamespace(entitlement.Namespace).
		SetFeatureID(entitlement.FeatureID).
		SetMetadata(entitlement.Metadata).
		SetSubjectKey(entitlement.SubjectKey).
		SetNillableMeasureUsageFrom(entitlement.MeasureUsageFrom).
		SetNillableIssueAfterReset(entitlement.IssueAfterReset).
		SetNillableIsSoftLimit(entitlement.IsSoftLimit)

	if entitlement.UsagePeriod != nil {
		dbInterval := db_entitlement.UsagePeriodInterval(entitlement.UsagePeriod.Interval)

		cmd.SetNillableUsagePeriodAnchor(&entitlement.UsagePeriod.Anchor).
			SetNillableUsagePeriodInterval(&dbInterval)
	}

	if entitlement.CurrentUsagePeriod != nil {
		cmd.SetNillableCurrentUsagePeriodStart(&entitlement.CurrentUsagePeriod.From).
			SetNillableCurrentUsagePeriodEnd(&entitlement.CurrentUsagePeriod.To)
	}

	if entitlement.Config != nil {
		var config map[string]interface{}
		if err := json.Unmarshal([]byte(*entitlement.Config), &config); err != nil {
			return nil, err
		}
		cmd.SetConfig(config)
	}

	res, err := cmd.Save(ctx)

	if err != nil {
		return nil, err
	}

	return mapEntitlementEntity(res), nil
}

func (a *entitlementDBAdapter) DeleteEntitlement(ctx context.Context, entitlementID models.NamespacedID) error {
	_, err := a.db.Entitlement.Update().
		Where(db_entitlement.ID(entitlementID.ID), db_entitlement.Namespace(entitlementID.Namespace)).
		SetDeletedAt(time.Now()).
		Save(ctx)
	return err
}

func (a *entitlementDBAdapter) GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey) ([]entitlement.Entitlement, error) {
	res, err := withLatestUsageReset(a.db.Entitlement.Query()).
		Where(
			db_entitlement.Or(db_entitlement.DeletedAtGT(time.Now()), db_entitlement.DeletedAtIsNil()),
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

}

func (a *entitlementDBAdapter) ListEntitlements(ctx context.Context, params entitlement.ListEntitlementsParams) ([]entitlement.Entitlement, error) {
	query := withLatestUsageReset(a.db.Entitlement.Query().
		Where(db_entitlement.Namespace(params.Namespace)))

	if !params.IncludeDeleted {
		query = query.Where(db_entitlement.Or(db_entitlement.DeletedAtGT(time.Now()), db_entitlement.DeletedAtIsNil()))
	}

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	switch params.OrderBy {
	case entitlement.ListEntitlementsOrderByCreatedAt:
		query = query.Order(db_entitlement.ByCreatedAt())
	case entitlement.ListEntitlementsOrderByUpdatedAt:
		query = query.Order(db_entitlement.ByUpdatedAt())
	}

	entities, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]entitlement.Entitlement, 0, len(entities))
	for _, e := range entities {
		result = append(result, *mapEntitlementEntity(e))
	}

	return result, nil

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
			EntitlementType: entitlement.EntitlementType(e.EntitlementType),
			Metadata:        e.Metadata,
		},
		MeasureUsageFrom: e.MeasureUsageFrom,
		IssueAfterReset:  e.IssueAfterReset,
		IsSoftLimit:      e.IsSoftLimit,
	}

	switch {
	case len(e.Edges.UsageReset) > 0:
		ent.LastReset = convert.ToPointer(e.Edges.UsageReset[0].ResetTime.In(time.UTC))
	case e.MeasureUsageFrom != nil:
		ent.LastReset = convert.ToPointer(e.MeasureUsageFrom.In(time.UTC))
	}

	if e.Config != nil {
		cStr, err := json.Marshal(e.Config)
		if err != nil {
			// TODO: handle error
			ent.Config = nil
		} else {
			ent.Config = convert.ToPointer(string(cStr))
		}
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

func (a *entitlementDBAdapter) LockEntitlementForTx(ctx context.Context, entitlementID models.NamespacedID) error {
	// TODO: check if we're actually in a transaction
	pgLockNotAvailableErrorCode := "55P03"

	_, err := a.db.Entitlement.Query().
		Where(
			db_entitlement.ID(entitlementID.ID),
			db_entitlement.Namespace(entitlementID.Namespace),
		).
		ForUpdate().
		Only(ctx)

	if err != nil {
		if db.IsNotFound(err) {
			return &entitlement.NotFoundError{
				EntitlementID: entitlementID,
			}
		}
		if strings.Contains(err.Error(), pgLockNotAvailableErrorCode) {
			// TODO: return a more specific error
			return err
		}
	}
	return err
}

func (a *entitlementDBAdapter) UpdateEntitlementUsagePeriod(ctx context.Context, entitlementID models.NamespacedID, params entitlement.UpdateEntitlementUsagePeriodParams) error {
	update := a.db.Entitlement.Update().
		Where(db_entitlement.ID(entitlementID.ID), db_entitlement.Namespace(entitlementID.Namespace)).
		SetCurrentUsagePeriodStart(params.CurrentUsagePeriod.From).
		SetCurrentUsagePeriodEnd(params.CurrentUsagePeriod.To)

	if params.NewAnchor != nil {
		update = update.SetUsagePeriodAnchor(*params.NewAnchor)
	}

	_, err := update.Save(ctx)
	return err
}

func (a *entitlementDBAdapter) ListEntitlementsWithExpiredUsagePeriod(ctx context.Context, namespace string, expiredBefore time.Time) ([]entitlement.Entitlement, error) {
	res, err := withLatestUsageReset(a.db.Entitlement.Query()).
		Where(
			db_entitlement.Namespace(namespace),
			db_entitlement.CurrentUsagePeriodEndNotNil(),
			db_entitlement.CurrentUsagePeriodEndLTE(expiredBefore),
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
}

func withLatestUsageReset(q *db.EntitlementQuery) *db.EntitlementQuery {
	return q.WithUsageReset(func(urq *db.UsageResetQuery) {
		urq.Order(usagereset.ByResetTime(sql.OrderDesc()))
		urq.Limit(1)
	})
}
