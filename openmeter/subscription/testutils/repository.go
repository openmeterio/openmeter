package subscriptiontestutils

import (
	"context"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/adapter"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func NewRepo(t *testing.T, dbDeps *DBDeps) testSubscriptionRepo {
	t.Helper()
	repo := adapter.NewSubscriptionRepo(dbDeps.dbClient)
	return testSubscriptionRepo{
		repo,
	}
}

type testSubscriptionRepo struct {
	subscription.Repository
}

func (r *testSubscriptionRepo) CreateExampleSubscription(t *testing.T, customerId string) subscription.Subscription {
	t.Helper()

	input := getExampleCreateSubscriptionInput(customerId)
	s, err := r.CreateSubscription(context.Background(), ExampleNamespace, input)
	if err != nil {
		t.Fatalf("failed to create example subscription: %v", err)
	}
	return s
}

func getExampleCreateSubscriptionInput(customerId string) subscription.CreateSubscriptionInput {
	return subscription.CreateSubscriptionInput{
		Plan:       ExamplePlanRef,
		CustomerId: customerId,
		Currency:   "USD",
		CadencedModel: models.CadencedModel{
			ActiveFrom: clock.Now(),
		},
	}
}
