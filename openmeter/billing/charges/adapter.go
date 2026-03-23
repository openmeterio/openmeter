package charges

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Adapter interface {
	ChargesSearchAdapter

	entutils.TxCreator
}

type ChargesSearchAdapter interface {
	GetTypesByIDs(ctx context.Context, ids meta.ChargeIDs) (GetTypesByIDsResult, error)
	ListCharges(ctx context.Context, input ListChargesInput) (pagination.Result[ChargeWithType], error)
}

type ChargeWithType struct {
	ChargeID meta.ChargeID
	Type     meta.ChargeType
}

type GetTypesByIDsResult []ChargeWithType
