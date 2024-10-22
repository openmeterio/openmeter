package appsandbox

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
)

type AutoProvisionInput struct {
	Namespace  string
	AppService app.Service
}

func (a AutoProvisionInput) Validate() error {
	if a.Namespace == "" {
		return errors.New("namespace is required")
	}

	if a.AppService == nil {
		return errors.New("app service is required")
	}

	return nil
}

func AutoProvision(ctx context.Context, input AutoProvisionInput) (appentity.App, error) {
	if err := input.Validate(); err != nil {
		return nil, app.ValidationError{
			Err: err,
		}
	}

	// Let's try to resolve the default app
	defaultApp, err := input.AppService.GetDefaultApp(ctx, appentity.GetDefaultAppInput{
		Namespace: input.Namespace,
		Type:      appentitybase.AppTypeSandbox,
	})
	if err != nil {
		if _, ok := lo.ErrorsAs[app.AppDefaultNotFoundError](err); ok {
			// Let's provision the new app
			defaultApp, err = input.AppService.CreateApp(ctx, appentity.CreateAppInput{
				Namespace:   input.Namespace,
				Name:        "Sandbox",
				Description: "Sandbox app",
				Type:        appentitybase.AppTypeSandbox,
			})
			if err != nil {
				return nil, fmt.Errorf("cannot create sandbox app: %w", err)
			}

			return defaultApp, nil
		}
		return nil, err
	}

	return defaultApp, nil
}
