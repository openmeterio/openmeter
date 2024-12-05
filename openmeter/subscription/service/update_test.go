package service_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func TestEdit(t *testing.T) {
	type TDeps struct {
		CurrentTime time.Time
		Customer    customerentity.Customer
		ExamplePlan subscription.Plan
		ServiceDeps subscriptiontestutils.ExposedServiceDeps
		Service     subscription.Service
	}

	tt := []struct {
		Name    string
		Handler func(t *testing.T, deps TDeps)
	}{
		{
			Name: "Should error if plan changes",
			Handler: func(t *testing.T, deps TDeps) {
				t.Skip("TODO")
			},
		},
		{
			Name: "Should error if customer changes",
			Handler: func(t *testing.T, deps TDeps) {
				t.Skip("TODO")
			},
		},
		{
			Name: "Should error if subscription start changes",
			Handler: func(t *testing.T, deps TDeps) {
				t.Skip("TODO")
			},
		},
		{
			Name: "Should update contents of future phase when phase end changes",
			Handler: func(t *testing.T, deps TDeps) {
				t.Skip("TODO")
			},
		},
		{
			Name: "Should update contents of future phase when phase start changes",
			Handler: func(t *testing.T, deps TDeps) {
				t.Skip("TODO")
			},
		},
		{
			Name: "Should delete item from future phase",
			Handler: func(t *testing.T, deps TDeps) {
				t.Skip("TODO")
			},
		},
		{
			Name: "Should add item to future phase",
			Handler: func(t *testing.T, deps TDeps) {
				t.Skip("TODO")
			},
		},
		{
			Name: "Should update item entitlement",
			Handler: func(t *testing.T, deps TDeps) {
				t.Skip("TODO")
			},
		},
		{
			Name: "Should update contents of current phase",
			Handler: func(t *testing.T, deps TDeps) {
				t.Skip("TODO")
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")
			clock.SetTime(currentTime)

			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			defer dbDeps.Cleanup()

			services, deps := subscriptiontestutils.NewService(t, dbDeps)
			service := services.Service

			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			require.NotNil(t, cust)

			_ = deps.FeatureConnector.CreateExampleFeature(t)
			examplePlan := subscriptiontestutils.GetExamplePlan()
			deps.PlanAdapter.AddPlan(t, examplePlan)

			tc.Handler(t, TDeps{
				CurrentTime: currentTime,
				Customer:    *cust,
				ExamplePlan: examplePlan,
				ServiceDeps: deps,
				Service:     service,
			})
		})
	}
}
