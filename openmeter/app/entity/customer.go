package appentity

import (
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
)

type CustomerData interface {
	GetAppID() appentitybase.AppID
	Validate() error
}
