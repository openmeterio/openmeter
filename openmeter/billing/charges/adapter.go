package charges

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	ChargeAdapter
	CreditRealizationAdapter
	StandardInvoiceRealizationAdapter

	entutils.TxCreator
}

type ChargeAdapter interface {
	GetChargeByID(ctx context.Context, input GetChargeByIDInput) (Charge, error)
	GetChargesByIDs(ctx context.Context, input GetChargesByIDsInput) (Charges, error)
	CreateCharges(ctx context.Context, input CreateChargeInputs) (Charges, error)
	UpdateFlatFeeCharge(ctx context.Context, charge FlatFeeCharge) (FlatFeeCharge, error)
}

type CreditRealizationAdapter interface {
	CreateCreditRealizations(ctx context.Context, chargeID ChargeID, realizations []CreditRealizationCreateInput) (CreditRealizations, error)
}

type StandardInvoiceRealizationAdapter interface {
	CreateStandardInvoicePaymentSettlement(ctx context.Context, chargeID ChargeID, paymentState StandardInvoicePaymentSettlement) (StandardInvoicePaymentSettlement, error)
	UpdateStandardInvoicePaymentSettlement(ctx context.Context, paymentState StandardInvoicePaymentSettlement) (StandardInvoicePaymentSettlement, error)

	CreateStandardInvoiceAccruedUsage(ctx context.Context, chargeID ChargeID, accruedUsage StandardInvoiceAccruedUsage) (StandardInvoiceAccruedUsage, error)
}
