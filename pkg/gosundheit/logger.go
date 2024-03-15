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

package gosundheit

import (
	"log/slog"

	health "github.com/AppsFlyer/go-sundheit"
)

type checkListener struct {
	logger *slog.Logger
}

func NewLogger(logger *slog.Logger) health.CheckListener {
	return checkListener{
		logger: logger,
	}
}

func (c checkListener) OnCheckRegistered(name string, result health.Result) {
	if result.Error != nil {
		c.logger.Error("initial health check failed", slog.String("check", name), slog.Any("error", result.Error))

		return
	}

	c.logger.Debug("health check registered", slog.String("check", name))
}

func (c checkListener) OnCheckStarted(name string) {
	c.logger.Debug("starting health check", slog.String("check", name))
}

func (c checkListener) OnCheckCompleted(name string, result health.Result) {
	if result.Error != nil {
		c.logger.Error("health check failed", slog.String("check", name), slog.Any("error", result.Error))

		return
	}

	c.logger.Debug("health check completed", slog.String("check", name))
}
