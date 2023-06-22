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

package streaming

import (
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type GetValuesParams struct {
	From       *time.Time
	To         *time.Time
	Subject    *string
	WindowSize *models.WindowSize
}

type Connector interface {
	Close() error
	Init(meter *models.Meter) error
	GetValues(meter *models.Meter, params *GetValuesParams) ([]*models.MeterValue, error)
	// Add more methods as needed ...
}
