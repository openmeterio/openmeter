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

// List features
// (GET /api/v1/features)
func (a *Router) ListFeatures(w http.ResponseWriter, r *http.Request, params api.ListFeaturesParams) {
	if !a.config.EntitlementsEnabled {
		unimplemented.ListFeatures(w, r, params)
		return
	}
	a.featureHandler.ListFeatures().With(params).ServeHTTP(w, r)
}

// Create feature
// (POST /api/v1/features)
func (a *Router) CreateFeature(w http.ResponseWriter, r *http.Request) {
	if !a.config.EntitlementsEnabled {
		unimplemented.CreateFeature(w, r)
		return
	}
	a.featureHandler.CreateFeature().ServeHTTP(w, r)
}

// Delete feature
// (DELETE /api/v1/features/{featureId})
func (a *Router) DeleteFeature(w http.ResponseWriter, r *http.Request, featureId api.FeatureId) {
	if !a.config.EntitlementsEnabled {
		unimplemented.DeleteFeature(w, r, featureId)
		return
	}
	a.featureHandler.DeleteFeature().With(featureId).ServeHTTP(w, r)
}

// Get feature
// (GET /api/v1/features/{featureId})
func (a *Router) GetFeature(w http.ResponseWriter, r *http.Request, featureId api.FeatureId) {
	if !a.config.EntitlementsEnabled {
		unimplemented.GetFeature(w, r, featureId)
		return
	}
	a.featureHandler.GetFeature().With(featureId).ServeHTTP(w, r)
}
