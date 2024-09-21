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
	"github.com/ThreeDotsLabs/watermill/message"
)

// RestoreContext ensures that the original context is restored after the handler is done processing the message.
//
// This helps with https://github.com/ThreeDotsLabs/watermill/issues/467
func RestoreContext(h message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		origCtx := msg.Context()
		defer func() {
			msg.SetContext(origCtx)
		}()

		return h(msg)
	}
}
