package realizations

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
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
	if err := charge.Intent.Settlement.Validate(); err != nil {
		return creditpurchase.Charge{}, err
	}

	if charge.Intent.Settlement.Type() == creditpurchase.SettlementTypePromotional {
		return creditpurchase.Charge{}, fmt.Errorf("promotional credit purchases do not use payment-backed credit grants")
	}

	if charge.Realizations.CreditGrantRealization != nil && charge.Realizations.CreditGrantRealization.TransactionGroupID != "" {
		return creditpurchase.Charge{}, fmt.Errorf("credit grant already realized [charge_id=%s, transaction_group_id=%s]", charge.ID, charge.Realizations.CreditGrantRealization.TransactionGroupID)
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

type AuthorizeInvoicedPaymentInput struct {
	Charge         creditpurchase.Charge
	LineWithHeader billing.StandardLineWithInvoiceHeader
}

var _ models.Validator = AuthorizeInvoicedPaymentInput{}

func (i AuthorizeInvoicedPaymentInput) Validate() error {
	var errs []error

	if i.Charge.Intent.Settlement.Type() != creditpurchase.SettlementTypeInvoice {
		errs = append(errs, errors.New("charge is not invoice settled"))
	}

	if err := i.LineWithHeader.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("line with invoice header: %w", err))
	}

	line := i.LineWithHeader.Line
	if i.LineWithHeader.Invoice.Namespace != i.Charge.Namespace {
		errs = append(errs, errors.New("invoice namespace must match charge namespace"))
	}

	if line != nil {
		if line.ChargeID == nil || *line.ChargeID != i.Charge.ID {
			errs = append(errs, errors.New("line charge ID must match charge"))
		}

		if line.Namespace != i.Charge.Namespace {
			errs = append(errs, errors.New("line namespace must match charge namespace"))
		}

		if line.InvoiceID != i.LineWithHeader.Invoice.ID {
			errs = append(errs, errors.New("line invoice ID must match invoice"))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (s *Service) AuthorizeInvoicedPayment(ctx context.Context, input AuthorizeInvoicedPaymentInput) (creditpurchase.Charge, error) {
	if err := input.Validate(); err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("validate authorize invoiced payment: %w", err)
	}

	charge := input.Charge
	lineWithHeader := input.LineWithHeader

	if charge.Realizations.InvoiceSettlement != nil {
		return creditpurchase.Charge{}, payment.ErrPaymentAlreadyAuthorized.
			WithAttrs(charge.ErrorAttributes()).
			WithAttrs(charge.Realizations.InvoiceSettlement.ErrorAttributes())
	}

	eventAt := clock.Now()
	ledgerTransactionGroupReference, err := s.handler.OnCreditPurchasePaymentAuthorized(ctx, creditpurchase.PaymentEventInput{
		Charge:     charge,
		EventAt:    eventAt,
		FiatAmount: lineWithHeader.Line.Totals.Total,
	})
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	paymentSettlement, err := s.adapter.CreateInvoicedPayment(ctx, charge.GetChargeID(), payment.InvoicedCreate{
		Namespace: charge.Namespace,
		Base: payment.Base{
			ServicePeriod: charge.Intent.ServicePeriod,
			FiatAmount:    lineWithHeader.Line.Totals.Total,
			Authorized: &ledgertransaction.TimedGroupReference{
				GroupReference: ledgerTransactionGroupReference,
				Time:           eventAt,
			},
			Status: payment.StatusAuthorized,
		},
		InvoiceID: lineWithHeader.Invoice.ID,
		LineID:    lineWithHeader.Line.ID,
	})
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	charge.Realizations.InvoiceSettlement = &paymentSettlement

	return charge, nil
}

func (s *Service) SettleInvoicedPayment(ctx context.Context, charge creditpurchase.Charge) (creditpurchase.Charge, error) {
	if charge.Realizations.InvoiceSettlement == nil {
		return creditpurchase.Charge{}, payment.ErrCannotSettleNotAuthorizedPayment.
			WithAttrs(charge.ErrorAttributes())
	}

	paymentSettlement := *charge.Realizations.InvoiceSettlement
	if paymentSettlement.Status != payment.StatusAuthorized {
		return creditpurchase.Charge{}, payment.ErrPaymentAlreadySettled.
			WithAttrs(charge.ErrorAttributes()).
			WithAttrs(paymentSettlement.ErrorAttributes())
	}

	eventAt := clock.Now()
	ledgerTransactionGroupReference, err := s.handler.OnCreditPurchasePaymentSettled(ctx, creditpurchase.PaymentEventInput{
		Charge:     charge,
		EventAt:    eventAt,
		FiatAmount: paymentSettlement.FiatAmount,
	})
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	paymentSettlement.Settled = &ledgertransaction.TimedGroupReference{
		GroupReference: ledgerTransactionGroupReference,
		Time:           eventAt,
	}
	paymentSettlement.Status = payment.StatusSettled

	paymentSettlement, err = s.adapter.UpdateInvoicedPayment(ctx, paymentSettlement)
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	charge.Realizations.InvoiceSettlement = &paymentSettlement

	return charge, nil
}

func (s *Service) AuthorizeExternalPayment(ctx context.Context, charge creditpurchase.Charge) (creditpurchase.Charge, error) {
	if charge.Realizations.ExternalPaymentSettlement != nil {
		return creditpurchase.Charge{}, payment.ErrPaymentAlreadyAuthorized.
			WithAttrs(charge.ErrorAttributes()).
			WithAttrs(charge.Realizations.ExternalPaymentSettlement.ErrorAttributes())
	}

	if _, err := charge.Intent.Settlement.AsExternalSettlement(); err != nil {
		return creditpurchase.Charge{}, err
	}

	resolvedCostBasis, err := charge.GetResolvedCostBasis()
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	fiatAmount := resolvedCostBasis.FiatAmount(charge.Intent.CreditAmount)

	eventAt := clock.Now()
	ledgerTransactionGroupReference, err := s.handler.OnCreditPurchasePaymentAuthorized(ctx, creditpurchase.PaymentEventInput{
		Charge:     charge,
		EventAt:    eventAt,
		FiatAmount: fiatAmount,
	})
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	newPaymentSettlement := payment.ExternalCreateInput{
		Namespace: charge.Namespace,
		Base: payment.Base{
			ServicePeriod: charge.Intent.ServicePeriod,
			FiatAmount:    fiatAmount,
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
		Charge:     charge,
		EventAt:    eventAt,
		FiatAmount: paymentSettlement.FiatAmount,
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
