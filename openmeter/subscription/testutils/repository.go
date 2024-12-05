package subscriptiontestutils

import (
	"context"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	repository "github.com/openmeterio/openmeter/openmeter/subscription/repo"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func NewSubscriptionRepo(t *testing.T, dbDeps *DBDeps) testSubscriptionRepo {
	t.Helper()
	repo := repository.NewSubscriptionRepo(dbDeps.dbClient)
	return testSubscriptionRepo{
		repo,
	}
}

type testSubscriptionRepo struct {
	subscription.SubscriptionRepository
}

func (r *testSubscriptionRepo) CreateExampleSubscription(t *testing.T, customerId string) subscription.Subscription {
	t.Helper()

	input := getExampleCreateSubscriptionInput(customerId)
	s, err := r.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("failed to create example subscription: %v", err)
	}
	return s
}

func getExampleCreateSubscriptionInput(customerId string) subscription.CreateSubscriptionEntityInput {
	return subscription.CreateSubscriptionEntityInput{
		Plan:       ExamplePlanRef,
		CustomerId: customerId,
		Currency:   "USD",
		CadencedModel: models.CadencedModel{
			ActiveFrom: clock.Now(),
		},
	}
}

func NewSubscriptionPhaseRepo(t *testing.T, dbDeps *DBDeps) testSubscriptionPhaseRepo {
	t.Helper()
	repo := repository.NewSubscriptionPhaseRepo(dbDeps.dbClient)
	return testSubscriptionPhaseRepo{
		repo,
	}
}

type testSubscriptionPhaseRepo struct {
	subscription.SubscriptionPhaseRepository
}

func NewSubscriptionItemRepo(t *testing.T, dbDeps *DBDeps) testSubscriptionItemRepo {
	t.Helper()
	repo := repository.NewSubscriptionItemRepo(dbDeps.dbClient)
	return testSubscriptionItemRepo{
		repo,
	}
}

type testSubscriptionItemRepo struct {
	subscription.SubscriptionItemRepository
}
