package rating

import (
	"context"
	"encoding/json"

	"github.com/alpacahq/alpacadecimal"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// TotalsReuse retains successful pure rating results. Usage and charge state must
// still be read on every calculation; this is not a cache of current balances.
type TotalsReuse struct {
	entries *lru.Cache[totalsReuseChargeKey, totalsReuseEntry]
}

type totalsReuseChargeKey struct {
	service *service
	charge  meta.ChargeID
}

type totalsReuseEntry struct {
	inputs string
	result GetTotalsForUsageResult
}

func NewTotalsReuse(capacity int) (*TotalsReuse, error) {
	entries, err := lru.New[totalsReuseChargeKey, totalsReuseEntry](capacity)
	if err != nil {
		return nil, err
	}

	return &TotalsReuse{entries: entries}, nil
}

type totalsReuseContextKey struct{}

type totalsReuseContext struct {
	reuse    *TotalsReuse
	onLookup func(context.Context, bool)
}

// WithTotalsReuse opts a calculation into bounded rating reuse. onLookup reports
// whether a fresh quantity and identical rating inputs reused a previous result.
func WithTotalsReuse(ctx context.Context, reuse *TotalsReuse, onLookup func(context.Context, bool)) context.Context {
	return context.WithValue(ctx, totalsReuseContextKey{}, totalsReuseContext{reuse: reuse, onLookup: onLookup})
}

// Keep both entry count and retained input size bounded. Unsupported inputs simply
// use normal rating; a memoization failure must never fail a calculation.
const maxTotalsReuseInputBytes = 64 * 1024

func totalsReuseInputs(intent usagebased.RateableIntent, ignoreMinimumCommitment bool) (string, bool) {
	currency := intent.Intent.Intent.Currency.Currency
	// These implementations derive their arithmetic from Details(). Unknown
	// implementations may have additional private inputs that we cannot compare.
	switch currency.(type) {
	case *currencyx.FiatCurrency, *currencyx.CustomCurrency:
	default:
		return "", false
	}

	inputs, err := json.Marshal(struct {
		Price                   *productcatalog.Price
		ServicePeriod           timeutil.ClosedPeriod
		Quantity                alpacadecimal.Decimal
		Name                    string
		Discounts               billing.Discounts
		StandardLineDiscounts   billing.StandardLineDiscounts
		UnitConfig              *productcatalog.UnitConfig
		CurrencyType            currencyx.CurrencyType
		Currency                currencyx.CurrencyDetails
		IgnoreMinimumCommitment bool
	}{
		Price:                   intent.GetPrice(),
		ServicePeriod:           intent.GetServicePeriod(),
		Quantity:                intent.MeterValue,
		Name:                    intent.GetName(),
		Discounts:               intent.GetRateCardDiscounts(),
		StandardLineDiscounts:   intent.GetStandardLineDiscounts(),
		UnitConfig:              intent.GetUnitConfig(),
		CurrencyType:            currency.Type(),
		Currency:                currency.Details(),
		IgnoreMinimumCommitment: ignoreMinimumCommitment,
	})
	if err != nil || len(inputs) > maxTotalsReuseInputBytes {
		return "", false
	}

	// Decimal JSON uses its exact decimal string; comparing these bytes never
	// collapses quantities or prices through float64 or a rounded value.
	return string(inputs), true
}
