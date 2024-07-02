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

package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/cors"
)

type corsOptions struct {
	cors.Options
	AllowedPaths []string
}

func corsHandler(options corsOptions) func(next http.Handler) http.Handler {
	ch := cors.Handler(options.Options)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If AllowedPaths is empty, apply CORS to all paths
			if len(options.AllowedPaths) == 0 {
				ch(next).ServeHTTP(w, r)
				return
			}

			// Check if the request path starts with any of the allowed prefixes
			for _, path := range options.AllowedPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					ch(next).ServeHTTP(w, r)
					return
				}
			}

			// If none of the prefixes match, call the next handler
			next.ServeHTTP(w, r)
		})
	}
}
