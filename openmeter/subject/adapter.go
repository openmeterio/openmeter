package subject

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Adapter interface {
	Create(ctx context.Context, input CreateInput) (*Subject, error)
	Update(ctx context.Context, input UpdateInput) (*Subject, error)
	GetByIdOrKey(ctx context.Context, orgId string, idOrKey string) (*Subject, error)
	List(ctx context.Context, orgId string, params ListParams) (pagination.PagedResponse[*Subject], error)
	DeleteById(ctx context.Context, id string) error
}
