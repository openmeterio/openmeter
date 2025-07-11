package adapter

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	entsubject "github.com/openmeterio/openmeter/openmeter/ent/db/subject"
	"github.com/openmeterio/openmeter/openmeter/subject"
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
			Where(entsubject.Namespace(input.Namespace)).
			Where(entsubject.ID(input.ID))

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

		// Get the updated entity
		sub, err := a.GetByIdOrKey(ctx, input.Namespace, input.ID)
		if err != nil {
			return subject.Subject{}, fmt.Errorf("failed to get updated subject: %w", err)
		}

		return sub, nil
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
		entity, err := tx.db.Subject.Query().
			Where(
				entsubject.Namespace(namespace),
				entsubject.Or(
					entsubject.ID(idOrKey),
					entsubject.Key(idOrKey),
				),
			).
			All(ctx)
		if err != nil {
			return subject.Subject{}, fmt.Errorf("failed to get subject: %w", err)
		}

		if len(entity) == 0 {
			return subject.Subject{}, models.NewGenericNotFoundError(
				fmt.Errorf("subject not found: %s", idOrKey),
			)
		}

		// Let's be deterministic regarding always preferring the ID match over the key match
		if len(entity) > 1 {
			subjectByID, found := lo.Find(entity, func(item *db.Subject) bool {
				return item.ID == idOrKey
			})

			if found {
				return mapEntity(subjectByID), nil
			}

			subjectByKey, found := lo.Find(entity, func(item *db.Subject) bool {
				return item.Key == idOrKey
			})

			if found {
				return mapEntity(subjectByKey), nil
			}

			return subject.Subject{}, models.NewGenericNotFoundError(
				fmt.Errorf("subject not found: %s", idOrKey),
			)
		}

		return mapEntity(entity[0]), nil
	})
}

func (a *adapter) GetByKey(ctx context.Context, key models.NamespacedKey) (subject.Subject, error) {
	if err := key.Validate(); err != nil {
		return subject.Subject{}, fmt.Errorf("invalid key: %w", models.NewGenericValidationError(err))
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (subject.Subject, error) {
		entity, err := tx.db.Subject.Query().
			Where(entsubject.Namespace(key.Namespace)).
			Where(entsubject.Key(key.Key)).
			All(ctx)
		if err != nil {
			return subject.Subject{}, fmt.Errorf("failed to get subject: %w", err)
		}

		if len(entity) == 0 {
			return subject.Subject{}, models.NewGenericNotFoundError(
				fmt.Errorf("subject not found: %s", key.Key),
			)
		}

		return mapEntity(entity[0]), nil
	})
}

func (a *adapter) GetById(ctx context.Context, id models.NamespacedID) (subject.Subject, error) {
	if err := id.Validate(); err != nil {
		return subject.Subject{}, fmt.Errorf("invalid id: %w", models.NewGenericValidationError(err))
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (subject.Subject, error) {
		entity, err := tx.db.Subject.Query().
			Where(entsubject.Namespace(id.Namespace)).
			Where(entsubject.ID(id.ID)).
			All(ctx)
		if err != nil {
			return subject.Subject{}, fmt.Errorf("failed to get subject: %w", err)
		}

		if len(entity) == 0 {
			return subject.Subject{}, models.NewGenericNotFoundError(
				fmt.Errorf("subject not found: %s", id.ID),
			)
		}

		return mapEntity(entity[0]), nil
	})
}

// List returns all subjects from database for a namespace
func (a *adapter) List(ctx context.Context, namespace string, params subject.ListParams) (pagination.PagedResponse[subject.Subject], error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (pagination.PagedResponse[subject.Subject], error) {
		query := tx.db.Subject.Query().
			Where(entsubject.Namespace(namespace))

		// Filter by keys
		if len(params.Keys) > 0 {
			query = query.Where(entsubject.KeyIn(params.Keys...))
		}

		// Search by key or display name
		if params.Search != "" {
			query = query.Where(
				entsubject.Or(
					entsubject.KeyContainsFold(params.Search),
					entsubject.DisplayNameContainsFold(params.Search),
				),
			)
		}

		// Sort
		switch params.SortBy {
		case subject.ListSortByKeyAsc:
			query.Order(db.Asc(entsubject.FieldKey))
		case subject.ListSortByKeyDesc:
			query.Order(db.Desc(entsubject.FieldKey))
		case subject.ListSortByDisplayNameAsc:
			// Sort by key first to make sure the order is consistent without display name
			query.Order(db.Asc(entsubject.FieldKey, entsubject.FieldDisplayName))
		case subject.ListSortByDisplayNameDesc:
			// Results have display name first, then key only in descending order
			query.Order(
				// Make sure null display names are last
				entsubject.ByDisplayName(
					sql.OrderNullsLast(),
					sql.OrderDesc(),
				),
				entsubject.ByKey(
					sql.OrderDesc(),
				),
			)
		default:
			query.Order(db.Asc(entsubject.FieldKey, entsubject.FieldDisplayName))
		}

		result, err := query.Paginate(ctx, params.Page)
		if err != nil {
			return pagination.PagedResponse[subject.Subject]{}, fmt.Errorf("failed to list subjects: %w", err)
		}

		return pagination.MapPagedResponse(result, mapEntity), nil
	})
}

// DeleteById deletes subject entity from database
// It does not delete usage for subject
func (a *adapter) Delete(ctx context.Context, id models.NamespacedID) error {
	if err := id.Validate(); err != nil {
		return fmt.Errorf("invalid id: %w", models.NewGenericValidationError(err))
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		_, err := tx.db.Subject.Delete().Where(
			entsubject.Namespace(id.Namespace),
			entsubject.ID(id.ID),
		).Exec(ctx)
		return err
	})
}

// mapEntity maps subject entity to subject model
func mapEntity(subjectEntity *db.Subject) subject.Subject {
	s := subject.Subject{
		Namespace: subjectEntity.Namespace,
		Id:        subjectEntity.ID,
		Key:       subjectEntity.Key,
		Metadata:  subjectEntity.Metadata,
	}

	if s.Metadata == nil {
		s.Metadata = make(map[string]interface{})
	}

	if subjectEntity.DisplayName != nil {
		s.DisplayName = subjectEntity.DisplayName
	}

	if subjectEntity.StripeCustomerID != nil {
		s.StripeCustomerId = subjectEntity.StripeCustomerID
	}

	return s
}
