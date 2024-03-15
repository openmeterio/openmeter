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

package main

import (
	"context"

	_ "github.com/benthosdev/benthos/v4/public/components/all"  // import all benthos components
	_ "github.com/benthosdev/benthos/v4/public/components/io"   // import io benthos components
	_ "github.com/benthosdev/benthos/v4/public/components/pure" // import pure benthos components
	"github.com/benthosdev/benthos/v4/public/service"

	_ "github.com/openmeterio/openmeter/collector/benthos/input"  // import input plugins
	_ "github.com/openmeterio/openmeter/collector/benthos/output" // import output plugins
)

func main() {
	service.RunCLI(context.Background())
}
