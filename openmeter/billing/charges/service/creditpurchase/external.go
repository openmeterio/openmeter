package creditpurchase

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func (s *service) onExternalCreditPurchase(ctx context.Context, charge charges.CreditPurchaseCharge) (charges.CreditPurchaseCharge, error) {
	externalCreditPurchaseSettlement, err := charge.Intent.Settlement.AsExternalCreditPurchaseSettlement()
	if err != nil {
		return charges.CreditPurchaseCharge{}, err
	}

	targetStatus := externalCreditPurchaseSettlement.InitialStatus

	// Let's handle the initial state
	ledgerTransactionGroupReference, err := s.creditPurchaseHandler.OnCreditPurchaseInitiated(ctx, charge)
	if err != nil {
		return charges.CreditPurchaseCharge{}, err
	}

	charge.State.CreditGrantRealization = &charges.TimedLedgerTransactionGroupReference{
		LedgerTransactionGroupReference: ledgerTransactionGroupReference,
		Time:                            clock.Now(),
	}

	charge.Status = charges.ChargeStatusActive

	charge, err = s.adapter.UpdateCreditPurchaseCharge(ctx, charge)
	if err != nil {
		return charges.CreditPurchaseCharge{}, err
	}

	// Let's handle the payment authorized state transition if requested
	if targetStatus.In(
		charges.AuthorizedInitialCreditPurchasePaymentSettlementStatus,
		charges.SettledInitialCreditPurchasePaymentSettlementStatus,
	) {
		charge, err = s.HandleExternalCreditPurchasePaymentAuthorized(ctx, charge)
		if err != nil {
			return charges.CreditPurchaseCharge{}, err
		}
	}

	// Let's handle the payment settled state transition if requested
	if targetStatus == charges.SettledInitialCreditPurchasePaymentSettlementStatus {
		charge, err = s.HandleExternalCreditPurchasePaymentSettled(ctx, charge)
		if err != nil {
			return charges.CreditPurchaseCharge{}, err
		}
	}

	return charge, nil
}

func (s *service) HandleExternalCreditPurchasePaymentAuthorized(ctx context.Context, charge charges.CreditPurchaseCharge) (charges.CreditPurchaseCharge, error) {
	if charge.State.ExternalPaymentSettlement != nil {
		return charges.CreditPurchaseCharge{}, charges.ErrPaymentAlreadyAuthorized.
			WithAttrs(charge.ErrorAttributes()).
			WithAttrs(charge.State.ExternalPaymentSettlement.ErrorAttributes())
	}

	ledgerTransactionGroupReference, err := s.creditPurchaseHandler.OnCreditPurchasePaymentAuthorized(ctx, charge)
	if err != nil {
		return charges.CreditPurchaseCharge{}, err
	}

	newPaymentSettlement := charges.ExternalPaymentSettlementCreateInput{
		Namespace: charge.Namespace,
		PaymentSettlementBase: charges.PaymentSettlementBase{
			ServicePeriod: charge.Intent.ServicePeriod,
			Amount:        charge.Intent.CreditAmount,
			Authorized: &charges.TimedLedgerTransactionGroupReference{
				LedgerTransactionGroupReference: ledgerTransactionGroupReference,
				Time:                            clock.Now(),
			},
			Status: charges.PaymentSettlementStatusAuthorized,
		},
	}

	paymentSettlement, err := s.adapter.CreateExternalPaymentSettlement(ctx, newPaymentSettlement)
	if err != nil {
		return charges.CreditPurchaseCharge{}, err
	}

	charge.State.ExternalPaymentSettlement = &paymentSettlement

	charge, err = s.adapter.UpdateCreditPurchaseCharge(ctx, charge)
	if err != nil {
		return charges.CreditPurchaseCharge{}, err
	}

	return charge, nil
}

func (s *service) HandleExternalCreditPurchasePaymentSettled(ctx context.Context, charge charges.CreditPurchaseCharge) (charges.CreditPurchaseCharge, error) {
	if charge.State.ExternalPaymentSettlement == nil {
		return charges.CreditPurchaseCharge{}, charges.ErrCannotSettleNotAuthorizedPayment.
			WithAttrs(charge.ErrorAttributes())
	}

	paymentSettlement := *charge.State.ExternalPaymentSettlement

	if paymentSettlement.Status != charges.PaymentSettlementStatusAuthorized {
		return charges.CreditPurchaseCharge{}, charges.ErrPaymentAlreadySettled.
			WithAttrs(charge.ErrorAttributes()).
			WithAttrs(paymentSettlement.ErrorAttributes())
	}

	ledgerTransactionGroupReference, err := s.creditPurchaseHandler.OnCreditPurchasePaymentSettled(ctx, charge)
	if err != nil {
		return charges.CreditPurchaseCharge{}, err
	}

	paymentSettlement.Settled = &charges.TimedLedgerTransactionGroupReference{
		LedgerTransactionGroupReference: ledgerTransactionGroupReference,
		Time:                            clock.Now(),
	}

	paymentSettlement.Status = charges.PaymentSettlementStatusSettled

	paymentSettlement, err = s.adapter.UpdateExternalPaymentSettlement(ctx, paymentSettlement)
	if err != nil {
		return charges.CreditPurchaseCharge{}, err
	}

	charge.State.ExternalPaymentSettlement = &paymentSettlement

	// Let's update the charge status to final
	charge.Status = charges.ChargeStatusFinal

	charge, err = s.adapter.UpdateCreditPurchaseCharge(ctx, charge)
	if err != nil {
		return charges.CreditPurchaseCharge{}, err
	}

	return charge, nil
}
