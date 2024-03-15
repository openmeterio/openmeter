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
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

// JSONResponseEncoder encodes a response as JSON.
func JSONResponseEncoder[Response any](_ context.Context, w http.ResponseWriter, response Response) error {
	return jsonResponseEncoder(w, http.StatusOK, response)
}

// JSONResponseEncoder encodes a response as JSON.
func jsonResponseEncoder[Response any](w http.ResponseWriter, statusCode int, response Response) error {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(true)

	if err := enc.Encode(response); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_, err := w.Write(buf.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func JSONResponseEncoderWithStatus[Response any](statusCode int) httptransport.ResponseEncoder[Response] {
	return func(ctx context.Context, w http.ResponseWriter, r Response) error {
		return jsonResponseEncoder(w, statusCode, r)
	}
}

// PlainTextResponseEncoder encodes a response as PlainText.
func PlainTextResponseEncoder[Response string](_ context.Context, w http.ResponseWriter, response Response) error {
	return plainTextResponseEncoder(w, http.StatusOK, response)
}

// PlainTextResponseEncoder encodes a response as PlainText.
func plainTextResponseEncoder[Response string](w http.ResponseWriter, statusCode int, response Response) error {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(statusCode)

	_, err := w.Write([]byte(response))
	if err != nil {
		return err
	}

	return nil
}

func EmptyResponseEncoder[Response any](statusCode int) httptransport.ResponseEncoder[Response] {
	return func(ctx context.Context, w http.ResponseWriter, r Response) error {
		w.WriteHeader(statusCode)
		return nil
	}
}
