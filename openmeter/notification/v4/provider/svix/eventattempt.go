package svix

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/samber/lo"
	svixv2 "github.com/svix/svix-webhooks/v2/go"
	svixmodels "github.com/svix/svix-webhooks/v2/go/models"

	notificationprovider "github.com/openmeterio/openmeter/openmeter/notification/v4/provider"
	"github.com/openmeterio/openmeter/pkg/models"
)

type listEventAttemptsInput struct {
	Namespace  string
	MessageID  string
	EndpointID string
}

var _ models.Validator = listEventAttemptsInput{}

func (i listEventAttemptsInput) Validate() error {
	var errs []error

	if err := (notificationprovider.GetChannelInput{Namespace: i.Namespace, ChannelID: i.EndpointID}).Validate(); err != nil {
		errs = append(errs, err)
	}

	if strings.TrimSpace(i.MessageID) == "" {
		errs = append(errs, errors.New("invalid message ID: required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (p *Provider) listEventAttempts(ctx context.Context, input listEventAttemptsInput) ([]svixmodels.MessageAttemptOut, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf(
			"invalid list message attempts input [svix.message.id=%s svix.endpoint.id=%s svix.application.uid=%s]: %w",
			input.MessageID,
			input.EndpointID,
			input.Namespace,
			err,
		)
	}

	options := &svixv2.MessageAttemptListByMsgOptions{
		Limit:            new(eventPageSize),
		EndpointId:       &input.EndpointID,
		WithContent:      new(true),
		ExpandedStatuses: new(true),
	}

	attempts := make([]svixmodels.MessageAttemptOut, 0)
	seenCursors := make(map[string]struct{})
	seenAttempts := make(map[string]struct{})

	for {
		page, err := p.client.MessageAttempt().ListByMsg(ctx, input.Namespace, input.MessageID, options)
		if err != nil {
			return nil, err
		}

		if page == nil {
			return nil, fmt.Errorf(
				"invalid message attempts response [svix.message.id=%s svix.endpoint.id=%s svix.application.uid=%s]: nil response",
				input.MessageID,
				input.EndpointID,
				input.Namespace,
			)
		}

		for _, attempt := range page.Data {
			var responseErr error

			switch {
			case attempt.Id == "":
				responseErr = errors.New("missing attempt ID")

			case attempt.EndpointId != input.EndpointID:
				responseErr = fmt.Errorf("unexpected endpoint ID: %s", attempt.EndpointId)

			case attempt.MsgId != input.MessageID:
				responseErr = fmt.Errorf("unexpected message ID: %s", attempt.MsgId)
			}

			if responseErr != nil {
				return nil, fmt.Errorf(
					"invalid message attempt [svix.attempt.id=%s svix.message.id=%s svix.endpoint.id=%s svix.application.uid=%s]: %w",
					attempt.Id,
					input.MessageID,
					input.EndpointID,
					input.Namespace,
					responseErr,
				)
			}

			if _, exists := seenAttempts[attempt.Id]; !exists {
				attempts = append(attempts, attempt)
				seenAttempts[attempt.Id] = struct{}{}
			}
		}

		if page.Done {
			return attempts, nil
		}

		cursor := lo.FromPtr(page.Iterator)
		if _, exists := seenCursors[cursor]; cursor == "" || exists {
			return nil, fmt.Errorf(
				"invalid message attempts pagination [svix.message.id=%s svix.endpoint.id=%s svix.application.uid=%s]: pagination did not advance",
				input.MessageID,
				input.EndpointID,
				input.Namespace,
			)
		}

		seenCursors[cursor] = struct{}{}
		options.Iterator = page.Iterator
	}
}
