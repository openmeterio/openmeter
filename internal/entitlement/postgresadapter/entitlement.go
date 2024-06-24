package postgresadapter

import (
	"context"
	"strings"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/internal/entitlement/postgresadapter/ent/db"
	db_entitlement "github.com/openmeterio/openmeter/internal/entitlement/postgresadapter/ent/db/entitlement"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
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
	res, err := a.db.Entitlement.Query().
		Where(
			db_entitlement.ID(entitlementID.ID),
			db_entitlement.Namespace(entitlementID.Namespace),
		).
		First(ctx)

	if err != nil {
		if db.IsNotFound(err) {
			return nil, &entitlement.EntitlementNotFoundError{EntitlementID: entitlementID}
		}
		return nil, err
	}

	return mapEntitlementEntity(res), nil
}

func (a *entitlementDBAdapter) CreateEntitlement(ctx context.Context, entitlement entitlement.EntitlementRepoCreateEntitlementInputs) (*entitlement.Entitlement, error) {
	res, err := a.db.Entitlement.Create().
		SetNamespace(entitlement.Namespace).
		SetFeatureID(entitlement.FeatureID).
		SetSubjectKey(entitlement.SubjectKey).
		SetMeasureUsageFrom(entitlement.MeasureUsageFrom).
		SetUsagePeriodAnchor(entitlement.UsagePeriod.Anchor).
		SetUsagePeriodInterval(db_entitlement.UsagePeriodInterval(entitlement.UsagePeriod.Period)).
		SetUsagePeriodNextReset(entitlement.UsagePeriod.NextReset).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return mapEntitlementEntity(res), nil
}

func (a *entitlementDBAdapter) UpdateEntitlementUsagePeriod(ctx context.Context, entitlementID models.NamespacedID, newAnchor *time.Time, nextReset time.Time) error {
	update := a.db.Entitlement.UpdateOneID(entitlementID.ID).
		SetUsagePeriodNextReset(nextReset)

	if newAnchor != nil {
		update = update.SetUsagePeriodAnchor(*newAnchor)
	}

	_, err := update.Save(ctx)
	return err
}

func (a *entitlementDBAdapter) GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey) ([]entitlement.Entitlement, error) {
	res, err := a.db.Entitlement.Query().
		Where(
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
	query := a.db.Entitlement.Query().
		Where(db_entitlement.Namespace(params.Namespace))

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

func (a *entitlementDBAdapter) ListEntitlementsWithDueReset(ctx context.Context, namespace string, at time.Time) ([]entitlement.Entitlement, error) {
	entities, err := a.db.Entitlement.Query().
		Where(
			db_entitlement.Namespace(namespace),
			db_entitlement.UsagePeriodNextResetLTE(at),
		).
		All(ctx)

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
	return &entitlement.Entitlement{
		NamespacedModel: models.NamespacedModel{
			Namespace: e.Namespace,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
			DeletedAt: convert.SafeToUTC(e.DeletedAt),
		},
		ID:               e.ID,
		SubjectKey:       e.SubjectKey,
		FeatureID:        e.FeatureID,
		MeasureUsageFrom: e.MeasureUsageFrom,
		UsagePeriod: entitlement.RecurrenceWithNextReset{
			Recurrence: entitlement.Recurrence{
				Period: credit.RecurrencePeriod(e.UsagePeriodInterval),
				Anchor: e.UsagePeriodAnchor,
			},
			NextReset: e.UsagePeriodNextReset,
		},
	}
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
			return &entitlement.EntitlementNotFoundError{
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
