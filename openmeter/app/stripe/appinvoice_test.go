package appstripe

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
)

func TestWouldStandardInvoiceBeEmpty(t *testing.T) {
	lineWithoutDetails := &billing.StandardLine{}
	lineWithDetails := &billing.StandardLine{
		DetailedLines: billing.DetailedLines{{}},
	}

	tests := []struct {
		name  string
		lines []*billing.StandardLine
		total int64
		want  bool
	}{
		{
			name: "no lines",
			want: true,
		},
		{
			name:  "lines without anything to bill",
			lines: []*billing.StandardLine{lineWithoutDetails},
			want:  true,
		},
		{
			name:  "billable lines that add up to zero",
			lines: []*billing.StandardLine{lineWithDetails},
			want:  true,
		},
		{
			name:  "billable lines with an amount to pay",
			lines: []*billing.StandardLine{lineWithDetails},
			total: 10,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			empty, err := appOperations{}.WouldStandardInvoiceBeEmpty(billing.StandardInvoice{
				Lines:  billing.NewStandardInvoiceLines(tt.lines),
				Totals: totals.Totals{Total: alpacadecimal.NewFromInt(tt.total)},
			})

			require.NoError(t, err)
			require.Equal(t, tt.want, empty)
		})
	}
}
