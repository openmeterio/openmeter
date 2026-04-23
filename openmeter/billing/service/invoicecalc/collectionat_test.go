package invoicecalc

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestGatheringInvoiceCollectionAt(t *testing.T) {
	anchor := mustTime(t, "2025-06-01T00:00:00Z")
	monthly := datetime.MustParseDuration(t, "P1M")

	tests := []struct {
		name    string
		invoice billing.GatheringInvoice
		deps    GatheringInvoiceCalculatorDependencies
		want    time.Time
		wantErr string
	}{
		{
			name: "subscription alignment uses earliest invoice at",
			invoice: newGatheringInvoice(
				newGatheringLine(t, "2025-06-15T12:00:00Z"),
				newGatheringLine(t, "2025-06-05T08:00:00Z"),
			),
			deps: GatheringInvoiceCalculatorDependencies{
				Collection: billing.CollectionConfig{
					Alignment: billing.AlignmentKindSubscription,
					Interval:  datetime.MustParseDuration(t, "PT1H"),
				},
			},
			want: mustTime(t, "2025-06-05T08:00:00Z"),
		},
		{
			name: "anchored alignment snaps to next anchor after earliest invoice at",
			invoice: newGatheringInvoice(
				newGatheringLine(t, "2025-06-15T12:00:00Z"),
				newGatheringLine(t, "2025-06-05T08:00:00Z"),
			),
			deps: GatheringInvoiceCalculatorDependencies{
				Collection: billing.CollectionConfig{
					Alignment: billing.AlignmentKindAnchored,
					AnchoredAlignmentDetail: &billing.AnchoredAlignmentDetail{
						Interval: monthly,
						Anchor:   anchor,
					},
					Interval: datetime.MustParseDuration(t, "PT1H"),
				},
			},
			want: mustTime(t, "2025-07-01T00:00:00Z"),
		},
		{
			name: "anchored alignment keeps exact anchor",
			invoice: newGatheringInvoice(
				newGatheringLine(t, "2025-07-01T00:00:00Z"),
			),
			deps: GatheringInvoiceCalculatorDependencies{
				Collection: billing.CollectionConfig{
					Alignment: billing.AlignmentKindAnchored,
					AnchoredAlignmentDetail: &billing.AnchoredAlignmentDetail{
						Interval: monthly,
						Anchor:   anchor,
					},
					Interval: datetime.MustParseDuration(t, "PT1H"),
				},
			},
			want: mustTime(t, "2025-07-01T00:00:00Z"),
		},
		{
			name: "anchored alignment can walk forward from future anchor basis",
			invoice: newGatheringInvoice(
				newGatheringLine(t, "2025-06-15T12:00:00Z"),
			),
			deps: GatheringInvoiceCalculatorDependencies{
				Collection: billing.CollectionConfig{
					Alignment: billing.AlignmentKindAnchored,
					AnchoredAlignmentDetail: &billing.AnchoredAlignmentDetail{
						Interval: monthly,
						Anchor:   mustTime(t, "2025-08-01T00:00:00Z"),
					},
					Interval: datetime.MustParseDuration(t, "PT1H"),
				},
			},
			want: mustTime(t, "2025-07-01T00:00:00Z"),
		},
		{
			name: "deleted lines are ignored",
			invoice: newGatheringInvoice(
				newDeletedGatheringLine(t, "2025-06-05T08:00:00Z"),
				newGatheringLine(t, "2025-06-15T12:00:00Z"),
			),
			deps: GatheringInvoiceCalculatorDependencies{
				Collection: billing.CollectionConfig{
					Alignment: billing.AlignmentKindSubscription,
					Interval:  datetime.MustParseDuration(t, "PT1H"),
				},
			},
			want: mustTime(t, "2025-06-15T12:00:00Z"),
		},
		{
			name:    "empty lines leaves zero collection at",
			invoice: billing.GatheringInvoice{Lines: billing.NewGatheringInvoiceLines([]billing.GatheringLine{})},
			deps: GatheringInvoiceCalculatorDependencies{
				Collection: billing.CollectionConfig{
					Alignment: billing.AlignmentKindSubscription,
					Interval:  datetime.MustParseDuration(t, "PT1H"),
				},
			},
			want: time.Time{},
		},
		{
			name: "errors when lines are not expanded",
			deps: GatheringInvoiceCalculatorDependencies{
				Collection: billing.CollectionConfig{
					Alignment: billing.AlignmentKindSubscription,
					Interval:  datetime.MustParseDuration(t, "PT1H"),
				},
			},
			wantErr: "lines must be expanded",
		},
		{
			name: "errors on anchored alignment without detail",
			invoice: newGatheringInvoice(
				newGatheringLine(t, "2025-06-15T12:00:00Z"),
			),
			deps: GatheringInvoiceCalculatorDependencies{
				Collection: billing.CollectionConfig{
					Alignment: billing.AlignmentKindAnchored,
					Interval:  datetime.MustParseDuration(t, "PT1H"),
				},
			},
			wantErr: "invalid collection config: anchored alignment detail must be set",
		},
		{
			name: "errors on invalid anchored recurrence",
			invoice: newGatheringInvoice(
				newGatheringLine(t, "2025-06-15T12:00:00Z"),
			),
			deps: GatheringInvoiceCalculatorDependencies{
				Collection: billing.CollectionConfig{
					Alignment: billing.AlignmentKindAnchored,
					AnchoredAlignmentDetail: &billing.AnchoredAlignmentDetail{
						Interval: datetime.ISODuration{},
						Anchor:   anchor,
					},
					Interval: datetime.MustParseDuration(t, "PT1H"),
				},
			},
			wantErr: "creating anchored alignment recurrence",
		},
		{
			name: "errors on unsupported alignment",
			invoice: newGatheringInvoice(
				newGatheringLine(t, "2025-06-15T12:00:00Z"),
			),
			deps: GatheringInvoiceCalculatorDependencies{
				Collection: billing.CollectionConfig{
					Alignment: billing.AlignmentKind("invalid"),
					Interval:  datetime.MustParseDuration(t, "PT1H"),
				},
			},
			wantErr: "invalid collection config: invalid alignment: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GatheringInvoiceCollectionAt(&tt.invoice, tt.deps)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			if tt.want.IsZero() {
				require.Nil(t, tt.invoice.NextCollectionAt)
				return
			}

			require.NotNil(t, tt.invoice.NextCollectionAt)
			require.Equal(t, tt.want, *tt.invoice.NextCollectionAt)
		})
	}
}

func TestStandardInvoiceCollectionAt(t *testing.T) {
	tests := []struct {
		name    string
		invoice billing.StandardInvoice
		want    time.Time
		wantErr string
	}{
		{
			name: "uses latest non deleted invoice at and adds collection interval",
			invoice: newStandardInvoice(
				mustTime(t, "2025-06-10T00:00:00Z"),
				datetime.MustParseDuration(t, "PT1H"),
				newStandardLine(t, "2025-06-05T08:00:00Z"),
				newStandardLine(t, "2025-06-15T12:00:00Z"),
			),
			want: mustTime(t, "2025-06-15T13:00:00Z"),
		},
		{
			name: "ignores flat fee lines when resolving collection at",
			invoice: newStandardInvoice(
				mustTime(t, "2025-06-10T00:00:00Z"),
				datetime.MustParseDuration(t, "PT1H"),
				newFlatFeeStandardLine(t, "2025-06-20T12:00:00Z"),
				newStandardLine(t, "2025-06-15T12:00:00Z"),
			),
			want: mustTime(t, "2025-06-15T13:00:00Z"),
		},
		{
			name: "ignores deleted lines when resolving latest invoice at",
			invoice: newStandardInvoice(
				mustTime(t, "2025-06-10T00:00:00Z"),
				datetime.MustParseDuration(t, "PT1H"),
				newDeletedStandardLine(t, "2025-06-20T12:00:00Z"),
				newStandardLine(t, "2025-06-15T12:00:00Z"),
			),
			want: mustTime(t, "2025-06-15T13:00:00Z"),
		},
		{
			name: "returns nil collection at when there are no non deleted metered lines",
			invoice: newStandardInvoice(
				mustTime(t, "2025-06-10T00:00:00Z"),
				datetime.MustParseDuration(t, "PT1H"),
				newDeletedStandardLine(t, "2025-06-20T12:00:00Z"),
			),
			want: time.Time{},
		},
		{
			name: "uses latest non deleted metered invoice at without interval when interval is zero",
			invoice: newStandardInvoice(
				mustTime(t, "2025-06-10T00:00:00Z"),
				datetime.ISODuration{},
				newStandardLine(t, "2025-06-05T08:00:00Z"),
				newStandardLine(t, "2025-06-15T12:00:00Z"),
			),
			want: mustTime(t, "2025-06-15T12:00:00Z"),
		},
		{
			name:    "errors when lines are not expanded",
			invoice: billing.StandardInvoice{},
			wantErr: "lines must be expanded",
		},
		{
			name: "returns nil collection at for flat fee only invoices",
			invoice: newStandardInvoice(
				mustTime(t, "2025-06-10T00:00:00Z"),
				datetime.MustParseDuration(t, "PT1H"),
				newFlatFeeStandardLine(t, "2025-06-20T12:00:00Z"),
			),
			want: time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := StandardInvoiceCollectionAt(&tt.invoice)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			if tt.want.IsZero() {
				require.Nil(t, tt.invoice.CollectionAt)
				return
			}

			require.NotNil(t, tt.invoice.CollectionAt)
			require.Equal(t, tt.want, *tt.invoice.CollectionAt)
		})
	}
}

func newGatheringInvoice(lines ...billing.GatheringLine) billing.GatheringInvoice {
	return billing.GatheringInvoice{
		Lines: billing.NewGatheringInvoiceLines(lines),
	}
}

func newGatheringLine(t *testing.T, invoiceAt string) billing.GatheringLine {
	t.Helper()

	at := mustTime(t, invoiceAt)

	return billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			InvoiceAt: at,
		},
	}
}

func newDeletedGatheringLine(t *testing.T, invoiceAt string) billing.GatheringLine {
	t.Helper()

	line := newGatheringLine(t, invoiceAt)
	deletedAt := mustTime(t, "2025-07-15T12:00:00Z")
	line.DeletedAt = &deletedAt

	return line
}

func newStandardInvoice(createdAt time.Time, interval datetime.ISODuration, lines ...*billing.StandardLine) billing.StandardInvoice {
	return billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			CreatedAt: createdAt,
			Workflow: billing.InvoiceWorkflow{
				Config: billing.WorkflowConfig{
					Collection: billing.CollectionConfig{
						Alignment: billing.AlignmentKindSubscription,
						Interval:  interval,
					},
				},
			},
		},
		Lines: billing.NewStandardInvoiceLines(lines),
	}
}

func newStandardLine(t *testing.T, invoiceAt string) *billing.StandardLine {
	t.Helper()

	at := mustTime(t, invoiceAt)

	return &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: billing.GatheringLineBase{}.ManagedResource,
			Currency:        currencyx.Code("USD"),
			ManagedBy:       billing.ManuallyManagedLine,
			InvoiceAt:       at,
			Period: timeutil.ClosedPeriod{
				From: at,
				To:   at,
			},
		},
		UsageBased: &billing.UsageBasedLine{
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
		},
	}
}

func newDeletedStandardLine(t *testing.T, invoiceAt string) *billing.StandardLine {
	t.Helper()

	line := newStandardLine(t, invoiceAt)
	deletedAt := mustTime(t, "2025-07-15T12:00:00Z")
	line.DeletedAt = &deletedAt

	return line
}

func newFlatFeeStandardLine(t *testing.T, invoiceAt string) *billing.StandardLine {
	t.Helper()

	at := mustTime(t, invoiceAt)

	return billing.NewFlatFeeLine(billing.NewFlatFeeLineInput{
		Namespace: "ns",
		InvoiceAt: at,
		Period: timeutil.ClosedPeriod{
			From: at,
			To:   at,
		},
		Name:      "Flat fee",
		Currency:  currencyx.Code("USD"),
		ManagedBy: billing.ManuallyManagedLine,

		PerUnitAmount: alpacadecimal.NewFromFloat(10),
		PaymentTerm:   productcatalog.InArrearsPaymentTerm,
	})
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)

	return parsed
}
