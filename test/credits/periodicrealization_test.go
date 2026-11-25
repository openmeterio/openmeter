package credits

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing/periodicrealization"
	periodicrealizationadapter "github.com/openmeterio/openmeter/openmeter/billing/periodicrealization/adapter"
	periodicrealizationservice "github.com/openmeterio/openmeter/openmeter/billing/periodicrealization/service"
	rundb "github.com/openmeterio/openmeter/openmeter/ent/db/billingperiodicrealizationrun"
	omtestutils "github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

func TestPeriodicRealizationSuite(t *testing.T) {
	suite.Run(t, new(PeriodicRealizationSuite))
}

type PeriodicRealizationSuite struct {
	BaseSuite

	PeriodicRealization periodicrealization.Service
}

func (s *PeriodicRealizationSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	logger := omtestutils.NewLogger(s.T())

	adapter, err := periodicrealizationadapter.New(periodicrealizationadapter.Config{
		Client: s.DBClient,
	})
	s.Require().NoError(err)

	locker, err := lockr.NewLocker(&lockr.LockerConfig{Logger: logger})
	s.Require().NoError(err)

	s.PeriodicRealization, err = periodicrealizationservice.New(periodicrealizationservice.Config{
		Adapter: adapter,
		Locker:  locker,
	})
	s.Require().NoError(err)
}

func (s *PeriodicRealizationSuite) TestSchedulesTheFirstRunOfANamespace() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("periodic-realization")

	// given: the namespace has no run

	// when: the run as of Nov 24, 10:00 is asked for
	run, err := s.PeriodicRealization.GetOrCreateRunAt(ctx, periodicrealization.GetOrCreateRunAtInput{
		Namespace: namespace,
		AsOf:      omtestutils.GetRFC3339Time(s.T(), "2026-11-24T10:00:00Z"),
	})
	s.Require().NoError(err)

	// then: the run for the next period end is stored, realizable once the collection delay has passed
	stored, err := s.DBClient.BillingPeriodicRealizationRun.Query().
		Where(rundb.Namespace(namespace)).
		Only(ctx)
	s.Require().NoError(err)

	s.Equal(run.ID, stored.ID)
	// The returned run carries its times in UTC, so they compare without conversion.
	s.Equal(omtestutils.GetRFC3339Time(s.T(), "2026-11-25T00:00:00Z"), run.PeriodTo)
	s.Equal(omtestutils.GetRFC3339Time(s.T(), "2026-11-25T00:01:00Z"), run.RealizeAfter)

	s.Equal(omtestutils.GetRFC3339Time(s.T(), "2026-11-25T00:00:00Z"), stored.PeriodTo.UTC())
	s.Equal(omtestutils.GetRFC3339Time(s.T(), "2026-11-25T00:01:00Z"), stored.RealizeAfter.UTC())
	s.Equal(periodicrealization.RunStatusScheduled, stored.Status)
	s.Nil(stored.PreviousRunID)
}

func (s *PeriodicRealizationSuite) TestSchedulesTheNextRunOfANamespace() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("periodic-realization")

	// given: the namespace has a run for the period ending Nov 25
	first, err := s.PeriodicRealization.GetOrCreateRunAt(ctx, periodicrealization.GetOrCreateRunAtInput{
		Namespace: namespace,
		AsOf:      omtestutils.GetRFC3339Time(s.T(), "2026-11-24T10:00:00Z"),
	})
	s.Require().NoError(err)

	// when: the run as of Nov 25, 10:00 is asked for
	next, err := s.PeriodicRealization.GetOrCreateRunAt(ctx, periodicrealization.GetOrCreateRunAtInput{
		Namespace: namespace,
		AsOf:      omtestutils.GetRFC3339Time(s.T(), "2026-11-25T10:00:00Z"),
	})
	s.Require().NoError(err)

	// then: a run for the period ending Nov 26 is stored, chained to the first one
	stored, err := s.DBClient.BillingPeriodicRealizationRun.Query().
		Where(rundb.Namespace(namespace), rundb.IDNEQ(first.ID)).
		Only(ctx)
	s.Require().NoError(err)

	s.Equal(next.ID, stored.ID)
	s.Equal(omtestutils.GetRFC3339Time(s.T(), "2026-11-26T00:00:00Z"), next.PeriodTo)
	s.Equal(omtestutils.GetRFC3339Time(s.T(), "2026-11-26T00:01:00Z"), next.RealizeAfter)
	s.Equal(periodicrealization.RunStatusScheduled, stored.Status)
	s.Require().NotNil(stored.PreviousRunID)
	s.Equal(first.ID, *stored.PreviousRunID)
}

func (s *PeriodicRealizationSuite) TestPeriodEndBelongsToTheRunEndingThere() {
	namespace := s.GetUniqueNamespace("periodic-realization")

	// given: the namespace has a run for the period ending Nov 25
	existing := s.runAsOf(namespace, "2026-11-24T10:00:00Z")

	// when: the run as of exactly Nov 25, 00:00 is asked for
	atPeriodEnd := s.runAsOf(namespace, "2026-11-25T00:00:00Z")

	// then: as-of is exclusive like the period end, so it is still the run ending there
	s.Equal(existing.ID, atPeriodEnd.ID)

	// when: the run as of a second later is asked for
	next := s.runAsOf(namespace, "2026-11-25T00:00:01Z")

	// then: that is past the period, so the next run is created
	s.NotEqual(existing.ID, next.ID)
	s.Equal(omtestutils.GetRFC3339Time(s.T(), "2026-11-26T00:00:00Z"), next.PeriodTo)
}

func (s *PeriodicRealizationSuite) TestReturnsTheRunOfAPeriodThatAlreadyHasOne() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("periodic-realization")

	// given: the namespace has a run for the period ending Nov 25
	existing := s.runAsOf(namespace, "2026-11-24T10:00:00Z")

	// when: the run as of a later time in the same period is asked for
	again := s.runAsOf(namespace, "2026-11-24T15:00:00Z")

	// then: the existing run is returned and nothing is created
	s.Equal(existing.ID, again.ID)

	count, err := s.DBClient.BillingPeriodicRealizationRun.Query().
		Where(rundb.Namespace(namespace)).
		Count(ctx)
	s.Require().NoError(err)
	s.Equal(1, count)
}

func (s *PeriodicRealizationSuite) TestSkippedPeriodIsCoveredByTheNextRun() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("periodic-realization")

	// given: runs for the periods ending Nov 25 and Nov 29, and none in between
	first := s.runAsOf(namespace, "2026-11-24T10:00:00Z")
	afterGap := s.runAsOf(namespace, "2026-11-28T10:00:00Z")

	s.Equal(omtestutils.GetRFC3339Time(s.T(), "2026-11-29T00:00:00Z"), afterGap.PeriodTo)
	s.Require().NotNil(afterGap.PreviousRunID)
	s.Equal(first.ID, *afterGap.PreviousRunID)

	// when: the run as of Nov 26, 10:00 is asked for, a time in a period without a run
	covering := s.runAsOf(namespace, "2026-11-26T10:00:00Z")

	// then: it is the Nov 29 run, and no run is created behind it
	s.Equal(afterGap.ID, covering.ID)

	count, err := s.DBClient.BillingPeriodicRealizationRun.Query().
		Where(rundb.Namespace(namespace)).
		Count(ctx)
	s.Require().NoError(err)
	s.Equal(2, count)
}

func (s *PeriodicRealizationSuite) TestTimeBeforeTheFirstRunIsCoveredByIt() {
	namespace := s.GetUniqueNamespace("periodic-realization")

	// given: the namespace's first run is for the period ending Nov 25
	first := s.runAsOf(namespace, "2026-11-24T10:00:00Z")

	// when: the run as of Nov 20 is asked for
	covering := s.runAsOf(namespace, "2026-11-20T10:00:00Z")

	// then: it is the first run
	s.Equal(first.ID, covering.ID)
}

// runAsOf returns the namespace's run as of the given RFC 3339 time.
func (s *PeriodicRealizationSuite) runAsOf(namespace string, asOf string) periodicrealization.Run {
	s.T().Helper()

	run, err := s.PeriodicRealization.GetOrCreateRunAt(s.T().Context(), periodicrealization.GetOrCreateRunAtInput{
		Namespace: namespace,
		AsOf:      omtestutils.GetRFC3339Time(s.T(), asOf),
	})
	s.Require().NoError(err)

	return run
}
