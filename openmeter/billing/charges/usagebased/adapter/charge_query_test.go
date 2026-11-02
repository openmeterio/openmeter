package adapter

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
)

func (s *DetailedLineAdapterSuite) TestRunQueriesIncludeNamespace() {
	ctx := s.T().Context()
	namespace := "usagebased-run-query-namespace"
	charge, run, _ := s.createChargeWithRun(namespace)

	var runQueries []string
	client := entdb.NewClient(
		entdb.Driver(s.testDB.EntDriver.Driver()),
		entdb.Debug(),
		entdb.Log(func(args ...any) {
			message := fmt.Sprint(args...)
			if strings.Contains(message, `FROM "charge_usage_based_runs" WHERE`) {
				_, predicate, _ := strings.Cut(message, " WHERE ")
				runQueries = append(runQueries, predicate)
			}
		}),
	)
	metaAdapter, err := metaadapter.New(metaadapter.Config{Client: client, Logger: slog.Default()})
	s.Require().NoError(err)
	reader, err := New(Config{Client: client, Logger: slog.Default(), MetaAdapter: metaAdapter})
	s.Require().NoError(err)

	s.Run("single charge", func() {
		// given: a charge with a persisted run; assert SQL because equivalent
		// results alone cannot detect the missing index-prefix predicate.
		runQueries = nil

		// when: realizations are loaded through the single-charge path.
		fetched, err := reader.GetByID(ctx, usagebased.GetByIDInput{
			ChargeID: charge.GetChargeID(),
			Expands:  meta.Expands{meta.ExpandRealizations},
		})
		s.Require().NoError(err)

		// then: the child query binds namespace as well as the charge ID.
		s.Require().Len(fetched.Realizations, 1)
		s.Equal(run.ID, fetched.Realizations[0].ID)
		s.Equal([]string{
			`("charge_usage_based_runs"."namespace" = $1 AND "charge_usage_based_runs"."deleted_at" IS NULL) AND "charge_usage_based_runs"."charge_id" IN ($2) args=[` + namespace + ` ` + charge.ID + `]`,
		}, runQueries)
	})

	s.Run("batch including deleted runs", func() {
		// given: a deleted run and an active run on another charge.
		_, err := s.adapter.UpdateRealizationRun(ctx, usagebased.UpdateRealizationRunInput{
			ID:        run.ID,
			DeletedAt: mo.Some(lo.ToPtr(time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC))),
		})
		s.Require().NoError(err)
		otherCharge, otherRun, _ := s.createChargeWithRun(namespace)
		runQueries = nil

		// when: the batch path explicitly includes deleted realizations.
		fetched, err := reader.GetByIDs(ctx, usagebased.GetByIDsInput{
			Namespace: namespace,
			IDs:       []string{charge.ID, otherCharge.ID},
			Expands:   meta.Expands{meta.ExpandRealizations, meta.ExpandDeletedRealizations},
		})
		s.Require().NoError(err)

		// then: deletion filtering is omitted, but namespace remains bound.
		s.Require().Len(fetched, 2)
		s.Require().Len(fetched[0].Realizations, 1)
		s.Require().Len(fetched[1].Realizations, 1)
		s.Equal(run.ID, fetched[0].Realizations[0].ID)
		s.Equal(otherRun.ID, fetched[1].Realizations[0].ID)
		s.Require().Len(runQueries, 1)
		s.Contains(runQueries[0], `"charge_usage_based_runs"."namespace" = $1 AND "charge_usage_based_runs"."charge_id" IN ($2, $3) args=[`+namespace+` `)
		s.Contains(runQueries[0], charge.ID)
		s.Contains(runQueries[0], otherCharge.ID)
	})
}
