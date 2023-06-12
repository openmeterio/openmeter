// Copyright © 2023 Tailfin Cloud Inc.
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

package api

import (
	"net/http"

	"github.com/go-chi/render"
)

// ErrResponse renderer type for handling all sorts of errors.
// In the best case scenario, the excellent github.com/pkg/errors package
// helps reveal information on the error, setting it on Err, and in the Render()
// method, using it to set the application-specific error code in AppCode.
type ErrResponse struct {
	Err error `json:"-"` // low-level runtime error

	StatusCode int    `json:"statusCode"`        // http response status code
	StatusText string `json:"status"`            // user-level status message
	AppCode    int64  `json:"code,omitempty"`    // application-specific error code
	Message    string `json:"message,omitempty"` // application-level error message, for debugging
}

func (e *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.StatusCode)
	return nil
}

var ErrNotFound = &ErrResponse{StatusCode: 404, StatusText: http.StatusText(404)}

func ErrBadRequest(err error) *ErrResponse {
	return &ErrResponse{
		Err:        err,
		StatusCode: 400,
		StatusText: http.StatusText(400),
		Message:    err.Error(),
	}
}

func ErrInternalServerError(err error) *ErrResponse {
	return &ErrResponse{
		Err:        err,
		StatusCode: 500,
		StatusText: http.StatusText(500),
		Message:    err.Error(),
	}
}

func ErrUnprocessableEntity(err error) render.Renderer {
	return &ErrResponse{
		Err:        err,
		StatusCode: 422,
		StatusText: http.StatusText(422),
		Message:    err.Error(),
	}
}
