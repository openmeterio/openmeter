package appstripeadapter

import (
	"context"

	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	"github.com/stripe/stripe-go/v80/client"
)

type StripeClient interface {
	GetAccount(ctx context.Context) (appstripeentity.StripeAccount, error)
	GetCustomer(ctx context.Context, stripeCustomerID string) (appstripeentity.StripeCustomer, error)
}

type stripeClient struct {
	client *client.API
}

func StripeClientFactory(apiKey string) StripeClient {
	client := &client.API{}
	client.Init(apiKey, nil)

	return &stripeClient{
		client: client,
	}
}

func (c *stripeClient) GetAccount(ctx context.Context) (appstripeentity.StripeAccount, error) {
	stripeAccount, err := c.client.Accounts.Get()
	if err != nil {
		return appstripeentity.StripeAccount{}, err
	}

	return appstripeentity.StripeAccount{
		StripeAccountID: stripeAccount.ID,
	}, nil
}

func (c *stripeClient) GetCustomer(ctx context.Context, stripeCustomerID string) (appstripeentity.StripeCustomer, error) {
	stripeCustomer, err := c.client.Customers.Get(stripeCustomerID, nil)
	if err != nil {
		return appstripeentity.StripeCustomer{}, err
	}

	return appstripeentity.StripeCustomer{
		StripeCustomerID: stripeCustomer.ID,
	}, nil
}
