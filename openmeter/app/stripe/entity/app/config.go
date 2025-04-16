package appstripeentityapp

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/app"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
)

type Configuration struct {
	SecretAPIKey *string
}

func (c Configuration) Validate() error {
	if c.SecretAPIKey != nil && *c.SecretAPIKey == "" {
		return errors.New("secretAPIKey cannot be empty")
	}

	return nil
}

func (a App) UpdateAppConfig(ctx context.Context, input app.AppConfigUpdate) error {
	configUpdate, ok := input.(Configuration)
	if !ok {
		return errors.New("invalid config update")
	}

	if configUpdate.SecretAPIKey != nil {
		return a.StripeAppService.UpdateAPIKey(ctx, appstripeentity.UpdateAPIKeyInput{
			AppID:  a.GetID(),
			APIKey: *configUpdate.SecretAPIKey,
		})
	}

	return nil
}
