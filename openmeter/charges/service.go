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
	CreateCharges(ctx context.Context, input CreateChargeInput) (Charges, error)
}
