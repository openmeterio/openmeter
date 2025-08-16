package billingadapter

import (
	"fmt"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

type idDiff = diff[string]

type lineDiffExpectation struct {
	LineBase   idDiff
	FlatFee    idDiff
	UsageBased idDiff

	AmountDiscounts idDiff

	AffectedLineIDs []string
	ChildrenDiff    *lineDiffExpectation
}

func TestInvoiceLineDiffing(t *testing.T) {
	template := []*billing.Line{
		{
			LineBase: billing.LineBase{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					ID: "1",
				}),
				Type: billing.InvoiceLineTypeFee,
			},
			FlatFee: &billing.FlatFeeLine{},
		},
		{
			LineBase: billing.LineBase{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					ID: "2",
				}),
				Type: billing.InvoiceLineTypeUsageBased,
			},
			UsageBased: &billing.UsageBasedLine{},
			Children: billing.NewLineChildren([]*billing.Line{
				{
					LineBase: billing.LineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							ID: "2.1",
						}),
						Type: billing.InvoiceLineTypeFee,
					},
					FlatFee:   &billing.FlatFeeLine{},
					Discounts: newAmountDiscountsWithIDs("D2.1.1"),
				},
				{
					LineBase: billing.LineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							ID: "2.2",
						}),
						Type: billing.InvoiceLineTypeFee,
					},
					FlatFee: &billing.FlatFeeLine{},
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
				AmountDiscounts: idDiff{
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
				AmountDiscounts: idDiff{
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

		changedLine.Discounts.Amount[0].ID = "D2.1.3"

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
				AmountDiscounts: idDiff{
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

		base[1].Children.GetByID("2.1").Discounts.Amount = nil

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			AffectedLineIDs: []string{"2", "2.1"},
			ChildrenDiff: &lineDiffExpectation{
				AmountDiscounts: idDiff{
					ToDelete: []string{"D2.1.1"},
				},
			},
		}, lineDiff)
	})

	t.Run("existing line hierarchy, one discount is changed", func(t *testing.T) {
		base := cloneLines(template)
		snapshotAsDBState(base)

		base[1].Children.GetByID("2.1").Discounts.Amount[0].Amount = alpacadecimal.NewFromFloat(20)

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			AffectedLineIDs: []string{"2", "2.1"},
			ChildrenDiff: &lineDiffExpectation{
				AmountDiscounts: idDiff{
					ToUpdate: []string{"D2.1.1"},
				},
			},
		}, lineDiff)
	})

	t.Run("existing line hierarchy, one discount is added/old one is removed", func(t *testing.T) {
		base := cloneLines(template)
		snapshotAsDBState(base)

		discounts := base[1].Children.GetByID("2.1").Discounts.Amount

		discounts[0].ID = ""
		discounts[0].Description = lo.ToPtr("D2.1.2")

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			AffectedLineIDs: []string{"2", "2.1"},
			ChildrenDiff: &lineDiffExpectation{
				AmountDiscounts: idDiff{
					ToCreate: []string{"D2.1.2"},
					ToDelete: []string{"D2.1.1"},
				},
			},
		}, lineDiff)
	})

	// DeletedAt handling
	t.Run("support for detailed lines being deleted using deletedAt", func(t *testing.T) {
		base := cloneLines(template)
		snapshotAsDBState(base)

		base[1].Children.GetByID("2.1").DeletedAt = lo.ToPtr(clock.Now())

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			AffectedLineIDs: []string{"2"},
			ChildrenDiff: &lineDiffExpectation{
				LineBase: idDiff{
					ToDelete: []string{"2.1"},
				},
				AmountDiscounts: idDiff{
					ToDelete: []string{"D2.1.1"},
				},
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
			LineBase: idDiff{
				ToDelete: []string{"2"},
			},
			ChildrenDiff: &lineDiffExpectation{
				LineBase: idDiff{
					ToDelete: []string{"2.1", "2.2"},
				},
				AmountDiscounts: idDiff{
					ToDelete: []string{"D2.1.1"},
				},
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
			LineBase: idDiff{
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

	t.Run("deleted, changed lines are not triggering updates", func(t *testing.T) {
		base := cloneLines(template)
		base[1].DeletedAt = lo.ToPtr(clock.Now())
		base[1].Children.GetByID("2.1").DeletedAt = lo.ToPtr(clock.Now())

		snapshotAsDBState(base)
		base[1].DeletedAt = nil
		base[1].Children.GetByID("2.1").DeletedAt = nil

		lineDiff, err := diffInvoiceLines(base)
		require.NoError(t, err)

		requireDiff(t, lineDiffExpectation{
			LineBase: idDiff{
				ToUpdate: []string{"2"},
			},
			ChildrenDiff: &lineDiffExpectation{
				LineBase: idDiff{
					ToUpdate: []string{"2.1"},
				},
			},
		}, lineDiff)
	})
}

func mapLinesToIDs(lines []*billing.Line) []string {
	return lo.Map(lines, func(line *billing.Line, _ int) string {
		// Use description as ID if it's set, so that we can predict the new line's ID for new
		// line testcases
		if line.Description != nil {
			return *line.Description
		}
		return line.ID
	})
}

func mapLineDiffToIDs(in diff[*billing.Line]) idDiff {
	return idDiff{
		ToCreate: mapLinesToIDs(in.ToCreate),
		ToUpdate: mapLinesToIDs(in.ToUpdate),
		ToDelete: mapLinesToIDs(in.ToDelete),
	}
}

func mapLineDiscountsToIDs(t *testing.T, discounts []withParent[billing.AmountLineDiscountManaged, *billing.Line]) []string {
	return lo.Map(discounts, func(d withParent[billing.AmountLineDiscountManaged, *billing.Line], _ int) string {
		if d.Discount.Description != nil {
			return *d.Discount.Description
		}

		return d.Discount.GetID()
	})
}

func mapLineDiscountDiffToIDs(t *testing.T, in diff[withParent[billing.AmountLineDiscountManaged, *billing.Line]]) idDiff {
	return idDiff{
		ToCreate: mapLineDiscountsToIDs(t, in.ToCreate),
		ToUpdate: mapLineDiscountsToIDs(t, in.ToUpdate),
		ToDelete: mapLineDiscountsToIDs(t, in.ToDelete),
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

func requireIdDiffMatches(t *testing.T, a, b idDiff, msgAndArgs ...interface{}) {
	t.Helper()

	require.ElementsMatch(t, a.ToCreate, b.ToCreate, msgPrefix("ToCreate", msgAndArgs...))
	require.ElementsMatch(t, a.ToUpdate, b.ToUpdate, msgPrefix("ToUpdate", msgAndArgs...))
	require.ElementsMatch(t, a.ToDelete, b.ToDelete, msgPrefix("ToDelete", msgAndArgs...))
}

func requireDiffWithoutChildren(t *testing.T, expected lineDiffExpectation, actual *invoiceLineDiff, prefix string) {
	t.Helper()

	requireIdDiffMatches(t, expected.LineBase, mapLineDiffToIDs(actual.LineBase), prefix+": LineBase")
	requireIdDiffMatches(t, expected.FlatFee, mapLineDiffToIDs(actual.FlatFee), prefix+": FlatFee")
	requireIdDiffMatches(t, expected.UsageBased, mapLineDiffToIDs(actual.UsageBased), prefix+": UsageBased")

	requireIdDiffMatches(t, expected.AmountDiscounts, mapLineDiscountDiffToIDs(t, actual.AmountDiscounts), prefix+": AmountDiscounts")
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

func cloneLines(lines []*billing.Line) []*billing.Line {
	return fixParentReferences(
		lo.Map(lines, func(line *billing.Line, _ int) *billing.Line {
			return line.Clone()
		}),
	)
}

func fixParentReferences(lines []*billing.Line) []*billing.Line {
	for _, line := range lines {
		for _, child := range line.Children.OrEmpty() {
			child.ParentLineID = lo.ToPtr(line.ID)
			child.ParentLine = line
		}
	}

	return lines
}

// snapshotAsDBState saves the current state of the lines as if they were in the database
func snapshotAsDBState(lines []*billing.Line) {
	for _, line := range lines {
		line.SaveDBSnapshot()
	}
}

func newAmountDiscountsWithIDs(ids ...string) billing.LineDiscounts {
	return billing.LineDiscounts{
		Amount: lo.Map(ids, func(id string, _ int) billing.AmountLineDiscountManaged {
			return billing.AmountLineDiscountManaged{
				ManagedModelWithID: models.ManagedModelWithID{
					ID: id,
				},
				AmountLineDiscount: billing.AmountLineDiscount{
					Amount: alpacadecimal.NewFromFloat(10),
				},
			}
		}),
	}
}
