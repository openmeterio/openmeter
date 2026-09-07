package billingservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingtestutils "github.com/openmeterio/openmeter/openmeter/billing/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestLimitGatheringLinesForInvoice(t *testing.T) {
	line := func(id, from, to string) gatheringLineWithBillablePeriod {
		gatheringLine := billing.GatheringLine{}
		gatheringLine.ID = id
		gatheringLine.ServicePeriod = timeutil.ClosedPeriod{
			From: mustTime(t, from),
			To:   mustTime(t, to),
		}

		return gatheringLineWithBillablePeriod{
			Line:           gatheringLine,
			BillablePeriod: gatheringLine.ServicePeriod,
		}
	}

	lines := []gatheringLineWithBillablePeriod{
		line("later", "2025-03-01T00:00:00Z", "2025-04-01T00:00:00Z"),
		line("tie-b", "2025-01-01T00:00:00Z", "2025-02-01T00:00:00Z"),
		line("earliest", "2024-12-01T00:00:00Z", "2025-01-01T00:00:00Z"),
		line("tie-a", "2025-01-01T00:00:00Z", "2025-02-01T00:00:00Z"),
	}

	t.Run("zero keeps all lines without reordering", func(t *testing.T) {
		got := limitGatheringLinesForInvoice(lines, 0)

		require.Equal(t, lines, got)
	})

	t.Run("positive limit keeps earliest service periods", func(t *testing.T) {
		got := limitGatheringLinesForInvoice(lines, 3)

		require.Equal(t, []string{"earliest", "tie-a", "tie-b"}, gatheringLineIDsForLimitTest(got))
	})
}

func TestGateInvoiceAssignment(t *testing.T) {
	t.Run("filters blocked lines and preserves input order", func(t *testing.T) {
		invoiceEngine := &syntheticGateLineEngine{
			NoopLineEngine: billingtestutils.NoopLineEngine{EngineType: billing.LineEngineTypeInvoice},
		}
		chargeEngine := &syntheticGateLineEngine{
			NoopLineEngine: billingtestutils.NoopLineEngine{EngineType: billing.LineEngineTypeChargeFlatFee},
		}
		invoiceEngine.gate = func(_ context.Context, input billing.GateInvoiceAssignmentInput) (billing.GateInvoiceAssignmentResult, error) {
			return billing.GateInvoiceAssignmentResult{
				input.Lines[1].GetLineID(): {ExcludeFromInvoice: true},
			}, nil
		}

		service := newGateTestService(t, invoiceEngine, chargeEngine)
		lines := []gatheringLineWithBillablePeriod{
			newGateTestGatheringLine("line-1", billing.LineEngineTypeInvoice),
			newGateTestGatheringLine("line-2", billing.LineEngineTypeInvoice),
			newGateTestGatheringLine("line-3", billing.LineEngineTypeChargeFlatFee),
		}

		result, err := service.gateInvoiceAssignment(t.Context(), lines)

		require.NoError(t, err)
		require.Equal(t, 1, result.ExcludedLineCount)
		require.Equal(t, []string{"line-1", "line-3"}, gatheringLineIDsForLimitTest(result.Lines))
		require.Len(t, invoiceEngine.inputs, 1)
		require.Len(t, invoiceEngine.inputs[0].Lines, 2)
		require.Len(t, chargeEngine.inputs, 1)
		require.Len(t, chargeEngine.inputs[0].Lines, 1)
	})

	t.Run("allows an empty candidate set without invoking an engine", func(t *testing.T) {
		service := &Service{lineEngines: newEngineRegistry()}

		result, err := service.gateInvoiceAssignment(t.Context(), nil)

		require.NoError(t, err)
		require.Empty(t, result.Lines)
		require.Zero(t, result.ExcludedLineCount)
	})

	t.Run("returns gate errors", func(t *testing.T) {
		gateErr := errors.New("gate failed")
		engine := &syntheticGateLineEngine{
			NoopLineEngine: billingtestutils.NoopLineEngine{EngineType: billing.LineEngineTypeInvoice},
			gate: func(context.Context, billing.GateInvoiceAssignmentInput) (billing.GateInvoiceAssignmentResult, error) {
				return billing.GateInvoiceAssignmentResult{}, gateErr
			},
		}
		service := newGateTestService(t, engine)

		_, err := service.gateInvoiceAssignment(t.Context(), []gatheringLineWithBillablePeriod{
			newGateTestGatheringLine("line-1", billing.LineEngineTypeInvoice),
		})

		require.ErrorIs(t, err, gateErr)
		require.ErrorContains(t, err, "gating invoice assignment with engine invoicing")
	})

	t.Run("allows omitted responses", func(t *testing.T) {
		engine := &syntheticGateLineEngine{
			NoopLineEngine: billingtestutils.NoopLineEngine{EngineType: billing.LineEngineTypeInvoice},
			gate: func(context.Context, billing.GateInvoiceAssignmentInput) (billing.GateInvoiceAssignmentResult, error) {
				return nil, nil
			},
		}
		service := newGateTestService(t, engine)

		result, err := service.gateInvoiceAssignment(t.Context(), []gatheringLineWithBillablePeriod{
			newGateTestGatheringLine("line-1", billing.LineEngineTypeInvoice),
		})

		require.NoError(t, err)
		require.Equal(t, []string{"line-1"}, gatheringLineIDsForLimitTest(result.Lines))
		require.Zero(t, result.ExcludedLineCount)
	})

	t.Run("rejects unknown decisions", func(t *testing.T) {
		engine := &syntheticGateLineEngine{
			NoopLineEngine: billingtestutils.NoopLineEngine{EngineType: billing.LineEngineTypeInvoice},
			gate: func(_ context.Context, input billing.GateInvoiceAssignmentInput) (billing.GateInvoiceAssignmentResult, error) {
				return billing.GateInvoiceAssignmentResult{
					{Namespace: "default", ID: "unknown"}: {},
				}, nil
			},
		}
		service := newGateTestService(t, engine)

		_, err := service.gateInvoiceAssignment(t.Context(), []gatheringLineWithBillablePeriod{
			newGateTestGatheringLine("line-1", billing.LineEngineTypeInvoice),
		})

		require.ErrorContains(t, err, "validating result from engine invoicing")
		require.ErrorContains(t, err, "unknown line ID: default/unknown")
	})
}

func TestResolvePendingLineCollectionCutoff(t *testing.T) {
	asOf := mustTime(t, "2025-06-15T12:00:00Z")
	anchor := mustTime(t, "2025-06-01T00:00:00Z")
	monthly := datetime.MustParseDuration(t, "P1M")

	tests := []struct {
		name       string
		opts       []billing.InvoicePendingLinesOption
		collection billing.CollectionConfig
		asOf       time.Time
		want       time.Time
		wantErr    string
	}{
		{
			name: "bypassing alignment returns as of unchanged for subscription alignment",
			opts: []billing.InvoicePendingLinesOption{
				billing.WithBypassCollectionAlignment(),
			},
			collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindSubscription,
				Interval:  datetime.MustParseDuration(t, "PT1H"),
			},
			asOf: asOf,
			want: asOf,
		},
		{
			name: "bypassing alignment returns as of unchanged for anchored alignment",
			opts: []billing.InvoicePendingLinesOption{
				billing.WithBypassCollectionAlignment(),
			},
			collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindAnchored,
				AnchoredAlignmentDetail: &billing.AnchoredAlignmentDetail{
					Interval: monthly,
					Anchor:   anchor,
				},
				Interval: datetime.MustParseDuration(t, "PT1H"),
			},
			asOf: asOf,
			want: asOf,
		},
		{
			name: "subscription alignment returns as of unchanged",
			collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindSubscription,
				Interval:  datetime.MustParseDuration(t, "PT1H"),
			},
			asOf: asOf,
			want: asOf,
		},
		{
			name: "anchored alignment returns previous anchor before as of",
			collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindAnchored,
				AnchoredAlignmentDetail: &billing.AnchoredAlignmentDetail{
					Interval: monthly,
					Anchor:   anchor,
				},
				Interval: datetime.MustParseDuration(t, "PT1H"),
			},
			asOf: mustTime(t, "2025-06-15T12:00:00Z"),
			want: mustTime(t, "2025-06-01T00:00:00Z"),
		},
		{
			name: "anchored alignment returns exact anchor when as of lands on anchor",
			collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindAnchored,
				AnchoredAlignmentDetail: &billing.AnchoredAlignmentDetail{
					Interval: monthly,
					Anchor:   anchor,
				},
				Interval: datetime.MustParseDuration(t, "PT1H"),
			},
			asOf: mustTime(t, "2025-07-01T00:00:00Z"),
			want: mustTime(t, "2025-07-01T00:00:00Z"),
		},
		{
			name: "anchored alignment can walk backwards from future anchor",
			collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindAnchored,
				AnchoredAlignmentDetail: &billing.AnchoredAlignmentDetail{
					Interval: monthly,
					Anchor:   mustTime(t, "2025-08-01T00:00:00Z"),
				},
				Interval: datetime.MustParseDuration(t, "PT1H"),
			},
			asOf: mustTime(t, "2025-06-15T12:00:00Z"),
			want: mustTime(t, "2025-06-01T00:00:00Z"),
		},
		{
			name: "anchored alignment errors without detail",
			collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindAnchored,
				Interval:  datetime.MustParseDuration(t, "PT1H"),
			},
			asOf:    asOf,
			wantErr: "anchored alignment detail is required",
		},
		{
			name: "anchored alignment errors for invalid recurrence interval",
			collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindAnchored,
				AnchoredAlignmentDetail: &billing.AnchoredAlignmentDetail{
					Interval: datetime.ISODuration{},
					Anchor:   anchor,
				},
				Interval: datetime.MustParseDuration(t, "PT1H"),
			},
			asOf:    asOf,
			wantErr: "creating anchored alignment recurrence",
		},
		{
			name: "errors for unsupported alignment",
			collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKind("invalid"),
				Interval:  datetime.MustParseDuration(t, "PT1H"),
			},
			asOf:    asOf,
			wantErr: "unsupported collection alignment: invalid",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := resolvePendingLineCollectionCutoff(billing.NewInvoicePendingLinesOptions(test.opts...), test.collection, test.asOf)

			if test.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.wantErr)
				require.True(t, got.IsZero())
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)

	return parsed
}

func gatheringLineIDsForLimitTest(lines []gatheringLineWithBillablePeriod) []string {
	ids := make([]string, 0, len(lines))
	for _, line := range lines {
		ids = append(ids, line.Line.ID)
	}

	return ids
}

type syntheticGateLineEngine struct {
	billingtestutils.NoopLineEngine

	gate   func(context.Context, billing.GateInvoiceAssignmentInput) (billing.GateInvoiceAssignmentResult, error)
	inputs []billing.GateInvoiceAssignmentInput
}

func (e *syntheticGateLineEngine) GateInvoiceAssignment(ctx context.Context, input billing.GateInvoiceAssignmentInput) (billing.GateInvoiceAssignmentResult, error) {
	e.inputs = append(e.inputs, input)
	if e.gate == nil {
		return nil, nil
	}

	return e.gate(ctx, input)
}

func newGateTestService(t *testing.T, engines ...billing.LineEngine) *Service {
	t.Helper()

	service := &Service{lineEngines: newEngineRegistry()}
	for _, engine := range engines {
		require.NoError(t, service.lineEngines.Register(engine))
	}

	return service
}

func newGateTestGatheringLine(id string, engineType billing.LineEngineType) gatheringLineWithBillablePeriod {
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	line := billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
		ID:            id,
		Namespace:     "default",
		Period:        period,
		InvoiceAt:     period.To,
		Name:          id,
		Currency:      currencyx.FiatCode("USD"),
		ManagedBy:     billing.ManuallyManagedLine,
		PerUnitAmount: alpacadecimal.NewFromInt(1),
		PaymentTerm:   productcatalog.InArrearsPaymentTerm,
	})
	line.Engine = engineType

	return gatheringLineWithBillablePeriod{
		Line:           line,
		BillablePeriod: period,
	}
}
