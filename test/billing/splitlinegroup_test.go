package billing

import (
	"context"
	"errors"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/invoicing/legacy/splitlinegroup"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *BillingAdapterTestSuite) createPersistedSplitLineHierarchy(namespace string) (*billing.StandardInvoice, splitlinegroup.SplitLineGroup, *billing.StandardLine) {
	s.T().Helper()

	ctx := s.T().Context()
	period := timeutil.ClosedPeriod{
		From: time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2023, 1, 20, 0, 0, 0, 0, time.UTC),
	}
	invoice := s.setupInvoice(ctx, namespace)
	group, err := s.SplitLineGroupService.CreateSplitLineGroup(ctx, splitlinegroup.SplitLineGroupCreate{
		Namespace: namespace,
		SplitLineGroupMutableFields: splitlinegroup.SplitLineGroupMutableFields{
			Name:          "namespace isolation group",
			ServicePeriod: period,
		},
		Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(1),
		}),
		Currency: currencyx.FiatCode(currency.USD),
	})
	require.NoError(s.T(), err)

	line := newLine(newLineInput{
		Namespace: namespace,
		Period:    period,
		Invoice:   invoice,
		Name:      "namespace isolation line",
	})
	line.SplitLineGroupID = lo.ToPtr(group.ID)

	lines, err := s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
		Namespace: namespace,
		Lines:     []*billing.StandardLine{line},
		InvoiceID: invoice.ID,
	})
	require.NoError(s.T(), err)
	require.Len(s.T(), lines, 1)

	return invoice, group, lines[0]
}

func (s *BillingAdapterTestSuite) TestSplitLineGroupServiceLifecycle() {
	// given: persisted group history with a live invoice line.
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("split-group-lifecycle")
	invoice, group, line := s.createPersistedSplitLineHierarchy(namespace)
	groupID := group.NamespacedID

	// when: a mutable update is validated against the group's existing price.
	invalidUpdate := group.ToUpdate()
	invalidUpdate.Name = ""
	_, err := s.SplitLineGroupService.UpdateSplitLineGroup(ctx, invalidUpdate)
	s.Require().Error(err)
	validUpdate := group.ToUpdate()
	validUpdate.Name = "updated group"
	updated, err := s.SplitLineGroupService.UpdateSplitLineGroup(ctx, validUpdate)
	s.Require().NoError(err)
	s.Equal("updated group", updated.Name)
	s.True(group.Price.Equal(updated.Price))

	// then: reads hide missing and cross-namespace groups with the same not-found contract.
	for _, missingID := range []models.NamespacedID{
		{Namespace: namespace, ID: ulid.Make().String()},
		{Namespace: s.GetUniqueNamespace("foreign-group"), ID: group.ID},
	} {
		_, err := s.SplitLineGroupService.GetSplitLineGroup(ctx, missingID)
		s.ErrorAs(err, new(billing.NotFoundError))
		err = s.SplitLineGroupService.DeleteSplitLineGroup(ctx, missingID)
		s.ErrorAs(err, new(billing.NotFoundError))
		_, err = s.SplitLineGroupService.UpdateSplitLineGroup(ctx, splitlinegroup.SplitLineGroupUpdate{
			NamespacedID:                missingID,
			SplitLineGroupMutableFields: validUpdate.SplitLineGroupMutableFields,
		})
		s.ErrorAs(err, new(billing.NotFoundError))
	}

	// when: deletion is attempted before and after deleting the invoice line.
	err = s.SplitLineGroupService.DeleteSplitLineGroup(ctx, groupID)
	s.Require().ErrorContains(err, "child lines must be deleted")
	line.DeletedAt = lo.ToPtr(time.Now())
	_, err = s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
		Namespace: namespace,
		InvoiceID: invoice.ID,
		Lines:     billing.StandardLines{line},
	})
	s.Require().NoError(err)
	err = s.SplitLineGroupService.DeleteSplitLineGroup(ctx, groupID)

	// then: historical deleted children permit deletion, and the group is gone.
	s.Require().NoError(err)
	_, err = s.SplitLineGroupService.GetSplitLineGroup(ctx, groupID)
	s.ErrorAs(err, new(billing.NotFoundError))
}

func (s *BillingAdapterTestSuite) TestSplitLineGroupDeletionPreservesAlreadyDeletedGuard() {
	// given: an already-marked-deleted group that still has a live child.
	ctx := s.T().Context()
	_, group, _ := s.createPersistedSplitLineHierarchy(s.GetUniqueNamespace("deleted-group"))
	err := s.DBClient.BillingInvoiceSplitLineGroup.UpdateOneID(group.ID).SetDeletedAt(time.Now()).Exec(ctx)
	s.Require().NoError(err)

	// when: service deletion is retried.
	err = s.SplitLineGroupService.DeleteSplitLineGroup(ctx, group.NamespacedID)

	// then: the existing early return leaves the group and live child intact.
	s.Require().NoError(err)
	hierarchy, err := s.SplitLineGroupService.GetSplitLineGroup(ctx, group.NamespacedID)
	s.Require().NoError(err)
	s.Require().NotNil(hierarchy.Group.DeletedAt)
	s.Require().Len(hierarchy.Lines, 1)
	s.Nil(hierarchy.Lines[0].DeletedAt)
}

func (s *BillingAdapterTestSuite) TestSplitLineGroupCreationRollsBackWithInvoiceWrite() {
	// given: a gathering line that will create its first split group under the customer lock.
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("split-group-create-rollback")
	invoice := s.setupInvoice(ctx, namespace)
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	line := billing.GatheringLine{GatheringLineBase: billing.GatheringLineBase{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{Namespace: namespace, Name: "split usage"}),
		Engine:          billing.LineEngineTypeInvoice,
		ManagedBy:       billing.ManuallyManagedLine,
		InvoiceID:       invoice.ID,
		Currency:        invoice.Currency,
		ServicePeriod:   period,
		InvoiceAt:       period.To,
		FeatureKey:      "test",
		Price:           *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
	}}
	var groupID models.NamespacedID

	// when: splitting succeeds, but the following invoice-line write violates its invoice reference.
	err := s.BillingService.WithLock(ctx, invoice.GetCustomerID(), func(ctx context.Context) error {
		split, err := s.LegacyBillingLineEngine.SplitGatheringLine(ctx, billing.SplitGatheringLineInput{
			Line:    line,
			SplitAt: period.From.Add(period.Duration() / 2),
		})
		if err != nil {
			return err
		}

		groupID = models.NamespacedID{Namespace: namespace, ID: *split.PreSplitAtLine.SplitLineGroupID}
		hierarchy, err := s.SplitLineGroupService.GetSplitLineGroup(ctx, groupID)
		s.Require().NoError(err)
		s.Equal(line.Name, hierarchy.Group.Name)
		invalidLine := newLine(newLineInput{Namespace: namespace, Invoice: invoice, Name: "failed invoice write", Period: period})
		invalidLine.InvoiceID = ulid.Make().String()
		invalidLine.SplitLineGroupID = &groupID.ID
		_, err = s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
			Namespace: namespace,
			InvoiceID: invalidLine.InvoiceID,
			Lines:     billing.StandardLines{invalidLine},
		})
		s.Require().ErrorContains(err, "foreign key")

		return err
	})

	// then: a separate read cannot find the group created by the failed billing attempt.
	s.Require().Error(err)
	s.Require().NotEmpty(groupID.ID)
	_, err = s.SplitLineGroupService.GetSplitLineGroup(ctx, groupID)
	s.ErrorAs(err, new(billing.NotFoundError))
}

func (s *BillingAdapterTestSuite) TestSplitLineGroupMutationsShareBillingTransaction() {
	// given: a group and its live invoice line committed before this attempt.
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("split-group-mutation-rollback")
	invoice, group, line := s.createPersistedSplitLineHierarchy(namespace)
	failedAttempt := errors.New("later reconciliation write failed")

	// when: invoice deletion, group update and group deletion succeed before a later failure.
	err := s.BillingService.WithLock(ctx, invoice.GetCustomerID(), func(ctx context.Context) error {
		update := group.ToUpdate()
		update.Name = "uncommitted name"
		_, err := s.SplitLineGroupService.UpdateSplitLineGroup(ctx, update)
		if err != nil {
			return err
		}

		hierarchy, err := s.SplitLineGroupService.GetSplitLineGroup(ctx, group.NamespacedID)
		s.Require().NoError(err)
		s.Equal(update.Name, hierarchy.Group.Name)
		s.Require().Nil(hierarchy.Lines[0].DeletedAt)
		deletedLine, err := line.Clone()
		s.Require().NoError(err)
		deletedLine.DeletedAt = lo.ToPtr(time.Now())
		_, err = s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
			Namespace: namespace,
			InvoiceID: invoice.ID,
			Lines:     billing.StandardLines{deletedLine},
		})
		if err != nil {
			return err
		}

		// Deletion must read the line's current transaction state, not the earlier live history.
		if err := s.SplitLineGroupService.DeleteSplitLineGroup(ctx, group.NamespacedID); err != nil {
			return err
		}

		return failedAttempt
	})

	// then: both group and invoice-line history retain their previously committed state.
	s.Require().ErrorIs(err, failedAttempt)
	hierarchy, err := s.SplitLineGroupService.GetSplitLineGroup(ctx, group.NamespacedID)
	s.Require().NoError(err)
	s.Equal(group.Name, hierarchy.Group.Name)
	s.Require().Len(hierarchy.Lines, 1)
	s.Nil(hierarchy.Lines[0].DeletedAt)
}
