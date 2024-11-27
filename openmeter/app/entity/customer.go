package appentity

import (
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
)

type CustomerData interface {
	GetAppID() appentitybase.AppID
	GetCustomerID() customerentity.CustomerID
	Validate() error
}
