package billingadapter

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
)

type idDiff = diff[string]

type lineDiffExpectation struct {
	LineBase   idDiff
	FlatFee    idDiff
	UsageBased idDiff

	Discounts idDiff

	AffectedLineIDs []string
	ChildrenDiff    *lineDiffExpectation
}

func TestInvoiceLineDiffing(t *testing.T) {
	template := []*billingentity.Line{
		{
			LineBase: billingentity.LineBase{
				ID:   "1",
				Type: billingentity.InvoiceLineTypeFee,
			},
		},
		{
			LineBase: billingentity.LineBase{
				ID:   "2",
				Type: billingentity.InvoiceLineTypeUsageBased,
			},
			Children: billingentity.NewLineChildren([]*billingentity.Line{
				{
					LineBase: billingentity.LineBase{
						ID:   "2.1",
						Type: billingentity.InvoiceLineTypeFee,
					},
					Discounts: billingentity.NewLineDiscounts([]billingentity.LineDiscount{
						{
							ID: "D2.1.1",
						},
					},
					),
				},
				{
					LineBase: billingentity.LineBase{
						ID:   "2.2",
						Type: billingentity.InvoiceLineTypeFee,
					},
				},
			}),
		},
	}

	t.Run("new line hierarchy (all lines are created)", func(t *testing.T) {
		base := cloneLines(template)

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			LineBase: idDiff{
				ToCreate: []string{"1", "2"},
			},
			FlatFee: idDiff{
				ToCreate: []string{"1"},
			},
			UsageBased: idDiff{
				ToCreate: []string{"2"},
			},
			ChildrenDiff: &lineDiffExpectation{
				LineBase: idDiff{
					ToCreate: []string{"2.1", "2.2"},
				},
				FlatFee: idDiff{
					ToCreate: []string{"2.1", "2.2"},
				},
				Discounts: idDiff{
					ToCreate: []string{"D2.1.1"},
				},
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

		require.True(t, base[1].Children.RemoveByID("2.1"), "child line 2.1 should be removed")

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			AffectedLineIDs: []string{"2"},
			ChildrenDiff: &lineDiffExpectation{
				LineBase: idDiff{
					ToDelete: []string{"2.1"},
				},
				// Flat fee is not deleted as it does not have soft delete, so it's enough to mark the line as deleted
				FlatFee: idDiff{},
				Discounts: idDiff{
					// Discounts also get deleted
					ToDelete: []string{"D2.1.1"},
				},
			},
		}, lineDiff)
	})

	t.Run("existing line hierarchy, one child line is changed", func(t *testing.T) {
		base := cloneLines(template)
		snapshotAsDBState(base)

		base[1].Children.GetByID("2.1").FlatFee.Quantity = alpacadecimal.NewFromFloat(10)

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			AffectedLineIDs: []string{"2"},
			ChildrenDiff: &lineDiffExpectation{
				LineBase: idDiff{
					ToUpdate: []string{"2.1"},
				},
				FlatFee: idDiff{
					ToUpdate: []string{"2.1"},
				},
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
			LineBase: idDiff{
				ToUpdate: []string{"2"},
			},
			UsageBased: idDiff{
				ToUpdate: []string{"2"},
			},
		}, lineDiff)
	})

	t.Run("a line is updated in the existing line hieararchy", func(t *testing.T) {
		base := cloneLines(template)
		snapshotAsDBState(base)

		// ID change should tirgger a delete/update
		changedLine := base[1].Children.GetByID("2.1")
		changedLine.ID = ""
		changedLine.Description = lo.ToPtr("2.3")

		discounts := changedLine.Discounts.Get()
		discounts[0].Description = lo.ToPtr("D2.1.3")
		changedLine.Discounts = billingentity.NewLineDiscounts(discounts)

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			AffectedLineIDs: []string{"2"},
			ChildrenDiff: &lineDiffExpectation{
				LineBase: idDiff{
					ToDelete: []string{"2.1"},
					ToCreate: []string{"2.3"},
				},
				FlatFee: idDiff{
					ToCreate: []string{"2.3"},
				},
				Discounts: idDiff{
					// The discount gets deleted + created
					ToCreate: []string{"D2.1.3"},
					ToDelete: []string{"D2.1.1"},
				},
			},
		}, lineDiff)
	})

	// Discount handling
	t.Run("existing line hierarchy, one discount is deleted", func(t *testing.T) {
		base := cloneLines(template)
		snapshotAsDBState(base)

		base[1].Children.GetByID("2.1").Discounts = billingentity.NewLineDiscounts(nil)

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			AffectedLineIDs: []string{"2", "2.1"},
			ChildrenDiff: &lineDiffExpectation{
				Discounts: idDiff{
					ToDelete: []string{"D2.1.1"},
				},
			},
		}, lineDiff)
	})

	t.Run("existing line hierarchy, one discount is changed", func(t *testing.T) {
		base := cloneLines(template)
		snapshotAsDBState(base)

		discounts := base[1].Children.GetByID("2.1").Discounts.Get()
		discounts[0].Amount = alpacadecimal.NewFromFloat(10)
		base[1].Children.GetByID("2.1").Discounts = billingentity.NewLineDiscounts(discounts)

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			AffectedLineIDs: []string{"2", "2.1"},
			ChildrenDiff: &lineDiffExpectation{
				Discounts: idDiff{
					ToUpdate: []string{"D2.1.1"},
				},
			},
		}, lineDiff)
	})

	t.Run("existing line hierarchy, one discount is added/old one is removed", func(t *testing.T) {
		base := cloneLines(template)
		snapshotAsDBState(base)

		discounts := base[1].Children.GetByID("2.1").Discounts.Get()
		discounts[0].ID = ""
		discounts[0].Description = lo.ToPtr("D2.1.2")
		base[1].Children.GetByID("2.1").Discounts = billingentity.NewLineDiscounts(discounts)

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			AffectedLineIDs: []string{"2", "2.1"},
			ChildrenDiff: &lineDiffExpectation{
				Discounts: idDiff{
					ToCreate: []string{"D2.1.2"},
					ToDelete: []string{"D2.1.1"},
				},
			},
		}, lineDiff)
	})
}

func mapLinesToIDs(lines []*billingentity.Line) []string {
	return lo.Map(lines, func(line *billingentity.Line, _ int) string {
		// Use description as ID if it's set, so that we can predict the new line's ID for new
		// line testcases
		if line.Description != nil {
			return *line.Description
		}
		return line.ID
	})
}

func mapLineDiffToIDs(in diff[*billingentity.Line]) idDiff {
	return idDiff{
		ToCreate: mapLinesToIDs(in.ToCreate),
		ToUpdate: mapLinesToIDs(in.ToUpdate),
		ToDelete: mapLinesToIDs(in.ToDelete),
	}
}

func mapLineDiscountsToIDs(discounts []discountWithLine) []string {
	return lo.Map(discounts, func(d discountWithLine, _ int) string {
		if d.Discount.Description != nil {
			return *d.Discount.Description
		}

		return d.Discount.ID
	})
}

func mapLineDiscountDiffToIDs(in diff[discountWithLine]) idDiff {
	return idDiff{
		ToCreate: mapLineDiscountsToIDs(in.ToCreate),
		ToUpdate: mapLineDiscountsToIDs(in.ToUpdate),
		ToDelete: mapLineDiscountsToIDs(in.ToDelete),
	}
}

func requireIdDiffMatches(t *testing.T, a, b idDiff, msgAndArgs ...interface{}) {
	t.Helper()

	require.ElementsMatch(t, a.ToCreate, b.ToCreate, msgAndArgs...)
	require.ElementsMatch(t, a.ToUpdate, b.ToUpdate, msgAndArgs...)
	require.ElementsMatch(t, a.ToDelete, b.ToDelete, msgAndArgs...)
}

func requireDiffWithoutChildren(t *testing.T, expected lineDiffExpectation, actual *invoiceLineDiff, prefix string) {
	t.Helper()

	requireIdDiffMatches(t, expected.LineBase, mapLineDiffToIDs(actual.LineBase), prefix+": LineBase")
	requireIdDiffMatches(t, expected.FlatFee, mapLineDiffToIDs(actual.FlatFee), prefix+": FlatFee")
	requireIdDiffMatches(t, expected.UsageBased, mapLineDiffToIDs(actual.UsageBased), prefix+": UsageBased")

	requireIdDiffMatches(t, expected.Discounts, mapLineDiscountDiffToIDs(actual.Discounts), prefix+": Discounts")
}

func requireDiff(t *testing.T, expected lineDiffExpectation, actual *invoiceLineDiff) {
	t.Helper()
	requireDiffWithoutChildren(t, expected, actual, "root diff")
	require.ElementsMatch(t, expected.AffectedLineIDs, actual.AffectedLineIDs.AsSlice(), "affectedLineIDs")

	childrenExpectation := expected.ChildrenDiff
	if childrenExpectation == nil {
		childrenExpectation = &lineDiffExpectation{}
	}

	requireDiffWithoutChildren(t, *childrenExpectation, actual.ChildrenDiff, "children diff")
}

func cloneLines(lines []*billingentity.Line) []*billingentity.Line {
	return fixParentReferences(
		lo.Map(lines, func(line *billingentity.Line, _ int) *billingentity.Line {
			return line.Clone()
		}),
	)
}

func fixParentReferences(lines []*billingentity.Line) []*billingentity.Line {
	for _, line := range lines {
		for _, child := range line.Children.Get() {
			child.ParentLineID = lo.ToPtr(line.ID)
			child.ParentLine = line
		}
	}

	return lines
}

// snapshotAsDBState saves the current state of the lines as if they were in the database
func snapshotAsDBState(lines []*billingentity.Line) {
	for _, line := range lines {
		line.SaveDBSnapshot()
	}
}
