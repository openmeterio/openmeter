package postgresadapter

import (
	"context"
	"encoding/json"
	"strings"
	"time"

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
			return nil, &entitlement.NotFoundError{EntitlementID: entitlementID}
		}
		return nil, err
	}

	return mapEntitlementEntity(res), nil
}

func (a *entitlementDBAdapter) GetEntitlementOfSubject(ctx context.Context, namespace string, subjectKey string, id string) (*entitlement.Entitlement, error) {
	res, err := a.db.Entitlement.Query().
		Where(
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

func (a *entitlementDBAdapter) CreateEntitlement(ctx context.Context, entitlement entitlement.CreateEntitlementInputs) (*entitlement.Entitlement, error) {
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
		ent.GenericProperties.UsagePeriod = &entitlement.UsagePeriod{
			Anchor:   e.UsagePeriodAnchor.In(time.UTC),
			Interval: entitlement.UsagePeriodInterval(*e.UsagePeriodInterval),
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
