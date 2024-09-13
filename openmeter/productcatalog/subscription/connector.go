package subscription

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/models"
	modelref "github.com/openmeterio/openmeter/pkg/models/ref"
	"github.com/samber/lo"
)

type Connector interface {
	// EndAt ends a subscription effective at the provided time.
	EndAt(ctx context.Context, subscriptionID string, at time.Time) (Subscription, error)

	// StartNew attempts to start a new subscription for a customer effective at the provided time.
	StartNew(ctx context.Context, customerID string, sub SubscriptionCreateInput, contents []ContentCreateInput) (Subscription, error)

	// ChangeContents changes the contents of a subscription effective retroactively.
	//
	// Effective retroactively means that the subscription contents are changed and no new version is created to replace the old.
	// For example, if you change some usage based price included in the subscription, the prior usage will also be billed based on the new price.
	ChangeContents(ctx context.Context, subscriptionID string, overrides SubscriptionOverrides) (Subscription, error)
}

type connector struct {
	customerSubscriptionRepo CustomerSubscriptionRepo
	subscriptionRepo         SubscriptionRepo
	contentRepo              ContentRepo
	planRepo                 PlanRepo

	lifecycleManager LifecycleManager

	entitlementConnector entitlement.Connector
}

var _ Connector = (*connector)(nil)

// EndAt ends a subscription effective at the provided time.
func (c *connector) EndAt(ctx context.Context, subscriptionID string, at time.Time) (Subscription, error) {
	sub, err := c.subscriptionRepo.GetByID(ctx, modelref.IDRef(subscriptionID))
	if err != nil {
		return Subscription{}, err
	}

	if sub.ActiveTo != nil {
		return Subscription{}, &models.GenericUserError{Message: "Subscription is already ended at a different time."}
	}

	if at.Before(sub.ActiveFrom) {
		return Subscription{}, &models.GenericUserError{Message: "End time is before start time."}
	}

	return c.subscriptionRepo.UpdateCadence(ctx, modelref.IDRef(subscriptionID), models.CadencedModel{
		ActiveFrom: sub.ActiveFrom,
		ActiveTo:   &at,
	})
}

func (c *connector) StartNew(ctx context.Context, customerID string, input SubscriptionCreateInput, inputContents []ContentCreateInput) (Subscription, error) {
	prevCustomerSubs, err := c.customerSubscriptionRepo.GetAll(ctx, modelref.IDRef(customerID), SubscriptionFilters{})
	if err != nil {
		return Subscription{}, err
	}

	// We need to validate that the new subscription meets lifecycle rules
	err = c.lifecycleManager.CanStartNew(ctx, customerID, prevCustomerSubs, input)
	if err != nil {
		return Subscription{}, err
	}

	activatesAt := input.CadencedModel.ActiveFrom

	// TODO: start a transaction here

	// Persist the subscription and its contents
	sub, err := c.subscriptionRepo.Create(ctx, input)
	if err != nil {
		return Subscription{}, err
	}

	contents, err := c.contentRepo.CreateMany(ctx, modelref.IDRef(sub.ID), lo.Map(inputContents, func(c ContentCreateInput, _ int) ContentCreateInput {
		// Contents have to become active alongside the subscription
		c.ActiveFrom = activatesAt
		return c
	}))
	if err != nil {
		return Subscription{}, err
	}

	// Create Entitlements for the subscription
	for _, content := range contents {
		_, err := c.entitlementConnector.CreateEntitlement(ctx, ContentToEntitlementCreateInput(content))
		if entitlementExistsError, ok := lo.ErrorsAs[*entitlement.AlreadyExistsError](err); ok {
			// TODO: there might be a cleaner upsert than using the override flow, probably a custom method is needed

			// FIXME: OverrideEntitlement closes the transaction used which is not okay for us
			//
			// Idea:
			// We could manage nested transactions with PG SAVEPOINTs
			// - commits turn into RELEASE SAVEPOINT
			// - rollbacks turn into ROLLBACK TO SAVEPOINT
			// for the nested operations, while the top level context can manage overall commit/rollback
			_, err = c.entitlementConnector.OverrideEntitlement(ctx, entitlementExistsError.SubjectKey, entitlementExistsError.EntitlementID, ContentToEntitlementCreateInput(content))
		}

		if err != nil {
			return Subscription{}, err
		}
	}

	return sub, nil
}

func (c *connector) ChangeContents(ctx context.Context, subscriptionID string, overrides SubscriptionOverrides) (Subscription, error) {
	panic("not implemented")
}
