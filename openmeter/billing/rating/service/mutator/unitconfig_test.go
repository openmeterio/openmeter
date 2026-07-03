package mutator

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/rate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

const (
	opMultiply = productcatalog.UnitConfigOperationMultiply
	opDivide   = productcatalog.UnitConfigOperationDivide

	roundCeiling = productcatalog.UnitConfigRoundingModeCeiling
	roundFloor   = productcatalog.UnitConfigRoundingModeFloor
	roundHalfUp  = productcatalog.UnitConfigRoundingModeHalfUp
	roundNone    = productcatalog.UnitConfigRoundingModeNone
)

// unitConfigTestLine is a StandardLineAccessor whose price and unit_config are
// overridable; the embedded StandardLine satisfies the rest of the interface. We
// use it (rather than a real RateableIntent) so a single fixture can exercise the
// Pre-populated cumulative case, which the charges path never produces at rating time.
type unitConfigTestLine struct {
	*billing.StandardLine

	price      *productcatalog.Price
	unitConfig *productcatalog.UnitConfig
}

func (l unitConfigTestLine) GetPrice() *productcatalog.Price           { return l.price }
func (l unitConfigTestLine) GetUnitConfig() *productcatalog.UnitConfig { return l.unitConfig }

func newUnitConfig(op productcatalog.UnitConfigOperation, factor float64, rounding productcatalog.UnitConfigRoundingMode) *productcatalog.UnitConfig {
	return &productcatalog.UnitConfig{
		Operation:        op,
		ConversionFactor: alpacadecimal.NewFromFloat(factor),
		Rounding:         rounding,
	}
}

func mutateUnitConfig(t *testing.T, price *productcatalog.Price, unitConfig *productcatalog.UnitConfig, quantity, preLinePeriod float64) rating.Usage {
	t.Helper()

	input := rate.PricerCalculateInput{
		StandardLineAccessor: unitConfigTestLine{
			StandardLine: &billing.StandardLine{},
			price:        price,
			unitConfig:   unitConfig,
		},
		Usage: &rating.Usage{
			Quantity:              alpacadecimal.NewFromFloat(quantity),
			PreLinePeriodQuantity: alpacadecimal.NewFromFloat(preLinePeriod),
		},
	}

	out, err := (&UnitConfig{}).Mutate(input)
	require.NoError(t, err)

	usage, err := out.GetUsage()
	require.NoError(t, err)

	return usage
}

func TestUnitConfigMutator(t *testing.T) {
	unitPrice := productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)})

	t.Run("no unit_config is a no-op", func(t *testing.T) {
		usage := mutateUnitConfig(t, unitPrice, nil, 1400, 0)
		require.Equal(t, float64(1400), usage.Quantity.InexactFloat64())
		require.Equal(t, float64(0), usage.PreLinePeriodQuantity.InexactFloat64())
	})

	t.Run("conversion and rounding modes", func(t *testing.T) {
		// given a single-period line (Pre = 0), the billed quantity is round(convert(Quantity)).
		cases := []struct {
			name           string
			cfg            *productcatalog.UnitConfig
			quantity       float64
			expectQuantity float64
		}{
			{"multiply, no rounding", newUnitConfig(opMultiply, 1.2, roundNone), 10, 12},
			{"divide, ceiling rounds up", newUnitConfig(opDivide, 1000, roundCeiling), 1400, 2},
			{"divide, floor rounds down", newUnitConfig(opDivide, 1000, roundFloor), 1900, 1},
			{"divide, half_up rounds half away from zero", newUnitConfig(opDivide, 1000, roundHalfUp), 1500, 2},
			{"divide, half_up below half rounds down", newUnitConfig(opDivide, 1000, roundHalfUp), 1400, 1},
			{"divide, none keeps precision", newUnitConfig(opDivide, 1000, roundNone), 1400, 1.4},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				usage := mutateUnitConfig(t, unitPrice, tc.cfg, tc.quantity, 0)
				require.Equal(t, tc.expectQuantity, usage.Quantity.InexactFloat64())
				require.Equal(t, float64(0), usage.PreLinePeriodQuantity.InexactFloat64())
			})
		}
	})

	t.Run("converts quantity and pre-line-period independently", func(t *testing.T) {
		// given:
		// - a split line carrying a non-zero pre-line-period, divide by 1000, ceiling.
		// when:
		// - the mutator converts each endpoint on its own (no cumulative delta).
		// then:
		// - Quantity = ceil(1300/1000) = 2 and PreLinePeriodQuantity = ceil(1400/1000) = 2,
		//   both in converted units so tiered pricers see converted tier boundaries. The
		//   retired cumulative-endpoint approach would instead have billed ceil(2.7)-ceil(1.4)=1;
		//   making split lines sum correctly under non-linear rounding is now the invoice
		//   layer's job, not the mutator's.
		usage := mutateUnitConfig(t, unitPrice, newUnitConfig(opDivide, 1000, roundCeiling), 1300, 1400)
		require.Equal(t, float64(2), usage.Quantity.InexactFloat64())
		require.Equal(t, float64(2), usage.PreLinePeriodQuantity.InexactFloat64())
	})

	t.Run("unsupported price type errors instead of billing raw", func(t *testing.T) {
		// A package price cannot carry a unit_config (the validator blocks it); if one
		// slips through, the mutator must surface the inconsistency rather than silently
		// bill the raw quantity.
		packagePrice := productcatalog.NewPriceFrom(productcatalog.PackagePrice{
			Amount:             alpacadecimal.NewFromInt(10),
			QuantityPerPackage: alpacadecimal.NewFromInt(1000),
		})

		input := rate.PricerCalculateInput{
			StandardLineAccessor: unitConfigTestLine{
				StandardLine: &billing.StandardLine{},
				price:        packagePrice,
				unitConfig:   newUnitConfig(opDivide, 1000, roundCeiling),
			},
			Usage: &rating.Usage{
				Quantity:              alpacadecimal.NewFromFloat(1400),
				PreLinePeriodQuantity: alpacadecimal.NewFromFloat(0),
			},
		}

		_, err := (&UnitConfig{}).Mutate(input)
		require.ErrorIs(t, err, ErrUnitConfigUnsupportedPrice)
	})

	t.Run("corrupt zero conversion factor errors instead of panicking", func(t *testing.T) {
		// Apply divides by the conversion factor with no zero guard; the authoring
		// validator makes a zero factor unreachable through normal writes, but a
		// corrupt import must fail the mutation instead of panicking the billing worker.
		input := rate.PricerCalculateInput{
			StandardLineAccessor: unitConfigTestLine{
				StandardLine: &billing.StandardLine{},
				price:        unitPrice,
				unitConfig:   newUnitConfig(opDivide, 0, roundCeiling),
			},
			Usage: &rating.Usage{
				Quantity:              alpacadecimal.NewFromFloat(1400),
				PreLinePeriodQuantity: alpacadecimal.NewFromFloat(0),
			},
		}

		_, err := (&UnitConfig{}).Mutate(input)
		require.ErrorContains(t, err, "invalid unit_config on line")
	})

	t.Run("does not mutate the caller's raw usage when re-rated", func(t *testing.T) {
		// Re-rating reads from the raw metered quantity each run; the mutator must not
		// double-convert, which it guarantees by never mutating the input usage in place.
		input := rate.PricerCalculateInput{
			StandardLineAccessor: unitConfigTestLine{
				StandardLine: &billing.StandardLine{},
				price:        unitPrice,
				unitConfig:   newUnitConfig(opDivide, 1000, roundCeiling),
			},
			Usage: &rating.Usage{
				Quantity:              alpacadecimal.NewFromFloat(1400),
				PreLinePeriodQuantity: alpacadecimal.NewFromFloat(0),
			},
		}

		out1, err := (&UnitConfig{}).Mutate(input)
		require.NoError(t, err)
		out2, err := (&UnitConfig{}).Mutate(input)
		require.NoError(t, err)

		u1, err := out1.GetUsage()
		require.NoError(t, err)
		u2, err := out2.GetUsage()
		require.NoError(t, err)

		require.Equal(t, float64(2), u1.Quantity.InexactFloat64())
		require.Equal(t, u1.Quantity.InexactFloat64(), u2.Quantity.InexactFloat64())
		require.Equal(t, float64(1400), input.Usage.Quantity.InexactFloat64(), "caller's raw usage stays untouched")
	})
}
