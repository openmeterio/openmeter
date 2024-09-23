package config

import (
	"errors"

	"github.com/spf13/viper"
)

// Namespace configuration
type NamespaceConfiguration struct {
	Default           string
	DisableManagement bool
}

func (c NamespaceConfiguration) Validate() error {
	var errs []error

	if c.Default == "" {
		errs = append(errs, errors.New("default namespace is required"))
	}

	return errors.Join(errs...)
}

// ConfigureNamespace configures some defaults in the Viper instance.
func ConfigureNamespace(v *viper.Viper) {
	v.SetDefault("namespace.default", "default")
	v.SetDefault("namespace.disableManagement", false)
}
