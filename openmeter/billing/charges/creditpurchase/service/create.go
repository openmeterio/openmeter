package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) Create(ctx context.Context, input creditpurchase.CreateInput) (creditpurchase.ChargeWithGatheringLine, error) {
	if err := input.Validate(); err != nil {
		return creditpurchase.ChargeWithGatheringLine{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (creditpurchase.ChargeWithGatheringLine, error) {
		// Let's create the credit purchase charge
		charge, err := s.adapter.CreateCharge(ctx, creditpurchase.CreateChargeInput(input))
		if err != nil {
			return creditpurchase.ChargeWithGatheringLine{}, err
		}

		// Let's activate the state machine for the credit purchase charge
		switch charge.Intent.Settlement.Type() {
		case creditpurchase.SettlementTypePromotional:
			charge, err = s.onPromotionalCreditPurchase(ctx, charge)
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

	// Clone metadata and add credit-purchase specific annotations
	annotations, err := intent.Annotations.Clone()
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
			InvoiceAt:     intent.ServicePeriod.From,
			TaxConfig:     invoiceSettlement.TaxConfig,
			ChargeID:      lo.ToPtr(charge.ID),
		},
	}, nil
}
