package realizations

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Service owns credit-purchase realization mechanics. It must not decide which
// lifecycle trigger should fire or which charge status should be entered.
type Service struct {
	adapter creditpurchase.Adapter
	handler creditpurchase.Handler
	lineage lineage.Service
}

type Config struct {
	Adapter creditpurchase.Adapter
	Handler creditpurchase.Handler
	Lineage lineage.Service
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter is required"))
	}

	if c.Handler == nil {
		errs = append(errs, errors.New("handler is required"))
	}

	if c.Lineage == nil {
		errs = append(errs, errors.New("lineage service is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		adapter: config.Adapter,
		handler: config.Handler,
		lineage: config.Lineage,
	}, nil
}

func (s *Service) GrantCredits(ctx context.Context, charge creditpurchase.Charge) (creditpurchase.Charge, error) {
	externalSettlement, err := charge.Intent.Settlement.AsExternalSettlement()
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	if err := externalSettlement.Validate(); err != nil {
		return creditpurchase.Charge{}, err
	}

	if charge.Realizations.CreditGrantRealization != nil && charge.Realizations.CreditGrantRealization.TransactionGroupID != "" {
		return creditpurchase.Charge{}, fmt.Errorf("external credit grant already realized [charge_id=%s, transaction_group_id=%s]", charge.ID, charge.Realizations.CreditGrantRealization.TransactionGroupID)
	}

	ledgerTransactionGroupReference, err := s.handler.OnCreditPurchaseInitiated(ctx, charge)
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	grantRealization, err := s.adapter.CreateCreditGrant(ctx, charge.GetChargeID(), creditpurchase.CreateCreditGrantInput{
		TransactionGroupID: ledgerTransactionGroupReference.TransactionGroupID,
		GrantedAt:          clock.Now(),
	})
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	charge.Realizations.CreditGrantRealization = &grantRealization

	if ledgerTransactionGroupReference.TransactionGroupID != "" {
		if err := s.lineage.BackfillAdvanceLineageSegments(ctx, lineage.BackfillAdvanceLineageSegmentsInput{
			Namespace:                 charge.Namespace,
			CustomerID:                charge.Intent.CustomerID,
			Currency:                  charge.Intent.Currency,
			Amount:                    charge.Intent.CreditAmount,
			BackingTransactionGroupID: ledgerTransactionGroupReference.TransactionGroupID,
			FeatureFilters:            charge.Intent.FeatureFilters.Normalize(),
		}); err != nil {
			return creditpurchase.Charge{}, err
		}
	}

	return charge, nil
}

func (s *Service) AuthorizeExternalPayment(ctx context.Context, charge creditpurchase.Charge) (creditpurchase.Charge, error) {
	if charge.Realizations.ExternalPaymentSettlement != nil {
		return creditpurchase.Charge{}, payment.ErrPaymentAlreadyAuthorized.
			WithAttrs(charge.ErrorAttributes()).
			WithAttrs(charge.Realizations.ExternalPaymentSettlement.ErrorAttributes())
	}

	eventAt := clock.Now()
	ledgerTransactionGroupReference, err := s.handler.OnCreditPurchasePaymentAuthorized(ctx, creditpurchase.PaymentEventInput{
		Charge:  charge,
		EventAt: eventAt,
	})
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
				Time:           eventAt,
			},
			Status: payment.StatusAuthorized,
		},
	}

	paymentSettlement, err := s.adapter.CreateExternalPayment(ctx, charge.GetChargeID(), newPaymentSettlement)
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	charge.Realizations.ExternalPaymentSettlement = &paymentSettlement

	return charge, nil
}

func (s *Service) SettleExternalPayment(ctx context.Context, charge creditpurchase.Charge) (creditpurchase.Charge, error) {
	if charge.Realizations.ExternalPaymentSettlement == nil {
		return creditpurchase.Charge{}, payment.ErrCannotSettleNotAuthorizedPayment.
			WithAttrs(charge.ErrorAttributes())
	}

	paymentSettlement := *charge.Realizations.ExternalPaymentSettlement

	if paymentSettlement.Status != payment.StatusAuthorized {
		return creditpurchase.Charge{}, payment.ErrPaymentAlreadySettled.
			WithAttrs(charge.ErrorAttributes()).
			WithAttrs(paymentSettlement.ErrorAttributes())
	}

	eventAt := clock.Now()
	ledgerTransactionGroupReference, err := s.handler.OnCreditPurchasePaymentSettled(ctx, creditpurchase.PaymentEventInput{
		Charge:  charge,
		EventAt: eventAt,
	})
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	paymentSettlement.Settled = &ledgertransaction.TimedGroupReference{
		GroupReference: ledgerTransactionGroupReference,
		Time:           eventAt,
	}

	paymentSettlement.Status = payment.StatusSettled

	paymentSettlement, err = s.adapter.UpdateExternalPayment(ctx, paymentSettlement)
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	charge.Realizations.ExternalPaymentSettlement = &paymentSettlement

	return charge, nil
}

func (s *Service) AuthorizeAndSettleExternalPayment(ctx context.Context, charge creditpurchase.Charge) (creditpurchase.Charge, error) {
	charge, err := s.AuthorizeExternalPayment(ctx, charge)
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	charge, err = s.SettleExternalPayment(ctx, charge)
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	return charge, nil
}
