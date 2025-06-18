package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/samber/lo"
	svix "github.com/svix/svix-webhooks/go"
)

var _ error = (*SvixError)(nil)

type SvixError struct {
	HTTPStatus int
	Code       *string
	Details    []string
}

func (e SvixError) Error() string {
	var out []byte
	buf := bytes.NewBuffer(out)

	buf.WriteString(lo.FromPtrOr(e.Code, "unknown svix error"))
	buf.WriteString(":")
	buf.WriteString(strings.Join(e.Details, ", "))

	return buf.String()
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

const HTTPStatusValidationError = 422

func AsSvixError(err error) error {
	if err == nil {
		return nil
	}

	svixErr, ok := lo.ErrorsAs[*svix.Error](err)
	if !ok {
		return err
	}

	switch svixErr.Status() {
	case HTTPStatusValidationError:
		var body SvixValidationErrorBody

		if e := json.Unmarshal(svixErr.Body(), &body); e != nil {
			return err
		}

		return SvixError{
			HTTPStatus: svixErr.Status(),
			Details: lo.Map(body.Detail, func(item SvixValidationError, _ int) string {
				return fmt.Sprintf("%s: %s", item.Type, item.Message)
			}),
		}
	default:
		var body SvixErrorBody

		if e := json.Unmarshal(svixErr.Body(), &body); e != nil {
			return err
		}

		return SvixError{
			HTTPStatus: svixErr.Status(),
			Code:       &body.Code,
			Details: []string{
				body.Detail,
			},
		}
	}
}
