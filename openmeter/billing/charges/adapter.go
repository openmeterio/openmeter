package charges

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	ChargeAdapter
	CreditRealizationAdapter

	entutils.TxCreator
}

type ChargeAdapter interface {
	GetChargeByID(ctx context.Context, input ChargeID) (Charge, error)
	GetChargesByIDs(ctx context.Context, namespace string, ids []string) (Charges, error)
	CreateCharges(ctx context.Context, input CreateChargeInputs) (Charges, error)
	UpdateFlatFeeCharge(ctx context.Context, charge FlatFeeCharge) (FlatFeeCharge, error)
}

type CreditRealizationAdapter interface {
	CreateCreditRealizations(ctx context.Context, chargeID ChargeID, realizations []CreditRealizationCreateInput) (CreditRealizations, error)
}
