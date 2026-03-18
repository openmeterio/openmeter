package usagebased

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	CreateCharges(ctx context.Context, charges CreateInput) ([]Charge, error)
	UpdateCharge(ctx context.Context, charge Charge) error
	GetByIDs(ctx context.Context, ids GetByIDsInput) ([]Charge, error)

	entutils.TxCreator
}
