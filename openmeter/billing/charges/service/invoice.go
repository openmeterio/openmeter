package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type chargeProcessorFn[T flatfee.Charge | creditpurchase.Charge | usagebased.Charge] func(ctx context.Context, charge T, lineWithHeader billing.StandardLineWithInvoiceHeader) error

func unsupported[T flatfee.Charge | creditpurchase.Charge | usagebased.Charge](err error) chargeProcessorFn[T] {
	return func(ctx context.Context, charge T, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
		return err
	}
}

func noop[T flatfee.Charge | creditpurchase.Charge | usagebased.Charge](ctx context.Context, charge T, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	return nil
}

type processorByType struct {
	flatFee        chargeProcessorFn[flatfee.Charge]
	creditPurchase chargeProcessorFn[creditpurchase.Charge]
	usageBased     chargeProcessorFn[usagebased.Charge]
}

func (s *service) handleStandardInvoiceUpdate(ctx context.Context, invoice billing.StandardInvoice) error {
	switch invoice.Status {
	case billing.StandardInvoiceStatusDraftCreated:
		return s.handleChargeEvent(ctx, invoice, processorByType{
			flatFee:        noop[flatfee.Charge],
			usageBased:     noop[usagebased.Charge],
			creditPurchase: s.creditPurchaseService.PostInvoiceDraftCreated,
		})

	case billing.StandardInvoiceStatusIssued:
		return s.handleChargeEvent(ctx, invoice, processorByType{
			flatFee:    s.flatFeeService.PostInvoiceIssued,
			usageBased: noop[usagebased.Charge],
			// Invoice credit purchase settlements are not requiring any special handling at this point
			creditPurchase: noop[creditpurchase.Charge],
		})

	case billing.StandardInvoiceStatusPaymentProcessingPending:
		return s.handleChargeEvent(ctx, invoice, processorByType{
			flatFee:        s.flatFeeService.PostInvoicePaymentAuthorized,
			usageBased:     noop[usagebased.Charge],
			creditPurchase: s.creditPurchaseService.PostInvoicePaymentAuthorized,
		})

	case billing.StandardInvoiceStatusPaid:
		return s.handleChargeEvent(ctx, invoice, processorByType{
			flatFee:        s.flatFeeService.PostInvoicePaymentSettled,
			usageBased:     noop[usagebased.Charge],
			creditPurchase: s.creditPurchaseService.PostInvoicePaymentSettled,
		})
	}

	return nil
}

func (s *service) handleChargeEvent(ctx context.Context, invoice billing.StandardInvoice, processorByType processorByType) error {
	linesWithCharges, err := s.getLinesWithChargesForStandardInvoice(ctx, invoice.Namespace, invoice)
	if err != nil {
		return err
	}

	for _, lineWithCharge := range linesWithCharges {
		switch lineWithCharge.Charge.Type() {
		case meta.ChargeTypeFlatFee:
			flatFee, err := lineWithCharge.Charge.AsFlatFeeCharge()
			if err != nil {
				return err
			}

			if processorByType.flatFee == nil {
				return fmt.Errorf("flat fee payment post processor is not supported")
			}

			err = processorByType.flatFee(ctx, flatFee, lineWithCharge.StandardLineWithInvoiceHeader)
			if err != nil {
				return err
			}
		case meta.ChargeTypeCreditPurchase:
			cp, err := lineWithCharge.Charge.AsCreditPurchaseCharge()
			if err != nil {
				return err
			}

			if processorByType.creditPurchase == nil {
				return fmt.Errorf("credit purchase payment post processor is not supported")
			}

			err = processorByType.creditPurchase(ctx, cp, lineWithCharge.StandardLineWithInvoiceHeader)
			if err != nil {
				return err
			}
		case meta.ChargeTypeUsageBased:
			usageBased, err := lineWithCharge.Charge.AsUsageBasedCharge()
			if err != nil {
				return err
			}

			if processorByType.usageBased == nil {
				return fmt.Errorf("usage based payment post processor is not supported")
			}

			err = processorByType.usageBased(ctx, usageBased, lineWithCharge.StandardLineWithInvoiceHeader)
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

func (h *standardInvoiceEventHandler) PostCreate(ctx context.Context, invoice *billing.StandardInvoice) error {
	// The creation can be treated as an update from out perspective for the draft.created state.
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

	referencedCharges, err := s.GetByIDs(ctx, charges.GetByIDsInput{
		Namespace: ns,
		IDs: lo.Map(linesWithChargeID, func(l billing.StandardLineWithInvoiceHeader, _ int) string {
			return lo.FromPtr(l.Line.ChargeID)
		}),
		Expands: meta.Expands{
			meta.ExpandRealizations,
		},
	})
	if err != nil {
		return nil, err
	}

	chargesByID := make(map[meta.ChargeID]charges.Charge, len(referencedCharges))
	for _, charge := range referencedCharges {
		id, err := charge.GetChargeID()
		if err != nil {
			return nil, err
		}
		chargesByID[id] = charge
	}

	return slicesx.MapWithErr(linesWithChargeID, func(lineWithHeader billing.StandardLineWithInvoiceHeader) (standardLineWithCharge, error) {
		chargeID := *lineWithHeader.Line.ChargeID

		charge, ok := chargesByID[meta.ChargeID{
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
		return err
	})
	if err != nil {
		return lo.Empty[T](), err
	}

	return out, nil
}
