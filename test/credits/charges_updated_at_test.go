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

	createCharge := func(reference string, serviceStart time.Time) meta.ChargeID {
		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.NewCreateChargeIntents(
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:          cust.GetID(),
					Currency:          USD,
					ServicePeriod:     timeutil.ClosedPeriod{From: serviceStart, To: serviceStart.AddDate(0, 1, 0)},
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

		return id
	}

	// given: two charges written before a sync checkpoint. Only the first one's
	// service period has started by the checkpoint.
	clock.FreezeTime(checkpoint.Add(-time.Hour))
	defer clock.UnFreeze()
	changed := createCharge("changed", checkpoint)
	createCharge("unchanged", checkpoint.AddDate(0, 1, 0))

	// when: advancing at the checkpoint updates only the started charge.
	clock.FreezeTime(checkpoint)
	defer clock.UnFreeze()

	_, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: cust.GetID()})
	s.Require().NoError(err)

	// then: only the updated charge matches. Its created_at is before the
	// checkpoint, so this can only match on updated_at.
	result, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Page:        pagination.NewPage(1, 10),
		Namespace:   ns,
		CustomerIDs: []string{cust.ID},
		ChargeTypes: []meta.ChargeType{meta.ChargeTypeUsageBased},
		UpdatedAt:   &filter.FilterTime{Gte: &checkpoint},
	})
	s.Require().NoError(err)
	s.Equal([]string{changed.ID}, lo.Map(result.Items, func(c charges.Charge, _ int) string {
		id, err := c.GetChargeID()
		s.Require().NoError(err)

		return id.ID
	}))
}
