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
	DeleteChargesByUniqueReferenceID(ctx context.Context, input DeleteChargesByUniqueReferenceIDInput) error
	UpsertChargesByChildUniqueReferenceID(ctx context.Context, input UpsertChargesByChildUniqueReferenceIDInput) (Charges, error)
}
