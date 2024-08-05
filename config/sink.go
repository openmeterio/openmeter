package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type SinkConfiguration struct {
	GroupId             string
	Dedupe              DedupeConfiguration
	MinCommitCount      int
	MaxCommitWait       time.Duration
	NamespaceRefetch    time.Duration
	IngestNotifications IngestNotificationsConfiguration
}

func (c SinkConfiguration) Validate() error {
	if c.MinCommitCount < 1 {
		return errors.New("MinCommitCount must be greater than 0")
	}

	if c.MaxCommitWait < 1 {
		return errors.New("MaxCommitWait must be greater than 0")
	}

	if c.NamespaceRefetch < 1 {
		return errors.New("NamespaceRefetch must be greater than 0")
	}

	if err := c.IngestNotifications.Validate(); err != nil {
		return fmt.Errorf("ingest notifications: %w", err)
	}

	return nil
}

type IngestNotificationsConfiguration struct {
	MaxEventsInBatch int
}

func (c IngestNotificationsConfiguration) Validate() error {
	if c.MaxEventsInBatch < 0 {
		return errors.New("ChunkSize must not be negative")
	}

	if c.MaxEventsInBatch > 1000 {
		return errors.New("ChunkSize must not be greater than 1000")
	}

	return nil
}

// Configure configures some defaults in the Viper instance.
func ConfigureSink(v *viper.Viper) {
	// Sink Dedupe
	v.SetDefault("sink.dedupe.enabled", false)
	v.SetDefault("sink.dedupe.driver", "memory")

	// Sink Dedupe Memory driver
	v.SetDefault("sink.dedupe.config.size", 128)

	// Sink Dedupe Redis driver
	v.SetDefault("sink.dedupe.config.address", "127.0.0.1:6379")
	v.SetDefault("sink.dedupe.config.database", 0)
	v.SetDefault("sink.dedupe.config.username", "")
	v.SetDefault("sink.dedupe.config.password", "")
	v.SetDefault("sink.dedupe.config.expiration", "24h")
	v.SetDefault("sink.dedupe.config.sentinel.enabled", false)
	v.SetDefault("sink.dedupe.config.sentinel.masterName", "")
	v.SetDefault("sink.dedupe.config.tls.enabled", false)
	v.SetDefault("sink.dedupe.config.tls.insecureSkipVerify", false)

	// Sink
	v.SetDefault("sink.groupId", "openmeter-sink-worker")
	v.SetDefault("sink.minCommitCount", 500)
	v.SetDefault("sink.maxCommitWait", "5s")
	v.SetDefault("sink.namespaceRefetch", "15s")
	v.SetDefault("sink.ingestNotifications.maxEventsInBatch", 500)
}
