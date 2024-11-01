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
// See https://en.wikipedia.org/wiki/Syslog#Severity_level
func mapLogLevel(level int) slog.Level {
	switch level {
	// Notice (5): Normal but significant conditions
	case 5:
		return slog.LevelInfo

	// Warning (4): Warning conditions
	case 4:
		return slog.LevelWarn

	// Error (3): Error conditions
	// Critical (2): Critical conditions
	// Alert (1): Action must be taken immediately
	// Emergency (0): System is unusable
	case 3, 2, 1, 0:
		return slog.LevelError

	// To reduce verbosity, we map all other levels to Debug.
	// Informal (6): Confirmation that the program is working as expected.
	// Debug (7): Debug-level messages
	default:
		return slog.LevelDebug
	}
}
