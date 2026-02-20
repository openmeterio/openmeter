package charges

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	ChargeAdapter
	StandardInvoiceRealizationAdapter
	CreditRealizationAdapter

	entutils.TxCreator
}

type ChargeAdapter interface {
	GetChargeByID(ctx context.Context, input ChargeID) (Charge, error)
	GetChargesByIDs(ctx context.Context, ns string, ids []string) (Charges, error)
	CreateCharges(ctx context.Context, input CreateChargeInput) (Charges, error)
}

type StandardInvoiceRealizationAdapter interface {
	CreateStandardInvoiceRealization(ctx context.Context, chargeID ChargeID, realization StandardInvoiceRealization) (StandardInvoiceRealization, error)
	UpdateStandardInvoiceRealizationByID(ctx context.Context, chargeID ChargeID, realization StandardInvoiceRealization) (StandardInvoiceRealization, error)
}

type CreditRealizationAdapter interface {
	CreateCreditRealizations(ctx context.Context, chargeID ChargeID, realizations []CreditRealizationCreateInput) (CreditRealizations, error)
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
