// Copyright © 2024 Tailfin Cloud Inc.
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

package config

import (
	"errors"
	"time"

	"github.com/spf13/viper"
)

type PortalCORSConfiguration struct {
	Enabled bool `mapstructure:"enabled"`
}

type PortalConfiguration struct {
	Enabled         bool                    `mapstructure:"enabled"`
	CORS            PortalCORSConfiguration `mapstructure:"cors"`
	TokenSecret     string                  `mapstructure:"tokenSecret"`
	TokenExpiration time.Duration           `mapstructure:"tokenExpiration"`
}

// Validate validates the configuration.
func (c PortalConfiguration) Validate() error {
	if c.Enabled && c.TokenSecret == "" {
		return errors.New("token secret is required")
	}

	if c.Enabled && c.TokenExpiration.Seconds() == 0 {
		return errors.New("token duration is required")
	}

	return nil
}

// ConfigurePortal configures some defaults in the Viper instance.
func ConfigurePortal(v *viper.Viper) {
	v.SetDefault("portal.enabled", false)
	v.SetDefault("portal.cors.enabled", true)
	v.SetDefault("portal.tokenSecret", "")
	v.SetDefault("portal.tokenExpiration", "1h")
}
