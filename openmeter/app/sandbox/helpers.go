package appsandbox

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/models"
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
		return nil, models.NewGenericValidationError(err)
	}

	// Get the sandbox app list
	sandboxAppList, err := input.AppService.ListApps(ctx, app.ListAppInput{
		Namespace: input.Namespace,
		Type:      lo.ToPtr(app.AppTypeSandbox),
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list apps: %w", err)
	}

	// If there is no sandbox app, we need to provision a new one
	if sandboxAppList.TotalCount == 0 {
		// Let's provision the new app
		appBase, err := input.AppService.CreateApp(ctx, app.CreateAppInput{
			Namespace:   input.Namespace,
			Name:        "Sandbox",
			Description: "OpenMeter Sandbox App to be used for testing purposes.",
			Type:        app.AppTypeSandbox,
		})
		if err != nil {
			return nil, fmt.Errorf("cannot create sandbox app: %w", err)
		}

		return input.AppService.GetApp(ctx, app.GetAppInput{
			Namespace: input.Namespace,
			ID:        appBase.GetID().ID,
		})
	}

	// If there is more than one sandbox app, we need to return the first one
	return sandboxAppList.Items[0], nil
}
