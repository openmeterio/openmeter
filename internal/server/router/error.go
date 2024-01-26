package router

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/openmeterio/openmeter/pkg/models"
)

// errorRespond responds with the problem and logs the problem
func errorRespond(logger *slog.Logger, problem models.Problem, w http.ResponseWriter, r *http.Request) {
	statusCode := http.StatusInternalServerError
	title := http.StatusText(statusCode)
	logLevel := slog.LevelError

	// Repond with the problem
	problem.Respond(w, r)

	// Extract the status code from status problem
	if sp, ok := problem.(*models.StatusProblem); ok {
		if sp.Status < 500 {
			logLevel = slog.LevelWarn
		}

		title = sp.Title
		statusCode = sp.Status
	}

	msg := fmt.Sprintf("request failed: %s", strings.ToLower(title))

	logger.LogAttrs(r.Context(), logLevel, msg,
		slog.Int("resp_status", statusCode),
		slog.String("req_method", r.Method),
		slog.String("req_path", r.URL.Path),
		slog.Any("error", problem.RawError()),
	)
}
