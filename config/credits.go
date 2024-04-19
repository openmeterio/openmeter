package config

type CreditsConfig struct {
	Enabled bool
}

// Validate validates the configuration.
func (c CreditsConfig) Validate() error {
	return nil
}
