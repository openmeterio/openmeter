package httpdriver

import (
	"time"

	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/portal"
)

// toAPIPortalToken maps a portal token to an API portal token.
func toAPIPortalToken(t *portal.PortalToken) apiv1models.PortalToken {
	apiPortalToken := apiv1models.PortalToken{
		Id:                t.Id,
		ExpiresAt:         t.ExpiresAt,
		Subject:           t.Subject,
		AllowedMeterSlugs: t.AllowedMeterSlugs,
		// We don't map token autpomatically because it's a security risk.
		// Token need to be added manually in create token handler.
	}

	if apiPortalToken.ExpiresAt != nil && time.Now().After(*apiPortalToken.ExpiresAt) {
		apiPortalToken.Expired = lo.ToPtr(true)
	}

	return apiPortalToken
}
