package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) onExternalCreditPurchase(ctx context.Context, charge creditpurchase.Charge) (creditpurchase.Charge, error) {
	externalCreditPurchaseSettlement, err := charge.Intent.Settlement.AsExternalSettlement()
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	targetStatus := externalCreditPurchaseSettlement.InitialStatus

	charge, err = transaction.Run(ctx, s.adapter, func(ctx context.Context) (creditpurchase.Charge, error) {
		ledgerTransactionGroupReference, err := s.handler.OnCreditPurchaseInitiated(ctx, charge)
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		charge.State.CreditGrantRealization = &ledgertransaction.TimedGroupReference{
			GroupReference: ledgerTransactionGroupReference,
			Time:           clock.Now(),
		}

		if ledgerTransactionGroupReference.TransactionGroupID != "" {
			if err := s.lineage.BackfillAdvanceLineageSegments(ctx, lineage.BackfillAdvanceLineageSegmentsInput{
				Namespace:                 charge.Namespace,
				CustomerID:                charge.Intent.CustomerID,
				Currency:                  charge.Intent.Currency,
				Amount:                    charge.Intent.CreditAmount,
				BackingTransactionGroupID: ledgerTransactionGroupReference.TransactionGroupID,
			}); err != nil {
				return creditpurchase.Charge{}, err
			}
		}

		charge.Status = meta.ChargeStatusActive

		updatedCharge, err := s.adapter.UpdateCharge(ctx, charge)
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		return updatedCharge, nil
	})
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	// Let's handle the payment authorized state transition if requested
	if targetStatus.In(
		creditpurchase.AuthorizedInitialPaymentSettlementStatus,
		creditpurchase.SettledInitialPaymentSettlementStatus,
	) {
		charge, err = s.HandleExternalPaymentAuthorized(ctx, charge)
		if err != nil {
			return creditpurchase.Charge{}, err
		}
	}

	// Let's handle the payment settled state transition if requested
	if targetStatus == creditpurchase.SettledInitialPaymentSettlementStatus {
		charge, err = s.HandleExternalPaymentSettled(ctx, charge)
		if err != nil {
			return creditpurchase.Charge{}, err
		}
	}

	return charge, nil
}

func (s *service) HandleExternalPaymentAuthorized(ctx context.Context, charge creditpurchase.Charge) (creditpurchase.Charge, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (creditpurchase.Charge, error) {
		if charge.State.ExternalPaymentSettlement != nil {
			return creditpurchase.Charge{}, payment.ErrPaymentAlreadyAuthorized.
				WithAttrs(charge.ErrorAttributes()).
				WithAttrs(charge.State.ExternalPaymentSettlement.ErrorAttributes())
		}

		ledgerTransactionGroupReference, err := s.handler.OnCreditPurchasePaymentAuthorized(ctx, charge)
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		newPaymentSettlement := payment.ExternalCreateInput{
			Namespace: charge.Namespace,
			Base: payment.Base{
				ServicePeriod: charge.Intent.ServicePeriod,
				Amount:        charge.Intent.CreditAmount,
				Authorized: &ledgertransaction.TimedGroupReference{
					GroupReference: ledgerTransactionGroupReference,
					Time:           clock.Now(),
				},
				Status: payment.StatusAuthorized,
			},
		}

		paymentSettlement, err := s.adapter.CreateExternalPayment(ctx, charge.GetChargeID(), newPaymentSettlement)
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		charge.State.ExternalPaymentSettlement = &paymentSettlement

		charge, err = s.adapter.UpdateCharge(ctx, charge)
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		return charge, nil
	})
}

func (s *service) HandleExternalPaymentSettled(ctx context.Context, charge creditpurchase.Charge) (creditpurchase.Charge, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (creditpurchase.Charge, error) {
		if charge.State.ExternalPaymentSettlement == nil {
			return creditpurchase.Charge{}, payment.ErrCannotSettleNotAuthorizedPayment.
				WithAttrs(charge.ErrorAttributes())
		}

		paymentSettlement := *charge.State.ExternalPaymentSettlement

		if paymentSettlement.Status != payment.StatusAuthorized {
			return creditpurchase.Charge{}, payment.ErrPaymentAlreadySettled.
				WithAttrs(charge.ErrorAttributes()).
				WithAttrs(paymentSettlement.ErrorAttributes())
		}

		ledgerTransactionGroupReference, err := s.handler.OnCreditPurchasePaymentSettled(ctx, charge)
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		paymentSettlement.Settled = &ledgertransaction.TimedGroupReference{
			GroupReference: ledgerTransactionGroupReference,
			Time:           clock.Now(),
		}

		paymentSettlement.Status = payment.StatusSettled

		paymentSettlement, err = s.adapter.UpdateExternalPayment(ctx, paymentSettlement)
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		charge.State.ExternalPaymentSettlement = &paymentSettlement

		// Let's update the charge status to final
		charge.Status = meta.ChargeStatusFinal

		charge, err = s.adapter.UpdateCharge(ctx, charge)
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		return charge, nil
	})
}
