package billing

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type BillingAdapterTestSuite struct {
	BaseSuite
}

func TestBillingAdapter(t *testing.T) {
	suite.Run(t, new(BillingAdapterTestSuite))
}

func (s *BillingAdapterTestSuite) setupInvoice(ctx context.Context, ns string) *billing.Invoice {
	s.T().Helper()
	// Given we have a customer
	customerEntity, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: ns,

		CustomerMutate: customer.CustomerMutate{
			Name:         "Test Customer",
			PrimaryEmail: lo.ToPtr("test@test.com"),
			BillingAddress: &models.Address{
				Country: lo.ToPtr(models.CountryCode("US")),
			},
			Currency: lo.ToPtr(currencyx.Code(currency.USD)),
			UsageAttribution: customer.CustomerUsageAttribution{
				SubjectKeys: []string{"test"},
			},
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), customerEntity)
	require.NotEmpty(s.T(), customerEntity.ID)

	// Given we have a profile
	_ = s.InstallSandboxApp(s.T(), ns)

	minimalCreateProfileInput := MinimalCreateProfileInputTemplate
	minimalCreateProfileInput.Namespace = ns

	profile, err := s.BillingService.CreateProfile(ctx, minimalCreateProfileInput)

	require.NoError(s.T(), err)
	require.NotNil(s.T(), profile)

	// Given we have an invoice
	invoice, err := s.BillingAdapter.CreateInvoice(ctx, billing.CreateInvoiceAdapterInput{
		Namespace: ns,
		Customer:  *customerEntity,

		Number:   "INV-123",
		Currency: currencyx.Code(currency.USD),
		Status:   billing.InvoiceStatusGathering,

		Profile:  *profile,
		IssuedAt: time.Now(),

		Type: billing.InvoiceTypeStandard,
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), invoice)

	return &invoice
}

func (s *BillingAdapterTestSuite) TestLineSplitting() {
	ctx := context.Background()
	ns := "ns-adapter-line-split"

	period := billing.Period{
		Start: lo.Must(time.Parse(time.RFC3339, "2023-01-10T00:00:00Z")),
		End:   lo.Must(time.Parse(time.RFC3339, "2023-01-20T00:00:00Z")),
	}

	invoice := s.setupInvoice(ctx, ns)

	var parentLine *billing.Line

	s.Run("Create a parent line", func() {
		// When we create a line
		parentLineIn := &billing.Line{
			LineBase: billing.LineBase{
				Namespace: ns,

				Type:        billing.InvoiceLineTypeUsageBased,
				ManagedBy:   billing.ManuallyManagedLine,
				InvoiceID:   invoice.ID,
				Name:        "Test Line Parent",
				Description: lo.ToPtr("Test Line Description"),
				Currency:    currencyx.Code(currency.USD),

				Period:    period,
				InvoiceAt: period.End,

				Status: billing.InvoiceLineStatusValid,
			},
			UsageBased: &billing.UsageBasedLine{
				Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromFloat(1),
				}),
				FeatureKey: "test",
			},
		}
		lines, err := s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
			Namespace: ns,
			Lines: []*billing.Line{
				parentLineIn,
			},
		})

		// Then the call succeeds
		require.NoError(s.T(), err)
		require.Len(s.T(), lines, 1)
		parentLine = lines[0]
		require.NotEmpty(s.T(), parentLine.ID)
		require.NotEmpty(s.T(), parentLine.UsageBased.ConfigID)

		// The created line matches the input with some additional fields
		parentLineIn.ID = parentLine.ID
		parentLineIn.CreatedAt = parentLine.CreatedAt
		parentLineIn.UpdatedAt = parentLine.UpdatedAt
		parentLineIn.UsageBased.ConfigID = parentLine.UsageBased.ConfigID
		parentLineIn.Children = billing.NewLineChildren(nil)

		require.Equal(s.T(), parentLineIn, parentLine.WithoutDBState())

		// Then fetching the line should return the same line
		fetchedLine, err := s.BillingAdapter.ListInvoiceLines(ctx, billing.ListInvoiceLinesAdapterInput{
			Namespace: ns,

			LineIDs: []string{parentLine.ID},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), fetchedLine, 1)
		require.Equal(s.T(), parentLine, fetchedLine[0])
	})

	parentLineID := parentLine.ID

	// When we split the line, and submit the split as a single upsert, then the changes must be reflected in the database
	var splitPost *billing.Line
	s.Run("Split the line", func() {
		asOf := period.Start.Add(time.Hour)
		splitPre := parentLine.CloneWithoutDependencies()
		splitPre.Name = "Test Line Split 1"
		splitPre.Period = billing.Period{
			Start: period.Start,
			End:   asOf,
		}
		splitPre.InvoiceAt = asOf
		splitPre.ParentLineID = lo.ToPtr(parentLine.ID)

		splitPost = parentLine.CloneWithoutDependencies()
		splitPost.Name = "Test Line Split 2"
		splitPost.Period = billing.Period{
			Start: asOf,
			End:   period.End,
		}
		splitPost.InvoiceAt = period.End
		splitPost.ParentLineID = lo.ToPtr(parentLine.ID)

		parentLine.Status = billing.InvoiceLineStatusSplit
		parentLine.Description = nil
		// TODO[later]: this is only required until we don't support child line updates
		parentLine.Children = billing.LineChildren{}

		// TODO[later]: We should allow for partial child syncs instead of specifying all children
		lines, err := s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
			Namespace: ns,
			Lines: []*billing.Line{
				splitPre,
				splitPost,
				parentLine,
			},
		})

		// Then the call succeeds
		require.NoError(s.T(), err)
		require.Len(s.T(), lines, 3)

		parentLine.UpdatedAt = lines[2].UpdatedAt
		parentLine.Children = lines[2].Children
		require.Equal(s.T(), parentLine.WithoutDBState(), lines[2].WithoutDBState(), "parentMatches")

		// Then all split lines are owned by the original parent line
		require.Equal(s.T(), parentLineID, *splitPre.ParentLineID, "split#1, in")
		require.Equal(s.T(), parentLineID, *lines[0].ParentLineID, "split#1, out")
		require.Equal(s.T(), parentLineID, *splitPost.ParentLineID, "split#2, in")
		require.Equal(s.T(), parentLineID, *lines[1].ParentLineID, "split#3, out")

		splitPre = mergeDBFields(splitPre, lines[0])
		splitPre.ParentLine = lines[0].ParentLine
		splitPre.Children = billing.NewLineChildren(nil)
		splitPre.Discounts = nil
		require.Equal(s.T(), splitPre, lines[0].WithoutDBState(), "preMatches")

		splitPost = mergeDBFields(splitPost, lines[1])
		splitPost.ParentLine = lines[1].ParentLine
		splitPost.Children = billing.NewLineChildren(nil)
		splitPost.Discounts = nil
		require.Equal(s.T(), splitPost, lines[1].WithoutDBState(), "postMatches")

		splitPost = lines[1]
	})

	// When we do the second split, and submit the split as a single upsert, then the changes must be reflected in the database
	s.Run("Split the line again", func() {
		asOf := splitPost.Period.Start.Add(2 * time.Hour)
		splitPost.Period.End = asOf
		splitPost.InvoiceAt = asOf

		newLastLine := splitPost.CloneWithoutDependencies()
		newLastLine.Period = billing.Period{
			Start: asOf,
			End:   period.End,
		}
		newLastLine.InvoiceAt = period.End
		newLastLine.ParentLineID = lo.ToPtr(parentLineID)
		newLastLine.ParentLine = parentLine
		newLastLine.Name = "Test Line Split 3"

		// Let's upsert
		lines, err := s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
			Namespace: ns,
			Lines: []*billing.Line{
				splitPost,
				newLastLine,
			},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), lines, 2)

		// Then all split lines are owned by the original parent line
		require.Equal(s.T(), parentLineID, *splitPost.ParentLineID, "split#2, in")
		require.Equal(s.T(), parentLineID, *lines[0].ParentLineID, "split#2, out")
		require.Equal(s.T(), parentLineID, *newLastLine.ParentLineID, "split#3, in")
		require.Equal(s.T(), parentLineID, *lines[1].ParentLineID, "split#3, out")

		// Then the returned lines match the input
		splitPost.UpdatedAt = lines[0].UpdatedAt
		splitPost.ParentLine = splitPost.ParentLine.WithoutDBState()
		splitPost.ParentLine.Children = billing.LineChildren{}
		splitPost.ParentLine.CreatedAt = lines[0].ParentLine.CreatedAt
		splitPost.ParentLine.UpdatedAt = lines[0].ParentLine.UpdatedAt

		require.Equal(s.T(), splitPost.WithoutDBState(), lines[0].WithoutDBState())

		newLastLine = mergeDBFields(newLastLine, lines[1])
		newLastLine.ParentLine = nil
		newLastLine.Children = billing.NewLineChildren(nil)
		newLastLine.Discounts = nil
		lines[1].ParentLine = nil
		require.Equal(s.T(), newLastLine.WithoutDBState(), lines[1].WithoutDBState())
	})
}

func mergeDBFields(in *billing.Line, dbInput *billing.Line) *billing.Line {
	in.ID = dbInput.ID
	in.CreatedAt = dbInput.CreatedAt
	in.UpdatedAt = dbInput.UpdatedAt
	in.DeletedAt = dbInput.DeletedAt

	in.UsageBased.ConfigID = dbInput.UsageBased.ConfigID
	return in
}

type usageBasedLineInput struct {
	Namespace              string
	Period                 billing.Period
	Invoice                *billing.Invoice
	Name                   string
	ChildUniqueReferenceID string
	DetailedLines          mo.Option[[]usageBasedLineInput]
}

func newUsageBasedLine(in usageBasedLineInput) *billing.Line {
	out := &billing.Line{
		LineBase: billing.LineBase{
			Namespace: in.Namespace,

			Type:      billing.InvoiceLineTypeUsageBased,
			ManagedBy: billing.ManuallyManagedLine,
			InvoiceID: in.Invoice.ID,
			Name:      in.Name,
			Currency:  in.Invoice.Currency,

			Period:    in.Period,
			InvoiceAt: in.Period.End,

			Status:                 billing.InvoiceLineStatusValid,
			ChildUniqueReferenceID: lo.EmptyableToPtr(in.ChildUniqueReferenceID),
		},
		UsageBased: &billing.UsageBasedLine{
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(1),
			}),
			FeatureKey: "test",
		},
	}

	if in.DetailedLines.IsPresent() {
		// Make the line present, but empty (so that it's present even if DetailedLines is only present)
		out.Children = billing.LineChildren{}

		for _, d := range in.DetailedLines.OrEmpty() {
			line := newUsageBasedLine(d)
			line.ParentLineID = lo.ToPtr(out.ID)
			line.ParentLine = out
			line.Status = billing.InvoiceLineStatusDetailed
			line.ManagedBy = billing.SystemManagedLine
			line.Type = billing.InvoiceLineTypeFee
			line.FlatFee = &billing.FlatFeeLine{
				PerUnitAmount: alpacadecimal.NewFromFloat(100),
				Quantity:      alpacadecimal.NewFromFloat(1),
				PaymentTerm:   productcatalog.InArrearsPaymentTerm,
				Category:      billing.FlatFeeCategoryRegular,
			}

			out.Children.Append(line)
		}
	}

	return out
}

func (s *BillingAdapterTestSuite) TestDetailedLineHandling() {
	ctx := context.Background()
	ns := "ns-adapter-detailed-line"
	// Given we have an invoice

	period := billing.Period{
		Start: lo.Must(time.Parse(time.RFC3339, "2023-01-10T00:00:00Z")),
		End:   lo.Must(time.Parse(time.RFC3339, "2023-01-20T00:00:00Z")),
	}

	invoice := s.setupInvoice(ctx, ns)

	// When we create a line with detailed fields those get persisted
	linesIn := []*billing.Line{
		newUsageBasedLine(usageBasedLineInput{
			Namespace: ns,
			Period:    period,
			Invoice:   invoice,
			Name:      "Test Line 1",
			DetailedLines: mo.Some([]usageBasedLineInput{
				{
					Namespace:              ns,
					Period:                 period,
					Invoice:                invoice,
					Name:                   "Test Line 1.1",
					ChildUniqueReferenceID: "ref1",
				},
				{
					Namespace:              ns,
					Period:                 period,
					Invoice:                invoice,
					Name:                   "Test Line 1.2",
					ChildUniqueReferenceID: "ref2",
				},
			}),
		}),
		newUsageBasedLine(usageBasedLineInput{
			Namespace: ns,
			Period:    period,
			Invoice:   invoice,
			Name:      "Test Line 2",
			DetailedLines: mo.Some([]usageBasedLineInput{
				{
					Namespace:              ns,
					Period:                 period,
					Invoice:                invoice,
					Name:                   "Test Line 2.1",
					ChildUniqueReferenceID: "ref1",
				},
			}),
		}),
		newUsageBasedLine(usageBasedLineInput{
			Namespace: ns,
			Period:    period,
			Invoice:   invoice,
			Name:      "Test Line 3",
			DetailedLines: mo.Some([]usageBasedLineInput{
				{
					Namespace:              ns,
					Period:                 period,
					Invoice:                invoice,
					Name:                   "Test Line 3.1",
					ChildUniqueReferenceID: "ref1",
				},
			}),
		}),
	}

	lines, err := s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
		Namespace: ns,
		Lines:     linesIn,
	})
	require.NoError(s.T(), err)
	require.Len(s.T(), lines, 3)

	// Then the lines are persisted as expected
	// Line 1
	require.Equal(s.T(), linesIn[0].Name, lines[0].Name)
	require.Empty(s.T(), linesIn[0].Children.MustGet()[0].ID)
	require.Len(s.T(), lines[0].Children.MustGet(), 2)
	require.ElementsMatch(s.T(),
		getUniqReferenceNames(linesIn[0].Children.MustGet()),
		getUniqReferenceNames(lines[0].Children.MustGet()))
	require.ElementsMatch(s.T(),
		getLineNames(linesIn[0].Children.MustGet()),
		getLineNames(lines[0].Children.MustGet()))

	require.Equal(s.T(), linesIn[1].Name, lines[1].Name)
	require.Len(s.T(), lines[1].Children.MustGet(), 1)
	require.ElementsMatch(s.T(),
		getUniqReferenceNames(linesIn[1].Children.MustGet()),
		getUniqReferenceNames(lines[1].Children.MustGet()))
	require.ElementsMatch(s.T(),
		getLineNames(linesIn[1].Children.MustGet()),
		getLineNames(lines[1].Children.MustGet()))

	require.Equal(s.T(), linesIn[2].Name, lines[2].Name)
	require.True(s.T(), lines[2].Children.IsPresent())
	require.Len(s.T(), lines[2].Children.MustGet(), 1)

	// When we execute an upsert the detailed lines are updated, but not duplicated
	s.Run("Detailed line upserting", func() {
		unchangedDetailedLineUpdatedAt := lo.FindOrElse[*billing.Line](lines[0].Children.MustGet(),
			&billing.Line{},
			func(l *billing.Line) bool {
				return *l.ChildUniqueReferenceID == "ref1"
			},
		).UpdatedAt

		newLine := newUsageBasedLine(usageBasedLineInput{
			Namespace:              ns,
			Period:                 period,
			Invoice:                invoice,
			Name:                   "Test Line 1.3",
			ChildUniqueReferenceID: "ref3",
		})
		newLine.Status = billing.InvoiceLineStatusDetailed
		newLine.Type = billing.InvoiceLineTypeFee
		newLine.ManagedBy = billing.SystemManagedLine
		newLine.FlatFee = &billing.FlatFeeLine{
			PerUnitAmount: alpacadecimal.NewFromFloat(100),
			Quantity:      alpacadecimal.NewFromFloat(1),
			PaymentTerm:   productcatalog.InArrearsPaymentTerm,
			Category:      billing.FlatFeeCategoryRegular,
		}

		lineChildren := lines[0].Children.MustGet()
		lineChildren = append(lineChildren, newLine)
		lo.ForEach(lineChildren, func(l *billing.Line, _ int) {
			l.ParentLineID = lo.ToPtr(lines[0].ID)
		})

		childrenWithIDReuse, err := lines[0].ChildrenWithIDReuse(
			lineChildren,
		)
		require.NoError(s.T(), err)
		lines[0].Children = childrenWithIDReuse

		// Not set => should be ignored
		lines[1].Children = billing.LineChildren{}
		// Set to empty array => detailed lines should be deleted
		lines[2].Children = billing.NewLineChildren([]*billing.Line{})

		// When we persist the changes
		lines, err = s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
			Namespace: ns,
			Lines:     lines,
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), lines, 3)

		// Then the lines are persisted as expected
		require.Len(s.T(), lines[0].Children.MustGet(), 3)
		require.ElementsMatch(s.T(),
			getUniqReferenceNames(lineChildren),
			getUniqReferenceNames(lines[0].Children.MustGet()))

		require.Equal(s.T(), lo.CountBy(lines[0].Children.MustGet(), func(l *billing.Line) bool {
			return l.ID != ""
		}), 3, "all lines must have IDs set")

		// Then ref1 has not been changed
		require.Equal(s.T(), unchangedDetailedLineUpdatedAt, lo.FindOrElse(lines[0].Children.MustGet(),
			&billing.Line{},
			func(l *billing.Line) bool {
				return *l.ChildUniqueReferenceID == "ref1"
			}).UpdatedAt)

		require.Len(s.T(), lines[1].Children.MustGet(), 1)
		require.ElementsMatch(s.T(),
			[]string{"ref1"},
			getUniqReferenceNames(lines[1].Children.MustGet()))

		require.Len(s.T(), lines[2].Children.MustGet(), 0)
		require.True(s.T(), lines[2].Children.IsPresent())
	})

	// When we remove a detailed line, the line gets deleted
	s.Run("Detailed line update (still a removal case)", func() {
		detailedLines := lines[0].Children.MustGet()

		slices.SortFunc(detailedLines, func(a, b *billing.Line) int {
			return strings.Compare(*a.ChildUniqueReferenceID, *b.ChildUniqueReferenceID)
		})

		require.Len(s.T(), detailedLines, 3)
		require.Equal(s.T(), "ref1", *detailedLines[0].ChildUniqueReferenceID)

		// Replace the first detailed line with a new child
		newLine := newUsageBasedLine(usageBasedLineInput{
			Namespace:              ns,
			Period:                 period,
			Invoice:                invoice,
			Name:                   "Test Line 1.4",
			ChildUniqueReferenceID: "ref4",
		})
		newLine.Status = billing.InvoiceLineStatusDetailed
		newLine.ManagedBy = billing.SystemManagedLine
		newLine.Type = billing.InvoiceLineTypeFee
		newLine.FlatFee = &billing.FlatFeeLine{
			PerUnitAmount: alpacadecimal.NewFromFloat(100),
			Quantity:      alpacadecimal.NewFromFloat(1),
			PaymentTerm:   productcatalog.InArrearsPaymentTerm,
			Category:      billing.FlatFeeCategoryRegular,
		}
		detailedLines[0] = newLine

		childrenWithIDReuse, err := lines[0].ChildrenWithIDReuse(
			detailedLines,
		)
		require.NoError(s.T(), err)
		lines[0].Children = childrenWithIDReuse

		// When we persist the changes
		lines, err := s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
			Namespace: ns,
			Lines:     []*billing.Line{lines[0]},
		})

		// Then we only get three lines
		require.NoError(s.T(), err)
		require.Len(s.T(), lines, 1)
		require.Len(s.T(), lines[0].Children.MustGet(), 3)

		require.ElementsMatch(s.T(),
			getUniqReferenceNames(detailedLines),
			getUniqReferenceNames(lines[0].Children.MustGet()))
		require.ElementsMatch(s.T(),
			getLineNames(detailedLines),
			getLineNames(lines[0].Children.MustGet()))

		// When we query the line's children, we get the 4 lines, one is deleted
		lines, err = s.BillingAdapter.ListInvoiceLines(ctx, billing.ListInvoiceLinesAdapterInput{
			Namespace:      ns,
			ParentLineIDs:  []string{lines[0].ID},
			IncludeDeleted: true,
		})

		// Then we get the 4 lines
		require.NoError(s.T(), err)
		require.Len(s.T(), lines, 4)
		require.ElementsMatch(s.T(),
			[]string{"ref1", "ref2", "ref3", "ref4"},
			getUniqReferenceNames(lines))

		deleted, found := lo.Find(lines, func(l *billing.Line) bool {
			return l.DeletedAt != nil
		})
		require.True(s.T(), found)
		require.Equal(s.T(), "ref1", *deleted.ChildUniqueReferenceID)
	})
}

func getUniqReferenceNames(lines []*billing.Line) []string {
	return lo.Map(lines, func(l *billing.Line, _ int) string {
		return *l.ChildUniqueReferenceID
	})
}

func getLineNames(lines []*billing.Line) []string {
	return lo.Map(lines, func(l *billing.Line, _ int) string {
		return l.Name
	})
}

// TestDiscountHandling tests the handling of discounts in the billing adapter
// this specific scenario simulates the detailed line calculation specifically (that's why we are
// using the detailed line's discounts for testing)
func (s *BillingAdapterTestSuite) TestDiscountHandling() {
	ctx := context.Background()
	ns := "ns-adapter-discount-handling"
	// Given we have an invoice

	period := billing.Period{
		Start: lo.Must(time.Parse(time.RFC3339, "2023-01-10T00:00:00Z")),
		End:   lo.Must(time.Parse(time.RFC3339, "2023-01-20T00:00:00Z")),
	}

	invoice := s.setupInvoice(ctx, ns)

	// When we create a line with detailed fields those get persisted
	lineIn := newUsageBasedLine(usageBasedLineInput{
		Namespace: ns,
		Period:    period,
		Invoice:   invoice,
		Name:      "Test Line 1",

		DetailedLines: mo.Some([]usageBasedLineInput{
			{
				Namespace: ns,
				Period:    period,
				Invoice:   invoice,
				Name:      "Test Line 1.1",
				// Warning: this is required, as otherwise the line would be considered a new line
				ChildUniqueReferenceID: "ref1",
			},
		}),
	})

	manualDiscountName := "Test Discount 3 - manual"

	lineIn.Children.MustGet()[0].Discounts = billing.NewLineDiscounts(
		billing.NewLineDiscountFrom(billing.AmountLineDiscount{
			Amount: alpacadecimal.NewFromFloat(10),
			LineDiscountBase: billing.LineDiscountBase{
				Description:            lo.ToPtr("Test Discount 1"),
				ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
				Reason:                 billing.LineDiscountReasonMaximumSpend,
			},
		}),
		billing.NewLineDiscountFrom(billing.AmountLineDiscount{
			Amount: alpacadecimal.NewFromFloat(20),
			LineDiscountBase: billing.LineDiscountBase{
				Description:            lo.ToPtr("Test Discount 2"),
				ChildUniqueReferenceID: lo.ToPtr("max-spend-multiline"),
				Reason:                 billing.LineDiscountReasonMaximumSpend,
			},
		}),
		billing.NewLineDiscountFrom(billing.AmountLineDiscount{
			Amount: alpacadecimal.NewFromFloat(30),
			LineDiscountBase: billing.LineDiscountBase{
				Description: lo.ToPtr(manualDiscountName),
				Reason:      billing.LineDiscountReasonRatecardDiscount,
			},
		}),
	)

	lines, err := s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
		Namespace: ns,
		Lines:     []*billing.Line{lineIn},
	})

	// Then the lines are persisted as expected
	require.NoError(s.T(), err)
	require.Len(s.T(), lines, 1)

	// Then the discounts are persisted as expected
	persistedDiscounts := lines[0].Children.MustGet()[0].Discounts
	require.Len(s.T(), persistedDiscounts, 3)

	// Remove the managed fields
	discountContents, err := persistedDiscounts.Mutate(billing.SetDiscountMangedFields(billing.LineSetDiscountMangedFields{}))
	require.NoError(s.T(), err)

	require.ElementsMatch(s.T(), discountContents, lineIn.Children.MustGet()[0].Discounts)

	// Let's update the discounts
	childLine := lines[0].Children.MustGet()[0].Clone()
	childLine.Discounts = billing.NewLineDiscounts(
		billing.NewLineDiscountFrom(billing.AmountLineDiscount{
			Amount: alpacadecimal.NewFromFloat(30),
			LineDiscountBase: billing.LineDiscountBase{
				Description:            lo.ToPtr("Test Discount 1 v2"),
				ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
				Reason:                 billing.LineDiscountReasonMaximumSpend,
			},
		}),
		// Maximum spend is deleted
		billing.NewLineDiscountFrom(billing.AmountLineDiscountManaged{
			LineSetDiscountMangedFields: billing.LineSetDiscountMangedFields{
				ID: func() string {
					for _, discount := range persistedDiscounts {
						base, err := discount.AsDiscountBase()
						require.NoError(s.T(), err)

						if *base.Description == manualDiscountName {
							return base.ID
						}
					}

					s.T().Fatal("manual discount not found")
					return ""
				}(),
			},
			AmountLineDiscount: billing.AmountLineDiscount{
				Amount: alpacadecimal.NewFromFloat(40),
				LineDiscountBase: billing.LineDiscountBase{
					Description: lo.ToPtr("Test Discount 3 - updated"),
					Reason:      billing.LineDiscountReasonRatecardDiscount,
				},
			},
		}),
		billing.NewLineDiscountFrom(billing.AmountLineDiscount{
			Amount: alpacadecimal.NewFromFloat(50),
			LineDiscountBase: billing.LineDiscountBase{
				Description: lo.ToPtr("Test Discount 4 - manual"),
				Reason:      billing.LineDiscountReasonRatecardDiscount,
			},
		}),
	)

	updateLineIn := lines[0].Clone()
	childrenWithIDReuse, err := updateLineIn.ChildrenWithIDReuse(
		[]*billing.Line{childLine},
	)
	require.NoError(s.T(), err)
	updateLineIn.Children = childrenWithIDReuse

	updatedLines, err := s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
		Namespace: ns,
		Lines:     []*billing.Line{updateLineIn},
	})

	// Then the discounts are persisted as expected
	require.NoError(s.T(), err)
	require.Len(s.T(), updatedLines, 1)

	previousVersionDiscounts := persistedDiscounts
	persistedDiscounts = updatedLines[0].Children.MustGet()[0].Discounts
	require.Len(s.T(), persistedDiscounts, 3)

	expectedChildLineDiscounts := childLine.Discounts
	// Line 0: we expect that the ID is set to the same value
	previousVersion, ok := previousVersionDiscounts.GetDiscountByChildUniqueReferenceID(billing.LineMaximumSpendReferenceID)
	require.True(s.T(), ok)

	currentVersion := s.findDiscountByDescription(persistedDiscounts, "Test Discount 1 v2")
	require.True(s.T(), expectedChildLineDiscounts[0].ContentsEqual(currentVersion))
	require.Equal(s.T(), currentVersion.GetManagedFields(), billing.LineSetDiscountMangedFields{
		ID: previousVersion.GetID(),
		ManagedModel: models.ManagedModel{
			// CreatedAt is unchanged
			CreatedAt: previousVersion.GetManagedFields().CreatedAt,
			UpdatedAt: currentVersion.GetManagedFields().UpdatedAt,
		},
	})

	// Line 1: maximum spend with retained id
	previousVersion = s.findDiscountByDescription(previousVersionDiscounts, "Test Discount 3 - manual")
	currentVersion = s.findDiscountByDescription(persistedDiscounts, "Test Discount 3 - updated")
	require.True(s.T(), expectedChildLineDiscounts[1].ContentsEqual(currentVersion))
	require.Equal(s.T(), currentVersion.GetManagedFields(), billing.LineSetDiscountMangedFields{
		ID: previousVersion.GetID(),
		ManagedModel: models.ManagedModel{
			// CreatedAt is unchanged
			CreatedAt: previousVersion.GetManagedFields().CreatedAt,
			UpdatedAt: currentVersion.GetManagedFields().UpdatedAt,
		},
	})

	// Line 2: new discount
	currentVersion = s.findDiscountByDescription(persistedDiscounts, "Test Discount 4 - manual")
	require.True(s.T(), expectedChildLineDiscounts[2].ContentsEqual(currentVersion))

	require.ElementsMatch(s.T(),
		lo.Must(
			expectedChildLineDiscounts.Mutate(
				billing.SetDiscountMangedFields(billing.LineSetDiscountMangedFields{}),
			),
		),
		lo.Must(
			persistedDiscounts.Mutate(
				billing.SetDiscountMangedFields(billing.LineSetDiscountMangedFields{}),
			),
		),
	)
}

func (s *BillingAdapterTestSuite) findDiscountByDescription(discounts []billing.LineDiscount, description string) billing.LineDiscount {
	s.T().Helper()

	return lo.FindOrElse(
		discounts,
		billing.NewLineDiscountFrom(billing.AmountLineDiscount{
			LineDiscountBase: billing.LineDiscountBase{
				Description: lo.ToPtr("notfound"),
			},
		}),
		func(d billing.LineDiscount) bool {
			base, err := d.AsDiscountBase()
			require.NoError(s.T(), err)

			return base.Description != nil && *base.Description == description
		})
}
