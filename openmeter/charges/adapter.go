package charges

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Adapter interface {
	ChargeAdapter

	entutils.TxCreator
}

type ChargeAdapter interface {
	GetChargeByID(ctx context.Context, input models.NamespacedID) (Charge, error)
	UpsertChargesByChildUniqueReferenceID(ctx context.Context, input UpsertChargesByChildUniqueReferenceIDInput) (Charges, error)
	DeleteChargesByUniqueReferenceID(ctx context.Context, input DeleteChargesByUniqueReferenceIDInput) error
}

type ProRatingModeAdapterEnum string

const (
	ProRatingAdapterModeEnumProratePrices ProRatingModeAdapterEnum = ProRatingModeAdapterEnum(productcatalog.ProRatingModeProratePrices)
	ProRatingAdapterModeEnumNoProrate     ProRatingModeAdapterEnum = "no_prorate"
)

func (e ProRatingModeAdapterEnum) Values() []string {
	return []string{
		string(ProRatingAdapterModeEnumProratePrices),
		string(ProRatingAdapterModeEnumNoProrate),
	}
}
