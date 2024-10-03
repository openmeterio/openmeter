package appstripeadapter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/client"

	app "github.com/openmeterio/openmeter/openmeter/app"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	"github.com/openmeterio/openmeter/pkg/models"
)

type StripeClient interface {
	GetAccount(ctx context.Context) (appstripeentity.StripeAccount, error)
	GetCustomer(ctx context.Context, stripeCustomerID string) (appstripeentity.StripeCustomer, error)
	GetCustomerPaymentMethods(ctx context.Context, stripeCustomerID string) ([]appstripeentity.StripePaymentMethod, error)
}

type StripeClientConfig struct {
	Namespace string
	APIKey    string
}

func (c *StripeClientConfig) Validate() error {
	if c.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	if c.APIKey == "" {
		return fmt.Errorf("api key is required")
	}

	return nil
}

type stripeClient struct {
	namespace string
	client    *client.API
}

func NewStripeClient(config StripeClientConfig) (StripeClient, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	backend := stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{
		LeveledLogger: leveledLogger{
			logger: slog.Default(),
		},
	})
	client := &client.API{}
	client.Init(config.APIKey, &stripe.Backends{
		API:     backend,
		Connect: backend,
		Uploads: backend,
	})

	return &stripeClient{
		namespace: config.Namespace,
		client:    client,
	}, nil
}

// leveledLogger is a logger that implements the stripe LeveledLogger interface
var _ stripe.LeveledLoggerInterface = (*leveledLogger)(nil)

type leveledLogger struct {
	logger *slog.Logger
}

func (l leveledLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debug(fmt.Sprintf(format, args...))
}

func (l leveledLogger) Infof(format string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, args...))
}

func (l leveledLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warn(fmt.Sprintf(format, args...))
}

func (l leveledLogger) Errorf(format string, args ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, args...))
}

// GetAccount returns the authorized stripe account
func (c *stripeClient) GetAccount(ctx context.Context) (appstripeentity.StripeAccount, error) {
	stripeAccount, err := c.client.Accounts.Get()
	if err != nil {
		return appstripeentity.StripeAccount{}, c.providerError(err)
	}

	return appstripeentity.StripeAccount{
		StripeAccountID: stripeAccount.ID,
	}, nil
}

// GetCustomer returns the stripe customer by stripe customer ID
func (c *stripeClient) GetCustomer(ctx context.Context, stripeCustomerID string) (appstripeentity.StripeCustomer, error) {
	stripeCustomer, err := c.client.Customers.Get(stripeCustomerID, nil)
	if err != nil {
		return appstripeentity.StripeCustomer{}, c.providerError(err)
	}

	return appstripeentity.StripeCustomer{
		StripeCustomerID: stripeCustomer.ID,
	}, nil
}

// GetCustomerPaymentMethods returns the payment methods for the stripe customer
func (c *stripeClient) GetCustomerPaymentMethods(ctx context.Context, stripeCustomerID string) ([]appstripeentity.StripePaymentMethod, error) {
	iterator := c.client.PaymentMethods.List(&stripe.PaymentMethodListParams{
		Customer: stripe.String(stripeCustomerID),
	})

	var paymentMethods []appstripeentity.StripePaymentMethod

	for iterator.Next() {
		if iterator.Err() != nil {
			return paymentMethods, c.providerError(iterator.Err())
		}

		stripePaymentMethod := iterator.PaymentMethod()

		paymentMethod := appstripeentity.StripePaymentMethod{
			ID: stripePaymentMethod.ID,
		}

		if stripePaymentMethod.BillingDetails != nil && stripePaymentMethod.BillingDetails.Address != nil {
			address := *stripePaymentMethod.BillingDetails.Address

			paymentMethod.BillingAddress = &models.Address{
				Country:    lo.ToPtr(models.CountryCode(address.Country)),
				City:       lo.ToPtr(address.City),
				State:      lo.ToPtr(address.State),
				PostalCode: lo.ToPtr(address.PostalCode),
				Line1:      lo.ToPtr(address.Line1),
				Line2:      lo.ToPtr(address.Line2),
			}
		}

		paymentMethods = append(paymentMethods, paymentMethod)
	}

	return paymentMethods, nil
}

// providerError returns a typed error for stripe provider errors
func (c *stripeClient) providerError(err error) error {
	if stripeErr, ok := err.(*stripe.Error); ok {
		if stripeErr.HTTPStatusCode == http.StatusUnauthorized {
			return app.AppProviderAuthenticationError{
				Namespace:     c.namespace,
				Type:          appentitybase.AppTypeStripe,
				ProviderError: errors.New(stripeErr.Msg),
			}
		}

		return app.AppProviderError{
			Namespace:     c.namespace,
			Type:          appentitybase.AppTypeStripe,
			ProviderError: errors.New(stripeErr.Msg),
		}
	}

	return err
}
