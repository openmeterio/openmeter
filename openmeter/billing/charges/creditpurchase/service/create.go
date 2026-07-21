package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) Create(ctx context.Context, input creditpurchase.CreateInput) (creditpurchase.ChargeWithGatheringLine, error) {
	if err := input.Validate(); err != nil {
		return creditpurchase.ChargeWithGatheringLine{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (creditpurchase.ChargeWithGatheringLine, error) {
		if input.Intent.Currency.Type() == currencyx.CurrencyTypeCustom {
			return creditpurchase.ChargeWithGatheringLine{}, fmt.Errorf("custom currency %s is not supported for credit purchases: %w", input.Intent.Currency.GetCode(), meta.ErrCustomCurrencyNotSupported)
		}

		input.Intent = input.Intent.Normalized()

		// Let's create the credit purchase charge
		charge, err := s.adapter.CreateCharge(ctx, input)
		if err != nil {
			return creditpurchase.ChargeWithGatheringLine{}, err
		}

		// Let's activate the state machine for the credit purchase charge
		switch charge.Intent.Settlement.Type() {
		case creditpurchase.SettlementTypePromotional:
			stateMachine, err := NewPromotionalCreditPurchaseStateMachine(StateMachineConfig{
				Charge:  charge,
				Adapter: s.adapter,
				Service: s,
			})
			if err != nil {
				return creditpurchase.ChargeWithGatheringLine{}, fmt.Errorf("new promotional state machine: %w", err)
			}

			advancedCharge, err := stateMachine.AdvanceUntilStateStable(ctx)
			if err != nil {
				return creditpurchase.ChargeWithGatheringLine{}, fmt.Errorf("advance promotional state machine: %w", err)
			}

			if advancedCharge != nil {
				charge = *advancedCharge
			}
		case creditpurchase.SettlementTypeInvoice:
			// noop, as we will transition to active state when the invoice is created, as
			// - invocing based charges are driven by the invoice state machine
			// - we should set the active state when the invoice is created, not when the credit purchase is created
		case creditpurchase.SettlementTypeExternal:
			charge, err = s.onExternalCreditPurchase(ctx, charge)
		default:
			return creditpurchase.ChargeWithGatheringLine{}, fmt.Errorf("invalid credit purchase settlement type: %s", charge.Intent.Settlement.Type())
		}
		if err != nil {
			return creditpurchase.ChargeWithGatheringLine{}, err
		}

		// For invoice settlement, prepare the gathering line (actual invoicing happens after TX commits)
		if charge.Intent.Settlement.Type() == creditpurchase.SettlementTypeInvoice {
			gatheringLine, err := s.buildInvoiceCreditPurchaseGatheringLine(charge)
			if err != nil {
				return creditpurchase.ChargeWithGatheringLine{}, fmt.Errorf("building invoice credit purchase gathering line: %w", err)
			}

			return creditpurchase.ChargeWithGatheringLine{
				Charge:                charge,
				GatheringLineToCreate: &gatheringLine,
			}, nil
		}

		return creditpurchase.ChargeWithGatheringLine{
			Charge: charge,
		}, nil
	})
}

func (s *service) buildInvoiceCreditPurchaseGatheringLine(charge creditpurchase.Charge) (billing.GatheringLine, error) {
	invoiceSettlement, err := charge.Intent.Settlement.AsInvoiceSettlement()
	if err != nil {
		return billing.GatheringLine{}, err
	}

	intent := charge.Intent

	// Total cost = credit amount * cost basis (e.g., 100 credits * $0.5 = $50)
	totalCost := intent.CreditAmount.Mul(invoiceSettlement.CostBasis)
	calc, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
		WithCode(invoiceSettlement.Currency).
		Build()
	if err != nil {
		return billing.GatheringLine{}, fmt.Errorf("creating currency calculator: %w", err)
	}
	totalCost = calc.RoundToPrecision(totalCost)

	// Clone metadata and add credit-purchase specific annotations
	annotations, err := charge.Intent.Annotations.Clone()
	if err != nil {
		return billing.GatheringLine{}, fmt.Errorf("cloning annotations: %w", err)
	}

	if annotations == nil {
		annotations = models.Annotations{}
	}

	annotations[billing.AnnotationKeyTaxable] = lo.ToPtr("false")
	annotations[billing.AnnotationKeyReason] = lo.ToPtr(billing.AnnotationValueReasonCreditPurchase)

	return billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace:   charge.Namespace,
				Name:        intent.Name,
				Description: intent.Description,
			}),
			Metadata:    intent.Metadata.Clone(),
			Annotations: annotations,
			ManagedBy:   intent.ManagedBy,
			Price: lo.FromPtr(
				productcatalog.NewPriceFrom(
					productcatalog.FlatPrice{
						Amount:      totalCost,
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					},
				),
			),
			Currency:      invoiceSettlement.Currency,
			ServicePeriod: intent.ServicePeriod,
			InvoiceAt:     intent.CalculateEffectiveAt(),
			TaxConfig:     lo.ToPtr(intent.TaxConfig.ToTaxConfig()),
			ChargeID:      lo.ToPtr(charge.ID),
			Engine:        billing.LineEngineTypeChargeCreditPurchase,
		},
	}, nil
}
