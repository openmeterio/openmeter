package subscriptionaddon

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

const (
	OrderByID        OrderBy = "id"
	OrderByCreatedAt OrderBy = "created_at"
	OrderByUpdatedAt OrderBy = "updated_at"
)

type OrderBy string

func (f OrderBy) Values() []OrderBy {
	return []OrderBy{
		OrderByID,
		OrderByCreatedAt,
		OrderByUpdatedAt,
	}
}

func (f OrderBy) Validate() error {
	if !slices.Contains(f.Values(), f) {
		return models.NewGenericValidationError(fmt.Errorf("invalid subscription addon order by: %s", f))
	}

	return nil
}

type Service interface {
	Create(ctx context.Context, namespace string, input CreateSubscriptionAddonInput) (*SubscriptionAddon, error)
	Get(ctx context.Context, id models.NamespacedID) (*SubscriptionAddon, error)
	List(ctx context.Context, namespace string, input ListSubscriptionAddonsInput) (pagination.Result[SubscriptionAddon], error)

	ChangeQuantity(ctx context.Context, id models.NamespacedID, input CreateSubscriptionAddonQuantityInput) (*SubscriptionAddon, error)
}

type ListSubscriptionAddonsInput struct {
	SubscriptionID string `json:"subscriptionID"`

	OrderBy OrderBy
	Order   sortx.Order

	pagination.Page
}

func (i ListSubscriptionAddonsInput) Validate() error {
	var errs []error

	if i.SubscriptionID == "" {
		errs = append(errs, errors.New("filter has to be provided, all values are empty"))
	}

	if i.OrderBy != "" {
		if err := i.OrderBy.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
