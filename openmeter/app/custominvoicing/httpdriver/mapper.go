package httpdriver

import (
	"fmt"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

func mapUpsertStandardInvoiceResultFromAPI(in *api.CustomInvoicingSyncResult) *billing.UpsertStandardInvoiceResult {
	if in == nil {
		return nil
	}

	res := billing.NewUpsertStandardInvoiceResult()

	if in.InvoiceNumber != nil {
		res.SetInvoiceNumber(*in.InvoiceNumber)
	}

	if in.ExternalId != nil {
		res.SetExternalID(*in.ExternalId)
	}

	if in.LineExternalIds != nil {
		for _, line := range *in.LineExternalIds {
			res.AddLineExternalID(line.LineId, line.ExternalId)
		}
	}

	if in.LineDiscountExternalIds != nil {
		for _, lineDiscount := range *in.LineDiscountExternalIds {
			res.AddLineDiscountExternalID(lineDiscount.LineDiscountId, lineDiscount.ExternalId)
		}
	}

	return res
}

func mapFinalizeStandardInvoiceResultFromAPI(in api.CustomInvoicingFinalizedRequest) *billing.FinalizeStandardInvoiceResult {
	res := billing.NewFinalizeStandardInvoiceResult()

	if in.Invoicing != nil {
		if in.Invoicing.InvoiceNumber != nil {
			res.SetInvoiceNumber(*in.Invoicing.InvoiceNumber)
		}

		if in.Invoicing.SentToCustomerAt != nil {
			res.SetSentToCustomerAt(*in.Invoicing.SentToCustomerAt)
		}
	}

	if in.Payment != nil {
		if in.Payment.ExternalId != nil {
			res.SetPaymentExternalID(*in.Payment.ExternalId)
		}
	}
	return res
}

func mapPaymentTriggerFromAPI(in api.CustomInvoicingPaymentTrigger) (billing.InvoiceTrigger, error) {
	if in == "" {
		return "", models.NewGenericValidationError(fmt.Errorf("payment trigger is required"))
	}

	// Map API trigger names to internal state machine triggers
	switch in {
	case api.CustomInvoicingPaymentTriggerPaid:
		return billing.TriggerPaid, nil
	case api.CustomInvoicingPaymentTriggerPaymentFailed:
		// Note: API uses "payment_failed" but internal trigger is "failed"
		return billing.TriggerFailed, nil
	case api.CustomInvoicingPaymentTriggerPaymentUncollectible:
		return billing.TriggerPaymentUncollectible, nil
	case api.CustomInvoicingPaymentTriggerPaymentOverdue:
		return billing.TriggerPaymentOverdue, nil
	case api.CustomInvoicingPaymentTriggerActionRequired:
		return billing.TriggerActionRequired, nil
	case api.CustomInvoicingPaymentTriggerVoid:
		return billing.TriggerVoid, nil
	default:
		return "", models.NewGenericValidationError(fmt.Errorf("unknown payment trigger: %s", in))
	}
}
