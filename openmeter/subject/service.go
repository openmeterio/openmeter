package subject

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Service interface {
	Create(ctx context.Context, input CreateInput) (*Subject, error)
	Update(ctx context.Context, input UpdateInput) (*Subject, error)
	GetByIdOrKey(ctx context.Context, orgId string, idOrKey string) (*Subject, error)
	GetByKeyWithFallback(ctx context.Context, namespacedKey models.NamespacedKey) (Subject, error)
	List(ctx context.Context, orgId string, params ListParams) (pagination.PagedResponse[*Subject], error)
	DeleteById(ctx context.Context, id string) error
}

type ListSortBy string

const (
	ListSortByKeyAsc          ListSortBy = "key_asc"
	ListSortByKeyDesc         ListSortBy = "key_desc"
	ListSortByDisplayNameAsc  ListSortBy = "display_name_asc"
	ListSortByDisplayNameDesc ListSortBy = "display_name_desc"
)

type ListParams struct {
	Keys   []string
	Search string
	Page   pagination.Page
	SortBy ListSortBy
}

// CreateInput is the input for creating a subject.
type CreateInput struct {
	Namespace        string
	Key              string
	DisplayName      *string
	StripeCustomerId *string
	Metadata         *map[string]string
}

func (i CreateInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.Key == "" {
		errs = append(errs, errors.New("key is required"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// UpdateInput is the input for updating a subject.
type UpdateInput struct {
	ID               string
	Namespace        string
	DisplayName      OptionalNullable[string]
	StripeCustomerId OptionalNullable[string]
	Metadata         OptionalNullable[map[string]string]
}

func (i UpdateInput) Validate() error {
	var errs []error

	if i.ID == "" {
		errs = append(errs, errors.New("id is required"))
	}

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// FIXME: this pattern is unique to this adapter, and should not be reused anywhere else
// OptionalNullable is a helper to differentiate between a nil value and a value that is not set.
type OptionalNullable[T any] struct {
	Value *T
	IsSet bool
}
