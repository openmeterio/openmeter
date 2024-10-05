package appservice

import (
	"github.com/openmeterio/openmeter/openmeter/appstripe"
)

var _ appstripe.AppService = (*Service)(nil)
