package config

type EntitlementsConfiguration struct {
	Enabled         bool
	BalanceSnapshot BalanceSnapshotConfig
}

type BalanceSnapshotConfig struct {
	Enabled bool
}

// Validate validates the configuration.
func (c EntitlementsConfiguration) Validate() error {
	return nil
}
