package subscription_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	pcsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	pcsubscriptionservice "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/service"
	subscription "github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
)

type testDeps struct {
	subscriptiontestutils.SubscriptionDependencies
	pcSubscriptionService       pcsubscription.PlanSubscriptionService
	subscriptionService         subscription.Service
	subscriptionWorkflowService subscriptionworkflow.Service
	cleanup                     func(t *testing.T) // Cleanup function
}

type setupConfig struct{}

func setup(t *testing.T, _ setupConfig) testDeps {
	t.Helper()

	// Let's build the dependencies
	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	require.NotNil(t, dbDeps)

	deps := subscriptiontestutils.NewService(t, dbDeps)

	pcSubsService := pcsubscriptionservice.New(pcsubscriptionservice.Config{
		WorkflowService:     deps.WorkflowService,
		SubscriptionService: deps.SubscriptionService,
		PlanService:         deps.PlanService,
		Logger:              testutils.NewLogger(t),
		CustomerService:     deps.CustomerService,
	})

	return testDeps{
		SubscriptionDependencies:    deps,
		pcSubscriptionService:       pcSubsService,
		subscriptionService:         deps.SubscriptionService,
		subscriptionWorkflowService: deps.WorkflowService,
		cleanup:                     dbDeps.Cleanup,
	}
}
