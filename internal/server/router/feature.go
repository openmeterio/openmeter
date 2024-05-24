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
	"net/http"

	"github.com/openmeterio/openmeter/api"
)

// Get feature, GET:/api/v1/features/{featureID}
func (a *Router) GetFeature(w http.ResponseWriter, r *http.Request, featureID api.FeatureID) {
	a.CreditHandlers.GetFeature.With(featureID).ServeHTTP(w, r)
}

// List features: GET /api/v1/features
func (a *Router) ListFeatures(w http.ResponseWriter, r *http.Request, apiParams api.ListFeaturesParams) {
	a.CreditHandlers.ListFeatures.With(apiParams).ServeHTTP(w, r)
}

// Create feature, POST: /api/v1/features
func (a *Router) CreateFeature(w http.ResponseWriter, r *http.Request) {
	a.CreditHandlers.CreateFeature.ServeHTTP(w, r)
}

// Delete feature, DELETE:/api/v1/features/{featureID}
func (a *Router) DeleteFeature(w http.ResponseWriter, r *http.Request, featureID api.FeatureID) {
	a.CreditHandlers.DeleteFeature.With(featureID).ServeHTTP(w, r)
}
