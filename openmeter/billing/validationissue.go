package billing

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ValidationIssueSeverity string

const (
	ValidationIssueSeverityCritical ValidationIssueSeverity = "critical"
	ValidationIssueSeverityWarning  ValidationIssueSeverity = "warning"

	ValidationComponentOpenMeter         = "openmeter"
	ValidationComponentOpenMeterMetering = "openmeter.metering"

	ValidationIssueCodeLineEngineCollectionCompletedFailed = "line_engine_collection_completed_failed"
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
	// TODO: migrate billing validation issues to models.ValidationIssue's FieldDescriptor-based path.
	// Deprecated: this field should be moved to models.ValidationIssue's FieldDescriptor-based path implementation.
	Path       string             `json:"path,omitempty"`
	Attributes models.Annotations `json:"attributes,omitempty"`
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

	if len(i.Attributes) > 0 {
		out["attributes"] = i.Attributes
	}

	return out
}

func (i ValidationIssue) Clone() (ValidationIssue, error) {
	clone := i

	attributes, err := i.Attributes.Clone()
	if err != nil {
		return ValidationIssue{}, fmt.Errorf("cloning attributes: %w", err)
	}

	clone.Attributes = attributes

	return clone, nil
}

func (i ValidationIssue) Error() string {
	return i.Message
}

// Is identifies coded validation issues independently of their contextual details.
func (i ValidationIssue) Is(target error) bool {
	other, ok := target.(ValidationIssue)

	return ok && i.Code != "" && i.Code == other.Code
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

// ValidationWithComponent wraps an error with a component name, if error is nil, it returns nil.
// This can be used to add context to an error when we are crossing service boundaries. When
// wrappers are nested, the outermost service boundary defines the extracted issue's component.
func ValidationWithComponent(component ComponentName, err error) error {
	if err == nil {
		return nil
	}

	return componentWrapper{
		component: component,
		err:       err,
	}
}

type messageWrapper struct {
	prefix string
	err    error
}

func (m messageWrapper) Error() string {
	return m.prefix + ": " + m.err.Error()
}

func (m messageWrapper) Unwrap() error {
	return m.err
}

// ValidationWithMessagef wraps an error with formatted context, if error is nil, it returns nil.
// Unlike component and field wrappers, message context does not classify an ordinary error as a
// validation issue. When the wrapped error contains validation issues, the context is added to each
// extracted issue's message.
func ValidationWithMessagef(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}

	return messageWrapper{
		prefix: fmt.Sprintf(format, args...),
		err:    err,
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

type attributesWrapper struct {
	attributes models.Annotations
	err        error
}

func (a attributesWrapper) Error() string {
	return a.err.Error()
}

func (a attributesWrapper) Unwrap() error {
	return a.err
}

// ValidationWithAttributes adds contextual attributes to validation issues extracted from err.
// When wrappers are nested, attributes from the outermost wrapper take precedence.
func ValidationWithAttributes(attributes models.Annotations, err error) error {
	if err == nil {
		return nil
	}

	return attributesWrapper{
		attributes: attributes,
		err:        err,
	}
}

type ValidationIssues []ValidationIssue

// ToValidationIssues converts an error into a list of validation issues
// If the error is nil, it returns nil
// If any error in the error tree is not wrapped in ValidationWithComponent,
// ValidationWithFieldPrefix, or ValidationWithAttributes and not an instance of ValidationIssue,
// it will return an error. This behavior allows us to have critical errors that are not validation issues.
func ToValidationIssues(errIn error) (ValidationIssues, error) {
	if errIn == nil {
		return nil, nil
	}

	issues, err := toValidationIssue(errIn, "", "", "", nil, false)
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

func (v ValidationIssues) Clone() (ValidationIssues, error) {
	if v == nil {
		return nil, nil
	}

	return lo.MapErr(v, func(issue ValidationIssue, _ int) (ValidationIssue, error) {
		return issue.Clone()
	})
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

func appendMessagePrefix(prefix string, message string) string {
	if prefix == "" {
		return message
	}

	return prefix + ": " + message
}

func toValidationIssue(err error, fieldPrefix string, component ComponentName, messagePrefix string, attributes models.Annotations, unknownAsValidationIssue bool) ([]ValidationIssue, error) {
	if err == nil {
		return nil, nil
	}

	// let's see if the current error requires special handling (as switch's
	// ordering is non-deterministic, we first have a typeswitch for the special cases)
	switch errT := err.(type) {
	case componentWrapper:
		issueComponent := component
		if issueComponent == "" {
			issueComponent = errT.component
		}

		return toValidationIssue(errT.err, fieldPrefix, issueComponent, messagePrefix, attributes, true)
	case fieldPrefixWrapper:
		return toValidationIssue(errT.err, appendToPrefix(fieldPrefix, errT.prefix), component, messagePrefix, attributes, true)
	case messageWrapper:
		return toValidationIssue(errT.err, fieldPrefix, component, appendMessagePrefix(messagePrefix, errT.prefix), attributes, unknownAsValidationIssue)
	case attributesWrapper:
		mergedAttributes, err := errT.attributes.Merge(attributes)
		if err != nil {
			return nil, fmt.Errorf("merging validation issue attributes: %w", err)
		}

		return toValidationIssue(errT.err, fieldPrefix, component, messagePrefix, mergedAttributes, true)
	case ValidationIssue:
		issueComponent := component
		if issueComponent == "" {
			issueComponent = errT.Component
		}

		mergedAttributes, err := errT.Attributes.Merge(attributes)
		if err != nil {
			return nil, fmt.Errorf("merging validation issue attributes: %w", err)
		}

		mergedAttributes, err = mergedAttributes.Clone()
		if err != nil {
			return nil, fmt.Errorf("cloning validation issue attributes: %w", err)
		}

		return []ValidationIssue{
			{
				Severity:   errT.Severity,
				Message:    appendMessagePrefix(messagePrefix, errT.Message),
				Code:       errT.Code,
				Path:       appendToPrefix(fieldPrefix, errT.Path),
				Component:  issueComponent,
				Attributes: mergedAttributes,
			},
		}, nil
	}

	switch errT := err.(type) {
	case errorsUnwrap:
		var issues []ValidationIssue
		for _, e := range errT.Unwrap() {
			out, err := toValidationIssue(e, fieldPrefix, component, messagePrefix, attributes, unknownAsValidationIssue)
			if err != nil {
				return nil, err
			}
			if len(out) > 0 {
				issues = append(issues, out...)
			}
		}

		return issues, nil
	case errorUnwrap:
		return toValidationIssue(errT.Unwrap(), fieldPrefix, component, messagePrefix, attributes, unknownAsValidationIssue)
	default:
		// Non-validation errors get coded as critical
		if unknownAsValidationIssue {
			issueAttributes, cloneErr := attributes.Clone()
			if cloneErr != nil {
				return nil, fmt.Errorf("cloning validation issue attributes: %w", cloneErr)
			}

			return []ValidationIssue{
				{
					Severity:   ValidationIssueSeverityCritical,
					Message:    appendMessagePrefix(messagePrefix, err.Error()),
					Path:       fieldPrefix,
					Component:  component,
					Attributes: issueAttributes,
				},
			}, nil
		} else {
			return nil, err
		}
	}
}
