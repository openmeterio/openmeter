package postgresadapter

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgresadapter/ent/db"
	db_grant "github.com/openmeterio/openmeter/internal/credit/postgresadapter/ent/db/grant"
	"github.com/openmeterio/openmeter/pkg/models"
)

type grantDBADapter struct {
	db *db.Client
}

func NewPostgresGrantDBAdapter(db *db.Client) credit.GrantDBConnector {
	return &grantDBADapter{
		db: db,
	}
}

func (g *grantDBADapter) CreateGrant(ctx context.Context, grant credit.DBCreateGrantInput) (credit.Grant, error) {
	// TODO: transaction and locking
	command := g.db.Grant.Create().
		SetOwnerID(grant.OwnerID).
		SetNamespace(grant.Namespace).
		SetAmount(grant.Amount).
		SetPriority(grant.Priority).
		SetEffectiveAt(grant.EffectiveAt).
		SetExpiration(grant.Expiration).
		SetExpiresAt(grant.ExpiresAt).
		SetMetadata(grant.Metadata).
		SetResetMaxRollover(grant.ResetMaxRollover)

	if grant.Recurrence != nil {
		command = command.
			SetNillableRecurrenceMaxRollover(&grant.Recurrence.MaxRolloverAmount).
			SetNillableRecurrencePeriod(&grant.Recurrence.Period).
			SetNillableRecurrenceAnchor(&grant.Recurrence.Anchor)
	}

	ent, err := command.Save(ctx)
	if err != nil {
		return credit.Grant{}, err
	}

	return mapGrantEntity(ent), nil
}

// translates to a delete
func (g *grantDBADapter) VoidGrant(ctx context.Context, grantID models.NamespacedID) error {
	// TODO: transaction and locking
	command := g.db.Grant.Update().Where(db_grant.ID(string(grantID.ID)), db_grant.Namespace(grantID.Namespace)).Mutation().Client().Grant.Create().SetVoidedAt(time.Now())
	return command.Exec(ctx)
}

func (g *grantDBADapter) ListGrants(ctx context.Context, params credit.ListGrantsParams) ([]credit.Grant, error) {
	query := g.db.Grant.Query().Where(db_grant.Namespace(params.Namespace))

	if params.OwnerID != nil {
		query = query.Where(db_grant.OwnerID(*params.OwnerID))
	}

	if !params.IncludeDeleted {
		query = query.Where(db_grant.DeletedAtIsNil())
	}

	if params.OrderBy != "" {
		switch params.OrderBy {
		case credit.GrantOrderByCreatedAt:
			query = query.Order(db_grant.ByCreatedAt())
		case credit.GrantOrderByUpdatedAt:
			query = query.Order(db_grant.ByUpdatedAt())
		case credit.GrantOrderByExpiresAt:
			query = query.Order(db_grant.ByExpiresAt())
		case credit.GrantOrderByEffectiveAt:
			query = query.Order(db_grant.ByEffectiveAt())
		case credit.GrantOrderByOwner:
			query = query.Order(db_grant.ByOwnerID())
		}
	}

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}

	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	entities, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	grants := make([]credit.Grant, 0, len(entities))
	for _, entity := range entities {
		grants = append(grants, mapGrantEntity(entity))
	}

	return grants, nil
}

func (g *grantDBADapter) ListActiveGrantsBetween(ctx context.Context, owner credit.NamespacedGrantOwner, from, to time.Time) ([]credit.Grant, error) {
	query := g.db.Grant.Query().
		Where(db_grant.And(db_grant.OwnerID(owner.ID), db_grant.Namespace(owner.Namespace))).
		Where(
			db_grant.Or(
				db_grant.EffectiveAtLT(to),
				db_grant.ExpiresAtGT(from),
			),
		).Where(
		db_grant.DeletedAtGTE(to),
		db_grant.VoidedAtGTE(to),
	)

	entities, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	grants := make([]credit.Grant, 0, len(entities))
	for _, entity := range entities {
		grants = append(grants, mapGrantEntity(entity))
	}

	return grants, nil
}

func (g *grantDBADapter) GetGrant(ctx context.Context, grantID models.NamespacedID) (credit.Grant, error) {
	ent, err := g.db.Grant.Query().Where(db_grant.ID(string(grantID.ID)), db_grant.Namespace(grantID.Namespace)).Only(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return credit.Grant{}, &credit.GrantNotFoundError{GrantID: grantID.ID}
		}
		return credit.Grant{}, err
	}

	return mapGrantEntity(ent), nil
}

func mapGrantEntity(entity *db.Grant) credit.Grant {
	g := credit.Grant{
		ManagedModel: models.ManagedModel{
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
		},
		NamespacedModel: models.NamespacedModel{
			Namespace: entity.Namespace,
		},
		ID:               entity.ID,
		OwnerID:          credit.GrantOwner(entity.OwnerID),
		Amount:           entity.Amount,
		Priority:         entity.Priority,
		VoidedAt:         entity.VoidedAt,
		EffectiveAt:      entity.EffectiveAt,
		Expiration:       entity.Expiration,
		ExpiresAt:        entity.ExpiresAt,
		Metadata:         entity.Metadata,
		ResetMaxRollover: entity.ResetMaxRollover,
	}

	if entity.RecurrencePeriod != nil && entity.RecurrenceAnchor != nil {
		g.Recurrence = &credit.Recurrence{
			Period: *entity.RecurrencePeriod,
			Anchor: *entity.RecurrenceAnchor,
		}

		if entity.RecurrenceMaxRollover != nil {
			g.Recurrence.MaxRolloverAmount = *entity.RecurrenceMaxRollover
		}
	}

	return g
}
