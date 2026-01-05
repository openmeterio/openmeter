package adapter

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	customerdb "github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	customersubjectsdb "github.com/openmeterio/openmeter/openmeter/ent/db/customersubjects"
	db_entitlement "github.com/openmeterio/openmeter/openmeter/ent/db/entitlement"
	db_grant "github.com/openmeterio/openmeter/openmeter/ent/db/grant"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type grantDBADapter struct {
	db *db.Client
}

func NewPostgresGrantRepo(db *db.Client) grant.Repo {
	return &grantDBADapter{
		db: db,
	}
}

func (g *grantDBADapter) CreateGrant(ctx context.Context, grant grant.RepoCreateInput) (*grant.Grant, error) {
	// TODO: transaction and locking
	command := g.db.Grant.Create().
		SetNamespace(grant.Namespace).
		SetOwnerID(grant.OwnerID).
		SetAmount(grant.Amount).
		SetPriority(grant.Priority).
		SetEffectiveAt(grant.EffectiveAt).
		SetExpiration(grant.Expiration).
		SetNillableExpiresAt(grant.ExpiresAt).
		SetAnnotations(grant.Annotations).
		SetMetadata(grant.Metadata).
		SetResetMaxRollover(grant.ResetMaxRollover).
		SetResetMinRollover(grant.ResetMinRollover)

	if grant.Recurrence != nil {
		command = command.
			SetNillableRecurrencePeriod(grant.Recurrence.Interval.ISOStringPtrOrNil()).
			SetNillableRecurrenceAnchor(&grant.Recurrence.Anchor)
	}

	ent, err := command.Save(ctx)
	if err != nil {
		return nil, err
	}

	mapped := mapGrantEntity(ent)

	return &mapped, nil
}

func (g *grantDBADapter) DeleteOwnerGrants(ctx context.Context, ownerID models.NamespacedID) error {
	command := g.db.Grant.Update().
		SetDeletedAt(clock.Now()).
		Where(db_grant.OwnerID(ownerID.ID), db_grant.Namespace(ownerID.Namespace))

	return command.Exec(ctx)
}

// translates to a delete
func (g *grantDBADapter) VoidGrant(ctx context.Context, grantID models.NamespacedID, at time.Time) error {
	// TODO: transaction and locking
	command := g.db.Grant.Update().
		SetVoidedAt(at).
		Where(db_grant.ID(grantID.ID), db_grant.Namespace(grantID.Namespace))
	return command.Exec(ctx)
}

func (g *grantDBADapter) ListGrants(ctx context.Context, params grant.ListParams) (pagination.Result[grant.Grant], error) {
	query := g.db.Grant.Query().Where(db_grant.Namespace(params.Namespace))

	if params.OwnerID != nil {
		query = query.Where(db_grant.OwnerID(*params.OwnerID))
	}

	if !params.IncludeDeleted {
		query = query.Where(
			db_grant.Or(db_grant.DeletedAtIsNil(), db_grant.DeletedAtGT(clock.Now())),
			db_grant.HasEntitlementWith(db_entitlement.Or(
				db_entitlement.DeletedAtIsNil(),
				db_entitlement.DeletedAtGT(clock.Now()),
			)),
		)
	}

	if len(params.CustomerIDs) > 0 {
		query = query.Where(db_grant.HasEntitlementWith(
			db_entitlement.HasCustomerWith(
				customerdb.IDIn(params.CustomerIDs...),
				customerdb.Or(
					customerdb.DeletedAtIsNil(),
					customerdb.DeletedAtGT(clock.Now()),
				),
			),
			db_entitlement.Or(
				db_entitlement.DeletedAtIsNil(),
				db_entitlement.DeletedAtGT(clock.Now()),
			),
		))
	}

	if len(params.SubjectKeys) > 0 {
		query = query.Where(db_grant.HasEntitlementWith(
			db_entitlement.HasCustomerWith(
				customerdb.HasSubjectsWith(
					customersubjectsdb.SubjectKeyIn(params.SubjectKeys...),
					customersubjectsdb.DeletedAtIsNil(),
				),
				customerdb.Or(
					customerdb.DeletedAtIsNil(),
					customerdb.DeletedAtGT(clock.Now()),
				),
			),
		))
	}

	if len(params.FeatureIdsOrKeys) > 0 {
		var ep predicate.Entitlement
		for i, key := range params.FeatureIdsOrKeys {
			p := db_entitlement.Or(db_entitlement.FeatureID(key), db_entitlement.FeatureKey(key))
			if i == 0 {
				ep = p
				continue
			}
			ep = db_entitlement.Or(ep, p)
		}
		query = query.Where(db_grant.HasEntitlementWith(ep))
	}

	if params.OrderBy != "" {
		order := []sql.OrderTermOption{}
		if !params.Order.IsDefaultValue() {
			order = entutils.GetOrdering(params.Order)
		}
		switch params.OrderBy {
		case grant.OrderByCreatedAt:
			query = query.Order(db_grant.ByCreatedAt(order...))
		case grant.OrderByUpdatedAt:
			query = query.Order(db_grant.ByUpdatedAt(order...))
		case grant.OrderByExpiresAt:
			query = query.Order(db_grant.ByExpiresAt(order...))
		case grant.OrderByEffectiveAt:
			query = query.Order(db_grant.ByEffectiveAt(order...))
		case grant.OrderByOwner:
			query = query.Order(db_grant.ByOwnerID(order...))
		}
	}

	response := pagination.Result[grant.Grant]{
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

		grants := make([]grant.Grant, 0, len(entities))
		for _, entity := range entities {
			grants = append(grants, mapGrantEntity(entity))
		}

		response.Items = grants
		return response, nil
	}

	paged, err := query.Paginate(ctx, params.Page)
	if err != nil {
		return response, err
	}

	grants := make([]grant.Grant, 0, len(paged.Items))
	for _, entity := range paged.Items {
		grants = append(grants, mapGrantEntity(entity))
	}

	response.Items = grants
	response.TotalCount = paged.TotalCount

	return response, nil
}

func (g *grantDBADapter) ListActiveGrantsBetween(ctx context.Context, owner models.NamespacedID, from, to time.Time) ([]grant.Grant, error) {
	query := g.db.Grant.Query().
		Where(db_grant.And(db_grant.OwnerID(owner.ID), db_grant.Namespace(owner.Namespace))).
		Where(db_grant.AmountGTE(0.0)). // For a time we allowed negative grant amounts with an undefined behavior, for continuity we just silently ignore them.
		Where(
			db_grant.Or(
				db_grant.And(db_grant.EffectiveAtLT(from), db_grant.Or(db_grant.ExpiresAtGT(from), db_grant.ExpiresAtIsNil())),
				db_grant.And(db_grant.EffectiveAtGTE(from), db_grant.EffectiveAtLT(to)),
				db_grant.EffectiveAt(from),
				db_grant.EffectiveAt(to),
			),
		).Where(
		db_grant.Or(db_grant.Not(db_grant.DeletedAtLT(from)), db_grant.DeletedAtIsNil()),
		db_grant.Or(db_grant.Not(db_grant.VoidedAtLT(from)), db_grant.VoidedAtIsNil()),
	)

	entities, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	grants := make([]grant.Grant, 0, len(entities))
	for _, entity := range entities {
		grants = append(grants, mapGrantEntity(entity))
	}

	return grants, nil
}

func (g *grantDBADapter) GetGrant(ctx context.Context, grantID models.NamespacedID) (grant.Grant, error) {
	ent, err := g.db.Grant.Query().Where(db_grant.ID(grantID.ID), db_grant.Namespace(grantID.Namespace)).Only(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return grant.Grant{}, &credit.GrantNotFoundError{GrantID: grantID.ID}
		}
		return grant.Grant{}, err
	}

	return mapGrantEntity(ent), nil
}

func mapGrantEntity(entity *db.Grant) grant.Grant {
	g := grant.Grant{
		ManagedModel: models.ManagedModel{
			CreatedAt: entity.CreatedAt.In(time.UTC),
			UpdatedAt: entity.UpdatedAt.In(time.UTC),
			DeletedAt: convert.SafeToUTC(entity.DeletedAt),
		},
		NamespacedModel: models.NamespacedModel{
			Namespace: entity.Namespace,
		},
		ID:       entity.ID,
		OwnerID:  entity.OwnerID,
		Amount:   entity.Amount,
		Priority: entity.Priority,
		VoidedAt: convert.SafeDeRef(entity.VoidedAt, func(t time.Time) *time.Time {
			return convert.ToPointer(t.In(time.UTC).Truncate(time.Minute)) // To avoid consistency errors for previous versions of the database where this value wasn't store truncated
		}),
		EffectiveAt:      entity.EffectiveAt,
		Expiration:       entity.Expiration,
		ExpiresAt:        entity.ExpiresAt,
		Annotations:      entity.Annotations,
		Metadata:         entity.Metadata,
		ResetMaxRollover: entity.ResetMaxRollover,
		ResetMinRollover: entity.ResetMinRollover,
	}

	if entity.RecurrencePeriod != nil && entity.RecurrenceAnchor != nil {
		parsed, _ := entity.RecurrencePeriod.Parse()

		g.Recurrence = &timeutil.Recurrence{
			Interval: timeutil.RecurrenceInterval{ISODuration: parsed},
			Anchor:   entity.RecurrenceAnchor.In(time.UTC),
		}
	}

	return g
}
