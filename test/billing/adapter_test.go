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

	profile := s.ProvisionBillingProfile(ctx, ns)

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

	lineIn.Children.MustGet()[0].Discounts = billing.LineDiscounts{
		Amount: []billing.AmountLineDiscountManaged{
			{
				AmountLineDiscount: billing.AmountLineDiscount{
					Amount: alpacadecimal.NewFromFloat(10),
					LineDiscountBase: billing.LineDiscountBase{
						Description:            lo.ToPtr("Test Discount 1"),
						ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
						Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
					},
				},
			},
			{
				AmountLineDiscount: billing.AmountLineDiscount{
					Amount: alpacadecimal.NewFromFloat(20),
					LineDiscountBase: billing.LineDiscountBase{
						Description:            lo.ToPtr("Test Discount 2"),
						ChildUniqueReferenceID: lo.ToPtr("max-spend-multiline"),
						Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
					},
				},
			},
			{
				AmountLineDiscount: billing.AmountLineDiscount{
					Amount: alpacadecimal.NewFromFloat(30),
					LineDiscountBase: billing.LineDiscountBase{
						Description: lo.ToPtr(manualDiscountName),
						Reason: billing.NewDiscountReasonFrom(productcatalog.PercentageDiscount{
							Percentage: models.NewPercentage(10),
						}),
					},
				},
			},
		},
	}

	lines, err := s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
		Namespace: ns,
		Lines:     []*billing.Line{lineIn},
	})

	// Then the lines are persisted as expected
	require.NoError(s.T(), err)
	require.Len(s.T(), lines, 1)

	// Then the discounts are persisted as expected
	persistedDiscounts := lines[0].Children.MustGet()[0].Discounts
	require.Len(s.T(), persistedDiscounts.Amount, 3)
	require.Len(s.T(), persistedDiscounts.Usage, 0)

	// Remove the managed fields
	discountContents, err := persistedDiscounts.Amount.Mutate(func(discount billing.AmountLineDiscountManaged) (billing.AmountLineDiscountManaged, error) {
		discount.ManagedModelWithID = models.ManagedModelWithID{}
		return discount, nil
	})
	require.NoError(s.T(), err)

	inputDiscountContents := lo.Must(lineIn.Children.MustGet()[0].Discounts.Amount.Mutate(
		func(discount billing.AmountLineDiscountManaged) (billing.AmountLineDiscountManaged, error) {
			discount.ManagedModelWithID = models.ManagedModelWithID{}
			return discount, nil
		},
	))

	require.ElementsMatch(s.T(),
		discountContents,
		inputDiscountContents,
	)

	// Let's update the discounts
	childLine := lines[0].Children.MustGet()[0].Clone()

	// Let's find the existing manual discount's ID
	existingDiscountID := ""
	for _, discount := range persistedDiscounts.Amount {
		if discount.Description != nil && *discount.Description == manualDiscountName {
			existingDiscountID = discount.ID
			break
		}
	}
	require.NotEmpty(s.T(), existingDiscountID)

	childLine.Discounts = billing.LineDiscounts{
		Amount: []billing.AmountLineDiscountManaged{
			{
				AmountLineDiscount: billing.AmountLineDiscount{
					Amount: alpacadecimal.NewFromFloat(30),
					LineDiscountBase: billing.LineDiscountBase{
						Description:            lo.ToPtr("Test Discount 1 v2"),
						ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
						Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
					},
				},
			},
			// Maximum spend is deleted
			{
				ManagedModelWithID: models.ManagedModelWithID{
					ID: existingDiscountID,
				},
				AmountLineDiscount: billing.AmountLineDiscount{
					Amount: alpacadecimal.NewFromFloat(40),
					LineDiscountBase: billing.LineDiscountBase{
						Description: lo.ToPtr("Test Discount 3 - updated"),
						Reason: billing.NewDiscountReasonFrom(productcatalog.PercentageDiscount{
							Percentage: models.NewPercentage(10),
						}),
					},
				},
			},
			{
				AmountLineDiscount: billing.AmountLineDiscount{
					Amount: alpacadecimal.NewFromFloat(50),
					LineDiscountBase: billing.LineDiscountBase{
						Description: lo.ToPtr("Test Discount 4 - manual"),
						Reason: billing.NewDiscountReasonFrom(productcatalog.PercentageDiscount{
							Percentage: models.NewPercentage(20),
						}),
					},
				},
			},
		},
	}

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
	require.Len(s.T(), persistedDiscounts.Amount, 3)

	expectedChildLineDiscounts := childLine.Discounts
	// Line 0: we expect that the ID is set to the same value
	previousVersion, ok := previousVersionDiscounts.Amount.GetDiscountByChildUniqueReferenceID(billing.LineMaximumSpendReferenceID)
	require.True(s.T(), ok)

	currentVersion := s.findAmountDiscountByDescription(persistedDiscounts.Amount, "Test Discount 1 v2")
	require.True(s.T(), expectedChildLineDiscounts.Amount[0].ContentsEqual(currentVersion))
	require.Equal(s.T(), currentVersion.ManagedModelWithID, models.ManagedModelWithID{
		ID: previousVersion.GetID(),
		ManagedModel: models.ManagedModel{
			// CreatedAt is unchanged
			CreatedAt: previousVersion.CreatedAt,
			UpdatedAt: currentVersion.UpdatedAt,
		},
	})

	// Line 1: maximum spend with retained id
	previousVersion = s.findAmountDiscountByDescription(previousVersionDiscounts.Amount, "Test Discount 3 - manual")
	currentVersion = s.findAmountDiscountByDescription(persistedDiscounts.Amount, "Test Discount 3 - updated")
	require.True(s.T(), expectedChildLineDiscounts.Amount[1].ContentsEqual(currentVersion))
	require.Equal(s.T(), currentVersion.ManagedModelWithID, models.ManagedModelWithID{
		ID: previousVersion.GetID(),
		ManagedModel: models.ManagedModel{
			// CreatedAt is unchanged
			CreatedAt: previousVersion.CreatedAt,
			UpdatedAt: currentVersion.UpdatedAt,
		},
	})

	// Line 2: new discount
	currentVersion = s.findAmountDiscountByDescription(persistedDiscounts.Amount, "Test Discount 4 - manual")
	require.True(s.T(), expectedChildLineDiscounts.Amount[2].ContentsEqual(currentVersion))

	require.ElementsMatch(s.T(),
		lo.Must(
			expectedChildLineDiscounts.Amount.Mutate(
				func(discount billing.AmountLineDiscountManaged) (billing.AmountLineDiscountManaged, error) {
					discount.ManagedModelWithID = models.ManagedModelWithID{}
					return discount, nil
				},
			),
		),
		lo.Must(
			persistedDiscounts.Amount.Mutate(
				func(discount billing.AmountLineDiscountManaged) (billing.AmountLineDiscountManaged, error) {
					discount.ManagedModelWithID = models.ManagedModelWithID{}
					return discount, nil
				},
			),
		),
	)
}

func (s *BillingAdapterTestSuite) findAmountDiscountByDescription(discounts []billing.AmountLineDiscountManaged, description string) billing.AmountLineDiscountManaged {
	s.T().Helper()

	for _, discount := range discounts {
		if discount.Description != nil && *discount.Description == description {
			return discount
		}
	}

	s.T().Fatalf("discount not found: %s", description)
	return billing.AmountLineDiscountManaged{}
}
