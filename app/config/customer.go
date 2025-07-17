package config

import (
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/pkg/models"
)

var _ models.Validator = (*CustomerConfiguration)(nil)

type CustomerConfiguration struct {
	EnableSubjectHook bool
	IgnoreErrors      bool
}

func (c CustomerConfiguration) Validate() error {
	return nil
}

// ConfigureCustomer configures some defaults in the Viper instance.
func ConfigureCustomer(v *viper.Viper, prefixes ...string) {
	prefixer := NewViperKeyPrefixer(prefixes...)

	v.SetDefault(prefixer("enableSubjectHook"), false)
	v.SetDefault(prefixer("ignoreErrors"), false)
}
