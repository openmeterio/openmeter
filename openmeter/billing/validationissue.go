package billing

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
)

type ValidationIssueSeverity string

const (
	ValidationIssueSeverityCritical ValidationIssueSeverity = "critical"
	ValidationIssueSeverityWarning  ValidationIssueSeverity = "warning"

	ValidationComponentOpenMeter         = "openmeter"
	ValidationComponentOpenMeterMetering = "openmeter.metering"
)

func (ValidationIssueSeverity) Values() []string {
	return []string{
		string(ValidationIssueSeverityCritical),
		string(ValidationIssueSeverityWarning),
	}
}

type ValidationIssue struct {
	ID        string     `json:"id,omitempty"`
	CreatedAt time.Time  `json:"createdAt,omitempty"`
	UpdatedAt time.Time  `json:"updatedAt,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`

	Severity  ValidationIssueSeverity `json:"severity"`
	Message   string                  `json:"message"`
	Code      string                  `json:"code,omitempty"`
	Component ComponentName           `json:"component,omitempty"`
	Path      string                  `json:"path,omitempty"`
}

func (i ValidationIssue) EncodeAsErrorExtension() map[string]interface{} {
	out := map[string]interface{}{
		"severity": i.Severity,
		"message":  i.Message,
	}

	if i.Component != "" {
		out["component"] = i.Component
	}

	if i.Path != "" {
		out["path"] = i.Path
	}

	if i.Code != "" {
		out["code"] = i.Code
	}

	return out
}

func (i ValidationIssue) Error() string {
	return i.Message
}

func NewValidationWarning(code, message string) ValidationIssue {
	return ValidationIssue{
		Severity: ValidationIssueSeverityWarning,
		Message:  message,
		Code:     code,
	}
}

func NewValidationError(code, message string) ValidationIssue {
	return ValidationIssue{
		Severity: ValidationIssueSeverityCritical,
		Message:  message,
		Code:     code,
	}
}

type ComponentName string

func AppTypeCapabilityToComponent(appType app.AppType, cap app.CapabilityType, op string) ComponentName {
	return ComponentName(fmt.Sprintf("app.%s.%s.%s", appType, cap, op))
}

type componentWrapper struct {
	component ComponentName
	err       error
}

func (c componentWrapper) Error() string {
	return string(c.component) + ": " + c.err.Error()
}

func (c componentWrapper) Unwrap() error {
	return c.err
}

// ValidationWithComponent wraps an error with a component name, if error is nil, it returns nil
// This can be used to add context to an error when we are crossing service boundaries.
func ValidationWithComponent(component ComponentName, err error) error {
	if err == nil {
		return nil
	}

	return componentWrapper{
		component: component,
		err:       err,
	}
}

type fieldPrefixWrapper struct {
	prefix string
	err    error
}

func (f fieldPrefixWrapper) Error() string {
	return f.prefix + ": " + f.err.Error()
}

func (f fieldPrefixWrapper) Unwrap() error {
	return f.err
}

// ValidationWithFieldPrefix wraps an error with a field prefix, if error is nil, it returns nil
// This can be used to delegate validation duties to a sub-entity. (e.g. lines don't need to know about
// the path in the invoice they are residing at)
func ValidationWithFieldPrefix(prefix string, err error) error {
	if err == nil {
		return nil
	}

	return fieldPrefixWrapper{
		prefix: prefix,
		err:    err,
	}
}

type ValidationIssues []ValidationIssue

// ToValidationIssues converts an error into a list of validation issues
// If the error is nil, it returns nil
// If any error in the error tree is not wrapped in ValidationWithComponent or ValidationWithFieldPrefix
// and not an instance of ValidationIssue, it will return an error. This behavior allows us to have
// critical errors that are not validation issues.
func ToValidationIssues(errIn error) (ValidationIssues, error) {
	if errIn == nil {
		return nil, nil
	}

	issues, err := toValidationIssue(errIn, "", "", false)
	if err != nil {
		return nil, errIn
	}

	return issues, nil
}

func (v ValidationIssues) RemoveMetaForCompare() ValidationIssues {
	return lo.Map(v, func(issue ValidationIssue, _ int) ValidationIssue {
		issue.CreatedAt = time.Time{}
		issue.UpdatedAt = time.Time{}
		issue.DeletedAt = nil
		issue.ID = ""

		return issue
	})
}

func (v ValidationIssues) Clone() ValidationIssues {
	return append(make(ValidationIssues, 0, len(v)), v...)
}

func (v ValidationIssues) AsError() error {
	if len(v) == 0 {
		return nil
	}

	return errors.Join(lo.Map(v, func(issue ValidationIssue, _ int) error {
		return issue
	})...)
}

func (v ValidationIssues) Map(f func(ValidationIssue, int) ValidationIssue) ValidationIssues {
	return lo.Map(v, f)
}

type errorsUnwrap interface {
	Unwrap() []error
}

type errorUnwrap interface {
	Unwrap() error
}

func addStartingSlashIfNeeded(path string) string {
	if path == "" {
		return ""
	}

	if path[0] == '/' {
		return path
	}

	return "/" + path
}

func appendToPrefix(prefix string, field string) string {
	if prefix == "" {
		return addStartingSlashIfNeeded(field)
	}

	if field == "" {
		return addStartingSlashIfNeeded(prefix)
	}

	return addStartingSlashIfNeeded(prefix + "/" + field)
}

func toValidationIssue(err error, fieldPrefix string, component ComponentName, unknownAsValidationIssue bool) ([]ValidationIssue, error) {
	if err == nil {
		return nil, nil
	}

	// let's see if the current error requires special handling (as switch's
	// ordering is non-deterministic, we first have a typeswitch for the special cases)
	switch errT := err.(type) {
	case componentWrapper:
		return toValidationIssue(errT.err, fieldPrefix, errT.component, true)
	case fieldPrefixWrapper:
		return toValidationIssue(errT.err, appendToPrefix(fieldPrefix, errT.prefix), component, true)
	case ValidationIssue:
		issueComponent := component
		if issueComponent == "" {
			issueComponent = errT.Component
		}

		return []ValidationIssue{
			{
				Severity:  errT.Severity,
				Message:   errT.Message,
				Code:      errT.Code,
				Path:      appendToPrefix(fieldPrefix, errT.Path),
				Component: issueComponent,
			},
		}, nil
	}

	switch errT := err.(type) {
	case errorsUnwrap:
		var issues []ValidationIssue
		for _, e := range errT.Unwrap() {
			out, err := toValidationIssue(e, fieldPrefix, component, unknownAsValidationIssue)
			if err != nil {
				return nil, err
			}
			if len(out) > 0 {
				issues = append(issues, out...)
			}
		}

		return issues, nil
	case errorUnwrap:
		return toValidationIssue(errT.Unwrap(), fieldPrefix, component, unknownAsValidationIssue)
	default:
		// Non-validation errors get coded as critical
		if unknownAsValidationIssue {
			return []ValidationIssue{
				{
					Severity:  ValidationIssueSeverityCritical,
					Message:   err.Error(),
					Path:      fieldPrefix,
					Component: component,
				},
			}, nil
		} else {
			return nil, err
		}
	}
}
