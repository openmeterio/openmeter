package taxcode

import "github.com/openmeterio/openmeter/openmeter/app"

type TaxCodeAppMapping struct {
	AppType app.AppType `json:"app_type"`
	TaxCode string      `json:"tax_code"`
}

type TaxCodeAppMappings []TaxCodeAppMapping
