package httpdriver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/equal"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/set"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
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

			lineEntities, err := slicesx.MapWithErr(req.Lines, func(line api.InvoicePendingLineCreate) (billing.GatheringLine, error) {
				return mapCreateGatheringLineToEntity(line, ns)
			})
			if err != nil {
				return CreatePendingLineRequest{}, billing.ValidationError{
					Err: fmt.Errorf("failed to map lines: %w", err),
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

			if res == nil {
				return CreatePendingLineResponse{}, fmt.Errorf("create pending invoice lines result is nil")
			}

			out := CreatePendingLineResponse{
				IsInvoiceNew: res.IsInvoiceNew,
			}

			// TODO: For the V3 api let's not return the invoice
			mergedProfile, err := h.service.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
				Customer: request.Customer,
				Expand: billing.CustomerOverrideExpand{
					Customer: true,
					Apps:     true,
				},
			})
			if err != nil {
				return CreatePendingLineResponse{}, fmt.Errorf("failed to get customer override: %w", err)
			}

			out.Invoice, err = MapGatheringInvoiceToAPI(res.Invoice, mergedProfile.Customer, mergedProfile.MergedProfile)
			if err != nil {
				return CreatePendingLineResponse{}, fmt.Errorf("failed to map invoice: %w", err)
			}

			out.Lines, err = slicesx.MapWithErr(res.Lines, mapGatheringInvoiceLineToAPI)
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

func mapCreateLineToEntity(line api.InvoicePendingLineCreate, ns string) (*billing.StandardLine, error) {
	rateCardParsed, err := mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{
		RateCard:   line.RateCard,
		Price:      line.Price,
		TaxConfig:  line.TaxConfig,
		FeatureKey: line.FeatureKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to map usage based line: %w", err)
	}

	return &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace:   ns,
				Name:        line.Name,
				Description: line.Description,
			}),

			Metadata:  lo.FromPtrOr(line.Metadata, map[string]string{}),
			ManagedBy: billing.ManuallyManagedLine,

			Period: billing.Period{
				Start: line.Period.From,
				End:   line.Period.To,
			},

			InvoiceAt:         line.InvoiceAt,
			TaxConfig:         rateCardParsed.TaxConfig,
			RateCardDiscounts: rateCardParsed.Discounts,
			Price:             lo.FromPtr(rateCardParsed.Price),
			FeatureKey:        rateCardParsed.FeatureKey,
		},
	}, nil
}

func mapCreateGatheringLineToEntity(line api.InvoicePendingLineCreate, ns string) (billing.GatheringLine, error) {
	rateCardParsed, err := mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{
		RateCard:   line.RateCard,
		Price:      line.Price,
		TaxConfig:  line.TaxConfig,
		FeatureKey: line.FeatureKey,
	})
	if err != nil {
		return billing.GatheringLine{}, fmt.Errorf("failed to map usage based line: %w", err)
	}

	if rateCardParsed.Price == nil {
		return billing.GatheringLine{}, fmt.Errorf("price is nil [line=%s]", line.Name)
	}

	return billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace:   ns,
				Name:        line.Name,
				Description: line.Description,
			}),

			Metadata:  lo.FromPtrOr(line.Metadata, map[string]string{}),
			ManagedBy: billing.ManuallyManagedLine,

			ServicePeriod: timeutil.ClosedPeriod{
				From: line.Period.From,
				To:   line.Period.To,
			},

			InvoiceAt:         line.InvoiceAt,
			TaxConfig:         rateCardParsed.TaxConfig,
			RateCardDiscounts: rateCardParsed.Discounts,
			Price:             lo.FromPtr(rateCardParsed.Price),
			FeatureKey:        rateCardParsed.FeatureKey,
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

func mapDetailedLinesToAPI(lines billing.DetailedLines, invoiceAt time.Time) (*[]api.InvoiceDetailedLine, error) {
	mappedLines, err := slicesx.MapWithErr(lines, func(line billing.DetailedLine) (api.InvoiceDetailedLine, error) {
		return mapDetailedLineToAPI(line, invoiceAt)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to map detailed lines: %w", err)
	}

	return lo.ToPtr(mappedLines), nil
}

func mapDetailedLineToAPI(line billing.DetailedLine, invoiceAt time.Time) (api.InvoiceDetailedLine, error) {
	amountDiscountsAPI, err := mapInvoiceLineAmountDiscountsToAPI(line.AmountDiscounts)
	if err != nil {
		return api.InvoiceDetailedLine{}, fmt.Errorf("failed to map amount discounts: %w", err)
	}

	var discountsAPI *api.InvoiceLineDiscounts
	if amountDiscountsAPI != nil {
		discountsAPI = &api.InvoiceLineDiscounts{
			Amount: amountDiscountsAPI,
		}
	}

	return api.InvoiceDetailedLine{
		Type: api.InvoiceDetailedLineTypeFlatFee,
		Id:   line.ID,

		CreatedAt: line.CreatedAt,
		DeletedAt: line.DeletedAt,
		UpdatedAt: line.UpdatedAt,
		InvoiceAt: invoiceAt,

		Currency: string(line.Currency),
		Status:   api.InvoiceLineStatusDetailed,

		Description: line.Description,
		Name:        line.Name,
		ManagedBy:   api.InvoiceLineManagedBySystem,

		Invoice: &api.InvoiceReference{
			Id: line.InvoiceID,
		},

		Period: api.Period{
			From: line.ServicePeriod.Start,
			To:   line.ServicePeriod.End,
		},

		PerUnitAmount: lo.ToPtr(line.PerUnitAmount.String()),
		Quantity:      lo.ToPtr(line.Quantity.String()),
		Category:      lo.ToPtr(api.InvoiceDetailedLineCostCategory(line.Category)),
		TaxConfig:     mapTaxConfigToAPI(line.TaxConfig),
		PaymentTerm:   lo.ToPtr(api.PricePaymentTerm(line.PaymentTerm)),

		RateCard: &api.InvoiceDetailedLineRateCard{
			TaxConfig: mapTaxConfigToAPI(line.TaxConfig),
			Price: &api.FlatPriceWithPaymentTerm{
				Type:        api.FlatPriceWithPaymentTermTypeFlat,
				PaymentTerm: lo.ToPtr(api.PricePaymentTerm(line.PaymentTerm)),
				Amount:      line.PerUnitAmount.String(),
			},
			Quantity: lo.ToPtr(line.Quantity.String()),
		},

		Discounts: discountsAPI,
		Totals:    mapTotalsToAPI(line.Totals),

		ExternalIds: mapLineAppExternalIdsToAPI(line.ExternalIDs),
	}, nil
}

func mapLineAppExternalIdsToAPI(externalIds billing.LineExternalIDs) *api.InvoiceLineAppExternalIds {
	if lo.IsEmpty(externalIds) {
		return nil
	}

	return &api.InvoiceLineAppExternalIds{
		Invoicing: lo.EmptyableToPtr(externalIds.Invoicing),
	}
}

func mapInvoiceLineToAPI(line *billing.StandardLine) (api.InvoiceLine, error) {
	price, err := productcataloghttp.FromRateCardUsageBasedPrice(line.Price)
	if err != nil {
		return api.InvoiceLine{}, fmt.Errorf("failed to map price: %w", err)
	}

	children, err := mapDetailedLinesToAPI(line.DetailedLines, line.InvoiceAt)
	if err != nil {
		return api.InvoiceLine{}, fmt.Errorf("failed to map children: %w", err)
	}

	discountsAPI, err := mapDiscountsToAPI(line.Discounts)
	if err != nil {
		return api.InvoiceLine{}, fmt.Errorf("failed to map discounts: %w", err)
	}

	invoiceLine := api.InvoiceLine{
		Type: api.InvoiceLineTypeUsageBased,
		Id:   line.ID,

		CreatedAt: line.CreatedAt,
		DeletedAt: line.DeletedAt,
		UpdatedAt: line.UpdatedAt,
		InvoiceAt: line.InvoiceAt,

		// TODO: deprecation
		Currency: string(line.Currency),
		Status:   api.InvoiceLineStatusValid,

		Description: line.Description,
		Name:        line.Name,
		ManagedBy:   api.InvoiceLineManagedBy(line.ManagedBy),

		// TODO: deprecation
		Invoice: &api.InvoiceReference{
			Id: line.InvoiceID,
		},

		Metadata: convert.MapToPointer(line.Metadata),
		Period: api.Period{
			From: line.Period.Start,
			To:   line.Period.End,
		},

		TaxConfig: mapTaxConfigToAPI(line.TaxConfig),

		FeatureKey:                   lo.EmptyableToPtr(line.FeatureKey),
		MeteredQuantity:              decimalPtrToStringPtrIfNotEqual(line.MeteredQuantity, line.Quantity),
		Quantity:                     decimalPtrToStringPtr(line.Quantity),
		PreLinePeriodQuantity:        decimalPtrToStringPtrIgnoringZeroValue(line.PreLinePeriodQuantity),
		MeteredPreLinePeriodQuantity: decimalPtrToStringPtrIgnoringZeroValue(line.MeteredPreLinePeriodQuantity),

		Price: lo.ToPtr(price),

		RateCard: &api.InvoiceUsageBasedRateCard{
			TaxConfig:  mapTaxConfigToAPI(line.TaxConfig),
			Price:      lo.ToPtr(price),
			FeatureKey: lo.EmptyableToPtr(line.FeatureKey),
		},

		Discounts: discountsAPI,
		Children:  children,
		Totals:    mapTotalsToAPI(line.Totals),

		ExternalIds:  mapLineAppExternalIdsToAPI(line.ExternalIDs),
		Subscription: mapSubscriptionReferencesToAPI(line.Subscription),
	}

	return invoiceLine, nil
}

func mapSubscriptionReferencesToAPI(optSub *billing.SubscriptionReference) *api.InvoiceLineSubscriptionReference {
	if optSub == nil {
		return nil
	}

	out := &api.InvoiceLineSubscriptionReference{
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

	out.BillingPeriod = api.Period{
		From: optSub.BillingPeriod.From,
		To:   optSub.BillingPeriod.To,
	}

	return out
}

func mapDiscountsToAPI(discounts billing.LineDiscounts) (*api.InvoiceLineDiscounts, error) {
	if discounts.IsEmpty() {
		return nil, nil
	}

	mappedAmountDiscounts, err := mapInvoiceLineAmountDiscountsToAPI(discounts.Amount)
	if err != nil {
		return nil, fmt.Errorf("failed to map amount discounts: %w", err)
	}

	mappedUsageDiscounts, err := mapInvoiceLineUsageDiscountsToAPI(discounts.Usage)
	if err != nil {
		return nil, fmt.Errorf("failed to map usage discounts: %w", err)
	}

	return &api.InvoiceLineDiscounts{
		Amount: mappedAmountDiscounts,
		Usage:  mappedUsageDiscounts,
	}, nil
}

func mapInvoiceLineAmountDiscountsToAPI(amountDiscounts billing.AmountLineDiscountsManaged) (*[]api.InvoiceLineAmountDiscount, error) {
	if len(amountDiscounts) == 0 {
		return nil, nil
	}

	mapped, err := slicesx.MapWithErr(amountDiscounts, func(discount billing.AmountLineDiscountManaged) (api.InvoiceLineAmountDiscount, error) {
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

	return lo.ToPtr(mapped), nil
}

func mapInvoiceLineUsageDiscountsToAPI(usageDiscounts billing.UsageLineDiscountsManaged) (*[]api.InvoiceLineUsageDiscount, error) {
	if len(usageDiscounts) == 0 {
		return nil, nil
	}

	mapped, err := slicesx.MapWithErr(usageDiscounts, func(discount billing.UsageLineDiscountManaged) (api.InvoiceLineUsageDiscount, error) {
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
		return nil, fmt.Errorf("failed to map usage discounts: %w", err)
	}

	return lo.ToPtr(mapped), nil
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

func mapSimulationLineToEntity(line api.InvoiceSimulationLine) (*billing.StandardLine, error) {
	rateCardParsed, err := mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{
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

	return &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				ID:          lo.FromPtr(line.Id),
				Name:        line.Name,
				Description: line.Description,
			}),
			Metadata:  lo.FromPtrOr(line.Metadata, map[string]string{}),
			ManagedBy: billing.ManuallyManagedLine,

			Period: billing.Period{
				Start: line.Period.From.Truncate(streaming.MinimumWindowSizeDuration),
				End:   line.Period.To.Truncate(streaming.MinimumWindowSizeDuration),
			},

			InvoiceAt:                    line.InvoiceAt.Truncate(streaming.MinimumWindowSizeDuration),
			TaxConfig:                    rateCardParsed.TaxConfig,
			RateCardDiscounts:            rateCardParsed.Discounts,
			Price:                        lo.FromPtr(rateCardParsed.Price),
			FeatureKey:                   rateCardParsed.FeatureKey,
			Quantity:                     &qty,
			MeteredQuantity:              &qty,
			PreLinePeriodQuantity:        &prePeriodQty,
			MeteredPreLinePeriodQuantity: &prePeriodQty,
		},
	}, nil
}

func standardLineFromInvoiceLineReplaceUpdate(line api.InvoiceLineReplaceUpdate, invoice *billing.StandardInvoice) (*billing.StandardLine, error) {
	rateCardParsed, err := mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{
		RateCard:   line.RateCard,
		Price:      line.Price,
		TaxConfig:  line.TaxConfig,
		FeatureKey: line.FeatureKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to map usage based line: %w", err)
	}

	return &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace:   invoice.Namespace,
				Name:        line.Name,
				Description: line.Description,
			}),

			Metadata:  lo.FromPtrOr(line.Metadata, map[string]string{}),
			ManagedBy: billing.ManuallyManagedLine,

			InvoiceID: invoice.ID,
			Currency:  invoice.Currency,

			Period: billing.Period{
				Start: line.Period.From.Truncate(streaming.MinimumWindowSizeDuration),
				End:   line.Period.To.Truncate(streaming.MinimumWindowSizeDuration),
			},
			InvoiceAt: line.InvoiceAt.Truncate(streaming.MinimumWindowSizeDuration),

			TaxConfig:         rateCardParsed.TaxConfig,
			RateCardDiscounts: rateCardParsed.Discounts,
			Price:             lo.FromPtr(rateCardParsed.Price),
			FeatureKey:        rateCardParsed.FeatureKey,
		},
	}, nil
}

func gatheringLineFromInvoiceLineReplaceUpdate(line api.InvoiceLineReplaceUpdate, invoice *billing.GatheringInvoice) (billing.GatheringLine, error) {
	rateCardParsed, err := mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{
		RateCard:   line.RateCard,
		Price:      line.Price,
		TaxConfig:  line.TaxConfig,
		FeatureKey: line.FeatureKey,
	})
	if err != nil {
		return billing.GatheringLine{}, fmt.Errorf("failed to map usage based line: %w", err)
	}

	if rateCardParsed.Price == nil {
		return billing.GatheringLine{}, billing.ValidationError{
			Err: fmt.Errorf("price is required for usage based lines"),
		}
	}

	return billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace:   invoice.Namespace,
				Name:        line.Name,
				Description: line.Description,
			}),

			Metadata:  lo.FromPtrOr(line.Metadata, map[string]string{}),
			ManagedBy: billing.ManuallyManagedLine,

			InvoiceID: invoice.ID,
			Currency:  invoice.Currency,

			ServicePeriod: timeutil.ClosedPeriod{
				From: line.Period.From.Truncate(streaming.MinimumWindowSizeDuration),
				To:   line.Period.To.Truncate(streaming.MinimumWindowSizeDuration),
			},
			InvoiceAt: line.InvoiceAt.Truncate(streaming.MinimumWindowSizeDuration),

			TaxConfig:         rateCardParsed.TaxConfig,
			RateCardDiscounts: rateCardParsed.Discounts,
			Price:             lo.FromPtr(rateCardParsed.Price),
			FeatureKey:        rateCardParsed.FeatureKey,
		},
	}, nil
}

func mergeStandardLineFromInvoiceLineReplaceUpdate(existing *billing.StandardLine, line api.InvoiceLineReplaceUpdate) (*billing.StandardLine, bool, error) {
	oldBase := existing.StandardLineBase.Clone()

	rateCardParsed, err := mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{
		RateCard:   line.RateCard,
		Price:      line.Price,
		TaxConfig:  line.TaxConfig,
		FeatureKey: line.FeatureKey,
	})
	if err != nil {
		return nil, false, billing.ValidationError{
			Err: fmt.Errorf("failed to map usage based line: %w", err),
		}
	}

	if rateCardParsed.Price == nil {
		return nil, false, billing.ValidationError{
			Err: fmt.Errorf("price is required for usage based lines"),
		}
	}

	existing.Metadata = lo.FromPtrOr(line.Metadata, api.Metadata(existing.Metadata))
	existing.Name = line.Name
	existing.Description = line.Description

	existing.Period.Start = line.Period.From.Truncate(streaming.MinimumWindowSizeDuration)
	existing.Period.End = line.Period.To.Truncate(streaming.MinimumWindowSizeDuration)
	existing.InvoiceAt = line.InvoiceAt.Truncate(streaming.MinimumWindowSizeDuration)

	existing.TaxConfig = rateCardParsed.TaxConfig
	existing.Price = lo.FromPtr(rateCardParsed.Price)
	existing.FeatureKey = rateCardParsed.FeatureKey

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

	wasChange := !oldBase.Equal(existing.StandardLineBase)
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

func mergeGatheringLineFromInvoiceLineReplaceUpdate(existing billing.GatheringLine, line api.InvoiceLineReplaceUpdate) (billing.GatheringLine, error) {
	old, err := existing.Clone()
	if err != nil {
		return billing.GatheringLine{}, fmt.Errorf("cloning existing line: %w", err)
	}

	rateCardParsed, err := mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{
		RateCard:   line.RateCard,
		Price:      line.Price,
		TaxConfig:  line.TaxConfig,
		FeatureKey: line.FeatureKey,
	})
	if err != nil {
		return billing.GatheringLine{}, billing.ValidationError{
			Err: fmt.Errorf("failed to map usage based line: %w", err),
		}
	}

	if line.Price == nil {
		return billing.GatheringLine{}, billing.ValidationError{
			Err: fmt.Errorf("price is required for usage based lines"),
		}
	}

	existing.Metadata = lo.FromPtrOr(line.Metadata, api.Metadata(existing.Metadata))
	existing.Name = line.Name
	existing.Description = line.Description

	existing.ServicePeriod.From = line.Period.From.Truncate(streaming.MinimumWindowSizeDuration)
	existing.ServicePeriod.To = line.Period.To.Truncate(streaming.MinimumWindowSizeDuration)
	existing.InvoiceAt = line.InvoiceAt.Truncate(streaming.MinimumWindowSizeDuration)

	existing.TaxConfig = rateCardParsed.TaxConfig
	existing.Price = lo.FromPtr(rateCardParsed.Price)
	existing.FeatureKey = rateCardParsed.FeatureKey

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
			return billing.GatheringLine{}, billing.ValidationError{
				Err: fmt.Errorf("line[%s]: %w", existing.ID, billing.ErrInvoiceLineProgressiveBillingUsageDiscountUpdateForbidden),
			}
		}
	}

	existing.RateCardDiscounts = rateCardParsed.Discounts

	if !old.Equal(existing) {
		existing.ManagedBy = billing.ManuallyManagedLine
	}

	// We are not allowing period change for split lines (or their children), as that would mess up the
	// calculation logic and/or we would need to update multiple invoices to correct all the references.
	//
	// Deletion is allowed.
	if old.SplitLineGroupID != nil && !old.ServicePeriod.Equal(existing.ServicePeriod) {
		return billing.GatheringLine{}, billing.ValidationError{
			Err: fmt.Errorf("line[%s]: %w", existing.ID, billing.ErrInvoiceLineNoPeriodChangeForSplitLine),
		}
	}

	return existing, nil
}

func (h *handler) mergeStandardInvoiceLinesFromAPI(ctx context.Context, invoice *billing.StandardInvoice, updatedLines []api.InvoiceLineReplaceUpdate) (billing.StandardInvoiceLines, error) {
	linesByID, _ := slicesx.UniqueGroupBy(invoice.Lines.OrEmpty(), func(line *billing.StandardLine) string {
		return line.ID
	})

	foundLines := set.New[string]()

	out := make([]*billing.StandardLine, 0, len(updatedLines))

	for _, line := range updatedLines {
		id := lo.FromPtr(line.Id)

		existingLine, existingLineFound := linesByID[id]

		if id == "" || !existingLineFound {
			// We allow injecting fake IDs for new lines, so that discounts can reference those,
			// but we are not persisting them to the database
			newLine, err := standardLineFromInvoiceLineReplaceUpdate(line, invoice)
			if err != nil {
				return billing.StandardInvoiceLines{}, fmt.Errorf("failed to create new line: %w", err)
			}

			if invoice.Status != billing.StandardInvoiceStatusGathering {
				newLine, err = h.service.SnapshotLineQuantity(ctx, billing.SnapshotLineQuantityInput{
					Invoice: invoice,
					Line:    newLine,
				})
				if err != nil {
					return billing.StandardInvoiceLines{}, fmt.Errorf("failed to snapshot quantity: %w", err)
				}
			}

			out = append(out, newLine)
			continue
		}

		foundLines.Add(id)
		mergedLine, changed, err := mergeStandardLineFromInvoiceLineReplaceUpdate(existingLine, line)
		if err != nil {
			return billing.StandardInvoiceLines{}, fmt.Errorf("failed to merge line: %w", err)
		}

		if changed && invoice.Status != billing.StandardInvoiceStatusGathering {
			mergedLine, err = h.service.SnapshotLineQuantity(ctx, billing.SnapshotLineQuantityInput{
				Invoice: invoice,
				Line:    mergedLine,
			})
			if err != nil {
				return billing.StandardInvoiceLines{}, fmt.Errorf("failed to snapshot quantity: %w", err)
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

	return billing.NewStandardInvoiceLines(out), nil
}

func (h *handler) mergeGatheringInvoiceLinesFromAPI(ctx context.Context, invoice *billing.GatheringInvoice, updatedLines []api.InvoiceLineReplaceUpdate) (billing.GatheringInvoiceLines, error) {
	linesByID, _ := slicesx.UniqueGroupBy(invoice.Lines.OrEmpty(), func(line billing.GatheringLine) string {
		return line.ID
	})

	foundLines := set.New[string]()

	out := make([]billing.GatheringLine, 0, len(updatedLines))

	for _, line := range updatedLines {
		id := lo.FromPtr(line.Id)

		existingLine, existingLineFound := linesByID[id]

		if id == "" || !existingLineFound {
			// We allow injecting fake IDs for new lines, so that discounts can reference those,
			// but we are not persisting them to the database
			newLine, err := gatheringLineFromInvoiceLineReplaceUpdate(line, invoice)
			if err != nil {
				return billing.GatheringInvoiceLines{}, fmt.Errorf("failed to create new line: %w", err)
			}

			out = append(out, newLine)
			continue
		}

		foundLines.Add(id)
		mergedLine, err := mergeGatheringLineFromInvoiceLineReplaceUpdate(existingLine, line)
		if err != nil {
			return billing.GatheringInvoiceLines{}, fmt.Errorf("failed to merge line: %w", err)
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

	return billing.NewGatheringInvoiceLines(out), nil
}
