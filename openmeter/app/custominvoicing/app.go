package custominvoicing

import (
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
)

const (
	AppType = "custom_invoicing"
)

type AppConfig struct {
	SkipDraftSync bool
	SkipFinalize  bool
	// TODO: secrets to be handled by a secret manager that only returns references
}

var Factory = app.NewAppFactory(app.Listing[App, AppConfig]{
	Type:        AppType,
	Name:        "Custom Invoicing",
	Description: "Custom Invoicing can be used to integrate with custom invoicing systems.",
})

type App struct {
	app.AppBase

	billingService billing.Service
}
