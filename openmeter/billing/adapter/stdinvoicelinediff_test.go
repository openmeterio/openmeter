package billingadapter

import (
	"fmt"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/entitydiff"
	"github.com/openmeterio/openmeter/pkg/models"
)

type idDiff struct {
	ToCreate []string
	ToUpdate []string
	ToDelete []string
}

type lineDiffExpectation struct {
	Line idDiff

	AmountDiscounts idDiff

	DetailedLine                idDiff
	DetailedLineAmountDiscounts idDiff

	AffectedLineIDs             []string
	DetailedLineAffectedLineIDs []string
}

func TestInvoiceLineDiffing(t *testing.T) {
	template := []*billing.StandardLine{
		{
			StandardLineBase: billing.StandardLineBase{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					ID: "1",
				}),
			},
			UsageBased: &billing.UsageBasedLine{},
		},
		{
			StandardLineBase: billing.StandardLineBase{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					ID: "2",
				}),
			},
			UsageBased: &billing.UsageBasedLine{},
			DetailedLines: billing.DetailedLines{
				{
					DetailedLineBase: billing.DetailedLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							ID: "2.1",
						}),
					},
					AmountDiscounts: newDetailedLineAmountDiscountsWithIDs("D2.1.1"),
				},
				{
					DetailedLineBase: billing.DetailedLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							ID: "2.2",
						}),
					},
				},
			},
		},
	}

	t.Run("new line hierarchy (all lines are created)", func(t *testing.T) {
		base := cloneLines(template)

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			Line: idDiff{
				ToCreate: []string{"1", "2"},
			},
			DetailedLine: idDiff{
				ToCreate: []string{"2.1", "2.2"},
			},
			DetailedLineAmountDiscounts: idDiff{
				ToCreate: []string{"D2.1.1"},
			},
		}, lineDiff)
	})

	t.Run("existing line hierarchy, no changes", func(t *testing.T) {
		base := cloneLines(template)
		snapshotAsDBState(base)

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{}, lineDiff)
	})

	t.Run("existing line hierarchy, one child line is deleted", func(t *testing.T) {
		base := cloneLines(template)
		snapshotAsDBState(base)

		require.True(t, removeDetailedLineByID(base[1], "2.1"), "child line 2.1 should be removed")

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			AffectedLineIDs: []string{"2"},
			DetailedLine: idDiff{
				ToDelete: []string{"2.1"},
			},
			DetailedLineAmountDiscounts: idDiff{
				ToDelete: []string{"D2.1.1"},
			},
		}, lineDiff)
	})

	t.Run("existing line hierarchy, one child line is changed", func(t *testing.T) {
		base := cloneLines(template)
		snapshotAsDBState(base)

		getDetailedLineByID(base[1], "2.1").Quantity = alpacadecimal.NewFromFloat(10)

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			AffectedLineIDs: []string{"2"},
			DetailedLine: idDiff{
				ToUpdate: []string{"2.1"},
			},
		}, lineDiff)
	})

	t.Run("existing line hierarchy, one parent line is changed", func(t *testing.T) {
		base := cloneLines(template)
		snapshotAsDBState(base)

		base[1].UsageBased.Quantity = lo.ToPtr(alpacadecimal.NewFromFloat(10))

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			Line: idDiff{
				ToUpdate: []string{"2"},
			},
		}, lineDiff)
	})

	t.Run("a line is updated in the existing line hieararchy", func(t *testing.T) {
		base := cloneLines(template)
		snapshotAsDBState(base)

		// ID change should tirgger a delete/update
		changedLine := getDetailedLineByID(base[1], "2.1")
		changedLine.ID = ""
		changedLine.Description = lo.ToPtr("2.3")

		changedLine.AmountDiscounts[0].ID = "D2.1.3"

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			AffectedLineIDs: []string{"2"},
			DetailedLine: idDiff{
				ToDelete: []string{"2.1"},
				ToCreate: []string{"2.3"},
			},
			DetailedLineAmountDiscounts: idDiff{
				// The discount gets deleted + created
				ToCreate: []string{"D2.1.3"},
				ToDelete: []string{"D2.1.1"},
			},
		}, lineDiff)
	})

	// Discount handling
	t.Run("existing line hierarchy, one discount is deleted", func(t *testing.T) {
		base := cloneLines(template)
		snapshotAsDBState(base)

		getDetailedLineByID(base[1], "2.1").AmountDiscounts = nil

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			AffectedLineIDs:             []string{"2"},
			DetailedLineAffectedLineIDs: []string{"2.1"},
			DetailedLineAmountDiscounts: idDiff{
				ToDelete: []string{"D2.1.1"},
			},
		}, lineDiff)
	})

	t.Run("existing line hierarchy, one discount is changed", func(t *testing.T) {
		base := cloneLines(template)
		snapshotAsDBState(base)

		getDetailedLineByID(base[1], "2.1").AmountDiscounts[0].Amount = alpacadecimal.NewFromFloat(20)

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			AffectedLineIDs:             []string{"2"},
			DetailedLineAffectedLineIDs: []string{"2.1"},
			DetailedLineAmountDiscounts: idDiff{
				ToUpdate: []string{"D2.1.1"},
			},
		}, lineDiff)
	})

	t.Run("existing line hierarchy, one discount is added/old one is removed", func(t *testing.T) {
		base := cloneLines(template)
		snapshotAsDBState(base)

		discounts := getDetailedLineByID(base[1], "2.1").AmountDiscounts

		discounts[0].ID = ""
		discounts[0].Description = lo.ToPtr("D2.1.2")

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			AffectedLineIDs:             []string{"2"},
			DetailedLineAffectedLineIDs: []string{"2.1"},
			DetailedLineAmountDiscounts: idDiff{
				ToCreate: []string{"D2.1.2"},
				ToDelete: []string{"D2.1.1"},
			},
		}, lineDiff)
	})

	// DeletedAt handling
	t.Run("support for detailed lines being deleted using deletedAt", func(t *testing.T) {
		base := cloneLines(template)
		snapshotAsDBState(base)

		getDetailedLineByID(base[1], "2.1").DeletedAt = lo.ToPtr(clock.Now())

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			AffectedLineIDs: []string{"2"},
			DetailedLine: idDiff{
				ToDelete: []string{"2.1"},
			},
			DetailedLineAmountDiscounts: idDiff{
				ToDelete: []string{"D2.1.1"},
			},
		}, lineDiff)
	})

	t.Run("support for parent lines with children being deleted using deletedAt", func(t *testing.T) {
		base := cloneLines(template)
		snapshotAsDBState(base)

		base[1].DeletedAt = lo.ToPtr(clock.Now())

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			Line: idDiff{
				ToDelete: []string{"2"},
			},
			DetailedLine: idDiff{
				ToDelete: []string{"2.1", "2.2"},
			},
			DetailedLineAmountDiscounts: idDiff{
				ToDelete: []string{"D2.1.1"},
			},
		}, lineDiff)
	})

	t.Run("support for parent lines without children being deleted using deletedAt", func(t *testing.T) {
		base := cloneLines(template)
		snapshotAsDBState(base)

		base[0].DeletedAt = lo.ToPtr(clock.Now())

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			Line: idDiff{
				ToDelete: []string{"1"},
			},
		}, lineDiff)
	})

	t.Run("deleted, changed lines are not triggering updates", func(t *testing.T) {
		base := cloneLines(template)
		base[0].DeletedAt = lo.ToPtr(clock.Now())
		snapshotAsDBState(base)
		base[0].Description = lo.ToPtr("test")

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{}, lineDiff)
	})
}

func mapDiffToIDs[T entitydiff.Entity](in entitydiff.Diff[T], getDescription func(T) *string) idDiff {
	return idDiff{
		ToCreate: lo.Map(in.Create, func(item T, _ int) string {
			return lo.FromPtrOr(getDescription(item), item.GetID())
		}),
		ToUpdate: lo.Map(in.Update, func(item entitydiff.DiffUpdate[T], _ int) string {
			return lo.FromPtrOr(getDescription(item.PersistedState), item.PersistedState.GetID())
		}),
		ToDelete: lo.Map(in.Delete, func(item T, _ int) string {
			return lo.FromPtrOr(getDescription(item), item.GetID())
		}),
	}
}

func msgPrefix(prefix string, in ...interface{}) []interface{} {
	if len(in) == 0 {
		return []interface{}{prefix}
	}

	if formatString, ok := in[0].(string); ok {
		formatString = fmt.Sprintf("%s: %s", prefix, formatString)
		return append([]interface{}{formatString}, in[1:]...)
	}

	return in
}

func requireIdDiffMatches[T entitydiff.Entity](t *testing.T, a idDiff, b entitydiff.Diff[T], getDescription func(T) *string, msgAndArgs ...interface{}) {
	t.Helper()

	idDiffB := mapDiffToIDs(b, getDescription)

	require.ElementsMatch(t, a.ToCreate, idDiffB.ToCreate, msgPrefix("ToCreate", msgAndArgs...))
	require.ElementsMatch(t, a.ToUpdate, idDiffB.ToUpdate, msgPrefix("ToUpdate", msgAndArgs...))
	require.ElementsMatch(t, a.ToDelete, idDiffB.ToDelete, msgPrefix("ToDelete", msgAndArgs...))
}

func requireDiff(t *testing.T, expected lineDiffExpectation, actual invoiceLineDiff) {
	t.Helper()

	requireIdDiffMatches(t, expected.Line, actual.Line, func(line *billing.StandardLine) *string { return line.GetDescription() }, "line diff")
	requireIdDiffMatches(t, expected.AmountDiscounts, actual.AmountDiscounts, func(discount amountLineDiscountManagedWithLine) *string { return discount.Entity.Description }, "amount discounts")

	requireIdDiffMatches(t, expected.DetailedLine, actual.DetailedLine, func(line detailedLineWithParent) *string { return line.Entity.GetDescription() }, "detailed line diff")
	requireIdDiffMatches(t, expected.DetailedLineAmountDiscounts, actual.DetailedLineAmountDiscounts, func(discount detailedLineAmountDiscountWithParent) *string { return discount.Entity.Description }, "detailed line amount discounts")

	require.ElementsMatch(t, expected.AffectedLineIDs, actual.AffectedLineIDs.AsSlice(), "affected line IDs")
	require.ElementsMatch(t, expected.DetailedLineAffectedLineIDs, actual.DetailedLineAffectedLineIDs.AsSlice(), "detailed line affected line IDs")
}

func cloneLines(lines []*billing.StandardLine) []*billing.StandardLine {
	return lo.Map(lines, func(line *billing.StandardLine, _ int) *billing.StandardLine {
		return line.Clone()
	})
}

// snapshotAsDBState saves the current state of the lines as if they were in the database
func snapshotAsDBState(lines []*billing.StandardLine) {
	for _, line := range lines {
		line.SaveDBSnapshot()
	}
}

func newDetailedLineAmountDiscountsWithIDs(ids ...string) billing.AmountLineDiscountsManaged {
	return lo.Map(ids, func(id string, _ int) billing.AmountLineDiscountManaged {
		return billing.AmountLineDiscountManaged{
			ManagedModelWithID: models.ManagedModelWithID{
				ID: id,
			},
			AmountLineDiscount: billing.AmountLineDiscount{
				Amount: alpacadecimal.NewFromFloat(10),
			},
		}
	})
}

func getDetailedLineByID(l *billing.StandardLine, id string) *billing.DetailedLine {
	for idx := range l.DetailedLines {
		if l.DetailedLines[idx].ID == id {
			return &l.DetailedLines[idx]
		}
	}
	return nil
}

func removeDetailedLineByID(l *billing.StandardLine, id string) bool {
	toBeRemoved := getDetailedLineByID(l, id)
	if toBeRemoved == nil {
		return false
	}

	l.DetailedLines = lo.Filter(l.DetailedLines, func(dl billing.DetailedLine, _ int) bool {
		return dl.ID != id
	})
	return true
}
