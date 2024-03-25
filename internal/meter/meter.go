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

package meter

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/models"
)

// Repository is an interface to the meter store.
type Repository interface {
	// ListAllMeters returns a list of meters.
	ListAllMeters(ctx context.Context) ([]models.Meter, error)

	// ListMeters returns a list of meters for the given namespace.
	ListMeters(ctx context.Context, namespace string) ([]models.Meter, error)

	// GetMeterByIDOrSlug returns a meter from the meter store by ID or slug.
	GetMeterByIDOrSlug(ctx context.Context, namespace string, idOrSlug string) (models.Meter, error)
}
