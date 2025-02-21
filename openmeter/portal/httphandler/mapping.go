package httpdriver

import (
	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/portal"
)

// toAPIPortalToken maps a portal token to an API portal token.
func toAPIPortalToken(t *portal.PortalToken) api.PortalToken {
	apiPortalToken := api.PortalToken{
		Id:                t.Id,
		Token:             t.Token,
		ExpiresAt:         t.ExpiresAt,
		Subject:           t.Subject,
		AllowedMeterSlugs: t.AllowedMeterSlugs,
	}

	return apiPortalToken
}
