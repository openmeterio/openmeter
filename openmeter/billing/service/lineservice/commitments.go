package lineservice

import "context"

type maxPriceCommitmentMiddleware struct {
	PricerMiddlewareBase
}

var _ PricerMiddleware = (*maxPriceCommitmentMiddleware)(nil)

func (m *maxPriceCommitmentMiddleware) AfterCalculate(ctx context.Context, l usageBasedLine, pricerResult newDetailedLinesInput) (newDetailedLinesInput, error) {
	return l, nil
}
