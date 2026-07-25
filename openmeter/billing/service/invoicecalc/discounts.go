package invoicecalc

import "github.com/openmeterio/openmeter/openmeter/billing"

func UpsertDiscountCorrelationIDs(invoice *billing.StandardInvoice) error {
	lines := invoice.Lines.OrEmpty()
	for _, line := range lines {
		line.RateCardDiscounts = line.RateCardDiscounts.UpsertCorrelationIDs()
	}

	return nil
}

func UpsertGatheringInvoiceDiscountCorrelationIDs(invoice *billing.GatheringInvoice) error {
	lines, err := invoice.Lines.MapWithErr(func(line billing.GatheringLine) (billing.GatheringLine, error) {
		line.RateCardDiscounts = line.RateCardDiscounts.UpsertCorrelationIDs()

		return line, nil
	})
	if err != nil {
		return err
	}

	invoice.Lines = lines

	return nil
}
