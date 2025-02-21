package common

import (
	"fmt"

	"github.com/google/wire"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/portal"
	portaladapter "github.com/openmeterio/openmeter/openmeter/portal/adapter"
)

var Portal = wire.NewSet(
	NewPortalService,
)

func NewPortalService(conf config.PortalConfiguration) (portal.Service, error) {
	if !conf.Enabled {
		return portaladapter.NewNoop(), nil
	}

	p, err := portaladapter.New(portaladapter.Config{
		Secret: conf.TokenSecret,
		Expire: conf.TokenExpiration,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create portal adapter: %w", err)
	}

	return p, nil
}
