package apps

import (
	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/billing"
)

func mapUpsertInvoiceResultFromAPI(in *api.BillingAppCustomInvoicingSyncResult) *billing.UpsertInvoiceResult {
	if in == nil {
		return nil
	}

	res := billing.NewUpsertInvoiceResult()

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

func mapFinalizeInvoiceResultFromAPI(in api.BillingAppCustomInvoicingFinalizedRequest) *billing.FinalizeInvoiceResult {
	res := billing.NewFinalizeInvoiceResult()

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
