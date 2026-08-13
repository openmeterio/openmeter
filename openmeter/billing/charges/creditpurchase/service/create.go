package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) Create(ctx context.Context, input creditpurchase.CreateInput) (creditpurchase.ChargeWithGatheringLine, error) {
	if err := input.Validate(); err != nil {
		return creditpurchase.ChargeWithGatheringLine{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (creditpurchase.ChargeWithGatheringLine, error) {
		input.Intent = input.Intent.Normalized()
		resolvedAt := clock.Now().UTC()

		initialCostBasisState, err := s.resolveInitialCostBasis(ctx, resolveInitialCostBasisInput{
			Currency:   input.Intent.Currency,
			CostBasis:  input.Intent.CostBasis,
			ResolvedAt: resolvedAt,
		})
		if err != nil {
			return creditpurchase.ChargeWithGatheringLine{}, err
		}

		// Let's create the credit purchase charge
		charge, err := s.adapter.CreateCharge(ctx, creditpurchase.CreateChargeAdapterInput{
			CreateInput:           input,
			InitialCostBasisState: initialCostBasisState,
		})
		if err != nil {
			return creditpurchase.ChargeWithGatheringLine{}, err
		}

		// Let's activate the state machine for the credit purchase charge
		switch charge.Intent.Settlement.Type() {
		case creditpurchase.SettlementTypePromotional:
			stateMachine, err := NewPromotionalCreditPurchaseStateMachine(StateMachineConfig{
				Charge:       charge,
				Adapter:      s.adapter,
				Realizations: s.realizations,
			})
			if err != nil {
				return creditpurchase.ChargeWithGatheringLine{}, fmt.Errorf("new promotional state machine: %w", err)
			}

			if err := stateMachine.AdvanceUntilStable(ctx); err != nil {
				return creditpurchase.ChargeWithGatheringLine{}, fmt.Errorf("advance promotional state machine: %w", err)
			}

			charge = stateMachine.GetCharge()
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
			gatheringLine, err := buildInvoiceCreditPurchaseGatheringLine(charge)
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

func buildInvoiceCreditPurchaseGatheringLine(charge creditpurchase.Charge) (billing.GatheringLine, error) {
	if charge.Intent.Settlement.Type() != creditpurchase.SettlementTypeInvoice {
		return billing.GatheringLine{}, errors.New("credit purchase is not invoice settled")
	}

	intent := charge.Intent

	fiatCurrency, err := intent.GetSettlementFiatCurrency()
	if err != nil {
		return billing.GatheringLine{}, fmt.Errorf("getting settlement fiat currency: %w", err)
	}

	// Dynamic cost basis can remain unresolved while the gathering line is
	// pending. Invoice creation replaces this temporary amount after resolution.
	totalCost := alpacadecimal.Zero

	if charge.State.ResolvedCostBasis != nil {
		totalCost, err = charge.GetFiatSettlementAmount()
		if err != nil {
			return billing.GatheringLine{}, fmt.Errorf("getting fiat settlement amount: %w", err)
		}
	}

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
				Name:        fmt.Sprintf("%s (%s %s credits)", intent.Name, intent.CreditAmount, intent.Currency.GetCode()),
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
			Currency:      fiatCurrency.GetFiatCode(),
			ServicePeriod: intent.ServicePeriod,
			InvoiceAt:     intent.CalculateEffectiveAt(),
			TaxConfig:     lo.ToPtr(intent.TaxConfig.ToTaxConfig()),
			ChargeID:      lo.ToPtr(charge.ID),
			Engine:        billing.LineEngineTypeChargeCreditPurchase,
		},
	}, nil
}
