package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	CreateLineByCustomerRequest  = billing.CreateInvoiceLinesInput
	CreateLineByCustomerResponse = api.BillingCreateLineResult
	CreateLineByCustomerHandler  httptransport.HandlerWithArgs[CreateLineByCustomerRequest, CreateLineByCustomerResponse, string]
)

func (h *handler) CreateLineByCustomer() CreateLineByCustomerHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerKeyOrId string) (CreateLineByCustomerRequest, error) {
			body := api.BillingCreateLineByCustomerJSONRequestBody{}

			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreateLineByCustomerRequest{}, fmt.Errorf("failed to decode request body: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateLineByCustomerRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			lines := make([]billingentity.Line, 0, len(body.Lines))
			for _, line := range body.Lines {
				line, err := mapCreateLineToEntity(line, ns)
				if err != nil {
					return CreateLineByCustomerRequest{}, fmt.Errorf("failed to map line: %w", err)
				}
				lines = append(lines, line)
			}

			return CreateLineByCustomerRequest{
				CustomerKeyOrID: customerKeyOrId,
				Namespace:       ns,
				Lines:           lines,
			}, nil
		},
		func(ctx context.Context, request CreateLineByCustomerRequest) (CreateLineByCustomerResponse, error) {
			lines, err := h.service.CreateInvoiceLines(ctx, request)
			if err != nil {
				return CreateLineByCustomerResponse{}, fmt.Errorf("failed to create invoice lines: %w", err)
			}

			res := CreateLineByCustomerResponse{
				Lines: make([]api.BillingInvoiceLine, 0, len(lines.Lines)),
			}

			for _, line := range lines.Lines {
				line, err := mapBillingLineToAPI(line)
				if err != nil {
					return CreateLineByCustomerResponse{}, fmt.Errorf("failed to map line: %w", err)
				}
				res.Lines = append(res.Lines, line)
			}

			return res, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateLineByCustomerResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("billingCreateLineByCustomer"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

func mapCreateLineToEntity(line api.BillingInvoiceLineCreateItem, ns string) (billingentity.Line, error) {
	// This should not fail, and we would have at least the discriminator unmarshaled
	manualFee, err := line.AsBillingManualFeeLineCreateItem()
	if err != nil {
		return billingentity.Line{}, fmt.Errorf("failed to map manual fee line: %w", err)
	}

	switch string(manualFee.Type) {
	case string(api.BillingManualFeeLineTypeManualFee):
		return mapCreateManualFeeLineToEntity(manualFee, ns)
	default:
		return billingentity.Line{}, fmt.Errorf("unsupported type: %s", manualFee.Type)
	}
}

func mapCreateManualFeeLineToEntity(line api.BillingManualFeeLineCreateItem, ns string) (billingentity.Line, error) {
	qty, err := mapStringPtrToDecimal(line.Quantity)
	if err != nil {
		return billingentity.Line{}, fmt.Errorf("failed to map quantity: %w", err)
	}

	price, err := alpacadecimal.NewFromString(line.Price)
	if err != nil {
		return billingentity.Line{}, fmt.Errorf("failed to parse price: %w", err)
	}

	invoiceId := ""
	if line.Invoice != nil {
		invoiceId = line.Invoice.Id
	}

	return billingentity.Line{
		LineBase: billingentity.LineBase{
			Namespace: ns,

			Metadata:    lo.FromPtrOr(line.Metadata, map[string]string{}),
			Name:        line.Name,
			Type:        billingentity.InvoiceLineTypeManualFee,
			Description: line.Description,

			InvoiceID: invoiceId,
			Status:    billingentity.InvoiceLineStatusValid, // This is not settable from outside
			Currency:  currencyx.Code(line.Currency),
			Period: billingentity.Period{
				Start: line.Period.Start,
				End:   line.Period.End,
			},

			InvoiceAt:    line.InvoiceAt,
			TaxOverrides: mapTaxConfigToEntity(line.TaxOverrides),
			Quantity:     qty,
		},
		ManualFee: &billingentity.ManualFeeLine{
			Price: price,
		},
	}, nil
}

func mapTaxConfigToEntity(tc *api.TaxConfig) *billingentity.TaxOverrides {
	if tc == nil {
		return nil
	}

	out := &billingentity.TaxOverrides{}

	if tc.Stripe != nil && tc.Stripe.Code != "" {
		out.Stripe = &billingentity.StripeTaxOverride{
			TaxCode: billingentity.StripeTaxCode(tc.Stripe.Code),
		}
	}

	return out
}

func mapTaxOverridesToAPI(to *billingentity.TaxOverrides) *api.TaxConfig {
	if to == nil {
		return nil
	}

	out := &api.TaxConfig{}

	if to.Stripe != nil {
		out.Stripe = &api.StripeTaxConfig{
			Code: string(to.Stripe.TaxCode),
		}
	}

	return out
}

func mapStringPtrToDecimal(s *string) (*alpacadecimal.Decimal, error) {
	if s == nil {
		return nil, nil
	}

	qty, err := alpacadecimal.NewFromString(*s)
	if err != nil {
		return nil, fmt.Errorf("failed to parse decimal: %w", err)
	}

	return &qty, nil
}

func mapBillingLineToAPI(line billingentity.Line) (api.BillingInvoiceLine, error) {
	switch line.Type {
	case billingentity.InvoiceLineTypeManualFee:
		return mapManualFeeLineToAPI(line)
	default:
		return api.BillingInvoiceLine{}, fmt.Errorf("unsupported type: %s", line.Type)
	}
}

func mapManualFeeLineToAPI(line billingentity.Line) (api.BillingInvoiceLine, error) {
	if line.ManualFee == nil {
		return api.BillingInvoiceLine{}, fmt.Errorf("manual fee line is nil")
	}

	feeLine := api.BillingManualFeeLine{
		Type: api.BillingManualFeeLineTypeManualFee,
		Id:   line.ID,

		CreatedAt: line.CreatedAt,
		DeletedAt: line.DeletedAt,
		UpdatedAt: line.UpdatedAt,
		InvoiceAt: line.InvoiceAt,

		Currency: string(line.Currency),

		Description: line.Description,
		Name:        line.Name,

		Invoice: &api.BillingInvoiceReference{
			Id: line.InvoiceID,
		},

		Metadata: lo.EmptyableToPtr(line.Metadata),
		Period: api.BillingPeriod{
			Start: line.Period.Start,
			End:   line.Period.End,
		},

		Price:        line.ManualFee.Price.String(),
		Quantity:     lo.ToPtr(line.Quantity.String()),
		TaxOverrides: mapTaxOverridesToAPI(line.TaxOverrides),
	}

	out := api.BillingInvoiceLine{}
	err := out.FromBillingManualFeeLine(feeLine)
	if err != nil {
		return api.BillingInvoiceLine{}, fmt.Errorf("failed to map manual fee line: %w", err)
	}

	return out, nil
}
