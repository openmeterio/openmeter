package client

import (
	"fmt"
	"log/slog"

	"github.com/stripe/stripe-go/v80"
)

// leveledLogger is a logger that implements the stripe LeveledLogger interface
var _ stripe.LeveledLoggerInterface = (*leveledLogger)(nil)

type leveledLogger struct {
	logger *slog.Logger
}

func (l leveledLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debug(fmt.Sprintf(format, args...), "source", "stripe-go")
}

func (l leveledLogger) Infof(format string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, args...), "source", "stripe-go")
}

func (l leveledLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warn(fmt.Sprintf(format, args...), "source", "stripe-go")
}

func (l leveledLogger) Errorf(format string, args ...interface{}) {
	// We don't want to pollute the logs with errors from the Stripe API as we are handling them in the application
	l.logger.Warn(fmt.Sprintf(format, args...), "source", "stripe-go")
}
