package credits

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *RatingTestSuite) TestListChargesUpdatedAt() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("credits-list-charges-updated-at")
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), ns)

	cust := s.CreateLedgerBackedCustomer(ns, "list-charges-updated-at")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	checkpoint := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	}

	createCharge := func(reference string) string {
		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.NewCreateChargeIntents(
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:          cust.GetID(),
					Currency:          USD,
					ServicePeriod:     servicePeriod,
					SettlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
					Price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
					Name:              reference,
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: reference,
					FeatureKey:        apiRequestsTotal.Feature.Key,
				}),
			),
		})
		s.Require().NoError(err)
		s.Require().Len(created, 1)

		id, err := created[0].GetChargeID()
		s.Require().NoError(err)

		return id.ID
	}

	list := func(updatedAt filter.FilterTime) pagination.Result[charges.Charge] {
		result, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
			Page:        pagination.NewPage(1, 1),
			Namespace:   ns,
			CustomerIDs: []string{cust.ID},
			ChargeTypes: []meta.ChargeType{meta.ChargeTypeUsageBased},
			UpdatedAt:   &updatedAt,
		})
		s.Require().NoError(err)

		return result
	}

	ids := func(result pagination.Result[charges.Charge]) []string {
		return lo.Map(result.Items, func(c charges.Charge, _ int) string {
			id, err := c.GetChargeID()
			s.Require().NoError(err)

			return id.ID
		})
	}

	// given: a charge written before a sync checkpoint and one written at it.
	clock.FreezeTime(checkpoint.Add(-time.Hour))
	defer clock.UnFreeze()
	before := createCharge("before-checkpoint")

	clock.FreezeTime(checkpoint)
	atCheckpoint := createCharge("at-checkpoint")

	// when/then: gte includes the boundary, gt excludes it, and lt selects the
	// earlier charge; page totals count only the filtered rows.
	gte := list(filter.FilterTime{Gte: &checkpoint})
	s.Equal([]string{atCheckpoint}, ids(gte))
	s.Equal(1, gte.TotalCount)

	s.Empty(list(filter.FilterTime{Gt: &checkpoint}).Items)

	lt := list(filter.FilterTime{Lt: &checkpoint})
	s.Equal([]string{before}, ids(lt))
	s.Equal(1, lt.TotalCount)
}
