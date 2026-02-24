package charges

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/models"
)

type Service interface {
	ChargeService
}

type ChargeService interface {
	GetChargeByID(ctx context.Context, input models.NamespacedID) (Charge, error)
	GetChargesByIDs(ctx context.Context, namespace string, ids []string) (Charges, error)
	CreateCharges(ctx context.Context, input CreateChargeInputs) (Charges, error)
}
