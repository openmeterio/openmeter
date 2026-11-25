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
	"github.com/openmeterio/openmeter/pkg/models"
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

	// then: the run of the period containing that time is stored, realizable once the collection delay has passed
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

func (s *PeriodicRealizationSuite) TestSchedulingIntoANamespaceWithARunIsNotSupportedYet() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("periodic-realization")

	// given: the namespace has its first run
	_, err := s.PeriodicRealization.GetOrCreateRunAt(ctx, periodicrealization.GetOrCreateRunAtInput{
		Namespace: namespace,
		AsOf:      omtestutils.GetRFC3339Time(s.T(), "2026-11-24T10:00:00Z"),
	})
	s.Require().NoError(err)

	// when: the run as of a day later is asked for
	_, err = s.PeriodicRealization.GetOrCreateRunAt(ctx, periodicrealization.GetOrCreateRunAtInput{
		Namespace: namespace,
		AsOf:      omtestutils.GetRFC3339Time(s.T(), "2026-11-25T10:00:00Z"),
	})

	// then: it is rejected and the namespace keeps its one run
	s.Require().Error(err)
	s.True(models.IsGenericNotImplementedError(err))

	count, err := s.DBClient.BillingPeriodicRealizationRun.Query().
		Where(rundb.Namespace(namespace)).
		Count(ctx)
	s.Require().NoError(err)
	s.Equal(1, count)
}
