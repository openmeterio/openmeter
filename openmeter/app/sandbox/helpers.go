package appsandbox

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
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

// AutoProvision creates a new default sandbox app if it doesn't exist, otherwise returns the existing one.
//
// We install the sandbox app by default in the system, so that the user can start trying out the system
// right away.
func AutoProvision(ctx context.Context, input AutoProvisionInput) (app.App, error) {
	if err := input.Validate(); err != nil {
		return nil, app.ValidationError{
			Err: err,
		}
	}

	// Let's try to resolve the default app
	defaultApp, err := input.AppService.GetDefaultApp(ctx, app.GetDefaultAppInput{
		Namespace: input.Namespace,
		Type:      app.AppTypeSandbox,
	})
	if err != nil {
		if _, ok := lo.ErrorsAs[app.AppDefaultNotFoundError](err); ok {
			// Let's provision the new app
			_, err := input.AppService.CreateApp(ctx, app.CreateAppInput{
				Namespace:   input.Namespace,
				Name:        "Sandbox",
				Description: "OpenMeter Sandbox App to be used for testing purposes.",
				Type:        app.AppTypeSandbox,
			})
			if err != nil {
				return nil, fmt.Errorf("cannot create sandbox app: %w", err)
			}

			return input.AppService.GetDefaultApp(ctx, app.GetDefaultAppInput{
				Namespace: input.Namespace,
				Type:      app.AppTypeSandbox,
			})
		}
		return nil, err
	}

	return defaultApp, nil
}
