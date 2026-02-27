package sync

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/llmcost"
)

const (
	// DefaultMinSourceAgreement is the default minimum number of sources that must agree on a price.
	DefaultMinSourceAgreement = 2

	// DefaultPriceTolerance is the default maximum allowed percentage difference between source prices.
	// 1% tolerance to account for rounding differences across sources.
	DefaultPriceTolerance = 0.01
)

// Reconciler groups source prices by (provider, model_id) and creates
// canonical global prices when multiple sources agree.
type Reconciler struct {
	repo           llmcost.Adapter
	logger         *slog.Logger
	minAgreement   int
	priceTolerance float64
}

func NewReconciler(repo llmcost.Adapter, logger *slog.Logger, minAgreement int, priceTolerance float64) *Reconciler {
	if minAgreement <= 0 {
		minAgreement = DefaultMinSourceAgreement
	}

	if priceTolerance < 0 {
		priceTolerance = DefaultPriceTolerance
	}

	return &Reconciler{
		repo:           repo,
		logger:         logger,
		minAgreement:   minAgreement,
		priceTolerance: priceTolerance,
	}
}

// providerModelKey is a grouping key for in-memory reconciliation.
type providerModelKey struct {
	Provider string
	ModelID  string
}

// Reconcile groups prices by (provider, model_id), checks for multi-source agreement,
// and upserts reconciled global prices.
func (r *Reconciler) Reconcile(ctx context.Context, prices []llmcost.SourcePrice) error {
	// Group prices by (provider, model_id)
	groups := make(map[providerModelKey][]llmcost.SourcePrice)
	for _, p := range prices {
		key := providerModelKey{Provider: string(p.Provider), ModelID: p.ModelID}
		groups[key] = append(groups[key], p)
	}

	now := time.Now().UTC()
	reconciled := 0
	skipped := 0

	for key, sourcePrices := range groups {
		if len(sourcePrices) < r.minAgreement {
			skipped++
			continue
		}

		// Find agreeing prices
		agreeing := r.findAgreement(sourcePrices)
		if agreeing == nil {
			r.logger.Debug("no price agreement",
				"provider", key.Provider,
				"model_id", key.ModelID,
				"sources", len(sourcePrices))

			skipped++

			continue
		}

		// Build the source prices map from agreeing sources
		sourcePricesMap := make(llmcost.SourcePricesMap, len(agreeing))
		for _, sp := range agreeing {
			sourcePricesMap[sp.Source] = llmcost.SourcePriceData{
				Pricing:   sp.Pricing,
				FetchedAt: sp.FetchedAt,
			}
		}

		// Average the agreeing prices for the canonical price
		avg := averagePrices(agreeing)

		price := llmcost.Price{
			Provider:      llmcost.Provider(key.Provider),
			ModelID:       key.ModelID,
			ModelName:     avg.ModelName,
			Pricing:       avg.Pricing,
			Currency:      "USD",
			Source:        llmcost.PriceSourceSystem,
			SourcePrices:  sourcePricesMap,
			EffectiveFrom: now,
		}

		if err := r.repo.UpsertGlobalPrice(ctx, price); err != nil {
			r.logger.Error("failed to upsert global price",
				"provider", key.Provider,
				"model_id", key.ModelID,
				"error", err)

			continue
		}

		reconciled++
	}

	r.logger.Info("reconciliation complete",
		"total_models", len(groups),
		"reconciled", reconciled,
		"skipped", skipped)

	return nil
}

// findAgreement checks if at least minAgreement sources agree on prices
// within priceTolerance. Returns the agreeing prices if found.
func (r *Reconciler) findAgreement(prices []llmcost.SourcePrice) []llmcost.SourcePrice {
	if len(prices) < r.minAgreement {
		return nil
	}

	tolerance := alpacadecimal.NewFromFloat(r.priceTolerance)

	// Compare each pair of source prices
	for i := 0; i < len(prices); i++ {
		agreeing := []llmcost.SourcePrice{prices[i]}

		for j := i + 1; j < len(prices); j++ {
			if pricesAgree(prices[i].Pricing, prices[j].Pricing, tolerance) {
				agreeing = append(agreeing, prices[j])
			}
		}

		if len(agreeing) >= r.minAgreement {
			return agreeing
		}
	}

	return nil
}

// pricesAgree checks if two ModelPricing values agree within tolerance.
func pricesAgree(a, b llmcost.ModelPricing, tolerance alpacadecimal.Decimal) bool {
	return decimalsAgree(a.InputPerToken, b.InputPerToken, tolerance) &&
		decimalsAgree(a.OutputPerToken, b.OutputPerToken, tolerance)
}

// decimalsAgree checks if two decimals are within tolerance of each other.
func decimalsAgree(a, b, tolerance alpacadecimal.Decimal) bool {
	if a.IsZero() && b.IsZero() {
		return true
	}

	if a.IsZero() || b.IsZero() {
		return false
	}

	diff := a.Sub(b).Abs()
	maxVal := a

	if b.GreaterThan(a) {
		maxVal = b
	}

	ratio := diff.Div(maxVal)

	return ratio.LessThanOrEqual(tolerance)
}

// averagePrices computes the average of agreeing source prices.
func averagePrices(prices []llmcost.SourcePrice) llmcost.SourcePrice {
	if len(prices) == 0 {
		panic(fmt.Sprintf("averagePrices called with empty slice"))
	}

	count := alpacadecimal.NewFromInt(int64(len(prices)))
	sumInput := alpacadecimal.NewFromInt(0)
	sumOutput := alpacadecimal.NewFromInt(0)

	for _, p := range prices {
		sumInput = sumInput.Add(p.Pricing.InputPerToken)
		sumOutput = sumOutput.Add(p.Pricing.OutputPerToken)
	}

	result := prices[0] // Use first price as base for model name etc.
	result.Pricing.InputPerToken = sumInput.Div(count)
	result.Pricing.OutputPerToken = sumOutput.Div(count)

	return result
}
