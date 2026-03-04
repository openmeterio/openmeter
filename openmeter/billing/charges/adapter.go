package charges

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	ChargeAdapter
	CreditRealizationAdapter
	CreditPurchaseAdapter
	StandardInvoiceRealizationAdapter
	ExternalPaymentSettlementAdapter

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

type CreditPurchaseAdapter interface {
	UpdateCreditPurchaseCharge(ctx context.Context, charge CreditPurchaseCharge) (CreditPurchaseCharge, error)
}

type ExternalPaymentSettlementAdapter interface {
	CreateExternalPaymentSettlement(ctx context.Context, chargeID ChargeID, paymentSettlement ExternalPaymentSettlement) (ExternalPaymentSettlement, error)
	UpdateExternalPaymentSettlement(ctx context.Context, paymentSettlement ExternalPaymentSettlement) (ExternalPaymentSettlement, error)
}
