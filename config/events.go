package config

import (
	"errors"

	"github.com/spf13/viper"
)

type EventsConfiguration struct {
	Enabled      bool
	SystemEvents EventSubsystemConfiguration
}

func (c EventsConfiguration) Validate() error {
	return c.SystemEvents.Validate()
}

type EventSubsystemConfiguration struct {
	Enabled bool
	Topic   string

	AutoProvision AutoProvisionConfiguration
}

func (c EventSubsystemConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Topic == "" {
		return errors.New("topic name is required")
	}
	return c.AutoProvision.Validate()
}

type AutoProvisionConfiguration struct {
	Enabled    bool
	Partitions int
}

func (c AutoProvisionConfiguration) Validate() error {
	if c.Enabled && c.Partitions < 1 {
		return errors.New("partitions must be greater than 0")
	}
	return nil
}

func ConfigureEvents(v *viper.Viper) {
	// TODO: after the system events are fully implemented, we should enable them by default
	v.SetDefault("events.enabled", false)
	v.SetDefault("events.systemEvents.enabled", true)
	v.SetDefault("events.systemEvents.topic", "om_sys.api_events")
	v.SetDefault("events.systemEvents.autoProvision.enabled", true)
	v.SetDefault("events.systemEvents.autoProvision.partitions", 4)
}
