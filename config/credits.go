package config

type CreditsConfiguration struct {
	Enabled bool
}

// Validate validates the configuration.
func (c CreditsConfiguration) Validate() error {
	return nil
}
