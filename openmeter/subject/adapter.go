package subject

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Adapter interface {
	Create(ctx context.Context, input CreateInput) (Subject, error)
	Update(ctx context.Context, input UpdateInput) (Subject, error)
	GetByIdOrKey(ctx context.Context, namespace string, idOrKey string) (Subject, error)
	GetByKey(ctx context.Context, key models.NamespacedKey) (Subject, error)
	GetById(ctx context.Context, id models.NamespacedID) (Subject, error)
	List(ctx context.Context, namespace string, params ListParams) (pagination.Result[Subject], error)
	Delete(ctx context.Context, id models.NamespacedID) error

	entutils.TxCreator
}

type GetSubjectAdapterInput struct {
	Namespace string
	ID        string
	Key       string
}

func (i GetSubjectAdapterInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.ID == "" && i.Key == "" {
		return errors.New("id or key is required")
	}

	return nil
}
