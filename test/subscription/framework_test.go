package subscription_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	pcsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	pcsubscriptionservice "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/service"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
)

type testDeps struct {
	subscriptiontestutils.ExposedServiceDeps
	pcSubscriptionService       pcsubscription.PlanSubscriptionService
	subscriptionService         subscription.Service
	subscriptionWorkflowService subscription.WorkflowService
	cleanup                     func(t *testing.T) // Cleanup function
}

type setupConfig struct{}

func setup(t *testing.T, _ setupConfig) testDeps {
	t.Helper()

	// Let's build the dependencies
	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	require.NotNil(t, dbDeps)

	services, deps := subscriptiontestutils.NewService(t, dbDeps)

	pcSubsService := pcsubscriptionservice.New(pcsubscriptionservice.Config{
		WorkflowService:     services.WorkflowService,
		SubscriptionService: services.Service,
		PlanService:         deps.PlanService,
		Logger:              testutils.NewLogger(t),
		CustomerService:     deps.CustomerService,
	})

	return testDeps{
		ExposedServiceDeps:          deps,
		pcSubscriptionService:       pcSubsService,
		subscriptionService:         services.Service,
		subscriptionWorkflowService: services.WorkflowService,
		cleanup:                     dbDeps.Cleanup,
	}
}
