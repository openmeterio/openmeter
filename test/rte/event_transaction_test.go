package rte_test

import (
	"context"
	"errors"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/eventoutbox"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *RTEIntegrationSuite) TestCalculationUpdateTransactions() {
	// Given both real calculators sharing the production outbox and state database.
	ctx := s.T().Context()
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()
	ns := s.GetUniqueNamespace("rte-events")
	customer := s.CreateLedgerBackedCustomer(ns, "rte-events")
	feature := s.SetupApiRequestsTotalFeature(ctx, ns)
	defer feature.Cleanup()
	ent, err := s.Entitlements.Entitlement.CreateCustomerEntitlement(ctx, entitlement.CreateCustomerEntitlementInput{
		CustomerID: customer.GetID(),
		Entitlement: entitlement.CreateEntitlementInputs{
			FeatureID:       &feature.Feature.ID,
			EntitlementType: entitlement.EntitlementTypeMetered,
			IssueAfterReset: lo.ToPtr(float64(100)),
			UsagePeriod:     lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{Anchor: start, Interval: timeutil.RecurrencePeriodMonth})),
		},
	})
	s.Require().NoError(err)
	pipeline := s.newEventPipeline()
	creditKey := creditbalance.Key{CustomerID: customer.GetID(), Currency: currencies.NewCurrencyReference("USD")}
	meteredKey := meteredentitlement.Key{EntitlementID: models.NamespacedID{Namespace: ns, ID: ent.ID}}
	creator := enttx.NewCreator(s.DBClient)
	rollback := errors.New("outer operation failed")

	for _, calculation := range []struct {
		name      string
		calculate func(context.Context) error
		get       func(context.Context) (time.Time, error)
	}{
		{
			name: "credit balance",
			calculate: func(ctx context.Context) error {
				_, err := pipeline.service.CreditBalances().Calculate(ctx, creditKey)
				return err
			},
			get: func(ctx context.Context) (time.Time, error) {
				result, err := pipeline.service.CreditBalances().Get(ctx, creditKey)
				return result.CalculatedAt, err
			},
		},
		{
			name: "metered entitlement",
			calculate: func(ctx context.Context) error {
				_, err := pipeline.service.MeteredEntitlements().Calculate(ctx, meteredKey)
				return err
			},
			get: func(ctx context.Context) (time.Time, error) {
				result, err := pipeline.service.MeteredEntitlements().Get(ctx, meteredKey)
				return result.CalculatedAt, err
			},
		},
	} {
		s.Run(calculation.name, func() {
			clock.FreezeTime(start)
			defer clock.UnFreeze()

			// When an outer operation rolls back after state and its event were written.
			err := transaction.RunWithNoValue(ctx, creator, func(ctx context.Context) error {
				s.Require().NoError(calculation.calculate(ctx))
				tx, err := entutils.GetDriverFromContext(ctx)
				s.Require().NoError(err)
				client := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()
				count, err := client.EventOutbox.Query().Where(eventoutbox.TransactionIDEQ(tx.ID())).Count(ctx)
				s.Require().NoError(err)
				s.Equal(1, count)

				return rollback
			})
			s.Require().ErrorIs(err, rollback)
			_, err = calculation.get(ctx)
			s.True(models.IsGenericNotFoundError(err))
			count, err := s.DBClient.EventOutbox.Query().Count(ctx)
			s.Require().NoError(err)
			s.Zero(count)

			// Then equal and older attempts keep the stored success and enqueue no updates.
			err = transaction.RunWithNoValue(ctx, creator, func(ctx context.Context) error {
				newer := start.Add(10 * time.Minute)
				clock.FreezeTime(newer)
				defer clock.UnFreeze()
				s.Require().NoError(calculation.calculate(ctx))
				s.Require().NoError(calculation.calculate(ctx))
				clock.FreezeTime(start.Add(5 * time.Minute))
				defer clock.UnFreeze()
				s.Require().NoError(calculation.calculate(ctx))
				storedAt, err := calculation.get(ctx)
				s.Require().NoError(err)
				s.True(storedAt.Equal(newer))
				tx, err := entutils.GetDriverFromContext(ctx)
				s.Require().NoError(err)
				client := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()
				count, err := client.EventOutbox.Query().Where(eventoutbox.TransactionIDEQ(tx.ID())).Count(ctx)
				s.Require().NoError(err)
				s.Equal(1, count)

				return rollback
			})
			s.Require().ErrorIs(err, rollback)

			// An outbox insertion failure must also roll back the candidate state.
			clock.FreezeTime(start.Add(20 * time.Minute))
			defer clock.UnFreeze()
			_, err = s.DBClient.ExecContext(ctx, "ALTER TABLE event_outboxes ADD CONSTRAINT reject_rte_test_event CHECK (topic <> 'rte-system-events')")
			s.Require().NoError(err)
			defer func() {
				_, err := s.DBClient.ExecContext(ctx, "ALTER TABLE event_outboxes DROP CONSTRAINT reject_rte_test_event")
				s.Require().NoError(err)
			}()
			err = calculation.calculate(ctx)
			s.Require().ErrorContains(err, "enqueue system event")
			_, err = calculation.get(ctx)
			s.True(models.IsGenericNotFoundError(err))
		})
	}
}
