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

package kafka

import (
	"context"
	"log/slog"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// LogEmitter emits logs from a [kafka.Consumer] or [kafka.Producer].
//
// Requires `go.logs.channel.enable` option set to true.
//
// This feature was implemented in [this PR].
//
// [this PR]: https://github.com/confluentinc/confluent-kafka-go/pull/421
type LogEmitter interface {
	Logs() chan kafka.LogEvent
}

// LogProcessor consumes logs from a [LogEmitter] and passes them to an [slog.Logger].
func LogProcessor(logEmitter LogEmitter, logger *slog.Logger) (execute func() error, interrupt func(error)) {
	ctx, cancel := context.WithCancel(context.Background())

	return func() error {
			for {
				select {
				case logEvent := <-logEmitter.Logs():
					processLog(logger, logEvent)
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		},
		func(error) {
			cancel()
		}
}

// ConsumeLogChannel is supposed to be called in a goroutine.
// It consumes a log channel returned by a [LogEmitter].
func ConsumeLogChannel(emitter LogEmitter, logger *slog.Logger) {
	for e := range emitter.Logs() {
		processLog(logger, e)
	}
}

func processLog(logger *slog.Logger, e kafka.LogEvent) {
	logger.Log(
		context.Background(),
		mapLogLevel(e.Level),
		e.Message,
		slog.String("name", e.Name),
		slog.String("tag", e.Tag),
		slog.Time("timestamp", e.Timestamp),
	)
}

// According to [kafka.LogEvent] the Level field is an int that contains a syslog severity level.
func mapLogLevel(level int) slog.Level {
	switch level {
	case 7:
		return slog.LevelDebug

	case 6, 5:
		return slog.LevelInfo

	case 4:
		return slog.LevelWarn

	case 3, 2, 1, 0:
		return slog.LevelError

	default:
		return slog.LevelInfo
	}
}
