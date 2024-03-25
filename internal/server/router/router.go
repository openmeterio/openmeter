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

package router

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/namespace"
	"github.com/openmeterio/openmeter/internal/server/authenticator"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/errorsx"
)

func init() {
	// See https://github.com/getkin/kin-openapi/issues/640
	openapi3filter.RegisterBodyDecoder("application/cloudevents+json", jsonBodyDecoder)
	openapi3filter.RegisterBodyDecoder("application/cloudevents-batch+json", jsonBodyDecoder)
}

func jsonBodyDecoder(body io.Reader, header http.Header, schema *openapi3.SchemaRef, encFn openapi3filter.EncodingFn) (interface{}, error) {
	var value interface{}
	if err := json.NewDecoder(body).Decode(&value); err != nil {
		return nil, &openapi3filter.ParseError{Kind: openapi3filter.KindInvalidFormat, Cause: err}
	}
	return value, nil
}

type IngestHandler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request, namespace string)
}

type Config struct {
	NamespaceManager    *namespace.Manager
	StreamingConnector  streaming.Connector
	IngestHandler       IngestHandler
	Meters              meter.Repository
	PortalCORSEnabled   bool
	PortalTokenStrategy *authenticator.PortalTokenStrategy
	ErrorHandler        errorsx.Handler
}

type Router struct {
	config Config
}

// Make sure we conform to ServerInterface
var _ api.ServerInterface = (*Router)(nil)

func NewRouter(config Config) (*Router, error) {
	return &Router{
		config: config,
	}, nil
}
