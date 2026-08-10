package billing

import (
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestGatheringInvoiceLineInvoiceAtAccessor(t *testing.T) {
	originalInvoiceAt := lo.Must(time.Parse(time.RFC3339, "2026-01-01T00:00:00Z"))
	line := GatheringLine{
		GatheringLineBase: GatheringLineBase{
			InvoiceAt: originalInvoiceAt,
		},
	}

	wrapper := &gatheringInvoiceLineGenericWrapper{GatheringLine: line}

	target := lo.Must(time.Parse(time.RFC3339, "2026-01-01T01:00:00Z"))
	wrapper.SetInvoiceAt(target)
	require.Equal(t, target, wrapper.GetInvoiceAt())

	accessor, ok := (GenericInvoiceLine)(wrapper).(InvoiceAtAccessor)
	if !ok {
		t.Fatalf("wrapper is not an InvoiceAtAccessor")
	}

	target = lo.Must(time.Parse(time.RFC3339, "2026-01-01T02:00:00Z"))
	accessor.SetInvoiceAt(target)
	require.Equal(t, target, accessor.GetInvoiceAt())
	require.Equal(t, originalInvoiceAt, line.InvoiceAt, "wrapped line should not be modified")
}

func TestGatheringLineUnitConfigSnapshotPropagation(t *testing.T) {
	unitConfig := &productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperationDivide,
		ConversionFactor: decimal.NewFromInt(1000),
		Rounding:         productcatalog.UnitConfigRoundingModeCeiling,
		DisplayUnit:      lo.ToPtr("GB"),
	}

	t.Run("AsNewStandardLine carries the unit_config snapshot as a deep copy", func(t *testing.T) {
		// given: a legacy gathering line carrying a unit_config snapshot
		gathering := GatheringLine{
			GatheringLineBase: GatheringLineBase{
				UnitConfig: unitConfig,
			},
		}

		// when: converting it to a standard line for billing
		std, err := gathering.AsNewStandardLine("invoice-id")
		require.NoError(t, err)

		// then: the snapshot is exposed via GetUnitConfig for the rating mutator...
		require.NotNil(t, std.UsageBased.UnitConfig)
		require.True(t, unitConfig.Equal(std.GetUnitConfig()))

		// ...and it is a deep copy, so mutating the standard line's config does not
		// leak back into the gathering line.
		std.UsageBased.UnitConfig.ConversionFactor = decimal.NewFromInt(1)
		require.True(t, unitConfig.Equal(gathering.UnitConfig))
	})

	t.Run("AsNewStandardLine leaves UnitConfig nil when the gathering line has none", func(t *testing.T) {
		// given: a legacy gathering line without a unit_config rate card
		gathering := GatheringLine{GatheringLineBase: GatheringLineBase{}}

		// when: converting it to a standard line
		std, err := gathering.AsNewStandardLine("invoice-id")
		require.NoError(t, err)

		// then: no config is applied, so rating bills raw (identity) as before
		require.Nil(t, std.UsageBased.UnitConfig)
		require.Nil(t, std.GetUnitConfig())
	})

	t.Run("Clone deep-copies the unit_config snapshot", func(t *testing.T) {
		base := GatheringLineBase{UnitConfig: unitConfig}

		cloned, err := base.Clone()
		require.NoError(t, err)
		require.True(t, unitConfig.Equal(cloned.UnitConfig))

		cloned.UnitConfig.ConversionFactor = decimal.NewFromInt(1)
		require.True(t, unitConfig.Equal(base.UnitConfig), "clone must not share the pointer with the original")
	})
}

func TestGatheringLinesCollectFeatureMeterRefs(t *testing.T) {
	// given:
	// - flat, usage-based, and featureless gathering lines
	// when:
	// - their feature meter references are collected
	// then:
	// - only feature-backed lines are returned with price-specific meter requirements
	featureKey := "api-requests"
	lines := GatheringLines{
		{
			GatheringLineBase: GatheringLineBase{
				FeatureKey: featureKey,
				Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      decimal.NewFromInt(1),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				})),
			},
		},
		{
			GatheringLineBase: GatheringLineBase{
				FeatureKey: featureKey,
				Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: decimal.NewFromInt(1),
				})),
			},
		},
		{
			GatheringLineBase: GatheringLineBase{
				Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      decimal.NewFromInt(1),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				})),
			},
		},
	}

	refs := lines.CollectFeatureMeterRefs()

	require.Len(t, refs, 2)
	require.Equal(t, featureKey, refs[0].IDOrKey.Key)
	require.False(t, refs[0].RequireMeter)
	require.Equal(t, featureKey, refs[1].IDOrKey.Key)
	require.True(t, refs[1].RequireMeter)
}
