package creditgrant

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type NoopService struct{}

var _ Service = NoopService{}

func NewNoopService() Service {
	return NoopService{}
}

func (NoopService) Create(context.Context, CreateInput) (creditpurchase.Charge, error) {
	return creditpurchase.Charge{}, nil
}

func (NoopService) Get(context.Context, GetInput) (creditpurchase.Charge, error) {
	return creditpurchase.Charge{}, nil
}

func (NoopService) List(context.Context, ListInput) (pagination.Result[creditpurchase.Charge], error) {
	return pagination.Result[creditpurchase.Charge]{}, nil
}
