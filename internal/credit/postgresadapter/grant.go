package postgresadapter

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgresadapter/ent/db"
	db_grant "github.com/openmeterio/openmeter/internal/credit/postgresadapter/ent/db/grant"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

type grantDBADapter struct {
	db *db.Client
}

func NewPostgresGrantRepo(db *db.Client) credit.GrantRepo {
	return &grantDBADapter{
		db: db,
	}
}

func (g *grantDBADapter) CreateGrant(ctx context.Context, grant credit.GrantRepoCreateGrantInput) (*credit.Grant, error) {
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
		SetResetMaxRollover(grant.ResetMaxRollover).
		SetResetMinRollover(grant.ResetMinRollover)

	if grant.Recurrence != nil {
		command = command.
			SetNillableRecurrencePeriod(&grant.Recurrence.Interval).
			SetNillableRecurrenceAnchor(&grant.Recurrence.Anchor)
	}

	ent, err := command.Save(ctx)
	if err != nil {
		return nil, err
	}

	mapped := mapGrantEntity(ent)

	return &mapped, nil
}

// translates to a delete
func (g *grantDBADapter) VoidGrant(ctx context.Context, grantID models.NamespacedID, at time.Time) error {
	// TODO: transaction and locking
	command := g.db.Grant.Update().
		SetVoidedAt(at).
		Where(db_grant.ID(string(grantID.ID)), db_grant.Namespace(grantID.Namespace))
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
		Where(
			db_grant.And(
				// Filter by owner, namespace
				db_grant.OwnerID(owner.ID),
				db_grant.Namespace(owner.Namespace),
				// We are not interested in grants that either expired before the period
				db_grant.Not(db_grant.EffectiveAtGT(to)),
				// We are not interested in grants that are not yet effective
				db_grant.Not(db_grant.ExpiresAtLT(from)),
				// We are not interested in deleted or voided grants
				db_grant.Or(db_grant.DeletedAtGTE(to), db_grant.DeletedAtIsNil()),
				db_grant.Or(db_grant.VoidedAtGTE(to), db_grant.VoidedAtIsNil()),
			),
		)

	entities, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	grants := make([]credit.Grant, 0, len(entities))
	for _, entity := range entities {
		// Let's prefilter these as they are not really active, but will make the engine
		// think they are active for that single point in time.
		if entity.EffectiveAt.Equal(entity.ExpiresAt) {
			continue
		}

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
			CreatedAt: entity.CreatedAt.In(time.UTC),
			UpdatedAt: entity.UpdatedAt.In(time.UTC),
			DeletedAt: convert.SafeToUTC(entity.DeletedAt),
		},
		NamespacedModel: models.NamespacedModel{
			Namespace: entity.Namespace,
		},
		ID:               entity.ID,
		OwnerID:          credit.GrantOwner(entity.OwnerID),
		Amount:           entity.Amount,
		Priority:         entity.Priority,
		VoidedAt:         convert.SafeToUTC(entity.VoidedAt),
		EffectiveAt:      entity.EffectiveAt,
		Expiration:       entity.Expiration,
		ExpiresAt:        entity.ExpiresAt,
		Metadata:         entity.Metadata,
		ResetMaxRollover: entity.ResetMaxRollover,
		ResetMinRollover: entity.ResetMinRollover,
	}

	if entity.RecurrencePeriod != nil && entity.RecurrenceAnchor != nil {
		g.Recurrence = &recurrence.Recurrence{
			Interval: *entity.RecurrencePeriod,
			Anchor:   entity.RecurrenceAnchor.In(time.UTC),
		}
	}

	return g
}
