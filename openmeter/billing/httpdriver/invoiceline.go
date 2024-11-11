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
	productcatalogmodel "github.com/openmeterio/openmeter/openmeter/productcatalog/model"
	planhttpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/httpdriver"
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
		func(ctx context.Context, r *http.Request, customerID string) (CreateLineByCustomerRequest, error) {
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
				CustomerID: customerID,
				Namespace:  ns,
				Lines:      lines,
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
	discriminator, err := line.Discriminator()
	if err != nil {
		return billingentity.Line{}, fmt.Errorf("failed to get type discriminator: %w", err)
	}

	switch discriminator {
	case string(api.BillingFlatFeeLineCreateItemTypeFlatFee):
		fee, err := line.AsBillingFlatFeeLineCreateItem()
		if err != nil {
			return billingentity.Line{}, fmt.Errorf("failed to map fee line: %w", err)
		}
		return mapCreateFlatFeeLineToEntity(fee, ns)
	case string(api.BillingUsageBasedLineCreateItemTypeUsageBased):
		usageBased, err := line.AsBillingUsageBasedLineCreateItem()
		if err != nil {
			return billingentity.Line{}, fmt.Errorf("failed to map usage based line: %w", err)
		}
		return mapCreateUsageBasedLineToEntity(usageBased, ns)
	default:
		return billingentity.Line{}, fmt.Errorf("unsupported type: %s", discriminator)
	}
}

func mapCreateFlatFeeLineToEntity(line api.BillingFlatFeeLineCreateItem, ns string) (billingentity.Line, error) {
	qty, err := alpacadecimal.NewFromString(line.Quantity)
	if err != nil {
		return billingentity.Line{}, fmt.Errorf("failed to map quantity: %w", err)
	}

	amount, err := alpacadecimal.NewFromString(line.Amount)
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
			Type:        billingentity.InvoiceLineTypeFee,
			Description: line.Description,

			InvoiceID: invoiceId,
			Status:    billingentity.InvoiceLineStatusValid, // This is not settable from outside
			Currency:  currencyx.Code(line.Currency),
			Period: billingentity.Period{
				Start: line.Period.Start,
				End:   line.Period.End,
			},

			InvoiceAt: line.InvoiceAt,
			TaxConfig: mapTaxConfigToEntity(line.TaxConfig),
		},
		FlatFee: billingentity.FlatFeeLine{
			Amount:      amount,
			PaymentTerm: lo.FromPtrOr((*productcatalogmodel.PaymentTermType)(line.PaymentTerm), productcatalogmodel.InAdvancePaymentTerm),
			Quantity:    qty,
		},
	}, nil
}

func mapCreateUsageBasedLineToEntity(line api.BillingUsageBasedLineCreateItem, ns string) (billingentity.Line, error) {
	invoiceId := ""
	if line.Invoice != nil {
		invoiceId = line.Invoice.Id
	}

	price, err := planhttpdriver.AsPrice(line.Price)
	if err != nil {
		return billingentity.Line{}, fmt.Errorf("failed to map price: %w", err)
	}

	return billingentity.Line{
		LineBase: billingentity.LineBase{
			Namespace: ns,

			Metadata:    lo.FromPtrOr(line.Metadata, map[string]string{}),
			Name:        line.Name,
			Type:        billingentity.InvoiceLineTypeFee,
			Description: line.Description,

			InvoiceID: invoiceId,
			Status:    billingentity.InvoiceLineStatusValid, // This is not settable from outside
			Currency:  currencyx.Code(line.Currency),
			Period: billingentity.Period{
				Start: line.Period.Start,
				End:   line.Period.End,
			},

			InvoiceAt: line.InvoiceAt,
			TaxConfig: mapTaxConfigToEntity(line.TaxConfig),
		},
		UsageBased: billingentity.UsageBasedLine{
			Price:      price,
			FeatureKey: line.FeatureKey,
		},
	}, nil
}

func mapTaxConfigToEntity(tc *api.TaxConfig) *billingentity.TaxConfig {
	if tc == nil {
		return nil
	}

	return lo.ToPtr(planhttpdriver.AsTaxConfig(*tc))
}

func mapTaxConfigToAPI(to *billingentity.TaxConfig) *api.TaxConfig {
	if to == nil {
		return nil
	}

	return lo.ToPtr(planhttpdriver.FromTaxConfig(*to))
}

func mapBillingLineToAPI(line billingentity.Line) (api.BillingInvoiceLine, error) {
	switch line.Type {
	case billingentity.InvoiceLineTypeFee:
		return mapFeeLineToAPI(line)
	case billingentity.InvoiceLineTypeUsageBased:
		return mapUsageBasedLineToAPI(line)
	default:
		return api.BillingInvoiceLine{}, fmt.Errorf("unsupported type: %s", line.Type)
	}
}

func mapFeeLineToAPI(line billingentity.Line) (api.BillingInvoiceLine, error) {
	feeLine := api.BillingFlatFeeLine{
		Type: api.BillingFlatFeeLineTypeFlatFee,
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

		Amount:    line.FlatFee.Amount.String(),
		Quantity:  line.FlatFee.Quantity.String(),
		TaxConfig: mapTaxConfigToAPI(line.TaxConfig),
	}

	out := api.BillingInvoiceLine{}
	err := out.FromBillingFlatFeeLine(feeLine)
	if err != nil {
		return api.BillingInvoiceLine{}, fmt.Errorf("failed to map fee line: %w", err)
	}

	return out, nil
}

func mapUsageBasedLineToAPI(line billingentity.Line) (api.BillingInvoiceLine, error) {
	price, err := mapPriceToAPI(line.UsageBased.Price)
	if err != nil {
		return api.BillingInvoiceLine{}, fmt.Errorf("failed to map price: %w", err)
	}

	ubpLine := api.BillingUsageBasedLine{
		Type: api.BillingUsageBasedLineTypeUsageBased,
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

		TaxConfig: mapTaxConfigToAPI(line.TaxConfig),

		FeatureKey: line.UsageBased.FeatureKey,
		Quantity:   decimalPtrToStringPtr(line.UsageBased.Quantity),
		Price:      price,
	}

	out := api.BillingInvoiceLine{}

	if err := out.FromBillingUsageBasedLine(ubpLine); err != nil {
		return api.BillingInvoiceLine{}, fmt.Errorf("failed to map fee line: %w", err)
	}

	return out, nil
}

func decimalPtrToStringPtr(d *alpacadecimal.Decimal) *string {
	if d == nil {
		return nil
	}

	return lo.ToPtr(d.String())
}

func decimalPtrToFloat64Ptr(d *alpacadecimal.Decimal) *float64 {
	if d == nil {
		return nil
	}

	return lo.ToPtr(d.InexactFloat64())
}

func mapPriceToAPI(price productcatalogmodel.Price) (api.RateCardUsageBasedPrice, error) {
	switch price.Type() {
	case productcatalogmodel.FlatPriceType:
		flatPrice, err := price.AsFlat()
		if err != nil {
			return api.RateCardUsageBasedPrice{}, fmt.Errorf("failed to map flat price: %w", err)
		}
		return mapFlatPriceToAPI(flatPrice)
	case productcatalogmodel.UnitPriceType:
		unitPriceType, err := price.AsUnit()
		if err != nil {
			return api.RateCardUsageBasedPrice{}, fmt.Errorf("failed to map unit price: %w", err)
		}

		return mapUnitPriceToAPI(unitPriceType)
	case productcatalogmodel.TieredPriceType:
		tieredPriceType, err := price.AsTiered()
		if err != nil {
			return api.RateCardUsageBasedPrice{}, fmt.Errorf("failed to map tiered price: %w", err)
		}

		return mapTieredPriceToAPI(tieredPriceType)
	default:
		return api.RateCardUsageBasedPrice{}, fmt.Errorf("unsupported price type: %s", price.Type())
	}
}

func mapFlatPriceToAPI(p productcatalogmodel.FlatPrice) (api.RateCardUsageBasedPrice, error) {
	out := api.RateCardUsageBasedPrice{}

	err := out.FromFlatPriceWithPaymentTerm(api.FlatPriceWithPaymentTerm{
		Amount:      p.Amount.String(),
		PaymentTerm: lo.ToPtr(api.PricePaymentTerm(p.PaymentTerm)),
		Type:        api.FlatPriceWithPaymentTermType(p.Type),
	})
	if err != nil {
		return api.RateCardUsageBasedPrice{}, fmt.Errorf("failed to map flat price: %w", err)
	}

	return out, nil
}

func mapUnitPriceToAPI(p productcatalogmodel.UnitPrice) (api.RateCardUsageBasedPrice, error) {
	out := api.RateCardUsageBasedPrice{}

	err := out.FromUnitPriceWithCommitments(api.UnitPriceWithCommitments{
		Amount:        p.Amount.String(),
		MaximumAmount: decimalPtrToStringPtr(p.MaximumAmount),
		MinimumAmount: decimalPtrToStringPtr(p.MinimumAmount),
		Type:          api.UnitPriceWithCommitmentsType(p.Type),
	})
	if err != nil {
		return api.RateCardUsageBasedPrice{}, fmt.Errorf("failed to map unit price: %w", err)
	}

	return out, nil
}

func mapTieredPriceToAPI(p productcatalogmodel.TieredPrice) (api.RateCardUsageBasedPrice, error) {
	out := api.RateCardUsageBasedPrice{}

	tiers := lo.Map(p.Tiers, func(t productcatalogmodel.PriceTier, _ int) api.PriceTier {
		res := api.PriceTier{
			UpToAmount: decimalPtrToFloat64Ptr(t.UpToAmount),
		}

		if t.FlatPrice != nil {
			res.FlatPrice = &api.FlatPrice{
				Amount: t.FlatPrice.Amount.String(),
				Type:   api.FlatPriceType(t.FlatPrice.Type),
			}
		}

		if t.UnitPrice != nil {
			res.UnitPrice = &api.UnitPrice{
				Amount: t.UnitPrice.Amount.String(),
				Type:   api.UnitPriceType(t.UnitPrice.Type),
			}
		}
		return res
	})

	err := out.FromTieredPriceWithCommitments(api.TieredPriceWithCommitments{
		Tiers:         tiers,
		Mode:          api.TieredPriceMode(p.Mode),
		MinimumAmount: decimalPtrToStringPtr(p.MinimumAmount),
		MaximumAmount: decimalPtrToStringPtr(p.MaximumAmount),
		Type:          api.TieredPriceWithCommitmentsType(p.Type),
	})
	if err != nil {
		return api.RateCardUsageBasedPrice{}, fmt.Errorf("failed to map tiered price: %w", err)
	}

	return out, nil
}
