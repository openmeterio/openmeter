package postgresadapter

import (
	"context"
	"strings"

	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/internal/entitlement/postgresadapter/ent/db"
	db_entitlement "github.com/openmeterio/openmeter/internal/entitlement/postgresadapter/ent/db/entitlement"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
)

type entitlementDBAdapter struct {
	db *db.Client
}

func NewPostgresEntitlementDBAdapter(db *db.Client) entitlement.EntitlementDBConnector {
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

func (a *entitlementDBAdapter) CreateEntitlement(ctx context.Context, entitlement entitlement.CreateEntitlementInputs) (*entitlement.Entitlement, error) {
	res, err := a.db.Entitlement.Create().
		SetNamespace(entitlement.Namespace).
		SetFeatureID(entitlement.FeatureID).
		SetMeasureUsageFrom(entitlement.MeasureUsageFrom).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return mapEntitlementEntity(res), nil
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
		FeatureID:        e.FeatureID,
		MeasureUsageFrom: e.MeasureUsageFrom,
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
