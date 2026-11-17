package billing

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/models/externalid"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestListStandardInvoicesInputValidateRequiresNamespace(t *testing.T) {
	// given:
	// - a standard invoice list input without a namespace
	// when:
	// - the input is validated
	// then:
	// - validation rejects the request
	err := (ListStandardInvoicesInput{}).Validate()

	require.ErrorContains(t, err, "namespace is required")
}

func TestStandardInvoiceStatusIsFailedIncludesValidationFailures(t *testing.T) {
	require.True(t, StandardInvoiceStatusDraftInvalidCreated.IsFailed())
	require.True(t, StandardInvoiceStatusDraftInvalid.IsFailed())
	require.False(t, StandardInvoiceStatusDraftWaitingForCollection.IsFailed())
}

func TestSortLines(t *testing.T) {
	lines := StandardLines{
		{
			StandardLineBase: StandardLineBase{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					Name:        "usage-based-line",
					Description: lo.ToPtr("index=1"),
				}),
				Period: timeutil.ClosedPeriod{
					From: time.Now().Add(time.Hour * 24),
				},
			},
			DetailedLines: DetailedLines{
				{
					DetailedLineBase: DetailedLineBase{
						Base: stddetailedline.Base{
							ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
								Name:        "child-2",
								Description: lo.ToPtr("index=1.1"),
							}),
							Index: lo.ToPtr(1),
						},
					},
				},
				{
					DetailedLineBase: DetailedLineBase{
						Base: stddetailedline.Base{
							ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
								Name:        "child-1",
								Description: lo.ToPtr("index=1.0"),
							}),
							Index: lo.ToPtr(0),
						},
					},
				},
			},
		},
		{
			StandardLineBase: StandardLineBase{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					Name:        "usage-based-line",
					Description: lo.ToPtr("index=0"),
				}),
				Period: timeutil.ClosedPeriod{
					From: time.Now(),
				},
			},
		},
	}

	lines.Sort()

	require.Equal(t, *lines[0].Description, "index=0")
	require.Equal(t, *lines[1].Description, "index=1")
	children := lines[1].DetailedLines
	require.Equal(t, *children[0].Description, "index=1.0")
	require.Equal(t, *children[1].Description, "index=1.1")
}

func TestStandardInvoiceLinesReplaceExact(t *testing.T) {
	newLine := func(id string) *StandardLine {
		line := validStandardLineForValidation()
		line.ID = id

		return &line
	}

	// given: an engine-owned line with an existing database snapshot
	dbState := newLine("line-1")
	dbState.Name = "persisted line"
	existing := newLine("line-1")
	existing.DBState = dbState
	unrelated := newLine("line-2")
	lines := NewStandardInvoiceLines(StandardLines{existing, unrelated})

	replacement := newLine("line-1")
	replacement.Name = "updated line"
	replacement.DBState = newLine("replacement-db-state")

	// when: the engine replaces exactly the lines it owns
	err := lines.ReplaceExact(ReplaceExactLinesInput{
		Existing:    StandardLines{existing},
		Replacement: StandardLines{replacement},
	})

	// then: only the owned line is replaced and its original database snapshot is preserved
	require.NoError(t, err)
	require.Equal(t, "updated line", lines.GetByID("line-1").Name)
	require.Same(t, dbState, lines.GetByID("line-1").DBState)
	require.Same(t, unrelated, lines.GetByID("line-2"))
}

func TestStandardInvoiceLinesReplaceExactRejectsInvalidReplacement(t *testing.T) {
	newLine := func(id string) *StandardLine {
		line := validStandardLineForValidation()
		line.ID = id

		return &line
	}

	tests := []struct {
		name          string
		lines         StandardInvoiceLines
		input         ReplaceExactLinesInput
		errorContains string
	}{
		{
			name:  "invoice lines are not expanded",
			lines: StandardInvoiceLines{},
			input: ReplaceExactLinesInput{
				Existing:    StandardLines{newLine("line-1")},
				Replacement: StandardLines{newLine("line-1")},
			},
			errorContains: "cannot replace lines without expanded invoice lines",
		},
		{
			name:  "replacement line is invalid",
			lines: NewStandardInvoiceLines(StandardLines{newLine("line-1")}),
			input: ReplaceExactLinesInput{
				Existing: StandardLines{newLine("line-1")},
				Replacement: StandardLines{func() *StandardLine {
					line := newLine("line-1")
					line.UsageBased = nil

					return line
				}()},
			},
			errorContains: "replacement lines: 0: usage based line is required",
		},
		{
			name:  "replacement omits an owned line",
			lines: NewStandardInvoiceLines(StandardLines{newLine("line-1"), newLine("line-2")}),
			input: ReplaceExactLinesInput{
				Existing:    StandardLines{newLine("line-1"), newLine("line-2")},
				Replacement: StandardLines{newLine("line-1")},
			},
			errorContains: "line ids mismatch",
		},
		{
			name:  "replacement adds an unowned line",
			lines: NewStandardInvoiceLines(StandardLines{newLine("line-1"), newLine("line-2")}),
			input: ReplaceExactLinesInput{
				Existing:    StandardLines{newLine("line-1")},
				Replacement: StandardLines{newLine("line-1"), newLine("line-2")},
			},
			errorContains: "line ids mismatch",
		},
		{
			name:  "replacement changes an owned line id",
			lines: NewStandardInvoiceLines(StandardLines{newLine("line-1")}),
			input: ReplaceExactLinesInput{
				Existing:    StandardLines{newLine("line-1")},
				Replacement: StandardLines{newLine("line-renamed")},
			},
			errorContains: "line ids mismatch",
		},
		{
			name:  "owned line is not present on invoice",
			lines: NewStandardInvoiceLines(StandardLines{newLine("line-2")}),
			input: ReplaceExactLinesInput{
				Existing:    StandardLines{newLine("line-1")},
				Replacement: StandardLines{newLine("line-1")},
			},
			errorContains: "replacing line[line-1]: line not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when: an invalid line replacement is attempted
			err := tt.lines.ReplaceExact(tt.input)

			// then: the replacement is rejected before mutating unrelated invoice lines
			require.ErrorContains(t, err, tt.errorContains)
		})
	}
}

func TestStandardInvoiceClearInvoicingAppReferences(t *testing.T) {
	stripeReference := externalid.LineExternalIDs{Invoicing: "il_123"}

	// given: an invoice with invoicing references on every level
	invoice := StandardInvoice{
		StandardInvoiceBase: StandardInvoiceBase{
			ExternalIDs: externalid.InvoiceExternalIDs{Invoicing: "in_123", Payment: "pi_123"},
		},
		Lines: NewStandardInvoiceLines(StandardLines{
			{
				StandardLineBase: StandardLineBase{ExternalIDs: stripeReference},
				Discounts: StandardLineDiscounts{
					Usage: UsageLineDiscountsManaged{
						{UsageLineDiscount: UsageLineDiscount{LineDiscountBase: LineDiscountBase{ExternalIDs: stripeReference}}},
					},
				},
				DetailedLines: DetailedLines{
					{
						DetailedLineBase: DetailedLineBase{Base: stddetailedline.Base{ExternalIDs: stripeReference}},
						AmountDiscounts: AmountLineDiscountsManaged{
							{AmountLineDiscount: AmountLineDiscount{LineDiscountBase: LineDiscountBase{ExternalIDs: stripeReference}}},
						},
					},
				},
			},
		}),
	}

	// when: the invoicing app references are cleared
	invoice.ClearInvoicingAppReferences()

	// then: only the invoicing references are gone, on every level
	require.Equal(t, externalid.InvoiceExternalIDs{Payment: "pi_123"}, invoice.ExternalIDs)

	line := invoice.Lines.OrEmpty()[0]
	require.Empty(t, line.ExternalIDs.Invoicing)
	require.Empty(t, line.Discounts.Usage[0].ExternalIDs.Invoicing)
	require.Empty(t, line.DetailedLines[0].ExternalIDs.Invoicing)
	require.Empty(t, line.DetailedLines[0].AmountDiscounts[0].ExternalIDs.Invoicing)
}
