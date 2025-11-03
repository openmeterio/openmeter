package commonhttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
)

type ErrorWithHTTPStatusCode struct {
	error
	StatusCode int
	Extensions []ExtendProblemFunc
}

func ExtendProblem(name, details string) ExtendProblemFunc {
	return func() map[string]interface{} {
		return map[string]interface{}{
			name: details,
		}
	}
}

type ExtendProblemFunc func() map[string]interface{}

func (e ExtendProblemFunc) apply(extensions map[string]interface{}) {
	for k, v := range e() {
		extensions[k] = v
	}
}

func NewHTTPError(statusCode int, err error, extensions ...ExtendProblemFunc) ErrorWithHTTPStatusCode {
	return ErrorWithHTTPStatusCode{
		StatusCode: statusCode,
		error:      err,
		Extensions: extensions,
	}
}

func (e ErrorWithHTTPStatusCode) EncodeError(ctx context.Context, w http.ResponseWriter) bool {
	problem := models.NewStatusProblem(ctx, e.error, e.StatusCode)
	for _, ext := range e.Extensions {
		ext.apply(problem.Extensions)
	}
	problem.Respond(w)
	return true
}

// ErrorEncoder encodes an error as HTTP 500 Internal Server Error.
func ErrorEncoder(ctx context.Context, _ error, w http.ResponseWriter) bool {
	models.NewStatusProblem(ctx, errors.New("something went wrong"), http.StatusInternalServerError).Respond(w)

	return false
}

// HandleErrorIfTypeMatches checks if the error is of the given type and encodes it as an HTTP error.
// Using the generic feature we can mandate that the error implements the error interface. This is a
// must, as the errors.As would panic if the error does not implement the error interface.
func HandleErrorIfTypeMatches[T error](ctx context.Context, statusCode int, err error, w http.ResponseWriter, extendedProblemFunc ...func(T) map[string]interface{}) bool {
	if err, ok := lo.ErrorsAs[T](err); ok {
		extendedProblemFuncs := make([]ExtendProblemFunc, 0, len(extendedProblemFunc))
		for _, f := range extendedProblemFunc {
			extendedProblemFuncs = append(extendedProblemFuncs, func() map[string]interface{} {
				return f(err)
			})
		}
		NewHTTPError(statusCode, err, extendedProblemFuncs...).EncodeError(ctx, w)
		return true
	}

	return false
}

type httpAttributeKey string

const httpStatusCodeErrorAttribute httpAttributeKey = "openmeter.http.status_code"

func WithHTTPStatusCodeAttribute(code int) models.ValidationIssueOption {
	return models.WithAttribute(httpStatusCodeErrorAttribute, code)
}

// Options for the Handler methods
type handleIssueIfHTTPStatusKnownOptions struct {
	statusPriorizationBehavior HTTPStatusAttributePriorizationBehavior
}

type HTTPStatusAttributePriorizationBehavior string

const (
	// In case there are multiple status codes, the errors won't be mapped. This is the default and recommended behavior
	HTTPStatusAttributePriorizationBehaviorSingular HTTPStatusAttributePriorizationBehavior = "singular"

	// This below options could be added but I fear they'd only cause more confusion
	//
	// // In case of conflicts, the status of the first error will be used
	// HTTPStatusAttributePriorizationBehaviorFirst HTTPStatusAttributePriorizationBehavior = "first"
	// // In case of conflicts, if there's an auth-authz error, it will be used, otherwise the status of the first error will be used
	// HTTPStatusAttributePriorizationBehaviorAuthAuthz HTTPStatusAttributePriorizationBehavior = "auth-authz"
)

type HandleIssueIfHTTPStatusKnownOptions func(*handleIssueIfHTTPStatusKnownOptions)

func WithHTTPStatusAttributePriorizationBehavior(behavior HTTPStatusAttributePriorizationBehavior) HandleIssueIfHTTPStatusKnownOptions {
	return func(o *handleIssueIfHTTPStatusKnownOptions) {
		o.statusPriorizationBehavior = behavior
	}
}

func HandleIssueIfHTTPStatusKnown(ctx context.Context, err error, w http.ResponseWriter, options ...HandleIssueIfHTTPStatusKnownOptions) bool {
	opts := &handleIssueIfHTTPStatusKnownOptions{
		statusPriorizationBehavior: HTTPStatusAttributePriorizationBehaviorSingular,
	}
	for _, opt := range options {
		opt(opts)
	}

	issues, err := models.AsValidationIssues(err)
	if err != nil {
		return false
	}

	if len(issues) == 0 {
		return false
	}

	issuesByCodeMap := make(map[int]models.ValidationIssues)

	for _, issue := range issues {
		code, ok := issue.Attributes()[httpStatusCodeErrorAttribute]
		if !ok {
			continue
		}

		issueCode, ok := code.(int)
		if !ok {
			slog.Default().DebugContext(ctx, "issue does have HTTP status code attribute but it's not an integer", "issue", issue)
			continue
		}

		issues := issuesByCodeMap[issueCode]
		issues = append(issues, issue)
		issuesByCodeMap[issueCode] = issues
	}

	if len(issuesByCodeMap) == 0 {
		return false
	}

	extendProblemFuncs := make([]ExtendProblemFunc, 0)
	responseStatusCode := 500 // default to internal server error

	switch opts.statusPriorizationBehavior {
	case HTTPStatusAttributePriorizationBehaviorSingular:
		if len(issuesByCodeMap) > 1 {
			return false
		}

		for code, issues := range issuesByCodeMap {
			responseStatusCode = code
			extendProblemFuncs = append(extendProblemFuncs, func() map[string]interface{} {
				return map[string]interface{}{
					// FIXME[galexi,chrisgacsal]: having everything under "validationErrors" makes no sense but we need it for backwards compatibility, otherwise its just hacky...
					// should migrate to more generic form like "errors"
					"validationErrors": lo.Map(issues, func(issue models.ValidationIssue, _ int) map[string]interface{} {
						// We don't want to expose private attributes to the client
						attrs := issue.Attributes()
						delete(attrs, httpStatusCodeErrorAttribute)
						delete(attrs, issue.Code())
						issue = issue.SetAttributes(attrs)

						return issue.AsErrorExtension()
					}),
				}
			})
		}
	default:
		slog.Default().ErrorContext(ctx, "Unknown HTTP status attribute priorization behavior, passing to next error handler", "behavior", opts.statusPriorizationBehavior)
		return false
	}

	// The returned error message will be a joined error(ValidationIssues.AsError().Error()), not sure if this is the best approach
	NewHTTPError(responseStatusCode, err, extendProblemFuncs...).EncodeError(ctx, w)

	return true
}
