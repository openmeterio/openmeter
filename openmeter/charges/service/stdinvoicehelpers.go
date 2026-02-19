package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/pkg/models"
)

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

type standardLineWithChargeID struct {
	ChargeID string
	billing.StandardLineWithInvoiceHeader
}

func (s *service) getLinesWithChargesForStandardInvoice(ctx context.Context, ns string, invoices ...billing.StandardInvoice) ([]standardLineWithCharge, error) {
	if ns == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	if len(invoices) == 0 {
		return nil, nil
	}

	totalLines := lo.SumBy(invoices, func(invoice billing.StandardInvoice) int {
		return invoice.Lines.NonDeletedLineCount()
	})

	linesWithChargeID := make([]standardLineWithChargeID, 0, totalLines)
	for _, invoice := range invoices {
		for _, line := range invoice.Lines.OrEmpty() {
			if line.ChargeID == nil {
				continue
			}

			linesWithChargeID = append(linesWithChargeID, standardLineWithChargeID{
				ChargeID: *line.ChargeID,
				StandardLineWithInvoiceHeader: billing.StandardLineWithInvoiceHeader{
					Line:    line,
					Invoice: invoice,
				},
			})
		}
	}

	referencedCharges, err := s.GetChargesByIDs(ctx,
		ns,
		lo.Map(linesWithChargeID, func(l standardLineWithChargeID, _ int) string {
			return l.ChargeID
		}),
	)
	if err != nil {
		return nil, err
	}

	chargesById := lo.SliceToMap(referencedCharges, func(c charges.Charge) (string, charges.Charge) {
		return c.ID, c
	})

	linesWithCharges := make([]standardLineWithCharge, 0, len(linesWithChargeID))
	for _, line := range linesWithChargeID {
		charge, ok := chargesById[line.ChargeID]
		if !ok {
			return nil, fmt.Errorf("charge not found [namespace=%s charge.id=%s]", ns, line.ChargeID)
		}

		linesWithCharges = append(linesWithCharges, standardLineWithCharge{
			Charge:                        charge,
			StandardLineWithInvoiceHeader: line.StandardLineWithInvoiceHeader,
		})
	}

	return linesWithCharges, nil
}

func (s *service) handleStandardInvoiceRealizations(ctx context.Context, invoice billing.StandardInvoice, fn func(ctx context.Context, charge charges.Charge, realization charges.StandardInvoiceRealizationWithLine) error) error {
	linesWithCharges, err := s.getLinesWithChargesForStandardInvoice(ctx, invoice.Namespace, invoice)
	if err != nil {
		return err
	}

	for _, ch := range linesWithCharges {
		realization, found := ch.Charge.Realizations.StandardInvoice.GetByLineID(ch.Line.ID)
		if !found {
			return fmt.Errorf("realization not found for line [namespace=%s charge.id=%s line.id=%s]", ch.Charge.Namespace, ch.Charge.ID, ch.Line.ID)
		}

		if err := fn(ctx, ch.Charge, charges.StandardInvoiceRealizationWithLine{
			StandardInvoiceRealization:    realization,
			StandardLineWithInvoiceHeader: ch.StandardLineWithInvoiceHeader,
		}); err != nil {
			return err
		}
	}

	return nil
}
