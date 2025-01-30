package internal

import "context"

func InitializeApplication(ctx context.Context, configFile string) error {
	var err error

	if err = loadConfig(configFile); err != nil {
		return err
	}

	App, AppShutdown, err = initializeApplication(ctx, Config)
	if err != nil {
		return err
	}

	return nil
}
