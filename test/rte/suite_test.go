package rte_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	creditstest "github.com/openmeterio/openmeter/test/credits"
)

// Reuse the credits fixture to exercise both real calculators and PostgreSQL state.
type RTEIntegrationSuite struct {
	creditstest.BaseSuite
}

func TestRTEIntegration(t *testing.T) {
	suite.Run(t, new(RTEIntegrationSuite))
}

func (s *RTEIntegrationSuite) newRTE() rte.Service {
	s.T().Helper()

	engine, err := common.NewRTEService(
		s.DBClient,
		s.CustomerBalanceSvc,
		s.Entitlements,
		testutils.NewDiscardLogger(s.T()),
		config.CreditsConfiguration{Enabled: true},
		eventbus.NewMock(s.T()),
	)
	s.Require().NoError(err)

	return engine
}
