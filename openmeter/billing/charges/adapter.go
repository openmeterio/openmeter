package charges

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Adapter interface {
	ChargeAdapter

	entutils.TxCreator
}

type ChargeAdapter interface {
	GetChargeByID(ctx context.Context, input models.NamespacedID) (Charge, error)
	GetChargesByIDs(ctx context.Context, namespace string, ids []string) (Charges, error)
	CreateCharges(ctx context.Context, input Charges) (Charges, error)
}
