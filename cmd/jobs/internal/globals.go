package internal

import "github.com/openmeterio/openmeter/app/config"

//nolint:gochecknoglobals
var (
	App         Application
	AppShutdown func()
	Config      config.Configuration
	ConfigFile  string
)
