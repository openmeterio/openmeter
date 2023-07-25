package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/exp/slog"
)

type StructuredLoggerOptions struct {
	// LogLevel defines the minimum level of severity that app should log.
	LogLevel slog.Level

	// SkipPaths defines a list of paths that should not be logged.
	SkipPaths []string

	// TimeFieldFormat defines the time format of the Time field, defaulting to "time.RFC3339Nano" see options at:
	// https://pkg.go.dev/time#pkg-constants
	TimeFieldFormat string

	// TimeFieldName sets the field name for the time field.
	// Some providers parse and search for different field names.
	TimeFieldName string
}

var DefaultStructuredLoggerOptions = &StructuredLoggerOptions{
	SkipPaths:       []string{},
	LogLevel:        slog.LevelDebug,
	TimeFieldFormat: time.RFC3339,
	TimeFieldName:   "timestamp",
}

type StructuredLogger struct {
	Logger  slog.Handler
	Options *StructuredLoggerOptions
}

func NewStructuredLogger(handler slog.Handler, options *StructuredLoggerOptions) func(next http.Handler) http.Handler {
	if options == nil {
		options = DefaultStructuredLoggerOptions
	}

	return middleware.RequestLogger(&StructuredLogger{Logger: handler, Options: options})
}

func (l *StructuredLogger) NewLogEntry(r *http.Request) middleware.LogEntry {
	logFields := []slog.Attr{
		slog.String(l.Options.TimeFieldName, time.Now().UTC().Format(l.Options.TimeFieldFormat)),
	}

	if reqID := middleware.GetReqID(r.Context()); reqID != "" {
		logFields = append(logFields, slog.String("req_id", reqID))
	}

	handler := l.Logger.WithAttrs(append(logFields,
		slog.String("http_scheme", r.URL.Scheme),
		slog.String("http_proto", r.Proto),
		slog.String("http_method", r.Method),
		slog.String("remote_addr", r.RemoteAddr),
		slog.String("user_agent", r.UserAgent()),
		slog.String("uri", r.URL.String())))

	return &StructuredLoggerEntry{logger: slog.New(handler), request: r, level: l.Options.LogLevel}
}

type StructuredLoggerEntry struct {
	// logger is the underlying logger.
	logger *slog.Logger

	// request is the original request, stored for later use when writing the log entry.
	request *http.Request

	// level is the default log level of the entry.
	level slog.Level
}

func (e *StructuredLoggerEntry) statusLevel(status int) slog.Level {
	switch {
	case status < 400:
		return e.level
	case status >= 400 && status < 500 && status != http.StatusNotFound:
		return slog.LevelWarn
	case status >= 500:
		return slog.LevelError
	default:
		return e.level
	}
}

func (e *StructuredLoggerEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	e.logger.LogAttrs(e.request.Context(), e.statusLevel(status), "request complete",
		slog.Int("resp_status", status),
		slog.Int("resp_byte_length", bytes),
		slog.Float64("resp_elapsed_ms", float64(elapsed.Nanoseconds())/1000000.0),
	)
}

func (e *StructuredLoggerEntry) Panic(v interface{}, stack []byte) {
	e.logger.LogAttrs(e.request.Context(), e.statusLevel(500), "request panicked",
		slog.String("stack", string(stack)),
		slog.String("panic", fmt.Sprintf("%+v", v)),
	)
}
