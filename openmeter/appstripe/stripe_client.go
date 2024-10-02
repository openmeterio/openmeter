package appstripe

import (
	"context"

	"github.com/stripe/stripe-go/v80/client"
)

type StripeClient interface {
	GetAccount(ctx context.Context) (StripeAccount, error)
	GetCustomer(ctx context.Context, stripeCustomerID string) (StripeCustomer, error)
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

func (c *stripeClient) GetAccount(ctx context.Context) (StripeAccount, error) {
	stripeAccount, err := c.client.Accounts.Get()
	if err != nil {
		return StripeAccount{}, err
	}

	return StripeAccount{
		StripeAccountID: stripeAccount.ID,
	}, nil
}

func (c *stripeClient) GetCustomer(ctx context.Context, stripeCustomerID string) (StripeCustomer, error) {
	stripeCustomer, err := c.client.Customers.Get(stripeCustomerID, nil)
	if err != nil {
		return StripeCustomer{}, err
	}

	return StripeCustomer{
		StripeCustomerID: stripeCustomer.ID,
	}, nil
}

type StripeAccount struct {
	StripeAccountID string
}

type StripeCustomer struct {
	StripeCustomerID string
}
