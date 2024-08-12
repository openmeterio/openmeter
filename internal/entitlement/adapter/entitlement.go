package adapter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/internal/ent/db"
	db_entitlement "github.com/openmeterio/openmeter/internal/ent/db/entitlement"
	db_feature "github.com/openmeterio/openmeter/internal/ent/db/feature"
	"github.com/openmeterio/openmeter/internal/ent/db/predicate"
	db_usagereset "github.com/openmeterio/openmeter/internal/ent/db/usagereset"
	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/internal/entitlement/balanceworker"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
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

func NewPostgresEntitlementRepo(db *db.Client) repo {
	return &entitlementDBAdapter{
		db: db,
	}
}

func (a *entitlementDBAdapter) GetEntitlement(ctx context.Context, entitlementID models.NamespacedID) (*entitlement.Entitlement, error) {
	res, err := withLatestUsageReset(a.db.Entitlement.Query()).
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
}

func (a *entitlementDBAdapter) GetEntitlementOfSubject(ctx context.Context, namespace string, subjectKey string, idOrFeatureKey string) (*entitlement.Entitlement, error) {
	res, err := withLatestUsageReset(a.db.Entitlement.Query()).
		Where(
			db_entitlement.Or(db_entitlement.DeletedAtGT(clock.Now()), db_entitlement.DeletedAtIsNil()),
			db_entitlement.SubjectKey(subjectKey),
			db_entitlement.Namespace(namespace),
			db_entitlement.Or(db_entitlement.ID(idOrFeatureKey), db_entitlement.FeatureKey(idOrFeatureKey)),
		).
		First(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, &entitlement.NotFoundError{
				EntitlementID: models.NamespacedID{
					Namespace: namespace,
					ID:        idOrFeatureKey,
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
		SetFeatureKey(entitlement.FeatureKey).
		SetNillableMeasureUsageFrom(entitlement.MeasureUsageFrom).
		SetNillableIssueAfterReset(entitlement.IssueAfterReset).
		SetNillableIssueAfterResetPriority(entitlement.IssueAfterResetPriority).
		SetNillablePreserveOverageAtReset(entitlement.PreserveOverageAtReset).
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
		cmd.SetConfig(entitlement.Config)
	}

	res, err := cmd.Save(ctx)
	if err != nil {
		return nil, err
	}

	return mapEntitlementEntity(res), nil
}

func (a *entitlementDBAdapter) DeleteEntitlement(ctx context.Context, entitlementID models.NamespacedID) error {
	affectedCount, err := a.db.Entitlement.Update().
		Where(db_entitlement.ID(entitlementID.ID), db_entitlement.Namespace(entitlementID.Namespace)).
		SetDeletedAt(clock.Now()).
		Save(ctx)
	if err != nil {
		return err
	}
	if affectedCount == 0 {
		return &entitlement.NotFoundError{EntitlementID: entitlementID}
	}
	return nil
}

func (a *entitlementDBAdapter) ListAffectedEntitlements(ctx context.Context, eventFilters []balanceworker.IngestEventQueryFilter) ([]balanceworker.IngestEventDataResponse, error) {
	if len(eventFilters) == 0 {
		return nil, fmt.Errorf("no eventFilters provided")
	}

	query := a.db.Entitlement.Query()

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
}

func (a *entitlementDBAdapter) GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey) ([]entitlement.Entitlement, error) {
	res, err := withLatestUsageReset(a.db.Entitlement.Query()).
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
}

func (a *entitlementDBAdapter) HasEntitlementForMeter(ctx context.Context, namespace string, meterSlug string) (bool, error) {
	exists, err := a.db.Entitlement.Query().
		Where(
			db_entitlement.Namespace(namespace),
			db_entitlement.HasFeatureWith(db_feature.MeterSlugEQ(meterSlug)),
		).
		Exist(ctx)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (a *entitlementDBAdapter) ListEntitlements(ctx context.Context, params entitlement.ListEntitlementsParams) (pagination.PagedResponse[entitlement.Entitlement], error) {
	query := a.db.Entitlement.Query()

	if len(params.Namespaces) > 0 {
		query = query.Where(db_entitlement.NamespaceIn(params.Namespaces...))
	}

	query = withLatestUsageReset(query)

	if len(params.SubjectKeys) > 0 {
		query = query.Where(db_entitlement.SubjectKeyIn(params.SubjectKeys...))
	}

	if len(params.EntitlementTypes) > 0 {
		query = query.Where(db_entitlement.EntitlementTypeIn(slicesx.Map(params.EntitlementTypes, func(t entitlement.EntitlementType) db_entitlement.EntitlementType {
			return db_entitlement.EntitlementType(t)
		})...))
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
			db_entitlement.Or(db_entitlement.DeletedAtIsNil(), db_entitlement.DeletedAtGT(clock.Now())),
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
		urq.Order(db_usagereset.ByResetTime(sql.OrderDesc()))
		urq.Limit(1)
	})
}
