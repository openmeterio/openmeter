package router

import (
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
)

// warningOnlyLogger is a logger that logs errors as warnings instead of errors.
// unfortunately, watermill only supports error and info level logging.
type warningOnlyLogger struct {
	watermill.LoggerAdapter

	logger *slog.Logger
}

func newWarningOnlyLogger(logger *slog.Logger) watermill.LoggerAdapter {
	return warningOnlyLogger{LoggerAdapter: watermill.NewSlogLogger(logger), logger: logger}
}

func (l warningOnlyLogger) With(fields watermill.LogFields) watermill.LoggerAdapter {
	logger := l.logger.With(l.slogAttrsFromFields(fields)...)

	return warningOnlyLogger{LoggerAdapter: watermill.NewSlogLogger(logger), logger: logger}
}

func (l warningOnlyLogger) slogAttrsFromFields(fields watermill.LogFields) []interface{} {
	args := make([]any, 0, len(fields)*2+2)

	for k, v := range fields {
		args = append(args, k, v)
	}

	return args
}

func (l warningOnlyLogger) Error(msg string, err error, fields watermill.LogFields) {
	args := append(l.slogAttrsFromFields(fields), "error", err)

	l.logger.Warn(msg, args...)
}
