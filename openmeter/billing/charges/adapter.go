package charges

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Adapter interface {
	ChargesSearchAdapter

	entutils.TxCreator
}

type ChargesSearchAdapter interface {
	GetTypesByIDs(ctx context.Context, input GetTypesByIDsInput) (GetTypesByIDsResult, error)
	ListCharges(ctx context.Context, input ListChargesInput) (pagination.Result[ChargeWithType], error)
}

type ChargeWithType struct {
	ID   string
	Type meta.ChargeType
}

type GetTypesByIDsResult []ChargeWithType

type GetTypesByIDsInput struct {
	Namespace string
	IDs       []string
}

func (i GetTypesByIDsInput) Validate() error {
	var errs []error
	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	for _, id := range i.IDs {
		if id == "" {
			errs = append(errs, errors.New("id is required"))
		}
	}

	return errors.Join(errs...)
}
