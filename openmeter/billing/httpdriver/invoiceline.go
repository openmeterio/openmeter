package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	planhttpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/httpdriver"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/set"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var _ InvoiceLineHandler = (*handler)(nil)

type (
	CreatePendingLineRequest  = billing.CreateInvoiceLinesInput
	CreatePendingLineResponse = []api.InvoiceLine
	CreatePendingLineHandler  = httptransport.Handler[CreatePendingLineRequest, CreatePendingLineResponse]
)

func (h *handler) CreatePendingLine() CreatePendingLineHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreatePendingLineRequest, error) {
			lines := []api.InvoicePendingLineCreate{}

			if err := commonhttp.JSONRequestBodyDecoder(r, &lines); err != nil {
				return CreatePendingLineRequest{}, fmt.Errorf("failed to decode request body: %w", err)
			}

			// TODO[OM-982]: limit to single depth, valid line creation

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreatePendingLineRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			if len(lines) == 0 {
				return CreatePendingLineRequest{}, billing.ValidationError{
					Err: fmt.Errorf("no lines provided"),
				}
			}

			lineEntities := make([]billing.LineWithCustomer, 0, len(lines))
			for _, line := range lines {
				lineEntity, err := mapCreateLineToEntity(line, ns)
				if err != nil {
					return CreatePendingLineRequest{}, fmt.Errorf("failed to map line: %w", err)
				}

				lineEntities = append(lineEntities, lineEntity)
			}

			return CreatePendingLineRequest{
				Namespace: ns,
				Lines:     lineEntities,
			}, nil
		},
		func(ctx context.Context, request CreatePendingLineRequest) (CreatePendingLineResponse, error) {
			lines, err := h.service.CreatePendingInvoiceLines(ctx, request)
			if err != nil {
				return CreatePendingLineResponse{}, fmt.Errorf("failed to create invoice lines: %w", err)
			}

			res := make(CreatePendingLineResponse, 0, len(lines))

			for _, line := range lines {
				line, err := mapBillingLineToAPI(line)
				if err != nil {
					return CreatePendingLineResponse{}, fmt.Errorf("failed to map line: %w", err)
				}
				res = append(res, line)
			}

			return res, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreatePendingLineResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("CreateInvoiceLineByCustomer"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

func mapCreateLineToEntity(line api.InvoicePendingLineCreate, ns string) (billing.LineWithCustomer, error) {
	// This should not fail, and we would have at least the discriminator unmarshaled
	discriminator, err := line.Discriminator()
	if err != nil {
		return billing.LineWithCustomer{}, fmt.Errorf("failed to get type discriminator: %w", err)
	}

	switch discriminator {
	case string(api.InvoiceFlatFeeLineTypeFlatFee):
		fee, err := line.AsInvoiceFlatFeePendingLineCreate()
		if err != nil {
			return billing.LineWithCustomer{}, fmt.Errorf("failed to map fee line: %w", err)
		}
		return mapCreatePendingFlatFeeLineToEntity(fee, ns)
	case string(api.InvoiceUsageBasedLineTypeUsageBased):
		usageBased, err := line.AsInvoiceUsageBasedPendingLineCreate()
		if err != nil {
			return billing.LineWithCustomer{}, fmt.Errorf("failed to map usage based line: %w", err)
		}
		return mapCreatePendingUsageBasedLineToEntity(usageBased, ns)
	default:
		return billing.LineWithCustomer{}, fmt.Errorf("unsupported type: %s", discriminator)
	}
}

func mapCreatePendingFlatFeeLineToEntity(line api.InvoiceFlatFeePendingLineCreate, ns string) (billing.LineWithCustomer, error) {
	qty, err := alpacadecimal.NewFromString(line.Quantity)
	if err != nil {
		return billing.LineWithCustomer{}, fmt.Errorf("failed to map quantity: %w", err)
	}

	perUnitAmount, err := alpacadecimal.NewFromString(line.PerUnitAmount)
	if err != nil {
		return billing.LineWithCustomer{}, fmt.Errorf("failed to parse price: %w", err)
	}

	return billing.LineWithCustomer{
		Line: billing.Line{
			LineBase: billing.LineBase{
				Namespace: ns,

				Metadata:    lo.FromPtrOr(line.Metadata, map[string]string{}),
				Name:        line.Name,
				Type:        billing.InvoiceLineTypeFee,
				Description: line.Description,
				ManagedBy:   billing.ManuallyManagedLine,

				Status:   billing.InvoiceLineStatusValid, // This is not settable from outside
				Currency: currencyx.Code(line.Currency),
				Period: billing.Period{
					Start: line.Period.From,
					End:   line.Period.To,
				},

				InvoiceAt: line.InvoiceAt,
				TaxConfig: mapTaxConfigToEntity(line.TaxConfig),
			},
			FlatFee: &billing.FlatFeeLine{
				PerUnitAmount: perUnitAmount,
				PaymentTerm:   lo.FromPtrOr((*productcatalog.PaymentTermType)(line.PaymentTerm), productcatalog.InAdvancePaymentTerm),
				Quantity:      qty,
				Category:      lo.FromPtrOr((*billing.FlatFeeCategory)(line.Category), billing.FlatFeeCategoryRegular),
			},
		},
		CustomerID: line.CustomerId,
	}, nil
}

func mapCreatePendingUsageBasedLineToEntity(line api.InvoiceUsageBasedPendingLineCreate, ns string) (billing.LineWithCustomer, error) {
	price, err := planhttpdriver.AsPrice(line.Price)
	if err != nil {
		return billing.LineWithCustomer{}, fmt.Errorf("failed to map price: %w", err)
	}

	return billing.LineWithCustomer{
		Line: billing.Line{
			LineBase: billing.LineBase{
				Namespace: ns,

				Metadata:    lo.FromPtrOr(line.Metadata, map[string]string{}),
				Name:        line.Name,
				Type:        billing.InvoiceLineTypeUsageBased,
				Description: line.Description,
				ManagedBy:   billing.ManuallyManagedLine,

				Status:   billing.InvoiceLineStatusValid, // This is not settable from outside
				Currency: currencyx.Code(line.Currency),
				Period: billing.Period{
					Start: line.Period.From,
					End:   line.Period.To,
				},

				InvoiceAt: line.InvoiceAt,
				TaxConfig: mapTaxConfigToEntity(line.TaxConfig),
			},
			UsageBased: &billing.UsageBasedLine{
				Price:      price,
				FeatureKey: line.FeatureKey,
			},
		},
		CustomerID: line.CustomerId,
	}, nil
}

func mapTaxConfigToEntity(tc *api.TaxConfig) *billing.TaxConfig {
	if tc == nil {
		return nil
	}

	return lo.ToPtr(planhttpdriver.AsTaxConfig(*tc))
}

func mapTaxConfigToAPI(to *billing.TaxConfig) *api.TaxConfig {
	if to == nil {
		return nil
	}

	return lo.ToPtr(planhttpdriver.FromTaxConfig(*to))
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

		Metadata: lo.EmptyableToPtr(line.Metadata),
		Period: api.Period{
			From: line.Period.Start,
			To:   line.Period.End,
		},

		PerUnitAmount: line.FlatFee.PerUnitAmount.String(),
		Quantity:      line.FlatFee.Quantity.String(),
		Category:      lo.ToPtr(api.InvoiceFlatFeeCategory(line.FlatFee.Category)),
		TaxConfig:     mapTaxConfigToAPI(line.TaxConfig),
		PaymentTerm:   lo.ToPtr(api.PricePaymentTerm(line.FlatFee.PaymentTerm)),

		Discounts: mapDiscountsToAPI(line.Discounts),
		Totals:    mapTotalsToAPI(line.Totals),
		Children:  children,

		ExternalIds: &api.InvoiceLineAppExternalIds{
			Invoicing: lo.EmptyableToPtr(line.ExternalIDs.Invoicing),
		},
		Subscriptions: mapSubscriptionReferencesToAPI(line.Subscription),
	}

	out := api.InvoiceLine{}
	err = out.FromInvoiceFlatFeeLine(feeLine)
	if err != nil {
		return api.InvoiceLine{}, fmt.Errorf("failed to map fee line: %w", err)
	}

	return out, nil
}

func mapUsageBasedLineToAPI(line *billing.Line) (api.InvoiceLine, error) {
	if line.UsageBased.Price == nil {
		return api.InvoiceLine{}, fmt.Errorf("price is nil [line=%s]", line.ID)
	}

	price, err := planhttpdriver.FromRateCardUsageBasedPrice(*line.UsageBased.Price)
	if err != nil {
		return api.InvoiceLine{}, fmt.Errorf("failed to map price: %w", err)
	}

	children, err := mapChildLinesToAPI(line.Children)
	if err != nil {
		return api.InvoiceLine{}, fmt.Errorf("failed to map children: %w", err)
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

		Metadata: lo.EmptyableToPtr(line.Metadata),
		Period: api.Period{
			From: line.Period.Start,
			To:   line.Period.End,
		},

		TaxConfig: mapTaxConfigToAPI(line.TaxConfig),

		FeatureKey:            line.UsageBased.FeatureKey,
		Quantity:              decimalPtrToStringPtr(line.UsageBased.Quantity),
		PreLinePeriodQuantity: decimalPtrToStringPtr(line.UsageBased.PreLinePeriodQuantity),
		Price:                 price,

		Discounts: mapDiscountsToAPI(line.Discounts),
		Children:  children,
		Totals:    mapTotalsToAPI(line.Totals),

		ExternalIds: lo.EmptyableToPtr(api.InvoiceLineAppExternalIds{
			Invoicing: lo.EmptyableToPtr(line.ExternalIDs.Invoicing),
		}),
		Subscriptions: mapSubscriptionReferencesToAPI(line.Subscription),
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

func mapDiscountsToAPI(optDiscounts billing.LineDiscounts) *[]api.InvoiceLineDiscount {
	if optDiscounts.IsAbsent() {
		return nil
	}

	discounts := optDiscounts.OrEmpty()

	out := make([]api.InvoiceLineDiscount, 0, len(discounts))

	for _, discount := range discounts {
		out = append(out, mapDiscountToAPI(discount))
	}

	return &out
}

func mapDiscountToAPI(discount billing.LineDiscount) api.InvoiceLineDiscount {
	return api.InvoiceLineDiscount{
		Id: discount.ID,

		CreatedAt: discount.CreatedAt,
		DeletedAt: discount.DeletedAt,
		UpdatedAt: discount.UpdatedAt,

		Description: discount.Description,
		Amount:      discount.Amount.String(),
		Code:        discount.ChildUniqueReferenceID,
		ExternalIds: &api.InvoiceLineAppExternalIds{
			Invoicing: lo.EmptyableToPtr(discount.ExternalIDs.Invoicing),
		},
	}
}

func decimalPtrToStringPtr(d *alpacadecimal.Decimal) *string {
	if d == nil {
		return nil
	}

	return lo.ToPtr(d.String())
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
	perUnitAmount, err := alpacadecimal.NewFromString(line.PerUnitAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to parse price: %w", err)
	}

	qty, err := alpacadecimal.NewFromString(line.Quantity)
	if err != nil {
		return nil, fmt.Errorf("failed to parse quantity: %w", err)
	}

	return &billing.Line{
		LineBase: billing.LineBase{
			ID:          lo.FromPtrOr(line.Id, ""),
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

			InvoiceAt: line.InvoiceAt,
			TaxConfig: mapTaxConfigToEntity(line.TaxConfig),
		},
		FlatFee: &billing.FlatFeeLine{
			PerUnitAmount: perUnitAmount,
			PaymentTerm:   lo.FromPtrOr((*productcatalog.PaymentTermType)(line.PaymentTerm), productcatalog.InAdvancePaymentTerm),
			Quantity:      qty,
			Category:      lo.FromPtrOr((*billing.FlatFeeCategory)(line.Category), billing.FlatFeeCategoryRegular),
		},
	}, nil
}

func mapUsageBasedSimulationLineToEntity(line api.InvoiceSimulationUsageBasedLine) (*billing.Line, error) {
	qty, err := alpacadecimal.NewFromString(line.Quantity)
	if err != nil {
		return nil, fmt.Errorf("failed to parse quantity: %w", err)
	}

	prePeriodQty := alpacadecimal.Zero
	if line.PreLinePeriodQuantity != nil {
		prePeriodQty, err = alpacadecimal.NewFromString(*line.PreLinePeriodQuantity)
		if err != nil {
			return nil, fmt.Errorf("failed to parse pre period quantity: %w", err)
		}
	}

	price, err := planhttpdriver.AsPrice(line.Price)
	if err != nil {
		return nil, fmt.Errorf("failed to map price: %w", err)
	}

	return &billing.Line{
		LineBase: billing.LineBase{
			ID:          lo.FromPtrOr(line.Id, ""),
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

			InvoiceAt: line.InvoiceAt,
			TaxConfig: mapTaxConfigToEntity(line.TaxConfig),
		},
		UsageBased: &billing.UsageBasedLine{
			Price:                 price,
			FeatureKey:            line.FeatureKey,
			Quantity:              &qty,
			PreLinePeriodQuantity: &prePeriodQty,
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
		return lo.FromPtrOr(v.Id, ""), nil
	case api.InvoiceUsageBasedLineReplaceUpdate:
		return lo.FromPtrOr(v.Id, ""), nil
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
		perUnitAmount, err := alpacadecimal.NewFromString(v.PerUnitAmount)
		if err != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("failed to parse per unit amount: %w", err),
			}
		}

		quantity, err := alpacadecimal.NewFromString(v.Quantity)
		if err != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("failed to parse quantity: %w", err),
			}
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

				TaxConfig: mapTaxConfigToEntity(v.TaxConfig),
			},
			FlatFee: &billing.FlatFeeLine{
				PerUnitAmount: perUnitAmount,
				Quantity:      quantity,

				PaymentTerm: lo.FromPtrOr((*productcatalog.PaymentTermType)(v.PaymentTerm), productcatalog.InAdvancePaymentTerm),
				Category:    lo.FromPtrOr((*billing.FlatFeeCategory)(v.Category), billing.FlatFeeCategoryRegular),
			},
		}, nil
	case api.InvoiceUsageBasedLineReplaceUpdate:
		price, err := planhttpdriver.AsPrice(v.Price)
		if err != nil {
			return nil, fmt.Errorf("failed to map price: %w", err)
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

				TaxConfig: mapTaxConfigToEntity(v.TaxConfig),
			},
			UsageBased: &billing.UsageBasedLine{
				Price:      price,
				FeatureKey: v.FeatureKey,

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

		perUnitAmount, err := alpacadecimal.NewFromString(v.PerUnitAmount)
		if err != nil {
			return nil, false, billing.ValidationError{
				Err: fmt.Errorf("failed to parse per unit amount: %w", err),
			}
		}

		quantity, err := alpacadecimal.NewFromString(v.Quantity)
		if err != nil {
			return nil, false, billing.ValidationError{
				Err: fmt.Errorf("failed to parse quantity: %w", err),
			}
		}

		existing.LineBase.Metadata = lo.FromPtrOr(v.Metadata, existing.Metadata)
		existing.LineBase.Name = v.Name
		existing.LineBase.Description = v.Description

		existing.Period.Start = v.Period.From
		existing.Period.End = v.Period.To
		existing.InvoiceAt = v.InvoiceAt

		existing.TaxConfig = mapTaxConfigToEntity(v.TaxConfig)

		existing.FlatFee.PerUnitAmount = perUnitAmount
		existing.FlatFee.Quantity = quantity
		existing.FlatFee.PaymentTerm = lo.FromPtrOr((*productcatalog.PaymentTermType)(v.PaymentTerm), existing.FlatFee.PaymentTerm)
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

		price, err := planhttpdriver.AsPrice(v.Price)
		if err != nil {
			return nil, false, billing.ValidationError{
				Err: fmt.Errorf("failed to map price: %w", err),
			}
		}

		existing.LineBase.Metadata = lo.FromPtrOr(v.Metadata, existing.Metadata)
		existing.LineBase.Name = v.Name
		existing.LineBase.Description = v.Description

		existing.Period.Start = v.Period.From
		existing.Period.End = v.Period.To
		existing.InvoiceAt = v.InvoiceAt

		existing.TaxConfig = mapTaxConfigToEntity(v.TaxConfig)

		existing.UsageBased.Price = price
		existing.UsageBased.FeatureKey = v.FeatureKey

		wasChange := !oldBase.Equal(existing.LineBase) || !oldUBP.Equal(existing.UsageBased)
		if wasChange {
			existing.ManagedBy = billing.ManuallyManagedLine
		}

		// We are not allowing period change for split lines (or their children), as that would mess up the
		// calculation logic and/or we would need to update multiple invoices to correct all the references.
		//
		// Deletion is allowed.
		if (oldBase.Status == billing.InvoiceLineStatusSplit || oldBase.ParentLineID != nil) && !oldBase.Period.Equal(existing.Period) {
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
