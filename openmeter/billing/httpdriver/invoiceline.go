package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/equal"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/set"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var _ InvoiceLineHandler = (*handler)(nil)

type (
	CreatePendingLineRequest  = billing.CreatePendingInvoiceLinesInput
	CreatePendingLineResponse = api.InvoicePendingLineCreateResponse
	CreatePendingLineParams   struct {
		CustomerID string
	}
	CreatePendingLineHandler = httptransport.HandlerWithArgs[CreatePendingLineRequest, CreatePendingLineResponse, CreatePendingLineParams]
)

const (
	defaultFlatFeePaymentTerm        = productcatalog.InAdvancePaymentTerm
	defaultFlatFeeQuantity    string = "1"
)

func (h *handler) CreatePendingLine() CreatePendingLineHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params CreatePendingLineParams) (CreatePendingLineRequest, error) {
			req := api.InvoicePendingLineCreateInput{}

			if err := commonhttp.JSONRequestBodyDecoder(r, &req); err != nil {
				return CreatePendingLineRequest{}, fmt.Errorf("failed to decode request body: %w", err)
			}

			// TODO[OM-982]: limit to single depth, valid line creation

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreatePendingLineRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			if len(req.Lines) == 0 {
				return CreatePendingLineRequest{}, billing.ValidationError{
					Err: fmt.Errorf("no lines provided"),
				}
			}

			lineEntities, err := slicesx.MapWithErr(req.Lines, func(line api.InvoicePendingLineCreate) (*billing.Line, error) {
				return mapCreateLineToEntity(line, ns)
			})
			if err != nil {
				return CreatePendingLineRequest{}, billing.ValidationError{
					Err: fmt.Errorf("failed to map lines: %w", err),
				}
			}

			if h.featureSwitches.UseUsageBasedFlatFeeLines {
				for _, line := range lineEntities {
					if line.Type == billing.InvoiceLineTypeFee {
						return CreatePendingLineRequest{}, billing.ValidationError{
							Err: fmt.Errorf("creating flat fee lines is not supported, please use usage based lines instead"),
						}
					}
				}
			}

			return CreatePendingLineRequest{
				Customer: customer.CustomerID{
					Namespace: ns,
					ID:        params.CustomerID,
				},
				Currency: currencyx.Code(req.Currency),
				Lines:    lineEntities,
			}, nil
		},
		func(ctx context.Context, request CreatePendingLineRequest) (CreatePendingLineResponse, error) {
			res, err := h.service.CreatePendingInvoiceLines(ctx, request)
			if err != nil {
				return CreatePendingLineResponse{}, fmt.Errorf("failed to create invoice lines: %w", err)
			}

			out := CreatePendingLineResponse{
				IsInvoiceNew: res.IsInvoiceNew,
			}

			out.Invoice, err = MapInvoiceToAPI(res.Invoice)
			if err != nil {
				return CreatePendingLineResponse{}, fmt.Errorf("failed to map invoice: %w", err)
			}

			out.Lines, err = slicesx.MapWithErr(res.Lines, func(line *billing.Line) (api.InvoiceLine, error) {
				return mapBillingLineToAPI(line)
			})
			if err != nil {
				return CreatePendingLineResponse{}, fmt.Errorf("failed to map lines: %w", err)
			}

			return out, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreatePendingLineResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("CreateInvoiceLineByCustomer"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

func mapCreateLineToEntity(line api.InvoicePendingLineCreate, ns string) (*billing.Line, error) {
	// This should not fail, and we would have at least the discriminator unmarshaled
	discriminator, err := line.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("failed to get type discriminator: %w", err)
	}

	switch discriminator {
	case string(api.InvoiceFlatFeeLineTypeFlatFee):
		fee, err := line.AsInvoiceFlatFeePendingLineCreate()
		if err != nil {
			return nil, fmt.Errorf("failed to map fee line: %w", err)
		}
		return mapCreatePendingFlatFeeLineToEntity(fee, ns)
	case string(api.InvoiceUsageBasedLineTypeUsageBased):
		usageBased, err := line.AsInvoiceUsageBasedPendingLineCreate()
		if err != nil {
			return nil, fmt.Errorf("failed to map usage based line: %w", err)
		}
		return mapCreatePendingUsageBasedLineToEntity(usageBased, ns)
	default:
		return nil, fmt.Errorf("unsupported type: %s", discriminator)
	}
}

func mapCreatePendingFlatFeeLineToEntity(line api.InvoiceFlatFeePendingLineCreate, ns string) (*billing.Line, error) {
	rateCardParsed, err := mapAndValidateFlatFeeRateCardDeprecatedFields(flatFeeRateCardItems{
		RateCard:      line.RateCard,
		PerUnitAmount: line.PerUnitAmount,
		PaymentTerm:   line.PaymentTerm,
		Quantity:      line.Quantity,
		TaxConfig:     line.TaxConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to map flat fee line: %w", err)
	}

	return &billing.Line{
		LineBase: billing.LineBase{
			Namespace: ns,

			Metadata:    lo.FromPtrOr(line.Metadata, map[string]string{}),
			Name:        line.Name,
			Type:        billing.InvoiceLineTypeFee,
			Description: line.Description,
			ManagedBy:   billing.ManuallyManagedLine,

			Status: billing.InvoiceLineStatusValid, // This is not settable from outside
			Period: billing.Period{
				Start: line.Period.From,
				End:   line.Period.To,
			},

			InvoiceAt:         line.InvoiceAt,
			TaxConfig:         rateCardParsed.TaxConfig,
			RateCardDiscounts: rateCardParsed.Discounts,
		},
		FlatFee: &billing.FlatFeeLine{
			PerUnitAmount: rateCardParsed.PerUnitAmount,
			PaymentTerm:   rateCardParsed.PaymentTerm,
			Quantity:      rateCardParsed.Quantity,
			Category:      lo.FromPtrOr((*billing.FlatFeeCategory)(line.Category), billing.FlatFeeCategoryRegular),
		},
	}, nil
}

func mapCreatePendingUsageBasedLineToEntity(line api.InvoiceUsageBasedPendingLineCreate, ns string) (*billing.Line, error) {
	rateCardParsed, err := mapAndValidateUsageBasedRateCardDeprecatedFields(usageBasedRateCardItems{
		RateCard:   line.RateCard,
		Price:      line.Price,
		TaxConfig:  line.TaxConfig,
		FeatureKey: line.FeatureKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to map usage based line: %w", err)
	}

	return &billing.Line{
		LineBase: billing.LineBase{
			Namespace: ns,

			Metadata:    lo.FromPtrOr(line.Metadata, map[string]string{}),
			Name:        line.Name,
			Type:        billing.InvoiceLineTypeUsageBased,
			Description: line.Description,
			ManagedBy:   billing.ManuallyManagedLine,

			Status: billing.InvoiceLineStatusValid, // This is not settable from outside
			Period: billing.Period{
				Start: line.Period.From,
				End:   line.Period.To,
			},

			InvoiceAt:         line.InvoiceAt,
			TaxConfig:         rateCardParsed.TaxConfig,
			RateCardDiscounts: rateCardParsed.Discounts,
		},
		UsageBased: &billing.UsageBasedLine{
			Price:      rateCardParsed.Price,
			FeatureKey: rateCardParsed.FeatureKey,
		},
	}, nil
}

func mapTaxConfigToEntity(tc *api.TaxConfig) *productcatalog.TaxConfig {
	if tc == nil {
		return nil
	}

	return lo.ToPtr(productcataloghttp.AsTaxConfig(*tc))
}

func mapTaxConfigToAPI(to *productcatalog.TaxConfig) *api.TaxConfig {
	if to == nil {
		return nil
	}

	return lo.ToPtr(productcataloghttp.FromTaxConfig(*to))
}

func mapBillingLineToAPI(line *billing.Line) (api.InvoiceLine, error) {
	switch line.Type {
	case billing.InvoiceLineTypeFee:
		return mapFeeLineToAPI(line)
	case billing.InvoiceLineTypeUsageBased:
		return mapUsageBasedLineToAPI(line)
	default:
		return api.InvoiceLine{}, fmt.Errorf("unsupported type: %s", line.Type)
	}
}

func mapChildLinesToAPI(optChildren billing.LineChildren) (*[]api.InvoiceLine, error) {
	if optChildren.IsAbsent() {
		return nil, nil
	}

	children := optChildren.OrEmpty()

	out := make([]api.InvoiceLine, 0, len(children))

	for _, child := range children {
		mappedLine, err := mapBillingLineToAPI(child)
		if err != nil {
			return nil, fmt.Errorf("failed to map child line: %w", err)
		}
		out = append(out, mappedLine)
	}

	return &out, nil
}

func mapFeeLineToAPI(line *billing.Line) (api.InvoiceLine, error) {
	children, err := mapChildLinesToAPI(line.Children)
	if err != nil {
		return api.InvoiceLine{}, fmt.Errorf("failed to map children: %w", err)
	}

	discountsAPI, err := mapDiscountsToAPI(line.Discounts)
	if err != nil {
		return api.InvoiceLine{}, fmt.Errorf("failed to map discounts: %w", err)
	}

	feeLine := api.InvoiceFlatFeeLine{
		Type: api.InvoiceFlatFeeLineTypeFlatFee,
		Id:   line.ID,

		CreatedAt: line.CreatedAt,
		DeletedAt: line.DeletedAt,
		UpdatedAt: line.UpdatedAt,
		InvoiceAt: line.InvoiceAt,

		Currency: string(line.Currency),
		Status:   api.InvoiceLineStatus(line.Status),

		Description: line.Description,
		Name:        line.Name,
		ManagedBy:   api.InvoiceLineManagedBy(line.ManagedBy),

		Invoice: &api.InvoiceReference{
			Id: line.InvoiceID,
		},

		Metadata: convert.MapToPointer(line.Metadata),
		Period: api.Period{
			From: line.Period.Start,
			To:   line.Period.End,
		},

		PerUnitAmount: lo.ToPtr(line.FlatFee.PerUnitAmount.String()),
		Quantity:      lo.ToPtr(line.FlatFee.Quantity.String()),
		Category:      lo.ToPtr(api.InvoiceFlatFeeCategory(line.FlatFee.Category)),
		TaxConfig:     mapTaxConfigToAPI(line.TaxConfig),
		PaymentTerm:   lo.ToPtr(api.PricePaymentTerm(line.FlatFee.PaymentTerm)),

		RateCard: &api.InvoiceFlatFeeRateCard{
			TaxConfig: mapTaxConfigToAPI(line.TaxConfig),
			Price: &api.FlatPriceWithPaymentTerm{
				Type:        api.FlatPriceWithPaymentTermTypeFlat,
				PaymentTerm: lo.ToPtr(api.PricePaymentTerm(line.FlatFee.PaymentTerm)),
				Amount:      line.FlatFee.PerUnitAmount.String(),
			},
			Quantity: lo.ToPtr(line.FlatFee.Quantity.String()),
		},

		Discounts: discountsAPI,
		Totals:    mapTotalsToAPI(line.Totals),
		Children:  children,

		ExternalIds:  mapLineAppExternalIdsToAPI(line.ExternalIDs),
		Subscription: mapSubscriptionReferencesToAPI(line.Subscription),
	}

	out := api.InvoiceLine{}
	err = out.FromInvoiceFlatFeeLine(feeLine)
	if err != nil {
		return api.InvoiceLine{}, fmt.Errorf("failed to map fee line: %w", err)
	}

	return out, nil
}

func mapLineAppExternalIdsToAPI(externalIds billing.LineExternalIDs) *api.InvoiceLineAppExternalIds {
	if lo.IsEmpty(externalIds) {
		return nil
	}

	return &api.InvoiceLineAppExternalIds{
		Invoicing: lo.EmptyableToPtr(externalIds.Invoicing),
	}
}

func mapUsageBasedLineToAPI(line *billing.Line) (api.InvoiceLine, error) {
	if line.UsageBased.Price == nil {
		return api.InvoiceLine{}, fmt.Errorf("price is nil [line=%s]", line.ID)
	}

	price, err := productcataloghttp.FromRateCardUsageBasedPrice(*line.UsageBased.Price)
	if err != nil {
		return api.InvoiceLine{}, fmt.Errorf("failed to map price: %w", err)
	}

	children, err := mapChildLinesToAPI(line.Children)
	if err != nil {
		return api.InvoiceLine{}, fmt.Errorf("failed to map children: %w", err)
	}

	discountsAPI, err := mapDiscountsToAPI(line.Discounts)
	if err != nil {
		return api.InvoiceLine{}, fmt.Errorf("failed to map discounts: %w", err)
	}

	ubpLine := api.InvoiceUsageBasedLine{
		Type: api.InvoiceUsageBasedLineTypeUsageBased,
		Id:   line.ID,

		CreatedAt: line.CreatedAt,
		DeletedAt: line.DeletedAt,
		UpdatedAt: line.UpdatedAt,
		InvoiceAt: line.InvoiceAt,

		Currency: string(line.Currency),
		Status:   api.InvoiceLineStatus(line.Status),

		Description: line.Description,
		Name:        line.Name,
		ManagedBy:   api.InvoiceLineManagedBy(line.ManagedBy),

		Invoice: &api.InvoiceReference{
			Id: line.InvoiceID,
		},

		Metadata: convert.MapToPointer(line.Metadata),
		Period: api.Period{
			From: line.Period.Start,
			To:   line.Period.End,
		},

		TaxConfig: mapTaxConfigToAPI(line.TaxConfig),

		FeatureKey:                   lo.EmptyableToPtr(line.UsageBased.FeatureKey),
		MeteredQuantity:              decimalPtrToStringPtrIfNotEqual(line.UsageBased.MeteredQuantity, line.UsageBased.Quantity),
		Quantity:                     decimalPtrToStringPtr(line.UsageBased.Quantity),
		PreLinePeriodQuantity:        decimalPtrToStringPtrIgnoringZeroValue(line.UsageBased.PreLinePeriodQuantity),
		MeteredPreLinePeriodQuantity: decimalPtrToStringPtrIgnoringZeroValue(line.UsageBased.MeteredPreLinePeriodQuantity),

		Price: lo.ToPtr(price),

		RateCard: &api.InvoiceUsageBasedRateCard{
			TaxConfig:  mapTaxConfigToAPI(line.TaxConfig),
			Price:      lo.ToPtr(price),
			FeatureKey: lo.EmptyableToPtr(line.UsageBased.FeatureKey),
		},

		Discounts: discountsAPI,
		Children:  children,
		Totals:    mapTotalsToAPI(line.Totals),

		ExternalIds:  mapLineAppExternalIdsToAPI(line.ExternalIDs),
		Subscription: mapSubscriptionReferencesToAPI(line.Subscription),
	}

	out := api.InvoiceLine{}

	if err := out.FromInvoiceUsageBasedLine(ubpLine); err != nil {
		return api.InvoiceLine{}, fmt.Errorf("failed to map fee line: %w", err)
	}

	return out, nil
}

func mapSubscriptionReferencesToAPI(optSub *billing.SubscriptionReference) *api.InvoiceLineSubscriptionReference {
	if optSub == nil {
		return nil
	}

	return &api.InvoiceLineSubscriptionReference{
		Item: api.IDResource{
			Id: optSub.SubscriptionID,
		},
		Phase: api.IDResource{
			Id: optSub.PhaseID,
		},
		Subscription: api.IDResource{
			Id: optSub.ItemID,
		},
	}
}

func mapDiscountsToAPI(discounts billing.LineDiscounts) (*api.InvoiceLineDiscounts, error) {
	if discounts.IsEmpty() {
		return nil, nil
	}
	out := api.InvoiceLineDiscounts{}

	if len(discounts.Amount) > 0 {
		mapped, err := slicesx.MapWithErr(discounts.Amount, func(discount billing.AmountLineDiscountManaged) (api.InvoiceLineAmountDiscount, error) {
			reason, err := mapDiscountReasonToAPI(discount.Reason)
			if err != nil {
				return api.InvoiceLineAmountDiscount{}, fmt.Errorf("failed to map discount reason: %w", err)
			}

			return api.InvoiceLineAmountDiscount{
				Id:          discount.ID,
				Amount:      discount.Amount.String(),
				CreatedAt:   discount.CreatedAt,
				DeletedAt:   discount.DeletedAt,
				UpdatedAt:   discount.UpdatedAt,
				Description: discount.Description,
				ExternalIds: mapLineAppExternalIdsToAPI(discount.ExternalIDs),
				Reason:      reason,
			}, nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to map amount discounts: %w", err)
		}

		out.Amount = lo.ToPtr(mapped)
	}

	if len(discounts.Usage) > 0 {
		mapped, err := slicesx.MapWithErr(discounts.Usage, func(discount billing.UsageLineDiscountManaged) (api.InvoiceLineUsageDiscount, error) {
			reason, err := mapDiscountReasonToAPI(discount.Reason)
			if err != nil {
				return api.InvoiceLineUsageDiscount{}, fmt.Errorf("failed to map discount reason: %w", err)
			}

			return api.InvoiceLineUsageDiscount{
				Id:                    discount.ID,
				Quantity:              discount.Quantity.String(),
				PreLinePeriodQuantity: decimalPtrToStringPtrIgnoringZeroValue(discount.PreLinePeriodQuantity),
				CreatedAt:             discount.CreatedAt,
				DeletedAt:             discount.DeletedAt,
				UpdatedAt:             discount.UpdatedAt,
				Description:           discount.Description,
				ExternalIds:           mapLineAppExternalIdsToAPI(discount.ExternalIDs),
				Reason:                reason,
			}, nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to map amount discounts: %w", err)
		}

		out.Usage = lo.ToPtr(mapped)
	}

	return &out, nil
}

func mapDiscountReasonToAPI(reason billing.DiscountReason) (api.BillingDiscountReason, error) {
	out := api.BillingDiscountReason{}
	switch reason.Type() {
	case billing.MaximumSpendDiscountReason:
		if err := out.FromDiscountReasonMaximumSpend(api.DiscountReasonMaximumSpend{
			Type: api.DiscountReasonMaximumSpendTypeMaximumSpend,
		}); err != nil {
			return out, err
		}
	case billing.RatecardPercentageDiscountReason:
		reason, err := reason.AsRatecardPercentage()
		if err != nil {
			return out, err
		}

		if err := out.FromDiscountReasonRatecardPercentage(api.DiscountReasonRatecardPercentage{
			Percentage:    reason.Percentage,
			CorrelationId: lo.EmptyableToPtr(reason.CorrelationID),
			Type:          api.DiscountReasonRatecardPercentageType(billing.RatecardPercentageDiscountReason),
		}); err != nil {
			return out, err
		}
	case billing.RatecardUsageDiscountReason:
		reason, err := reason.AsRatecardUsage()
		if err != nil {
			return out, err
		}

		if err := out.FromDiscountReasonRatecardUsage(api.DiscountReasonRatecardUsage{
			Quantity:      reason.Quantity.String(),
			CorrelationId: lo.EmptyableToPtr(reason.CorrelationID),
			Type:          api.DiscountReasonRatecardUsageType(billing.RatecardUsageDiscountReason),
		}); err != nil {
			return out, err
		}
	default:
		return api.BillingDiscountReason{}, fmt.Errorf("unknown discount reason type: %s", reason.Type())
	}

	return out, nil
}

func decimalPtrToStringPtr(d *alpacadecimal.Decimal) *string {
	if d == nil {
		return nil
	}

	return lo.ToPtr(d.String())
}

func decimalPtrToStringPtrIgnoringZeroValue(d *alpacadecimal.Decimal) *string {
	if d == nil {
		return nil
	}

	if d.IsZero() {
		return nil
	}

	return lo.ToPtr(d.String())
}

// decimalPtrToStringPtrIfNotEqual returns a pointer to the string representation of the decimal if it is not equal to the other decimal.
func decimalPtrToStringPtrIfNotEqual(value *alpacadecimal.Decimal, other *alpacadecimal.Decimal) *string {
	if value == nil {
		return nil
	}

	if other == nil {
		return lo.ToPtr(value.String())
	}

	if value.Equal(lo.FromPtr(other)) {
		return nil
	}

	return lo.ToPtr(value.String())
}

func mapSimulationLineToEntity(line api.InvoiceSimulationLine) (*billing.Line, error) {
	lineType, err := line.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("failed to get type discriminator: %w", err)
	}

	switch lineType {
	case string(api.InvoiceFlatFeeLineTypeFlatFee):
		flatFee, err := line.AsInvoiceSimulationFlatFeeLine()
		if err != nil {
			return nil, fmt.Errorf("failed to map flat fee line: %w", err)
		}

		return mapSimulationFlatFeeLineToEntity(flatFee)

	case string(api.InvoiceUsageBasedLineTypeUsageBased):
		usageBased, err := line.AsInvoiceSimulationUsageBasedLine()
		if err != nil {
			return nil, fmt.Errorf("failed to map usage based line: %w", err)
		}

		return mapUsageBasedSimulationLineToEntity(usageBased)
	default:
		return nil, fmt.Errorf("unsupported type: %s", lineType)
	}
}

func mapSimulationFlatFeeLineToEntity(line api.InvoiceSimulationFlatFeeLine) (*billing.Line, error) {
	rateCardParsed, err := mapAndValidateFlatFeeRateCardDeprecatedFields(flatFeeRateCardItems{
		RateCard:      line.RateCard,
		PerUnitAmount: line.PerUnitAmount,
		PaymentTerm:   line.PaymentTerm,
		Quantity:      line.Quantity,
		TaxConfig:     line.TaxConfig,
	})
	if err != nil {
		return nil, err
	}

	return &billing.Line{
		LineBase: billing.LineBase{
			ID:          lo.FromPtr(line.Id),
			Metadata:    lo.FromPtrOr(line.Metadata, map[string]string{}),
			Name:        line.Name,
			Type:        billing.InvoiceLineTypeFee,
			Description: line.Description,
			ManagedBy:   billing.ManuallyManagedLine,

			Status: billing.InvoiceLineStatusValid,
			Period: billing.Period{
				Start: line.Period.From,
				End:   line.Period.To,
			},

			InvoiceAt:         line.InvoiceAt,
			TaxConfig:         rateCardParsed.TaxConfig,
			RateCardDiscounts: rateCardParsed.Discounts,
		},
		FlatFee: &billing.FlatFeeLine{
			PerUnitAmount: rateCardParsed.PerUnitAmount,
			PaymentTerm:   rateCardParsed.PaymentTerm,
			Quantity:      rateCardParsed.Quantity,
			Category:      lo.FromPtrOr((*billing.FlatFeeCategory)(line.Category), billing.FlatFeeCategoryRegular),
		},
	}, nil
}

func mapUsageBasedSimulationLineToEntity(line api.InvoiceSimulationUsageBasedLine) (*billing.Line, error) {
	rateCardParsed, err := mapAndValidateUsageBasedRateCardDeprecatedFields(usageBasedRateCardItems{
		RateCard:   line.RateCard,
		Price:      line.Price,
		TaxConfig:  line.TaxConfig,
		FeatureKey: line.FeatureKey,
	})
	if err != nil {
		return nil, err
	}

	if rateCardParsed.Price == nil {
		return nil, billing.ValidationError{
			Err: fmt.Errorf("price is required for usage based lines"),
		}
	}

	var qty, prePeriodQty alpacadecimal.Decimal
	if rateCardParsed.Price.Type() == productcatalog.FlatPriceType {
		qty = alpacadecimal.NewFromInt(1)
		prePeriodQty = alpacadecimal.Zero
	} else {
		var err error

		qty, err = alpacadecimal.NewFromString(line.Quantity)
		if err != nil {
			return nil, billing.ValidationError{Err: fmt.Errorf("failed to parse quantity: %w", err)}
		}

		prePeriodQty, err = alpacadecimal.NewFromString(lo.FromPtrOr(line.PreLinePeriodQuantity, "0"))
		if err != nil {
			return nil, billing.ValidationError{Err: fmt.Errorf("failed to parse preLinePeriodQuantity: %w", err)}
		}
	}

	return &billing.Line{
		LineBase: billing.LineBase{
			ID:          lo.FromPtr(line.Id),
			Metadata:    lo.FromPtrOr(line.Metadata, map[string]string{}),
			Name:        line.Name,
			Type:        billing.InvoiceLineTypeUsageBased,
			Description: line.Description,
			ManagedBy:   billing.ManuallyManagedLine,

			Status: billing.InvoiceLineStatusValid,
			Period: billing.Period{
				Start: line.Period.From,
				End:   line.Period.To,
			},

			InvoiceAt:         line.InvoiceAt,
			TaxConfig:         rateCardParsed.TaxConfig,
			RateCardDiscounts: rateCardParsed.Discounts,
		},
		UsageBased: &billing.UsageBasedLine{
			Price:                        rateCardParsed.Price,
			FeatureKey:                   rateCardParsed.FeatureKey,
			Quantity:                     &qty,
			MeteredQuantity:              &qty,
			PreLinePeriodQuantity:        &prePeriodQty,
			MeteredPreLinePeriodQuantity: &prePeriodQty,
		},
	}, nil
}

func getIDFromLineReplace(line api.InvoiceLineReplaceUpdate) (string, error) {
	value, err := line.ValueByDiscriminator()
	if err != nil {
		return "", err
	}

	switch v := value.(type) {
	case api.InvoiceFlatFeeLineReplaceUpdate:
		return lo.FromPtr(v.Id), nil
	case api.InvoiceUsageBasedLineReplaceUpdate:
		return lo.FromPtr(v.Id), nil
	default:
		return "", fmt.Errorf("unknown line type: %T", value)
	}
}

func lineFromInvoiceLineReplaceUpdate(line api.InvoiceLineReplaceUpdate, invoice *billing.Invoice) (*billing.Line, error) {
	value, err := line.ValueByDiscriminator()
	if err != nil {
		return nil, err
	}

	switch v := value.(type) {
	case api.InvoiceFlatFeeLineReplaceUpdate:
		rateCardParsed, err := mapAndValidateFlatFeeRateCardDeprecatedFields(flatFeeRateCardItems{
			RateCard:      v.RateCard,
			PerUnitAmount: v.PerUnitAmount,
			PaymentTerm:   v.PaymentTerm,
			Quantity:      v.Quantity,
			TaxConfig:     v.TaxConfig,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to map flat fee line: %w", err)
		}

		return &billing.Line{
			LineBase: billing.LineBase{
				Namespace: invoice.Namespace,

				Metadata:    lo.FromPtrOr(v.Metadata, map[string]string{}),
				Name:        v.Name,
				Description: v.Description,
				ManagedBy:   billing.ManuallyManagedLine,
				Status:      billing.InvoiceLineStatusValid,

				Type: billing.InvoiceLineTypeFee,

				InvoiceID: invoice.ID,
				Currency:  invoice.Currency,

				Period: billing.Period{
					Start: v.Period.From,
					End:   v.Period.To,
				},
				InvoiceAt: v.InvoiceAt,

				TaxConfig:         rateCardParsed.TaxConfig,
				RateCardDiscounts: rateCardParsed.Discounts,
			},
			FlatFee: &billing.FlatFeeLine{
				PerUnitAmount: rateCardParsed.PerUnitAmount,
				Quantity:      rateCardParsed.Quantity,

				PaymentTerm: rateCardParsed.PaymentTerm,
				Category:    lo.FromPtrOr((*billing.FlatFeeCategory)(v.Category), billing.FlatFeeCategoryRegular),
			},
		}, nil
	case api.InvoiceUsageBasedLineReplaceUpdate:
		rateCardParsed, err := mapAndValidateUsageBasedRateCardDeprecatedFields(usageBasedRateCardItems{
			RateCard:   v.RateCard,
			Price:      v.Price,
			TaxConfig:  v.TaxConfig,
			FeatureKey: v.FeatureKey,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to map usage based line: %w", err)
		}

		return &billing.Line{
			LineBase: billing.LineBase{
				Namespace: invoice.Namespace,

				Metadata:    lo.FromPtrOr(v.Metadata, map[string]string{}),
				Name:        v.Name,
				Description: v.Description,
				ManagedBy:   billing.ManuallyManagedLine,
				Status:      billing.InvoiceLineStatusValid,

				Type: billing.InvoiceLineTypeUsageBased,

				InvoiceID: invoice.ID,
				Currency:  invoice.Currency,

				Period: billing.Period{
					Start: v.Period.From,
					End:   v.Period.To,
				},
				InvoiceAt: v.InvoiceAt,

				TaxConfig:         rateCardParsed.TaxConfig,
				RateCardDiscounts: rateCardParsed.Discounts,
			},
			UsageBased: &billing.UsageBasedLine{
				Price:      rateCardParsed.Price,
				FeatureKey: rateCardParsed.FeatureKey,

				// TODO: snapshotting
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown line type: %T", value)
	}
}

func mergeLineFromInvoiceLineReplaceUpdate(existing *billing.Line, line api.InvoiceLineReplaceUpdate) (*billing.Line, bool, error) {
	value, err := line.ValueByDiscriminator()
	if err != nil {
		return nil, false, err
	}

	switch v := value.(type) {
	case api.InvoiceFlatFeeLineReplaceUpdate:
		if existing.Type != billing.InvoiceLineTypeFee {
			return nil, false, billing.ValidationError{
				Err: fmt.Errorf("line type change is not supported for line %s", existing.ID),
			}
		}

		oldBase := existing.LineBase.Clone()
		oldFee := existing.FlatFee.Clone()

		rateCardParsed, err := mapAndValidateFlatFeeRateCardDeprecatedFields(flatFeeRateCardItems{
			RateCard:      v.RateCard,
			PerUnitAmount: v.PerUnitAmount,
			PaymentTerm:   v.PaymentTerm,
			Quantity:      v.Quantity,
			TaxConfig:     v.TaxConfig,
		})
		if err != nil {
			return nil, false, fmt.Errorf("failed to map flat fee line: %w", err)
		}

		existing.LineBase.Metadata = lo.FromPtrOr(v.Metadata, existing.Metadata)
		existing.LineBase.Name = v.Name
		existing.LineBase.Description = v.Description

		existing.Period.Start = v.Period.From
		existing.Period.End = v.Period.To
		existing.InvoiceAt = v.InvoiceAt

		existing.TaxConfig = rateCardParsed.TaxConfig
		existing.RateCardDiscounts = rateCardParsed.Discounts

		existing.FlatFee.PerUnitAmount = rateCardParsed.PerUnitAmount
		existing.FlatFee.Quantity = rateCardParsed.Quantity
		existing.FlatFee.PaymentTerm = rateCardParsed.PaymentTerm
		existing.FlatFee.Category = lo.FromPtrOr((*billing.FlatFeeCategory)(v.Category), existing.FlatFee.Category)

		wasChange := !oldBase.Equal(existing.LineBase) || !oldFee.Equal(existing.FlatFee)
		if wasChange {
			existing.ManagedBy = billing.ManuallyManagedLine
		}

		return existing, wasChange, nil
	case api.InvoiceUsageBasedLineReplaceUpdate:
		if existing.Type != billing.InvoiceLineTypeUsageBased {
			return nil, false, billing.ValidationError{
				Err: fmt.Errorf("line type change is not supported for line %s", existing.ID),
			}
		}

		oldBase := existing.LineBase.Clone()
		oldUBP := existing.UsageBased.Clone()

		rateCardParsed, err := mapAndValidateUsageBasedRateCardDeprecatedFields(usageBasedRateCardItems{
			RateCard:   v.RateCard,
			Price:      v.Price,
			TaxConfig:  v.TaxConfig,
			FeatureKey: v.FeatureKey,
		})
		if err != nil {
			return nil, false, billing.ValidationError{
				Err: fmt.Errorf("failed to map usage based line: %w", err),
			}
		}

		existing.LineBase.Metadata = lo.FromPtrOr(v.Metadata, existing.Metadata)
		existing.LineBase.Name = v.Name
		existing.LineBase.Description = v.Description

		existing.Period.Start = v.Period.From
		existing.Period.End = v.Period.To
		existing.InvoiceAt = v.InvoiceAt

		existing.TaxConfig = rateCardParsed.TaxConfig
		existing.UsageBased.Price = rateCardParsed.Price
		existing.UsageBased.FeatureKey = rateCardParsed.FeatureKey

		// Rate card discounts are not allowed to be updated on a progressively billed line (e.g. if there is
		// already a partial invoice created), as we might go short on the discount quantity.
		//
		// If this is ever requested:
		// - we should introduce the concept of a "discount pool" that is shared across invoices and
		// - editing the discount edits the pool
		// - editing requires that the discount pool's quantity cannot be less than the already used
		//   quantity.

		if existing.SplitLineGroupID != nil && rateCardParsed.Discounts.Usage != nil && existing.RateCardDiscounts.Usage != nil {
			if !equal.PtrEqual(rateCardParsed.Discounts.Usage, existing.RateCardDiscounts.Usage) {
				return nil, false, billing.ValidationError{
					Err: fmt.Errorf("line[%s]: %w", existing.ID, billing.ErrInvoiceLineProgressiveBillingUsageDiscountUpdateForbidden),
				}
			}
		}

		existing.RateCardDiscounts = rateCardParsed.Discounts

		wasChange := !oldBase.Equal(existing.LineBase) || !oldUBP.Equal(existing.UsageBased)
		if wasChange {
			existing.ManagedBy = billing.ManuallyManagedLine
		}

		// We are not allowing period change for split lines (or their children), as that would mess up the
		// calculation logic and/or we would need to update multiple invoices to correct all the references.
		//
		// Deletion is allowed.
		if oldBase.SplitLineGroupID != nil && !oldBase.Period.Equal(existing.Period) {
			return nil, false, billing.ValidationError{
				Err: fmt.Errorf("line[%s]: %w", existing.ID, billing.ErrInvoiceLineNoPeriodChangeForSplitLine),
			}
		}

		return existing, wasChange, nil
	}

	return nil, false, fmt.Errorf("unknown line type: %T", value)
}

func (h *handler) mergeInvoiceLinesFromAPI(ctx context.Context, invoice *billing.Invoice, updatedLines []api.InvoiceLineReplaceUpdate) (billing.LineChildren, error) {
	linesByID, _ := slicesx.UniqueGroupBy(invoice.Lines.OrEmpty(), func(line *billing.Line) string {
		return line.ID
	})

	foundLines := set.New[string]()

	out := make([]*billing.Line, 0, len(updatedLines))

	for _, line := range updatedLines {
		id, err := getIDFromLineReplace(line)
		if err != nil {
			return billing.LineChildren{}, fmt.Errorf("failed to get line ID: %w", err)
		}

		existingLine, existingLineFound := linesByID[id]

		if id == "" || !existingLineFound {
			// We allow injecting fake IDs for new lines, so that discounts can reference those,
			// but we are not persisting them to the database
			newLine, err := lineFromInvoiceLineReplaceUpdate(line, invoice)
			if err != nil {
				return billing.LineChildren{}, fmt.Errorf("failed to create new line: %w", err)
			}

			if h.featureSwitches.UseUsageBasedFlatFeeLines && newLine.Type == billing.InvoiceLineTypeFee {
				return billing.LineChildren{}, billing.ValidationError{
					Err: fmt.Errorf("creating flat fee lines is not supported, please use usage based lines instead"),
				}
			}

			if invoice.Status != billing.InvoiceStatusGathering {
				newLine, err = h.service.SnapshotLineQuantity(ctx, billing.SnapshotLineQuantityInput{
					Invoice: invoice,
					Line:    newLine,
				})
				if err != nil {
					return billing.LineChildren{}, fmt.Errorf("failed to snapshot quantity: %w", err)
				}
			}

			out = append(out, newLine)
			continue
		}

		foundLines.Add(id)
		mergedLine, changed, err := mergeLineFromInvoiceLineReplaceUpdate(existingLine, line)
		if err != nil {
			return billing.LineChildren{}, fmt.Errorf("failed to merge line: %w", err)
		}

		if h.featureSwitches.UseUsageBasedFlatFeeLines && changed && mergedLine.Type == billing.InvoiceLineTypeFee {
			return billing.LineChildren{}, billing.ValidationError{
				Err: fmt.Errorf("updating flat fee lines is not supported, please use usage based lines instead"),
			}
		}

		if changed && invoice.Status != billing.InvoiceStatusGathering {
			mergedLine, err = h.service.SnapshotLineQuantity(ctx, billing.SnapshotLineQuantityInput{
				Invoice: invoice,
				Line:    mergedLine,
			})
			if err != nil {
				return billing.LineChildren{}, fmt.Errorf("failed to snapshot quantity: %w", err)
			}
		}

		out = append(out, mergedLine)
	}

	lineIDs := set.New(lo.Keys(linesByID)...)

	deletedLines := set.Subtract(lineIDs, foundLines).AsSlice()
	for _, id := range deletedLines {
		existingLine := linesByID[id]
		existingLine.DeletedAt = lo.ToPtr(clock.Now())
		out = append(out, existingLine)
	}

	return billing.NewLineChildren(out), nil
}
