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

package nopublisher

import (
	"errors"

	"github.com/ThreeDotsLabs/watermill/message"
)

var ErrMessagesProduced = errors.New("messages produced by no publisher handler")

func NoPublisherHandlerToHandlerFunc(h message.NoPublishHandlerFunc) message.HandlerFunc {
	return func(message *message.Message) ([]*message.Message, error) {
		return nil, h(message)
	}
}

func HandlerFuncToNoPublisherHandler(h message.HandlerFunc) message.NoPublishHandlerFunc {
	return func(message *message.Message) error {
		outMessages, err := h(message)
		if len(outMessages) > 0 {
			return ErrMessagesProduced
		}
		return err
	}
}
