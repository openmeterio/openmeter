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

package commonhttp

import (
	"context"
	"errors"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/errorsx"
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

// HandleErrorIfTypeMatches checks if the error is of the given type and encodes it as an HTTP error.
// Using the generic feature we can mandate that the error implements the error interface. This is a
// must, as the errors.As would panic if the error does not implement the error interface.
func HandleErrorIfTypeMatches[T error](ctx context.Context, statusCode int, err error, w http.ResponseWriter, extendedProblemFunc ...func(T) (string, string)) bool {
	if err, ok := errorsx.ErrorAs[T](err); ok {
		extendedProblemFuncs := make([]ExtendProblemFunc, 0, len(extendedProblemFunc))
		for _, f := range extendedProblemFunc {
			extendedProblemFuncs = append(extendedProblemFuncs, func() (string, string) {
				return f(err)
			})
		}
		NewHTTPError(statusCode, err, extendedProblemFuncs...).EncodeError(ctx, w)
		return true
	}

	return false
}
