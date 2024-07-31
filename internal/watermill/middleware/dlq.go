package middleware

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/openmeterio/openmeter/internal/watermill"
	"github.com/pkg/errors"
)

const (
	DLQErrorReasonKey   = "reason_dlq"
	DLQOriginalTopicKey = "original_topic"
)

// DLQMsgProcessorFunc determines whether any messages should be published to the DLQ based on the original message and the error.
// Even if DLQMsgProcessorFunc returns a non-nil slice of messages, if error is nil no messages will be published to the DLQ.
type DLQMsgProcessorFunc func(original *message.Message, err error) []*message.Message

// defaultProcessor listens for RetryableErrors and publishes the messages contained in that
var defaultProcessor DLQMsgProcessorFunc = func(original *message.Message, err error) []*message.Message {
	if retryable, ok := err.(watermill.RetryableError); ok {
		return retryable.RetryMessages()
	}

	return nil
}

type deadLetterQueue struct {
	topic string
	pub   message.Publisher

	msgProcessor DLQMsgProcessorFunc
}

// DLQ provides a middleware that salvages unprocessable messages and published them on a separate topic.
// The main middleware chain then continues on, business as usual.
func DLQ(pub message.Publisher, topic string, msgProcessor DLQMsgProcessorFunc) (message.HandlerMiddleware, error) {
	if topic == "" {
		return nil, fmt.Errorf("invalid DLQ topic")
	}

	if msgProcessor == nil {
		msgProcessor = defaultProcessor
	}

	dlq := deadLetterQueue{
		topic:        topic,
		pub:          pub,
		msgProcessor: msgProcessor,
	}

	return dlq.Middleware, nil
}

func (dlq deadLetterQueue) publishWithErr(msg *message.Message, err error) error {
	// Sanity check, we couldn't annotate the metadata without the error
	if err == nil {
		return nil
	}

	// Add context why it was poisoned
	msg.Metadata.Set(DLQErrorReasonKey, err.Error())
	msg.Metadata.Set(DLQOriginalTopicKey, message.SubscribeTopicFromCtx(msg.Context()))

	// Don't intercept error from publish. Can't help you if the publisher is down as well.
	return dlq.pub.Publish(dlq.topic, msg)
}

func (dlq deadLetterQueue) Middleware(h message.HandlerFunc) message.HandlerFunc {
	// Capture return values from the handler
	return func(msg *message.Message) (events []*message.Message, err error) {
		defer func() {
			msgs := dlq.msgProcessor(msg, err)

			// Attempting to publish all messages to the DLQ
			dlqSuccess := len(msgs) > 0
			for _, m := range msgs {
				if publishErr := dlq.publishWithErr(m, err); publishErr != nil {
					dlqSuccess = false
					publishErr = errors.Wrap(publishErr, "cannot publish message to poison queue")
					err = multierror.Append(err, publishErr)
				}
			}

			// If all messages were successfully published then we no longer have an error
			if dlqSuccess {
				err = nil
			}
		}()

		return h(msg)
	}
}
