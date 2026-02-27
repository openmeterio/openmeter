package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type paymentPostProcessorFn[T charges.FlatFeeCharge | charges.UsageBasedCharge | charges.CreditPurchaseCharge] func(ctx context.Context, charge T, lineWithHeader billing.StandardLineWithInvoiceHeader) error

type processorByType struct {
	flatFee        paymentPostProcessorFn[charges.FlatFeeCharge]
	usageBased     paymentPostProcessorFn[charges.UsageBasedCharge]
	creditPurchase paymentPostProcessorFn[charges.CreditPurchaseCharge]
}

func (s *service) handleStandardInvoiceUpdate(ctx context.Context, invoice billing.StandardInvoice) error {
	if invoice.Status == billing.StandardInvoiceStatusPaymentProcessingPending {
		return s.handlePaymentEvent(ctx, invoice, processorByType{
			flatFee: s.flatFeeService.PostPaymentAuthorized,
		})
	}

	if invoice.Status == billing.StandardInvoiceStatusPaid {
		return s.handlePaymentEvent(ctx, invoice, processorByType{
			flatFee: s.flatFeeService.PostPaymentSettled,
		})
	}

	return nil
}

func (s *service) handlePaymentEvent(ctx context.Context, invoice billing.StandardInvoice, processorByType processorByType) error {
	linesWithCharges, err := s.getLinesWithChargesForStandardInvoice(ctx, invoice.Namespace, invoice)
	if err != nil {
		return err
	}

	for _, lineWithCharge := range linesWithCharges {
		switch lineWithCharge.Charge.Type() {
		case charges.ChargeTypeFlatFee:
			ff, err := lineWithCharge.Charge.AsFlatFeeCharge()
			if err != nil {
				return err
			}

			if processorByType.flatFee == nil {
				return fmt.Errorf("flat fee payment post processor is not supported")
			}

			err = processorByType.flatFee(ctx, ff, lineWithCharge.StandardLineWithInvoiceHeader)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported charge type: %s", lineWithCharge.Charge.Type())
		}
	}

	return nil
}

var _ billing.StandardInvoiceHook = (*standardInvoiceEventHandler)(nil)

// standardInvoiceEventHandler implements the billing.StandardInvoiceHook interface and channels the update events
// to the charges service.
type standardInvoiceEventHandler struct {
	models.NoopServiceHook[billing.StandardInvoice]
	chargesService *service
}

func (h *standardInvoiceEventHandler) PostUpdate(ctx context.Context, invoice *billing.StandardInvoice) error {
	return h.chargesService.handleStandardInvoiceUpdate(ctx, *invoice)
}

type standardLineWithCharge struct {
	billing.StandardLineWithInvoiceHeader
	Charge charges.Charge
}

func (s *service) getLinesWithChargesForStandardInvoice(ctx context.Context, ns string, invoice billing.StandardInvoice) ([]standardLineWithCharge, error) {
	if ns == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	linesWithChargeID := lo.FilterMap(invoice.Lines.OrEmpty(), func(line *billing.StandardLine, _ int) (billing.StandardLineWithInvoiceHeader, bool) {
		if line.ChargeID == nil {
			return billing.StandardLineWithInvoiceHeader{}, false
		}

		return billing.StandardLineWithInvoiceHeader{
			Line:    line,
			Invoice: invoice,
		}, true
	})

	referencedCharges, err := s.GetChargesByIDs(ctx,
		ns,
		lo.Map(linesWithChargeID, func(l billing.StandardLineWithInvoiceHeader, _ int) string {
			return *l.Line.ChargeID
		}),
	)
	if err != nil {
		return nil, err
	}

	chargesByID := make(map[charges.ChargeID]charges.Charge, len(referencedCharges))
	for _, charge := range referencedCharges {
		id, err := charge.GetChargeID()
		if err != nil {
			return nil, err
		}
		chargesByID[id] = charge
	}

	return slicesx.MapWithErr(linesWithChargeID, func(lineWithHeader billing.StandardLineWithInvoiceHeader) (standardLineWithCharge, error) {
		chargeID := *lineWithHeader.Line.ChargeID

		charge, ok := chargesByID[charges.ChargeID{
			Namespace: ns,
			ID:        chargeID,
		}]
		if !ok {
			return standardLineWithCharge{}, fmt.Errorf("charge not found [namespace=%s charge.id=%s]", ns, chargeID)
		}

		return standardLineWithCharge{
			Charge:                        charge,
			StandardLineWithInvoiceHeader: lineWithHeader,
		}, nil
	})
}

func withBillingTransactionForInvoiceManipulation[T any](ctx context.Context, s *service, customerID customer.CustomerID, fn func(ctx context.Context) (T, error)) (T, error) {
	var out T

	err := s.billingService.WithLock(ctx, customerID, func(ctx context.Context) error {
		var err error
		out, err = fn(ctx)
		if err != nil {
			return err
		}

		return err
	})
	if err != nil {
		return lo.Empty[T](), err
	}

	return out, nil
}
