package appstripe

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
)

func TestCreateAppStripeInputValidateBindsSecretsToApp(t *testing.T) {
	// given:
	// - a valid Stripe app creation input with app-owned API-key and webhook secrets
	// when:
	// - an app ID, secret owner, or semantic secret key is replaced
	// then:
	// - validation accepts only references belonging to the exact app
	appID := app.AppID{Namespace: "namespace", ID: "app-id"}
	validInput := CreateAppStripeInput{
		CreateAppInput: app.CreateAppInput{
			ID:        &appID,
			Namespace: appID.Namespace,
			Name:      "Stripe",
			Type:      app.AppTypeStripe,
		},
		StripeAccountID: "acct_123",
		APIKey: secretentity.NewSecretID(
			appID,
			"api-key-secret-id",
			APIKeySecretKey,
		),
		MaskedAPIKey:    "****",
		StripeWebhookID: "we_123",
		WebhookSecret: secretentity.NewSecretID(
			appID,
			"webhook-secret-id",
			WebhookSecretKey,
		),
	}

	for _, testCase := range []struct {
		name    string
		mutate  func(*CreateAppStripeInput)
		wantErr bool
	}{
		{
			name: "exact references",
		},
		{
			name: "missing app id",
			mutate: func(input *CreateAppStripeInput) {
				input.ID = nil
			},
			wantErr: true,
		},
		{
			name: "app id namespace differs from app namespace",
			mutate: func(input *CreateAppStripeInput) {
				input.Namespace = "another-namespace"
			},
			wantErr: true,
		},
		{
			name: "api key belongs to another app",
			mutate: func(input *CreateAppStripeInput) {
				input.APIKey = secretentity.NewSecretID(
					app.AppID{Namespace: appID.Namespace, ID: "another-app-id"},
					input.APIKey.ID,
					input.APIKey.Key,
				)
			},
			wantErr: true,
		},
		{
			name: "api key has another semantic key",
			mutate: func(input *CreateAppStripeInput) {
				input.APIKey.Key = "another-key"
			},
			wantErr: true,
		},
		{
			name: "webhook secret belongs to another app",
			mutate: func(input *CreateAppStripeInput) {
				input.WebhookSecret = secretentity.NewSecretID(
					app.AppID{Namespace: appID.Namespace, ID: "another-app-id"},
					input.WebhookSecret.ID,
					input.WebhookSecret.Key,
				)
			},
			wantErr: true,
		},
		{
			name: "webhook secret has another semantic key",
			mutate: func(input *CreateAppStripeInput) {
				input.WebhookSecret.Key = "another-key"
			},
			wantErr: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			input := validInput
			if testCase.mutate != nil {
				testCase.mutate(&input)
			}

			err := input.Validate()
			if testCase.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
