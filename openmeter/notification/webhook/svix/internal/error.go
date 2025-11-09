package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/samber/lo"
	svix "github.com/svix/svix-webhooks/go"

	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
)

var _ error = (*SvixError)(nil)

type SvixError struct {
	HTTPStatus int
	Code       *string
	Details    []string

	RetryAfter time.Duration
}

func (e SvixError) Error() string {
	var out []byte
	buf := bytes.NewBuffer(out)

	buf.WriteString(lo.FromPtrOr(e.Code, "unknown svix error"))
	buf.WriteString(": ")
	buf.WriteString(strings.Join(e.Details, ", "))

	return buf.String()
}

func (e SvixError) Wrap() error {
	switch e.HTTPStatus {
	case http.StatusBadRequest, http.StatusConflict, http.StatusUnprocessableEntity:
		return webhook.NewUnrecoverableError(webhook.NewValidationError(e))
	case http.StatusNotFound:
		return webhook.NewNotFoundError(e)
	case http.StatusTooManyRequests:
		return webhook.NewRetryableError(e, e.RetryAfter)
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return webhook.NewRetryableError(e, e.RetryAfter)
	default:
		return e
	}
}

type SvixErrorBody struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

type SvixValidationErrorBody struct {
	Detail []SvixValidationError `json:"detail"`
}

type SvixValidationError struct {
	Loc     []string `json:"loc"`
	Message string   `json:"msg"`
	Type    string   `json:"type"`
}

func WrapSvixError(err error) error {
	if err == nil {
		return nil
	}

	svixErr, ok := lo.ErrorsAs[*svix.Error](err)
	if !ok {
		return err
	}

	switch svixErr.Status() {
	case http.StatusUnprocessableEntity:
		var body SvixValidationErrorBody

		if e := json.Unmarshal(svixErr.Body(), &body); e != nil {
			return fmt.Errorf("failed to parse Svix error response: %w", err)
		}

		return SvixError{
			HTTPStatus: svixErr.Status(),
			Code:       lo.ToPtr("validation_error"),
			Details: lo.Map(body.Detail, func(item SvixValidationError, _ int) string {
				loc := strings.Join(item.Loc, ".")

				return fmt.Sprintf("[location=%s type=%s]: %s", loc, item.Type, item.Message)
			}),
		}.Wrap()
	default:
		var body SvixErrorBody

		if e := json.Unmarshal(svixErr.Body(), &body); e != nil {
			return fmt.Errorf("failed to parse Svix error response: %w", err)
		}

		return SvixError{
			HTTPStatus: svixErr.Status(),
			Code:       &body.Code,
			Details: []string{
				body.Detail,
			},
		}.Wrap()
	}
}
