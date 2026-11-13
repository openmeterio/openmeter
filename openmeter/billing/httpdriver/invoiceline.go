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
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
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
				Currency: currencyx.FiatCode(req.Currency),
				Lines:    billing.NewCreatePendingInvoiceLines(lineEntities),
			}, nil
		},
		func(ctx context.Context, request CreatePendingLineRequest) (CreatePendingLineResponse, error) {
			res, err := h.createPendingInvoiceLines(ctx, request)
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
			mergedProfile, err := h.service.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{
				Customer: request.Customer,
				Expand: billing.CustomerBillingProfileExpand{
					Customer: true,
					Apps:     true,
				},
			})
			if err != nil {
				return CreatePendingLineResponse{}, fmt.Errorf("failed to get customer override: %w", err)
			}

			out.Invoice, err = MapGatheringInvoiceToAPI(res.Invoice, mergedProfile.Customer, mergedProfile.EffectiveProfile)
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

func (h *handler) createPendingInvoiceLines(ctx context.Context, request CreatePendingLineRequest) (*billing.CreatePendingInvoiceLinesResult, error) {
	useCharges, err := h.shouldCreatePendingLinesWithCharges(request.Customer.Namespace)
	if err != nil {
		return nil, err
	}

	if useCharges {
		return h.chargeService.CreatePendingInvoiceLines(ctx, request)
	}

	return h.service.CreatePendingInvoiceLines(ctx, request)
}

func (h *handler) shouldCreatePendingLinesWithCharges(namespace string) (bool, error) {
	if h.chargeService == nil {
		return false, nil
	}

	if !h.credits.Enabled {
		return false, nil
	}

	if !h.credits.EnableCreditThenInvoice {
		return false, nil
	}

	if h.featureGate == nil {
		return true, nil
	}

	return h.featureGate.Enabled(namespace, h.featureGate.Flags.Credits())
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
			Engine:    billing.LineEngineTypeInvoice,

			Period: timeutil.ClosedPeriod{
				From: line.Period.From,
				To:   line.Period.To,
			},

			InvoiceAt:         line.InvoiceAt,
			TaxConfig:         billing.FromProductCatalog(rateCardParsed.TaxConfig),
			RateCardDiscounts: billing.DiscountsFromProductCatalog(rateCardParsed.Discounts),
		},
		UsageBased: &billing.UsageBasedLine{
			Price:      rateCardParsed.Price,
			FeatureKey: rateCardParsed.FeatureKey,
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
			Engine:    billing.LineEngineTypeInvoice,

			ServicePeriod: timeutil.ClosedPeriod{
				From: line.Period.From,
				To:   line.Period.To,
			},

			InvoiceAt:         line.InvoiceAt,
			TaxConfig:         rateCardParsed.TaxConfig,
			RateCardDiscounts: billing.DiscountsFromProductCatalog(rateCardParsed.Discounts),
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
			Engine:    billing.LineEngineTypeInvoice,

			Period: timeutil.ClosedPeriod{
				From: line.Period.From.Truncate(streaming.MinimumWindowSizeDuration),
				To:   line.Period.To.Truncate(streaming.MinimumWindowSizeDuration),
			},

			InvoiceAt:         line.InvoiceAt.Truncate(streaming.MinimumWindowSizeDuration),
			TaxConfig:         billing.FromProductCatalog(rateCardParsed.TaxConfig),
			RateCardDiscounts: billing.DiscountsFromProductCatalog(rateCardParsed.Discounts),
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

			Metadata: lo.FromPtrOr(line.Metadata, map[string]string{}),

			InvoiceID: invoice.ID,
			Currency:  invoice.Currency,

			Period: timeutil.ClosedPeriod{
				From: line.Period.From.Truncate(streaming.MinimumWindowSizeDuration),
				To:   line.Period.To.Truncate(streaming.MinimumWindowSizeDuration),
			},
			InvoiceAt: line.InvoiceAt.Truncate(streaming.MinimumWindowSizeDuration),

			TaxConfig:         billing.FromProductCatalog(rateCardParsed.TaxConfig),
			RateCardDiscounts: billing.DiscountsFromProductCatalog(rateCardParsed.Discounts),
		},
		UsageBased: &billing.UsageBasedLine{
			Price:      rateCardParsed.Price,
			FeatureKey: rateCardParsed.FeatureKey,

			// TODO: snapshotting
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

			Metadata: lo.FromPtrOr(line.Metadata, map[string]string{}),

			InvoiceID: invoice.ID,
			Currency:  invoice.Currency,

			ServicePeriod: timeutil.ClosedPeriod{
				From: line.Period.From.Truncate(streaming.MinimumWindowSizeDuration),
				To:   line.Period.To.Truncate(streaming.MinimumWindowSizeDuration),
			},
			InvoiceAt: line.InvoiceAt.Truncate(streaming.MinimumWindowSizeDuration),

			TaxConfig:         rateCardParsed.TaxConfig,
			RateCardDiscounts: billing.DiscountsFromProductCatalog(rateCardParsed.Discounts),
			Price:             lo.FromPtr(rateCardParsed.Price),
			FeatureKey:        rateCardParsed.FeatureKey,
		},
	}, nil
}

func mergeStandardLineFromInvoiceLineReplaceUpdate(existing *billing.StandardLine, line api.InvoiceLineReplaceUpdate) (*billing.StandardLine, error) {
	rateCardParsed, err := mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{
		RateCard:   line.RateCard,
		Price:      line.Price,
		TaxConfig:  line.TaxConfig,
		FeatureKey: line.FeatureKey,
	})
	if err != nil {
		return nil, billing.ValidationError{
			Err: fmt.Errorf("failed to map usage based line: %w", err),
		}
	}

	existing.Metadata = lo.FromPtrOr(line.Metadata, api.Metadata(existing.Metadata))
	existing.Name = line.Name
	existing.Description = line.Description

	existing.Period.From = line.Period.From.Truncate(streaming.MinimumWindowSizeDuration)
	existing.Period.To = line.Period.To.Truncate(streaming.MinimumWindowSizeDuration)
	existing.InvoiceAt = line.InvoiceAt.Truncate(streaming.MinimumWindowSizeDuration)

	existing.TaxConfig = billing.FromProductCatalog(rateCardParsed.TaxConfig)
	existing.UsageBased.Price = rateCardParsed.Price
	existing.UsageBased.FeatureKey = rateCardParsed.FeatureKey
	existing.RateCardDiscounts = existing.RateCardDiscounts.ReplaceFromProductCatalog(rateCardParsed.Discounts)

	return existing, nil
}

func mergeGatheringLineFromInvoiceLineReplaceUpdate(existing billing.GatheringLine, line api.InvoiceLineReplaceUpdate) (billing.GatheringLine, error) {
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

	if rateCardParsed.Price == nil {
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
	existing.RateCardDiscounts = existing.RateCardDiscounts.ReplaceFromProductCatalog(rateCardParsed.Discounts)

	return existing, nil
}

func (h *handler) mergeStandardInvoiceLinesFromAPI(_ context.Context, invoice *billing.StandardInvoice, updatedLines []api.InvoiceLineReplaceUpdate) (billing.StandardInvoiceLines, error) {
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

			out = append(out, newLine)

			continue
		}

		foundLines.Add(id)
		mergedLine, err := mergeStandardLineFromInvoiceLineReplaceUpdate(existingLine, line)
		if err != nil {
			return billing.StandardInvoiceLines{}, fmt.Errorf("failed to merge line: %w", err)
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
