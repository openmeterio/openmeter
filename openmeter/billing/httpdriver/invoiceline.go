package httpdriver

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingconvert "github.com/openmeterio/openmeter/openmeter/billing/apiconvert"
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
	CreatePendingLineResponse = apiv1models.InvoicePendingLineCreateResponse
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
			req := apiv1models.InvoicePendingLineCreateInput{}

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

			lineEntities, err := slicesx.MapWithErr(req.Lines, func(line apiv1models.InvoicePendingLineCreate) (billing.GatheringLine, error) {
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

				// API callers author the tax code references they send, so a deleted
				// Tax Code ID is rejected.
				ChangeSource: billing.ChangeSourceAPIRequest,
			}, nil
		},
		func(ctx context.Context, request CreatePendingLineRequest) (CreatePendingLineResponse, error) {
			var (
				resp       CreatePendingLineResponse
				res        *billing.CreatePendingInvoiceLinesResult
				customerID = request.Customer
			)

			// The service call and the response mapping share one transaction: the stripe.code
			// each line sends is resolved to a Tax Code (creating missing ones) that commits or
			// rolls back with the lines.
			err := h.taxCodeResolver.WithResolver(ctx, func(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver) error {
				for i := range request.Lines {
					resolved, err := resolveLineTaxConfig(ctx, resolver, request.Customer.Namespace, request.Lines[i].TaxConfig)
					if err != nil {
						return err
					}

					request.Lines[i].TaxConfig = resolved
				}

				var err error
				res, err = h.createPendingInvoiceLines(ctx, request)
				if err != nil {
					return err
				}

				if res == nil {
					return fmt.Errorf("create pending invoice lines result is nil")
				}

				// TODO: For the V3 api let's not return the invoice
				mergedProfile, err := h.service.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{
					Customer: customerID,
					Expand: billing.CustomerBillingProfileExpand{
						Customer: true,
						Apps:     true,
					},
				})
				if err != nil {
					return fmt.Errorf("failed to get customer override: %w", err)
				}

				taxCodes, err := billingconvert.ResolveInvoiceTaxCodes(ctx, h.taxCodeResolver, billingconvert.ResolveInvoiceTaxCodesInput{
					Namespace: request.Customer.Namespace,
					Workflows: []billing.WorkflowConfig{mergedProfile.EffectiveProfile.WorkflowConfig},
					// The response renders the whole gathering invoice, so lines added by earlier
					// requests need their Tax Codes too.
					GatheringLines: append(slices.Clone(res.Invoice.Lines.OrEmpty()), res.Lines...),
				})
				if err != nil {
					return err
				}

				response := CreatePendingLineResponse{
					IsInvoiceNew: res.IsInvoiceNew,
				}

				response.Invoice, err = MapGatheringInvoiceToAPI(res.Invoice, mergedProfile.Customer, mergedProfile.EffectiveProfile, taxCodes)
				if err != nil {
					return fmt.Errorf("failed to map invoice: %w", err)
				}

				response.Lines, err = slicesx.MapWithErr(res.Lines, func(line billing.GatheringLine) (apiv1models.InvoiceLine, error) {
					return mapGatheringInvoiceLineToAPI(line, taxCodes)
				})
				if err != nil {
					return fmt.Errorf("failed to map lines: %w", err)
				}

				resp = response

				return nil
			})
			if err != nil {
				return CreatePendingLineResponse{}, fmt.Errorf("failed to create invoice lines: %w", err)
			}

			return resp, nil
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

func mapCreateGatheringLineToEntity(line apiv1models.InvoicePendingLineCreate, ns string) (billing.GatheringLine, error) {
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

func mapTaxConfigToEntity(tc *apiv1models.TaxConfig) *productcatalog.TaxConfig {
	if tc == nil {
		return nil
	}

	return lo.ToPtr(productcataloghttp.AsTaxConfig(*tc))
}

func mapSimulationLineToEntity(line apiv1models.InvoiceSimulationLine) (*billing.StandardLine, error) {
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

func standardLineFromInvoiceLineReplaceUpdate(line apiv1models.InvoiceLineReplaceUpdate, invoice *billing.StandardInvoice, resolvedTaxConfig *productcatalog.TaxConfig) (*billing.StandardLine, error) {
	rateCardParsed, err := mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{
		RateCard:   line.RateCard,
		Price:      line.Price,
		TaxConfig:  line.TaxConfig,
		FeatureKey: line.FeatureKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to map usage based line: %w", err)
	}

	// The handler resolved the request's tax config before the service call; nil keeps the
	// line without a tax config, which the invoice's default fills in at calculation time.
	taxConfig := resolvedTaxConfig
	if taxConfig == nil {
		taxConfig = rateCardParsed.TaxConfig
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

			TaxConfig:         billing.FromProductCatalog(taxConfig),
			RateCardDiscounts: billing.DiscountsFromProductCatalog(rateCardParsed.Discounts),
		},
		UsageBased: &billing.UsageBasedLine{
			Price:      rateCardParsed.Price,
			FeatureKey: rateCardParsed.FeatureKey,

			// TODO: snapshotting
		},
	}, nil
}

func gatheringLineFromInvoiceLineReplaceUpdate(line apiv1models.InvoiceLineReplaceUpdate, invoice *billing.GatheringInvoice, resolvedTaxConfig *productcatalog.TaxConfig) (billing.GatheringLine, error) {
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

	// The handler resolved the request's tax config before the service call; nil keeps the
	// line without a tax config, which the invoice's default fills in at calculation time.
	taxConfig := resolvedTaxConfig
	if taxConfig == nil {
		taxConfig = rateCardParsed.TaxConfig
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

			TaxConfig:         taxConfig,
			RateCardDiscounts: billing.DiscountsFromProductCatalog(rateCardParsed.Discounts),
			Price:             lo.FromPtr(rateCardParsed.Price),
			FeatureKey:        rateCardParsed.FeatureKey,
		},
	}, nil
}

func mergeStandardLineFromInvoiceLineReplaceUpdate(existing *billing.StandardLine, line apiv1models.InvoiceLineReplaceUpdate, resolvedTaxConfig *productcatalog.TaxConfig) (*billing.StandardLine, error) {
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

	existing.Metadata = lo.FromPtrOr(line.Metadata, apiv1models.Metadata(existing.Metadata))
	existing.Name = line.Name
	existing.Description = line.Description

	existing.Period.From = line.Period.From.Truncate(streaming.MinimumWindowSizeDuration)
	existing.Period.To = line.Period.To.Truncate(streaming.MinimumWindowSizeDuration)
	existing.InvoiceAt = line.InvoiceAt.Truncate(streaming.MinimumWindowSizeDuration)

	// The handler resolved the request's tax config before the service call; nil keeps the
	// line without a tax config, which the invoice's default fills in at calculation time.
	taxConfig := resolvedTaxConfig
	if taxConfig == nil {
		taxConfig = rateCardParsed.TaxConfig
	}

	existing.TaxConfig = billing.FromProductCatalog(taxConfig)
	existing.UsageBased.Price = rateCardParsed.Price
	existing.UsageBased.FeatureKey = rateCardParsed.FeatureKey
	existing.RateCardDiscounts = existing.RateCardDiscounts.ReplaceFromProductCatalog(rateCardParsed.Discounts)

	return existing, nil
}

func mergeGatheringLineFromInvoiceLineReplaceUpdate(existing billing.GatheringLine, line apiv1models.InvoiceLineReplaceUpdate, resolvedTaxConfig *productcatalog.TaxConfig) (billing.GatheringLine, error) {
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

	existing.Metadata = lo.FromPtrOr(line.Metadata, apiv1models.Metadata(existing.Metadata))
	existing.Name = line.Name
	existing.Description = line.Description

	existing.ServicePeriod.From = line.Period.From.Truncate(streaming.MinimumWindowSizeDuration)
	existing.ServicePeriod.To = line.Period.To.Truncate(streaming.MinimumWindowSizeDuration)
	existing.InvoiceAt = line.InvoiceAt.Truncate(streaming.MinimumWindowSizeDuration)

	// The handler resolved the request's tax config before the service call; nil keeps the
	// line without a tax config, which the invoice's default fills in at calculation time.
	taxConfig := resolvedTaxConfig
	if taxConfig == nil {
		taxConfig = rateCardParsed.TaxConfig
	}

	existing.TaxConfig = taxConfig
	existing.Price = lo.FromPtr(rateCardParsed.Price)
	existing.FeatureKey = rateCardParsed.FeatureKey
	existing.RateCardDiscounts = existing.RateCardDiscounts.ReplaceFromProductCatalog(rateCardParsed.Discounts)

	return existing, nil
}

func (h *handler) mergeStandardInvoiceLinesFromAPI(_ context.Context, invoice *billing.StandardInvoice, updatedLines []apiv1models.InvoiceLineReplaceUpdate, resolved []resolvedLineTaxConfig) (billing.StandardInvoiceLines, error) {
	resolvedTaxConfigs, err := taxConfigsForRequestLines(resolved, updatedLines)
	if err != nil {
		return billing.StandardInvoiceLines{}, err
	}

	linesByID, _ := slicesx.UniqueGroupBy(invoice.Lines.OrEmpty(), func(line *billing.StandardLine) string {
		return line.ID
	})

	foundLines := set.New[string]()

	out := make([]*billing.StandardLine, 0, len(updatedLines))

	for i, line := range updatedLines {
		id := lo.FromPtr(line.Id)

		existingLine, existingLineFound := linesByID[id]

		if id == "" || !existingLineFound {
			// We allow injecting fake IDs for new lines, so that discounts can reference those,
			// but we are not persisting them to the database
			newLine, mergeErr := standardLineFromInvoiceLineReplaceUpdate(line, invoice, resolvedTaxConfigs[i])
			if mergeErr != nil {
				return billing.StandardInvoiceLines{}, fmt.Errorf("failed to create new line: %w", mergeErr)
			}

			out = append(out, newLine)

			continue
		}

		foundLines.Add(id)
		mergedLine, mergeErr := mergeStandardLineFromInvoiceLineReplaceUpdate(existingLine, line, resolvedTaxConfigs[i])
		if mergeErr != nil {
			return billing.StandardInvoiceLines{}, fmt.Errorf("failed to merge line: %w", mergeErr)
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

func (h *handler) mergeGatheringInvoiceLinesFromAPI(ctx context.Context, invoice *billing.GatheringInvoice, updatedLines []apiv1models.InvoiceLineReplaceUpdate, resolved []resolvedLineTaxConfig) (billing.GatheringInvoiceLines, error) {
	resolvedTaxConfigs, err := taxConfigsForRequestLines(resolved, updatedLines)
	if err != nil {
		return billing.GatheringInvoiceLines{}, err
	}

	linesByID, _ := slicesx.UniqueGroupBy(invoice.Lines.OrEmpty(), func(line billing.GatheringLine) string {
		return line.ID
	})

	foundLines := set.New[string]()

	out := make([]billing.GatheringLine, 0, len(updatedLines))

	for i, line := range updatedLines {
		id := lo.FromPtr(line.Id)

		existingLine, existingLineFound := linesByID[id]

		if id == "" || !existingLineFound {
			// We allow injecting fake IDs for new lines, so that discounts can reference those,
			// but we are not persisting them to the database
			newLine, mergeErr := gatheringLineFromInvoiceLineReplaceUpdate(line, invoice, resolvedTaxConfigs[i])
			if mergeErr != nil {
				return billing.GatheringInvoiceLines{}, fmt.Errorf("failed to create new line: %w", mergeErr)
			}

			out = append(out, newLine)

			continue
		}

		foundLines.Add(id)
		mergedLine, mergeErr := mergeGatheringLineFromInvoiceLineReplaceUpdate(existingLine, line, resolvedTaxConfigs[i])
		if mergeErr != nil {
			return billing.GatheringInvoiceLines{}, fmt.Errorf("failed to merge line: %w", mergeErr)
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
