package config

import (
	"errors"

	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/pkg/buildinfo"
)

// CreditsWriterConfiguration identifies the build that posts ledger groups.
type CreditsWriterConfiguration struct {
	Revision string `yaml:"revision"`
	Version  string `yaml:"version"`
}

// CommitSHA falls back to Go's VCS metadata for binaries built in a checkout.
func (c CreditsWriterConfiguration) CommitSHA() string {
	if c.Revision != "" {
		return c.Revision
	}

	revision, _ := buildinfo.Revision("")

	return revision
}

// SetDefaults allows config files and environment to override binary metadata.
func (c CreditsWriterConfiguration) SetDefaults(v *viper.Viper, prefixes ...string) {
	prefixer := NewViperKeyPrefixer(prefixes...)
	v.SetDefault(prefixer("writer.revision"), c.Revision)
	v.SetDefault(prefixer("writer.version"), c.Version)
}

type CreditsConfiguration struct {
	Writer                  CreditsWriterConfiguration `yaml:"writer"`
	Enabled                 bool                       `yaml:"enabled"`
	EnableCreditThenInvoice bool                       `yaml:"enableCreditThenInvoice"`
}

func (c CreditsConfiguration) Validate() error {
	var errs []error

	// Add any validation logic for CreditsConfiguration here if needed.

	return errors.Join(errs...)
}

func ConfigureCredits(v *viper.Viper, prefixes ...string) {
	prefixer := NewViperKeyPrefixer(prefixes...)

	(CreditsWriterConfiguration{}).SetDefaults(v, prefixes...)
	v.SetDefault(prefixer("enabled"), false)
	v.SetDefault(prefixer("enableCreditThenInvoice"), false)
}
