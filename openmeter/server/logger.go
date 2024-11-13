package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

type StructuredLogger struct {
	Logger slog.Handler
}

func NewStructuredLogger(handler slog.Handler) func(next http.Handler) http.Handler {
	return middleware.RequestLogger(&StructuredLogger{Logger: handler})
}

func (l *StructuredLogger) NewLogEntry(r *http.Request) middleware.LogEntry {
	return &StructuredLoggerEntry{logger: slog.New(l.Logger), request: r}
}

type StructuredLoggerEntry struct {
	// logger is the underlying logger.
	logger *slog.Logger

	// request is the original request, stored for later use when writing the log entry.
	request *http.Request
}

func (e *StructuredLoggerEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	e.logger.LogAttrs(e.request.Context(), slog.LevelDebug, "request complete",
		slog.Int(string(semconv.HTTPResponseStatusCodeKey), status),
		slog.Int(string(semconv.HTTPResponseSizeKey), bytes),
		slog.Float64("http.response.duration_ms", float64(elapsed.Nanoseconds())/1000000.0),
	)
}

func (e *StructuredLoggerEntry) Panic(v interface{}, stack []byte) {
	e.logger.LogAttrs(e.request.Context(), slog.LevelError, "request panicked",
		slog.String("stack", string(stack)),
		slog.String("panic", fmt.Sprintf("%+v", v)),
	)
}
