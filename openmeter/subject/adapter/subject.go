package adapter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	entitlementdb "github.com/openmeterio/openmeter/openmeter/ent/db/entitlement"
	subjectdb "github.com/openmeterio/openmeter/openmeter/ent/db/subject"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// Create creates a subject entity in database
func (a *adapter) Create(ctx context.Context, input subject.CreateInput) (subject.Subject, error) {
	if err := input.Validate(); err != nil {
		return subject.Subject{}, fmt.Errorf("invalid input: %w", models.NewGenericValidationError(err))
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (subject.Subject, error) {
		query := tx.db.Subject.Create().
			SetNamespace(input.Namespace).
			SetKey(input.Key).
			SetNillableDisplayName(input.DisplayName).
			SetNillableStripeCustomerID(input.StripeCustomerId)

		if input.Metadata != nil {
			query.SetMetadata(*input.Metadata)
		}

		subjectEntity, err := query.Save(ctx)
		if err != nil {
			if db.IsConstraintError(err) {
				return subject.Subject{}, models.NewGenericConflictError(
					fmt.Errorf("subject with key already exists: %s", input.Key),
				)
			}

			return subject.Subject{}, fmt.Errorf("failed to create subject: %w", err)
		}

		return mapEntity(subjectEntity), nil
	})
}

// Update creates a subject entity in database
func (a *adapter) Update(ctx context.Context, input subject.UpdateInput) (subject.Subject, error) {
	if err := input.Validate(); err != nil {
		return subject.Subject{}, fmt.Errorf("invalid input: %w", models.NewGenericValidationError(err))
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (subject.Subject, error) {
		query := tx.db.Subject.Update().
			Where(
				subjectdb.Namespace(input.Namespace),
				subjectdb.ID(input.ID),
			)

		// FIXME: this pattern is unique to this adapter, and should be refactored
		// Check if field is present in body
		// Ent doesn't null out fields when nil is provided, so we have to do it manually
		// https://github.com/ent/ent/issues/2108
		if input.DisplayName.IsSet {
			if input.DisplayName.Value != nil {
				query.SetDisplayName(*input.DisplayName.Value)
			} else {
				query.ClearDisplayName()
			}
		}

		if input.Metadata.IsSet {
			if input.Metadata.Value != nil {
				query.SetMetadata(*input.Metadata.Value)
			} else {
				query.ClearMetadata()
			}
		}

		if input.StripeCustomerId.IsSet {
			if input.StripeCustomerId.Value != nil {
				query.SetStripeCustomerID(*input.StripeCustomerId.Value)
			} else {
				query.ClearStripeCustomerID()
			}
		}

		_, err := query.Save(ctx)
		if err != nil {
			return subject.Subject{}, fmt.Errorf("failed to update subject: %w", err)
		}

		// Return the updated entity
		return tx.GetById(ctx, models.NamespacedID{
			Namespace: input.Namespace,
			ID:        input.ID,
		})
	})
}

// Get returns subject entity from database
func (a *adapter) GetByIdOrKey(ctx context.Context, namespace string, idOrKey string) (subject.Subject, error) {
	if namespace == "" {
		return subject.Subject{}, models.NewGenericValidationError(errors.New("namespace is required"))
	}

	if idOrKey == "" {
		return subject.Subject{}, models.NewGenericValidationError(errors.New("id or key is required"))
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (subject.Subject, error) {
		now := clock.Now().UTC()

		sub, err := tx.db.Subject.Query().
			Where(
				subjectdb.Namespace(namespace),
				subjectdb.Or(
					subjectdb.ID(idOrKey),
					subjectdb.And(
						subjectdb.Key(idOrKey),
						subjectdb.Or(
							subjectdb.DeletedAtIsNil(),
							subjectdb.DeletedAtGTE(now),
						),
					),
				),
			).
			First(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return subject.Subject{}, models.NewGenericNotFoundError(
					fmt.Errorf("subject not found [namespace=%s subject.idOrKey=%s]", namespace, idOrKey),
				)
			}

			return subject.Subject{},
				fmt.Errorf("failed to get subject[namespace=%s subject.idOrKey=%s]: %w", namespace, idOrKey, err)
		}

		return mapEntity(sub), nil
	})
}

func (a *adapter) GetByKey(ctx context.Context, key models.NamespacedKey) (subject.Subject, error) {
	if err := key.Validate(); err != nil {
		return subject.Subject{}, fmt.Errorf("invalid key: %w", models.NewGenericValidationError(err))
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (subject.Subject, error) {
		now := clock.Now().UTC()

		sub, err := tx.db.Subject.Query().
			Where(
				subjectdb.Namespace(key.Namespace),
				subjectdb.Key(key.Key),
				subjectdb.Or(
					subjectdb.DeletedAtIsNil(),
					subjectdb.DeletedAtGTE(now),
				),
			).
			First(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return subject.Subject{}, models.NewGenericNotFoundError(
					fmt.Errorf("subject not found [namespace=%s subject.key=%s]", key.Namespace, key.Key),
				)
			}

			return subject.Subject{}, fmt.Errorf("failed to get subject: %w", err)
		}

		return mapEntity(sub), nil
	})
}

func (a *adapter) GetById(ctx context.Context, id models.NamespacedID) (subject.Subject, error) {
	if err := id.Validate(); err != nil {
		return subject.Subject{}, fmt.Errorf("invalid id: %w", models.NewGenericValidationError(err))
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (subject.Subject, error) {
		entity, err := tx.db.Subject.Query().
			Where(
				subjectdb.Namespace(id.Namespace),
				subjectdb.ID(id.ID),
			).
			First(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return subject.Subject{}, models.NewGenericNotFoundError(
					fmt.Errorf("subject not found [namespace=%s subject.id=%s]", id.Namespace, id.ID),
				)
			}

			return subject.Subject{}, fmt.Errorf("failed to get subject: %w", err)
		}

		return mapEntity(entity), nil
	})
}

// List returns all subjects from database for a namespace
func (a *adapter) List(ctx context.Context, namespace string, params subject.ListParams) (pagination.Result[subject.Subject], error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (pagination.Result[subject.Subject], error) {
		now := clock.Now().UTC()

		query := tx.db.Subject.Query().
			Where(
				subjectdb.Namespace(namespace),
				subjectdb.Or(
					subjectdb.DeletedAtIsNil(),
					subjectdb.DeletedAtGTE(now),
				),
			)

		// Filter by keys
		if len(params.Keys) > 0 {
			query = query.Where(subjectdb.KeyIn(params.Keys...))
		}

		// Search by key or display name
		if params.Search != "" {
			query = query.Where(
				subjectdb.Or(
					subjectdb.KeyContainsFold(params.Search),
					subjectdb.DisplayNameContainsFold(params.Search),
				),
			)
		}

		// Sort
		switch params.SortBy {
		case subject.ListSortByKeyAsc:
			query.Order(db.Asc(subjectdb.FieldKey))
		case subject.ListSortByKeyDesc:
			query.Order(db.Desc(subjectdb.FieldKey))
		case subject.ListSortByDisplayNameAsc:
			// Sort by key first to make sure the order is consistent without display name
			query.Order(db.Asc(subjectdb.FieldKey, subjectdb.FieldDisplayName))
		case subject.ListSortByDisplayNameDesc:
			// Results have display name first, then key only in descending order
			query.Order(
				// Make sure null display names are last
				subjectdb.ByDisplayName(
					sql.OrderNullsLast(),
					sql.OrderDesc(),
				),
				subjectdb.ByKey(
					sql.OrderDesc(),
				),
			)
		default:
			query.Order(db.Asc(subjectdb.FieldID))
		}

		result, err := query.Paginate(ctx, params.Page)
		if err != nil {
			return pagination.Result[subject.Subject]{}, fmt.Errorf("failed to list subjects: %w", err)
		}

		return pagination.MapResult(result, mapEntity), nil
	})
}

// DeleteById deletes subject entity from database
// It does not delete usage for subject
func (a *adapter) Delete(ctx context.Context, id models.NamespacedID) error {
	if err := id.Validate(); err != nil {
		return fmt.Errorf("invalid id: %w", models.NewGenericValidationError(err))
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		now := clock.Now().UTC()

		sub, err := tx.db.Subject.Query().
			Where(subjectdb.Namespace(id.Namespace)).
			Where(subjectdb.ID(id.ID)).
			WithEntitlements(func(query *db.EntitlementQuery) {
				query.Where(
					entitlementdb.Namespace(id.Namespace),
					entitlementdb.SubjectID(id.ID),
					entitlementdb.Or(
						entitlementdb.DeletedAtGTE(now),
						entitlementdb.DeletedAtIsNil(),
					),
					entitlementdb.ActiveFromLTE(now),
					entitlementdb.Or(
						entitlementdb.ActiveToGTE(now),
						entitlementdb.ActiveToIsNil(),
					),
				)
			}).
			First(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return models.NewGenericNotFoundError(
					fmt.Errorf("subject not found [namespace=%s subject.id=%s]", id.Namespace, id.ID),
				)
			}

			return fmt.Errorf("failed to get subject: %w", err)
		}

		// Return if Subject is already deleted
		if sub.DeletedAt != nil && sub.DeletedAt.Before(now) {
			return nil
		}

		if len(sub.Edges.Entitlements) > 0 {
			entitlementIDs := lo.FilterMap(sub.Edges.Entitlements, func(item *db.Entitlement, _ int) (string, bool) {
				if item != nil {
					return item.ID, true
				}

				return "", false
			})
			return models.NewGenericPreConditionFailedError(
				fmt.Errorf("subject has active entitlements with ids: %s", strings.Join(entitlementIDs, ", ")),
			)
		}

		return tx.db.Subject.
			Update().
			SetDeletedAt(now).
			Where(
				subjectdb.Namespace(id.Namespace),
				subjectdb.ID(id.ID),
			).
			Exec(ctx)
	})
}

// mapEntity maps subject entity to subject model
func mapEntity(e *db.Subject) subject.Subject {
	s := subject.Subject{
		ManagedModel: models.ManagedModel{
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
			DeletedAt: e.DeletedAt,
		},
		Namespace: e.Namespace,
		Id:        e.ID,
		Key:       e.Key,
		Metadata:  e.Metadata,
	}

	if s.Metadata == nil {
		s.Metadata = make(map[string]interface{})
	}

	if e.DisplayName != nil {
		s.DisplayName = e.DisplayName
	}

	if e.StripeCustomerID != nil {
		s.StripeCustomerId = e.StripeCustomerID
	}

	return s
}
