package usagebased

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	CreateCharges(ctx context.Context, charges CreateInput) ([]Charge, error)
	UpdateCharge(ctx context.Context, charge Charge) error
	GetByMetas(ctx context.Context, ids GetByMetasInput) ([]Charge, error)

	entutils.TxCreator
}
