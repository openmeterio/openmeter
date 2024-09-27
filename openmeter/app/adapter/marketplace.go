package appadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
)

var _ app.MarketplaceAdapter = (*adapter)(nil)

// ListListings lists market

// InstallAppWithAPIKey installs an app with an API key
func (a adapter) InstallAppWithAPIKey(ctx context.Context, input appentity.InstallAppWithAPIKeyInput) (appentity.App, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetOauth2InstallURL gets an OAuth2 install URL
func (a adapter) GetOauth2InstallURL(ctx context.Context, input appentity.GetOauth2InstallURLInput) (appentity.GetOauth2InstallURLOutput, error) {
	return appentity.GetOauth2InstallURLOutput{}, fmt.Errorf("not implemented")
}

// AuthorizeOauth2Install authorizes an OAuth2 install
func (a adapter) AuthorizeOauth2Install(ctx context.Context, input appentity.AuthorizeOauth2InstallInput) error {
	return fmt.Errorf("not implemented")
}
