package adapter

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	entsubject "github.com/openmeterio/openmeter/openmeter/ent/db/subject"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// Create creates a subject entity in database
func (a adapter) Create(ctx context.Context, input subject.CreateInput) (*subject.Subject, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", models.NewGenericValidationError(err))
	}

	query := a.ent.Subject.Create().
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
			return nil, models.NewGenericConflictError(
				fmt.Errorf("subject with key already exists: %s", input.Key),
			)
		}

		return nil, fmt.Errorf("failed to create subject: %w", err)
	}

	return mapEntity(subjectEntity), nil
}

// Update creates a subject entity in database
func (a adapter) Update(ctx context.Context, input subject.UpdateInput) (*subject.Subject, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", models.NewGenericValidationError(err))
	}

	query := a.ent.Subject.Update().
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
		return nil, fmt.Errorf("failed to update subject: %w", err)
	}

	// Get the updated entity
	sub, err := a.GetByIdOrKey(ctx, input.Namespace, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated subject: %w", err)
	}

	return sub, nil
}

// GetByIdOrKey returns subject entity from database
func (a adapter) GetByIdOrKey(ctx context.Context, namespace string, idOrKey string) (*subject.Subject, error) {
	entity, err := a.ent.Subject.Query().
		Where(
			entsubject.Namespace(namespace),
			entsubject.Or(
				entsubject.ID(idOrKey),
				entsubject.Key(idOrKey),
			),
		).
		Only(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, models.NewGenericNotFoundError(
				fmt.Errorf("subject not found: %s", idOrKey),
			)
		}

		return nil, fmt.Errorf("failed to get subject: %w", err)
	}

	return mapEntity(entity), err
}

// List returns all subjects from database for a namespace
func (a adapter) List(ctx context.Context, namespace string, params subject.ListParams) (pagination.PagedResponse[*subject.Subject], error) {
	query := a.ent.Subject.Query().
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
		return pagination.PagedResponse[*subject.Subject]{}, fmt.Errorf("failed to list subjects: %w", err)
	}

	return pagination.MapPagedResponse(result, mapEntity), nil
}

// DeleteById deletes subject entity from database
// It does not delete usage for subject
func (a adapter) DeleteById(ctx context.Context, id string) error {
	return a.ent.Subject.DeleteOneID(id).Exec(ctx)
}

// mapEntity maps subject entity to subject model
func mapEntity(subjectEntity *db.Subject) *subject.Subject {
	subject := subject.Subject{
		Id:       subjectEntity.ID,
		Key:      subjectEntity.Key,
		Metadata: subjectEntity.Metadata,
	}

	if subject.Metadata == nil {
		subject.Metadata = make(models.Metadata)
	}

	if subjectEntity.DisplayName != nil {
		subject.DisplayName = subjectEntity.DisplayName
	}

	if subjectEntity.StripeCustomerID != nil {
		subject.StripeCustomerId = subjectEntity.StripeCustomerID
	}

	return &subject
}
