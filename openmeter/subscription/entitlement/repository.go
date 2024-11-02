package subscriptionentitlement

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbentitlement "github.com/openmeterio/openmeter/openmeter/ent/db/entitlement"
	dbsubscriptionentitlement "github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionentitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionEntitlement struct {
	models.ManagedResource
	subscription.SubscriptionItemRef
	EntitlementId string `json:"entitlementId"`
}

type CreateSubscriptionEntitlementInput struct {
	Namespace           string
	EntitlementId       string
	SubscriptionItemRef subscription.SubscriptionItemRef
}

type Repository interface {
	Create(ctx context.Context, ent CreateSubscriptionEntitlementInput) (*SubscriptionEntitlement, error)
	Get(ctx context.Context, id string) (SubscriptionEntitlement, error)
	Delete(ctx context.Context, id string) error
	// The `at` time reffers to the active time of the entitlement
	GetBySubscriptionItem(ctx context.Context, namespace string, ref subscription.SubscriptionItemRef, at time.Time) (SubscriptionEntitlement, error)
	GetForSubscription(ctx context.Context, subscriptionId models.NamespacedID, at time.Time) ([]SubscriptionEntitlement, error)
}

type repository struct {
	db *db.Client
}

var _ Repository = &repository{}

func NewRepository(db *db.Client) *repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, ent CreateSubscriptionEntitlementInput) (*SubscriptionEntitlement, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *repository) (*SubscriptionEntitlement, error) {
		ent, err := repo.db.SubscriptionEntitlement.Create().
			SetNamespace(ent.Namespace).
			SetEntitlementID(ent.EntitlementId).
			SetSubscriptionID(ent.SubscriptionItemRef.SubscriptionId).
			SetSubscriptionPhaseKey(ent.SubscriptionItemRef.PhaseKey).
			SetSubscriptionItemKey(ent.SubscriptionItemRef.ItemKey).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create subscription entitlement: %w", err)
		}

		return mapDBSubscriptionEntitlementToSubscriptionEntitlement(ent), nil
	})
}

func (r *repository) Delete(ctx context.Context, id string) error {
	_, err := entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *repository) (any, error) {
		err := repo.db.SubscriptionEntitlement.DeleteOneID(id).Exec(ctx)
		if db.IsNotFound(err) {
			return nil, &NotFoundError{ID: id}
		}
		if err != nil {
			return nil, fmt.Errorf("failed to delete subscription entitlement: %w", err)
		}

		return nil, nil
	})
	return err
}

func (r *repository) Get(ctx context.Context, id string) (SubscriptionEntitlement, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *repository) (SubscriptionEntitlement, error) {
		ent, err := repo.db.SubscriptionEntitlement.Get(ctx, id)
		if db.IsNotFound(err) {
			return SubscriptionEntitlement{}, &NotFoundError{ID: id}
		}
		if err != nil {
			return SubscriptionEntitlement{}, fmt.Errorf("failed to get subscription entitlement: %w", err)
		}

		sE := mapDBSubscriptionEntitlementToSubscriptionEntitlement(ent)
		return *sE, nil
	})
}

func (r *repository) GetBySubscriptionItem(ctx context.Context, namespace string, ref subscription.SubscriptionItemRef, at time.Time) (SubscriptionEntitlement, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *repository) (SubscriptionEntitlement, error) {
		ent, err := repo.db.SubscriptionEntitlement.Query().
			Where(
				dbsubscriptionentitlement.HasEntitlementWith(
					// Entitlement Activity is NOT relevant here because we fetch all entitlements: past, present and future.
					dbentitlement.Or(
						dbentitlement.DeletedAtIsNil(),
						dbentitlement.DeletedAtGT(at),
					),
				),
			).
			Where(
				dbsubscriptionentitlement.Or(
					dbsubscriptionentitlement.DeletedAtIsNil(),
					dbsubscriptionentitlement.DeletedAtGT(at),
				),
			).
			Where(
				dbsubscriptionentitlement.Namespace(namespace),
				dbsubscriptionentitlement.SubscriptionID(ref.SubscriptionId),
				dbsubscriptionentitlement.SubscriptionPhaseKey(ref.PhaseKey),
				dbsubscriptionentitlement.SubscriptionItemKey(ref.ItemKey),
			).Only(ctx)

		if db.IsNotFound(err) {
			return SubscriptionEntitlement{}, &NotFoundError{ItemRef: ref, At: at}
		}
		if db.IsNotSingular(err) {
			return SubscriptionEntitlement{}, fmt.Errorf("failed to get subscription entitlement, found more than one results for %+v: %w", map[string]any{
				"ref": ref,
				"at":  at,
			}, err)
		}

		if err != nil {
			return SubscriptionEntitlement{}, fmt.Errorf("failed to get subscription entitlement: %w", err)
		}

		sE := mapDBSubscriptionEntitlementToSubscriptionEntitlement(ent)
		return *sE, nil
	})
}

func (r *repository) GetForSubscription(ctx context.Context, subscriptionId models.NamespacedID, at time.Time) ([]SubscriptionEntitlement, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *repository) ([]SubscriptionEntitlement, error) {
		ents, err := repo.db.SubscriptionEntitlement.Query().
			Where(
				dbsubscriptionentitlement.HasEntitlementWith(
					// Entitlement Activity is NOT relevant here because we fetch all entitlements: past, present and future.
					dbentitlement.Or(
						dbentitlement.DeletedAtIsNil(),
						dbentitlement.DeletedAtGT(at),
					),
				),
			).
			Where(
				dbsubscriptionentitlement.Or(
					dbsubscriptionentitlement.DeletedAtIsNil(),
					dbsubscriptionentitlement.DeletedAtGT(at),
				),
			).
			Where(
				dbsubscriptionentitlement.SubscriptionID(subscriptionId.ID),
				dbsubscriptionentitlement.Namespace(subscriptionId.Namespace),
			).All(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get subscription entitlements: %w", err)
		}

		var sEs []SubscriptionEntitlement
		for _, ent := range ents {
			if ent == nil {
				return nil, fmt.Errorf("unexpected nil subscription entitlement")
			}
			sE := mapDBSubscriptionEntitlementToSubscriptionEntitlement(ent)

			if sE == nil {
				return nil, fmt.Errorf("unexpected nil subscription entitlement after mapping")
			}

			sEs = append(sEs, *sE)
		}

		return sEs, nil
	})
}

func (r *repository) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := r.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (r *repository) WithTx(ctx context.Context, tx *entutils.TxDriver) *repository {
	txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return NewRepository(txClient.Client())
}

func mapDBSubscriptionEntitlementToSubscriptionEntitlement(ent *db.SubscriptionEntitlement) *SubscriptionEntitlement {
	if ent == nil {
		return nil
	}
	return &SubscriptionEntitlement{
		ManagedResource: models.ManagedResource{
			ID: ent.ID,
			NamespacedModel: models.NamespacedModel{
				Namespace: ent.Namespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: ent.CreatedAt.UTC(),
				UpdatedAt: ent.UpdatedAt.UTC(),
				DeletedAt: convert.SafeToUTC(ent.DeletedAt),
			},
		},
		SubscriptionItemRef: subscription.SubscriptionItemRef{
			SubscriptionId: ent.SubscriptionID,
			PhaseKey:       ent.SubscriptionPhaseKey,
			ItemKey:        ent.SubscriptionItemKey,
		},
		EntitlementId: ent.EntitlementID,
	}
}
