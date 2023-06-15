// Copyright © 2023 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"

	"github.com/openmeterio/openmeter/pkg/models"
)

// configuration holds any kind of configuration that comes from the outside world and
// is necessary for running the application.
// TODO: improve configuration options
type configuration struct {
	Address    string
	Broker     string
	KSQLDB     string
	Schema     string
	Partitions int

	Log logConfiguration

	// Telemetry configuration
	Telemetry struct {
		// Telemetry HTTP server address
		Address string
	}

	Meters []*models.Meter
}

// Validate validates the configuration.
func (c configuration) Validate() error {
	if c.Address == "" {
		return errors.New("server address is required")
	}

	if c.Broker == "" {
		return errors.New("kafka broker is required")
	}

	if c.KSQLDB == "" {
		return errors.New("ksqldb URL is required")
	}

	if c.Schema == "" {
		return errors.New("schema registry URL is required")
	}

	if err := c.Log.Validate(); err != nil {
		return err
	}

	if c.Telemetry.Address == "" {
		return errors.New("telemetry http server address is required")
	}

	if len(c.Meters) == 0 {
		return errors.New("at least one meter is required")
	}

	for _, m := range c.Meters {
		// set default window size
		if m.WindowSize == "" {
			m.WindowSize = models.WindowSizeMinute
		}

		if err := m.Validate(); err != nil {
			return err
		}
	}

	return nil
}

type logConfiguration struct {
	// Format specifies the output log format.
	// Accepted values are: json, text
	Format string

	// Level is the minimum log level that should appear on the output.
	Level string
}

// Validate validates the configuration.
func (c logConfiguration) Validate() error {
	if !slices.Contains([]string{"json", "text", "tint"}, c.Format) {
		return fmt.Errorf("invalid format: %q", c.Format)
	}

	if !slices.Contains([]string{"debug", "info", "warn", "error"}, c.Level) {
		return fmt.Errorf("invalid format: %q", c.Level)
	}

	return nil
}

// configure configures some defaults in the Viper instance.
func configure(v *viper.Viper, flags *pflag.FlagSet) {
	// Viper settings
	v.AddConfigPath(".")

	// Environment variable settings
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	// Server configuration
	flags.String("address", ":8888", "Server address")
	_ = v.BindPFlag("address", flags.Lookup("address"))
	v.SetDefault("address", ":8888")

	// Kafka configuration
	flags.String("broker", "127.0.0.1:29092", "Kafka broker")
	_ = v.BindPFlag("broker", flags.Lookup("broker"))
	v.SetDefault("broker", "127.0.0.1:29092")

	// Kafka partition count
	flags.Int("partitions", 100, "Kafka Partitions")
	_ = v.BindPFlag("partitions", flags.Lookup("partitions"))
	// TODO: default to 100 in prod
	v.SetDefault("partitions", 1)

	// kSQL configuration
	// TODO: improve this section
	flags.String("ksqldb-url", "http://127.0.0.1:8088", "KSQLDB to connect to")
	_ = v.BindPFlag("ksqldb", flags.Lookup("ksqldb-url"))
	v.SetDefault("ksqldb", "http://127.0.0.1:8088")

	// Schema configuration
	flags.String("schema-registry-url", "http://127.0.0.1:8081", "Schema Registry")
	_ = v.BindPFlag("schema", flags.Lookup("schema-registry-url"))
	v.SetDefault("schema", "http://127.0.0.1:8081")

	// Log configuration
	v.SetDefault("log.format", "json")
	v.SetDefault("log.level", "info")
	//
	// Telemetry configuration
	flags.String("telemetry-address", ":10000", "Telemetry HTTP server address")
	_ = v.BindPFlag("telemetry.address", flags.Lookup("telemetry-address"))
	v.SetDefault("telemetry.address", ":10000")
}
