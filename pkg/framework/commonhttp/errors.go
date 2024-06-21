package commonhttp

import (
	"context"
	"errors"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/models"
)

type ErrorWithHTTPStatusCode struct {
	error
	StatusCode int
	Extensions []ExtendProblemFunc
}

func ExtendProblem(name, details string) ExtendProblemFunc {
	return func() (string, string) {
		return name, details
	}
}

type ExtendProblemFunc func() (name string, details string)

func (e ExtendProblemFunc) apply(extensions map[string]interface{}) {
	name, details := e()
	extensions[name] = details
}

func NewHTTPError(statusCode int, err error, extensions ...ExtendProblemFunc) ErrorWithHTTPStatusCode {
	return ErrorWithHTTPStatusCode{
		StatusCode: statusCode,
		error:      err,
		Extensions: extensions,
	}
}

func (e ErrorWithHTTPStatusCode) EncodeError(ctx context.Context, w http.ResponseWriter) bool {
	problem := models.NewStatusProblem(ctx, e.error, e.StatusCode)
	for _, ext := range e.Extensions {
		ext.apply(problem.Extensions)
	}
	problem.Respond(w)
	return true
}

// ErrorEncoder encodes an error as HTTP 500 Internal Server Error.
func ErrorEncoder(ctx context.Context, _ error, w http.ResponseWriter) bool {
	models.NewStatusProblem(ctx, errors.New("something went wrong"), http.StatusInternalServerError).Respond(w)

	return false
}
