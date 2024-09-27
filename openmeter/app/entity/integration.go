package appentity

import "context"

type AppFactory interface {
	NewIntegration(context.Context, AppBase) (App, error)
	Capabilities() []CapabilityType
}

type Integration struct {
	Listing MarketplaceListing
	Factory AppFactory
}
