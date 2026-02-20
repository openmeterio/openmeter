package service

import (
	"context"
	"fmt"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/charges"
)

func (s *service) addStandardInvoiceRealization(ctx context.Context, charge charges.Charge, l billing.StandardLineWithInvoiceHeader) (charges.Charge, charges.StandardInvoiceRealization, error) {
	realization := charges.StandardInvoiceRealization{
		LineID:        l.Line.ID,
		ServicePeriod: l.Line.Period.ToClosedPeriod(),
		Status:        mapInvoiceStateToRealizationStatus(l.Invoice.Status),

		MeteredServicePeriodQuantity: lo.FromPtr(
			lo.CoalesceOrEmpty(
				l.Line.UsageBased.MeteredQuantity,
				l.Line.UsageBased.Quantity,
			),
		),
		MeteredPreServicePeriodQuantity: lo.FromPtr(
			lo.CoalesceOrEmpty(
				l.Line.UsageBased.MeteredPreLinePeriodQuantity,
				l.Line.UsageBased.PreLinePeriodQuantity,
			),
		),

		Totals: l.Line.Totals,
	}

	realization, err := s.adapter.CreateStandardInvoiceRealization(ctx, charge.GetChargeID(), realization)
	if err != nil {
		return charge, realization, err
	}

	charge.Realizations.StandardInvoice = append(charge.Realizations.StandardInvoice, realization)

	return charge, realization, nil
}

func (s *service) updateStandardInvoiceRealizationByID(ctx context.Context, charge charges.Charge, realization charges.StandardInvoiceRealization) (charges.Charge, error) {
	// TODO: We might want to make this an intent based API such as set realization state to prevent accidental updates to the underlying realization.
	realization, err := s.adapter.UpdateStandardInvoiceRealizationByID(ctx, charge.GetChargeID(), realization)
	if err != nil {
		return charge, err
	}

	for idx, r := range charge.Realizations.StandardInvoice {
		if r.ID == realization.ID {
			charge.Realizations.StandardInvoice[idx] = realization
			return charge, nil
		}
	}

	return charge, fmt.Errorf("realization not found [namespace=%s charge.id=%s realization.id=%s]", charge.Namespace, charge.ID, realization.ID)
}

func mapInvoiceStateToRealizationStatus(state billing.StandardInvoiceStatus) charges.StandardInvoiceRealizationStatus {
	if state.IsFinal() {
		return charges.StandardInvoiceRealizationStatusSettled
	}

	shortStatus := state.ShortStatus()
	if slices.Contains(billing.StandardInvoiceMutableStatusCategories, billing.StandardInvoiceStatusCategory(shortStatus)) {
		return charges.StandardInvoiceRealizationStatusDraft
	}

	return charges.StandardInvoiceRealizationStatusAuthorized
}
